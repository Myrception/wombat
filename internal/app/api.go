package app

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	goruntime "runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"github.com/mitchellh/mapstructure"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/runtime/protoiface"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	defaultStateKey          = "state_default"
	defaultWorkspaceKey      = "wksp_default"
	workspacePrefix          = "wksp_"
	metadataKeyPrefix        = "md_"
	reflectMetadataKeyPrefix = "rmd_"
	messageKeyPrefix         = "msg_"
)

type api struct {
	ctx              context.Context
	client           *client
	store            *store
	protofiles       *protoregistry.Files
	streamReq        chan proto.Message
	cancelMonitoring context.CancelFunc
	cancelInFlight   context.CancelFunc
	mu               sync.Mutex // protect in-flight requests
	inFlight         bool
	appData          string
	state            *workspaceState
}

type statsHandler struct {
	*api
}

type storeLogger struct {
	ctx     context.Context
	log     *slog.Logger
	logFunc func(string, ...interface{})
}

// Create a new storeLogger
func newStoreLogger(ctx context.Context) *storeLogger {
	return &storeLogger{
		ctx: ctx,
		log: slog.Default(),
	}
}

// Debugf implements badger.Logger interface
func (s *storeLogger) Debugf(format string, args ...interface{}) {
	// Use slog's Debug level
	s.log.Debug(format, args...)
}

// Infof implements badger.Logger interface
func (s *storeLogger) Infof(format string, args ...interface{}) {
	// Use slog's Info level
	s.log.Info(format, args...)
}

// Warningf implements badger.Logger interface
func (s *storeLogger) Warningf(format string, args ...interface{}) {
	// Use slog's Warn level and also send to Wails runtime
	s.log.Warn(format, args...)
}

// Errorf implements badger.Logger interface
func (s *storeLogger) Errorf(format string, args ...interface{}) {
	// Use slog's Error level and also send to Wails runtime
	s.log.Error(format, args...)
}

func NewApp() *api {
	return &api{}
}

// Startup is the initialization function for the Wails v2 runtime
func (a *api) Startup(ctx context.Context) {
	a.ctx = ctx

	var err error
	a.store, err = newStore(a.appData, newStoreLogger(ctx))
	if err != nil {
		runtime.LogError(ctx,
			fmt.Errorf("app: failed to create database: %v", err).Error())
		runtime.LogInfof(ctx, "appData: %s", a.appData)
	}
	a.state = a.getCurrentState()

	opts, err := a.GetWorkspaceOptions()
	if err != nil {
		runtime.LogError(ctx, err.Error())
	}
	hds, err := a.GetReflectMetadata(opts.Addr)
	if err != nil {
		runtime.LogError(ctx, err.Error())
	}

	if err := a.Connect(opts, hds, false); err != nil {
		runtime.LogError(ctx, err.Error())
	}

	go a.checkForUpdate()
}

func (a *api) checkForUpdate() {
	r, err := checkForUpdate()
	if err != nil {
		if err == noUpdate {
			runtime.LogInfo(a.ctx, err.Error())
			return
		}
		runtime.LogWarning(a.ctx, fmt.Sprintf("failed to check for updates: %v", err))
		return
	}
	runtime.EventsEmit(a.ctx, eventUpdateAvailable, r)
}

// Shutdown is called when the application is closing
func (a *api) Shutdown(ctx context.Context) {
	a.store.close()
	if a.cancelMonitoring != nil {
		a.cancelMonitoring()
	}
	if a.cancelInFlight != nil {
		a.cancelInFlight()
	}
	if a.client != nil {
		a.client.close()
	}
}

func (a *api) emitError(title, msg string) {
	runtime.EventsEmit(a.ctx, eventError, errorMsg{title, msg})
}

func (a *api) getCurrentState() *workspaceState {
	rtn := &workspaceState{
		CurrentID: defaultWorkspaceKey,
	}
	val, err := a.store.get([]byte(defaultStateKey))
	if err != nil && err != errKeyNotFound {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to get current state from store: %v", err))
	}
	if len(val) == 0 {
		return rtn
	}
	dec := gob.NewDecoder(bytes.NewBuffer(val))
	if err := dec.Decode(rtn); err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to decode state: %v", err))
	}
	return rtn
}

// GetWorkspaceOptions gets the workspace options from the store
func (a *api) GetWorkspaceOptions() (*options, error) {
	wo := &options{
		ID: a.state.CurrentID,
	}

	val, err := a.store.get([]byte(wo.ID))
	if err != nil && err != errKeyNotFound {
		return nil, err
	}

	if len(val) == 0 {
		return wo, nil
	}

	dec := gob.NewDecoder(bytes.NewBuffer(val))
	err = dec.Decode(wo)
	if err != nil {
		return nil, err
	}

	if wo.ID == "" {
		wo.ID = defaultWorkspaceKey
	}

	return wo, nil
}

// WailsShutdown is the shutdown function that is called when wails shuts down
func (a *api) WailsShutdown() {
	a.store.close()
	if a.cancelMonitoring != nil {
		a.cancelMonitoring()
	}
	if a.cancelInFlight != nil {
		a.cancelInFlight()
	}
	if a.client != nil {
		a.client.close()
	}
}

// GetReflectMetadata gets the reflection metadata from the store by addr
func (a *api) GetReflectMetadata(addr string) (headers, error) {
	val, err := a.store.get([]byte(reflectMetadataKeyPrefix + hash(addr)))
	if err != nil {
		return nil, err
	}
	var hds headers
	dec := gob.NewDecoder(bytes.NewBuffer(val))
	err = dec.Decode(&hds)

	return hds, err
}

// GetMetadata gets the metadata from the store by addr
func (a *api) GetMetadata(addr string) (headers, error) {
	val, err := a.store.get([]byte(metadataKeyPrefix + hash(addr)))
	if err != nil {
		return nil, err
	}
	var hds headers
	dec := gob.NewDecoder(bytes.NewBuffer(val))
	err = dec.Decode(&hds)

	return hds, err
}

// ListWorkspaces returns a list of workspaces as their options
func (a *api) ListWorkspaces() ([]options, error) {
	items, err := a.store.list([]byte(workspacePrefix))
	if err != nil {
		return nil, err
	}
	var opts []options
	hasDefault := false
	for _, val := range items {
		opt := options{}
		dec := gob.NewDecoder(bytes.NewBuffer(val))
		if err := dec.Decode(&opt); err != nil {
			return opts, err
		}
		// opts.ID was added in v0.3.0 so need to double check
		if opt.ID == defaultWorkspaceKey || opt.ID == "" {
			hasDefault = true
			opt.ID = defaultWorkspaceKey
			opts = append([]options{opt}, opts...)
			continue
		}
		opts = append(opts, opt)
	}
	if !hasDefault {
		opts = append([]options{{ID: defaultWorkspaceKey}}, opts...)
	}
	return opts, nil
}

// SelectWorkspace changes the current workspace by ID
func (a *api) SelectWorkspace(id string) (rerr error) {
	if a.state.CurrentID == id {
		return nil
	}

	defer func() {
		if rerr != nil {
			runtime.LogError(a.ctx, rerr.Error())
			a.emitError("Workspace Error", rerr.Error())
		}
	}()

	if a.client != nil {
		if a.cancelMonitoring != nil {
			a.cancelMonitoring()
			time.Sleep(100 * time.Millisecond)
		}
		a.client.close()
		a.client = nil
	}

	a.changeWorkspace(id)
	opts, err := a.GetWorkspaceOptions()
	if err != nil {
		return err
	}

	hds, err := a.GetReflectMetadata(opts.Addr)
	if err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("failed to get reflection metadata: %v", err))
	}

	// Ignoring error as Connect will already emit errors to the frontend
	a.Connect(opts, hds, false)

	return nil
}

// DeleteWorkspace will remove a workspace from the store and switch to
// the default workspace, if the deleted workspace is current.
func (a *api) DeleteWorkspace(id string) error {
	a.store.del([]byte(id))
	if a.state.CurrentID == id {
		a.SelectWorkspace(defaultWorkspaceKey)
	}
	return nil
}

// GetRawMessageState gets the message state by method full name
func (a *api) GetRawMessageState(method string) (string, error) {
	opts, err := a.GetWorkspaceOptions()
	if err != nil {
		return "", fmt.Errorf("failed to get message state, no workspace options: %v", err)
	}

	val, err := a.store.get([]byte(messageKeyPrefix + hash(opts.Addr, method)))
	return string(val), err
}

// FindProtoFiles opens a directory dialog to search for proto files
func (a *api) FindProtoFiles() (files []string, rerr error) {
	defer func() {
		if rerr != nil {
			const errTitle = "Not found"
			runtime.LogError(a.ctx, rerr.Error())
			a.emitError(errTitle, rerr.Error())
		}
	}()

	dir, err := a.SelectDirectory()
	if err != nil {
		const errTitle = "Not found"
		runtime.LogError(a.ctx, err.Error())
		a.emitError(errTitle, err.Error())
	}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".proto" {
			files = append(files, path)
		}
		return nil
	})

	if len(files) == 0 {
		return nil, errors.New("no *.proto files found")
	}

	return files, nil
}

// SelectDirectory opens a directory dialog and returns the path of the selected directory
func (a *api) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Proto Files Directory",
	})
}

// Connect will attempt to connect a grpc server and parse any proto files
func (a *api) Connect(data, rawHeaders interface{}, save bool) (rerr error) {
	defer func() {
		if rerr != nil {
			const errTitle = "Connection error"
			runtime.LogError(a.ctx, rerr.Error())
			runtime.EventsEmit(a.ctx, eventClientStateChanged, connectivity.Shutdown.String())
			a.emitError(errTitle, rerr.Error())
		}
	}()

	var opts options
	if err := mapstructure.Decode(data, &opts); err != nil {
		return err
	}

	// reset all things
	runtime.EventsEmit(a.ctx, eventClientConnectStarted, opts.Addr)
	runtime.EventsEmit(a.ctx, eventServicesSelectChanged)
	runtime.EventsEmit(a.ctx, eventMethodInputChanged)

	if a.client != nil {
		if err := a.client.close(); err != nil {
			return fmt.Errorf("failed to close previous connection: %v", err)
		}
	}
	a.client = &client{}

	if a.cancelMonitoring != nil {
		a.cancelMonitoring()
	}
	ctx := context.Background()
	ctx, a.cancelMonitoring = context.WithCancel(ctx)
	go a.monitorStateChanges(ctx)

	var hds headers
	if err := mapstructure.Decode(rawHeaders, &hds); err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("unable to decode reflection metadata headers: %v", err))
	}

	if err := a.client.connect(opts, statsHandler{a}); err != nil {
		// Still try to parse proto definitions. Will fail silently
		// if using reflection services as there is no connection
		// to a valid server.
		a.cancelMonitoring()
		a.client = nil
		go a.loadProtoFiles(opts, hds, true)

		return fmt.Errorf("failed to connect to server: %v", err)
	}

	runtime.EventsEmit(a.ctx, eventClientConnected, opts.Addr)

	go a.loadProtoFiles(opts, hds, false)

	if !save {
		return nil
	}

	if opts.ID == "" {
		id := uuid.Must(uuid.NewV4())
		opts.ID = workspacePrefix + id.String()
		a.changeWorkspace(opts.ID)
	}

	go a.setWorkspaceOptions(opts)
	go a.setMetadata(reflectMetadataKeyPrefix+hash(opts.Addr), hds)

	return nil
}

func (a *api) changeWorkspace(id string) {
	a.state.CurrentID = id
	var val bytes.Buffer
	enc := gob.NewEncoder(&val)
	enc.Encode(a.state)

	a.store.set([]byte(defaultStateKey), val.Bytes())
}

func (a *api) loadProtoFiles(opts options, reflectHeaders headers, silent bool) (rerr error) {
	defer func() {
		if rerr != nil {
			const errTitle = "Failed to load RPC schema"
			runtime.LogError(a.ctx, rerr.Error())
			if !silent {
				runtime.EventsEmit(a.ctx, eventError, errorMsg{errTitle, rerr.Error()})
			}
		}
	}()

	a.protofiles = nil

	var err error
	if opts.Reflect {
		if a.client == nil {
			return errors.New("unable to load proto files via reflection: client is <nil>")
		}
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.New(nil))
		for _, h := range reflectHeaders {
			if h.Key == "" {
				continue
			}
			ctx = metadata.AppendToOutgoingContext(ctx, h.Key, h.Val)
			fmt.Printf("h.Val = %+v\n", h.Val)
		}

		ctx = context.WithValue(ctx, ctxInternalKey{}, struct{}{})
		if a.protofiles, err = protoFilesFromReflectionAPI(ctx, a.client.conn); err != nil {
			return fmt.Errorf("error getting proto files from reflection API: %v", err)
		}
	}
	if !opts.Reflect && len(opts.Protos.Files) > 0 {
		if a.protofiles, err = protoFilesFromDisk(opts.Protos.Roots, opts.Protos.Files); err != nil {
			return fmt.Errorf("error parsing proto files from disk: %v", err)
		}
	}

	return a.emitServicesSelect("", "", nil)
}

func (a *api) emitServicesSelect(method string, data string, metadata headers) error {
	if a.protofiles == nil {
		return nil
	}

	var targetMd protoreflect.MethodDescriptor
	var ss servicesSelect
	a.protofiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		sds := fd.Services()
		for i := 0; i < sds.Len(); i++ {
			var s serviceSelect
			sd := sds.Get(i)
			s.FullName = string(sd.FullName())

			mds := sd.Methods()
			for j := 0; j < mds.Len(); j++ {
				md := mds.Get(j)
				fname := fmt.Sprintf("/%s/%s", sd.FullName(), md.Name())
				if fname == method {
					targetMd = md
				}
				s.Methods = append(s.Methods, methodSelect{
					Name:         string(md.Name()),
					FullName:     fname,
					ClientStream: md.IsStreamingClient(),
					ServerStream: md.IsStreamingServer(),
				})
			}
			sort.SliceStable(s.Methods, func(i, j int) bool {
				return s.Methods[i].Name < s.Methods[j].Name
			})
			ss = append(ss, s)
		}
		return true
	})

	if len(ss) == 0 {
		return nil
	}

	sort.SliceStable(ss, func(i, j int) bool {
		return ss[i].FullName < ss[j].FullName
	})

	if method != "" && targetMd == nil {
		return fmt.Errorf("method %q not found", method)
	}

	// Use Wails v2 EventsEmit to send the services select data to the frontend
	runtime.EventsEmit(a.ctx, eventServicesSelectChanged, ss, method, data, metadata)
	return nil
}

func (a *api) setWorkspaceOptions(opts options) {
	if opts.ID == "" {
		opts.ID = defaultWorkspaceKey
	}

	var val bytes.Buffer
	enc := gob.NewEncoder(&val)
	if err := enc.Encode(opts); err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to encode workspace options: %v", err))
		return
	}

	if err := a.store.set([]byte(opts.ID), val.Bytes()); err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to store workspace options: %v", err))
	}
}

func (a *api) setMetadata(key string, hds headers) {
	var toSet headers
	for _, h := range hds {
		if h.Key == "" {
			continue
		}
		toSet = append(toSet, h)
	}

	var val bytes.Buffer
	enc := gob.NewEncoder(&val)
	if err := enc.Encode(toSet); err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to encode metadata: %v", err))
		return
	}

	if err := a.store.set([]byte(key), val.Bytes()); err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to store metadata: %v", err))
	}
}

func (a *api) setMessage(method string, rawJSON []byte) {
	opts, err := a.GetWorkspaceOptions()
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to set message, no workspace options: %v", err))
		return
	}

	if err := a.store.set([]byte(messageKeyPrefix+hash(opts.Addr, method)), rawJSON); err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to store message: %v", err))
	}
}

func (a *api) monitorStateChanges(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			// This will panic if we are waiting for a state change and the client (and its connection)
			// get GC'd without this context being canceled
			runtime.LogError(a.ctx, fmt.Sprintf("panic monitoring state changes: %v", r))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			runtime.LogDebug(a.ctx, "ending monitoring of state changes")
			return
		default:
			if a.client == nil || a.client.conn == nil {
				// If client or connection is nil, wait a bit and check again
				time.Sleep(500 * time.Millisecond)
				continue
			}

			state := a.client.conn.GetState()
			runtime.EventsEmit(a.ctx, eventClientStateChanged, state.String())

			timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			ok := a.client.conn.WaitForStateChange(timeoutCtx, state)
			cancel()

			if !ok {
				runtime.LogDebug(a.ctx, "ending monitoring of state changes")
				time.Sleep(500 * time.Millisecond)
				return
			}
		}
	}
}

func (a *api) getMethodDesc(fullname string) (protoreflect.MethodDescriptor, error) {
	if a.protofiles == nil {
		return nil, fmt.Errorf("no proto files loaded")
	}

	name := strings.Replace(fullname[1:], "/", ".", 1)
	desc, err := a.protofiles.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, fmt.Errorf("failed to find descriptor: %v", err)
	}

	methodDesc, ok := desc.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor was not a method: %T", desc)
	}

	return methodDesc, nil
}

// SelectMethod is called when the user selects a new method by the given name
func (a *api) SelectMethod(fullname string, initState string, metadata interface{}) (rerr error) {
	defer func() {
		if rerr != nil {
			const errTitle = "Failed to select method"
			runtime.LogError(a.ctx, rerr.Error())
			runtime.EventsEmit(a.ctx, eventError, errorMsg{errTitle, rerr.Error()})
			runtime.EventsEmit(a.ctx, eventMethodInputChanged)
		}
	}()

	methodDesc, err := a.getMethodDesc(fullname)
	if err != nil {
		return err
	}

	in, err := messageViewFromDesc(methodDesc.Input(), &cyclicDetector{})
	if err != nil {
		return err
	}

	m := methodInput{
		FullName: fullname,
		Message:  in,
	}

	var hs headers
	if err := mapstructure.Decode(metadata, &hs); err != nil {
		runtime.LogDebug(a.ctx, fmt.Sprintf("failed to decode metadata: %v", err))
		runtime.EventsEmit(a.ctx, eventMethodInputChanged, m, initState)
	} else {
		runtime.EventsEmit(a.ctx, eventMethodInputChanged, m, initState, hs)
	}
	return nil
}

func messageViewFromDesc(md protoreflect.MessageDescriptor, cd *cyclicDetector) (*messageDesc, error) {
	//(rogchap) this is a recursive function, therefore we should make sure we
	// don't get a stack overflow. The protobuf wireformat does not support
	// cyclic data objects: protocolbuffers/protobuf#5504
	if err := cd.detect(md); err != nil {
		return nil, err
	}
	var rtn messageDesc
	rtn.Name = string(md.Name())
	rtn.FullName = string(md.FullName())

	fds := md.Fields()
	var err error
	rtn.Fields, err = fieldViewsFromDesc(fds, false, cd)
	if err != nil {
		return nil, err
	}

	return &rtn, nil
}

func setFieldDescBasics(fdesc *fieldDesc, fd protoreflect.FieldDescriptor) {
	fdesc.Name = string(fd.Name())
	fdesc.Kind = fd.Kind().String()
	fdesc.FullName = string(fd.FullName())
	fdesc.Repeated = fd.IsList()

	if emd := fd.Enum(); emd != nil {
		evals := emd.Values()
		for i := 0; i < evals.Len(); i++ {
			eval := evals.Get(i)
			fdesc.Enum = append(fdesc.Enum, string(eval.Name()))
		}
	}
}

func fieldViewsFromDesc(fds protoreflect.FieldDescriptors, isOneof bool, cd *cyclicDetector) ([]fieldDesc, error) {
	var fields []fieldDesc

	seenOneof := make(map[protoreflect.Name]struct{})
	for i := 0; i < fds.Len(); i++ {

		fd := fds.Get(i)
		fdesc := fieldDesc{}
		setFieldDescBasics(&fdesc, fd)

		if fd.IsMap() {
			fdesc.Kind = "map"
			fdesc.MapKey = &fieldDesc{}
			setFieldDescBasics(fdesc.MapKey, fd.MapKey())

			fdesc.MapValue = &fieldDesc{}
			mapVal := fd.MapValue()
			setFieldDescBasics(fdesc.MapValue, mapVal)
			if fmd := mapVal.Message(); fmd != nil {
				var err error
				fdesc.MapValue.Message, err = messageViewFromDesc(fmd, cd)
				if err != nil {
					return nil, err
				}
				cd.reset()
			}
			goto appendField
		}

		if !isOneof {
			if oneof := fd.ContainingOneof(); oneof != nil {
				if _, ok := seenOneof[oneof.Name()]; ok {
					continue
				}
				fdesc.Name = string(oneof.Name())
				fdesc.Kind = "oneof"
				var err error
				fdesc.Oneof, err = fieldViewsFromDesc(oneof.Fields(), true, cd)
				if err != nil {
					return nil, err
				}

				seenOneof[oneof.Name()] = struct{}{}
				goto appendField
			}
		}

		if fmd := fd.Message(); fmd != nil {
			var err error
			const structFullName = "google.protobuf.Struct"
			if fmd.FullName() == structFullName {
				fdesc.Kind = "message"
				fdesc.Message = &messageDesc{
					Name:     string(fmd.Name()),
					FullName: structFullName,
					Fields: []fieldDesc{{
						Name:     "value",
						FullName: "google.protobuf.Struct.value",
						Kind:     "string",
					}},
				}
				goto appendField
			}

			fdesc.Message, err = messageViewFromDesc(fmd, cd)
			if err != nil {
				return nil, err
			}
			cd.reset()
		}

	appendField:
		fields = append(fields, fdesc)
	}
	return fields, nil
}

func (a *api) RetryConnection() {
	if a.client == nil || a.client.conn == nil {
		runtime.LogError(a.ctx, "cannot retry connection: client or connection is nil")
		return
	}

	state := a.client.conn.GetState()
	if state == connectivity.TransientFailure || state == connectivity.Shutdown {
		// State is currently disconnected. Do a quick retry in case the server restarted recently.
		runtime.LogInfo(a.ctx, "connection in failed state, attempting to reset connection backoff")
		a.client.conn.ResetConnectBackoff()

		// Setup a one-time event handler to detect state change
		stateChanged := make(chan bool, 1)
		unsubscribe := runtime.EventsOn(a.ctx, eventClientStateChanged, func(data ...interface{}) {
			select {
			case stateChanged <- true:
				// Signal sent
			default:
				// Channel already has a value, no need to send again
			}
		})

		// Wait for at least one retry to complete or timeout after 5 seconds
		select {
		case <-stateChanged:
			// State changed, continue
		case <-time.After(5 * time.Second):
			runtime.LogWarning(a.ctx, "retry connection timed out waiting for state change")
		}

		// Clean up event handler
		unsubscribe()
	}
}

func (a *api) Send(method string, stringJSON string, rawHeaders interface{}) (rerr error) {
	rawJSON := []byte(stringJSON)
	defer func() {
		if rerr != nil {
			const errTitle = "Unable to send request"
			runtime.LogError(a.ctx, rerr.Error())
			runtime.EventsEmit(a.ctx, eventError, errorMsg{errTitle, rerr.Error()})
		}
	}()

	a.RetryConnection()

	md, err := a.getMethodDesc(method)
	if err != nil {
		const errTitle = "getMethodDesc"
		runtime.LogError(a.ctx, err.Error())
		runtime.EventsEmit(a.ctx, eventError, errorMsg{errTitle, err.Error()})
		return err
	}

	req := dynamicpb.NewMessage(md.Input())
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, req); err != nil {
		const errTitle = "unmarshal"
		runtime.LogError(a.ctx, err.Error())
		runtime.EventsEmit(a.ctx, eventError, errorMsg{errTitle, err.Error()})
		return fmt.Errorf("failed to unmarshal request: %v", err)
	}

	// Store message for later use
	go a.setMessage(method, rawJSON)

	if a.inFlight && md.IsStreamingClient() {
		a.streamReq <- req
		return nil
	}

	a.mu.Lock()
	a.inFlight = true
	defer func() {
		a.mu.Unlock()
		a.inFlight = false
	}()

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.New(nil))

	var hs headers
	if err := mapstructure.Decode(rawHeaders, &hs); err != nil {
		return fmt.Errorf("failed to decode headers: %v", err)
	}

	opts, err := a.GetWorkspaceOptions()
	if err != nil {
		return err
	}
	go a.setMetadata(metadataKeyPrefix+hash(opts.Addr), hs)

	for _, h := range hs {
		if h.Key == "" {
			continue
		}
		ctx = metadata.AppendToOutgoingContext(ctx, h.Key, h.Val)
	}

	ctx, a.cancelInFlight = context.WithCancel(ctx)

	runtime.EventsEmit(a.ctx, eventRPCStarted, rpcStart{
		ClientStream: md.IsStreamingClient(),
		ServerStream: md.IsStreamingServer(),
	})

	if md.IsStreamingClient() && md.IsStreamingServer() {
		stream, err := a.client.invokeBidiStream(ctx, method)
		if err != nil {
			return fmt.Errorf("failed to invoke bidirectional stream: %v", err)
		}

		a.streamReq = make(chan proto.Message)
		go func() {
			for r := range a.streamReq {
				if err := stream.SendMsg(r); err != nil {
					runtime.LogError(a.ctx, fmt.Sprintf("failed to send message to stream: %v", err))
					close(a.streamReq)
					a.streamReq = nil
				}
			}
			stream.CloseSend()
		}()
		a.streamReq <- req

		for {
			resp := dynamicpb.NewMessage(md.Output())
			if err := stream.RecvMsg(resp); err != nil {
				if err != io.EOF {
					runtime.LogDebug(a.ctx, fmt.Sprintf("stream receive ended with: %v", err))
				}
				break
			}
		}

		return nil
	}

	if md.IsStreamingClient() {
		stream, err := a.client.invokeClientStream(ctx, method)
		if err != nil {
			return fmt.Errorf("failed to invoke client stream: %v", err)
		}
		a.streamReq = make(chan proto.Message, 1)
		a.streamReq <- req
		done := ctx.Done()

	wait:
		for {
			select {
			case <-done:
				a.CloseSend()
				return nil
			case r := <-a.streamReq:
				if r == nil {
					break wait
				}
				if err := stream.SendMsg(r); err != nil {
					runtime.LogError(a.ctx, fmt.Sprintf("failed to send message to stream: %v", err))
					close(a.streamReq)
					a.streamReq = nil
					break wait
				}
			}
		}
		stream.CloseSend()
		resp := dynamicpb.NewMessage(md.Output())
		if err := stream.RecvMsg(resp); err != nil {
			if err != io.EOF {
				return fmt.Errorf("error receiving message: %v", err)
			}
		}
		if err := stream.RecvMsg(nil); err != io.EOF {
			runtime.LogWarning(a.ctx, fmt.Sprintf("unexpected message received after EOF: %v", err))
		}

		return nil
	}

	if md.IsStreamingServer() {
		stream, err := a.client.invokeServerStream(ctx, method, req)
		if err != nil {
			return fmt.Errorf("failed to invoke server stream: %v", err)
		}
		for {
			resp := dynamicpb.NewMessage(md.Output())
			if err := stream.RecvMsg(resp); err != nil {
				if err != io.EOF {
					runtime.LogDebug(a.ctx, fmt.Sprintf("stream receive ended with: %v", err))
				}
				break
			}
		}

		return nil
	}

	// Standard unary call
	resp := dynamicpb.NewMessage(md.Output())
	if err := a.client.invoke(ctx, method, req, resp); err != nil {
		return fmt.Errorf("failed to invoke RPC: %v", err)
	}
	return nil
}

// TagConn implements the stats.Handler interface
func (statsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	// noop
	return ctx
}

// HandleConn implements the stats.Handler interface
func (statsHandler) HandleConn(context.Context, stats.ConnStats) {
	// noop
}

// TagRPC implements the stats.Handler interface
func (statsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	// noop
	return ctx
}

// HandleRPC implements the stats.Handler interface
func (a statsHandler) HandleRPC(ctx context.Context, stat stats.RPCStats) {
	if internal := ctx.Value(ctxInternalKey{}); internal != nil {
		return
	}

	switch s := stat.(type) {
	case *stats.Begin:
		runtime.EventsEmit(a.ctx, eventStatBegin, s)
	case *stats.OutHeader:
		runtime.EventsEmit(a.ctx, eventStatOutHeader, rpcStatOutHeader{s, fmt.Sprintf("%+v", s.Header)})
	case *stats.OutPayload:
		if p, err := formatPayload(s.Payload); err == nil {
			s.Payload = p
		}
		runtime.EventsEmit(a.ctx, eventStatOutPayload, rpcStatOutPayload{s, fmt.Sprintf("%+v", s.Payload)})
		runtime.EventsEmit(a.ctx, eventOutPayloadReceived, s.Payload)
	case *stats.OutTrailer:
		runtime.EventsEmit(a.ctx, eventStatOutTrailer, rpcStatOutTrailer{s, fmt.Sprintf("%+v", s.Trailer)})
	case *stats.InHeader:
		runtime.EventsEmit(a.ctx, eventStatInHeader, rpcStatInHeader{s, fmt.Sprintf("%+v", s.Header)})
		runtime.EventsEmit(a.ctx, eventInHeaderReceived, s.Header)
	case *stats.InPayload:
		txt, err := formatPayload(s.Payload)
		if err != nil {
			runtime.LogError(a.ctx, fmt.Errorf("failed to marshal in payload to proto text: %v", err).Error())
			return
		}
		s.Payload = txt
		runtime.EventsEmit(a.ctx, eventStatInPayload, rpcStatInPayload{s, fmt.Sprintf("%+v", s.Payload)})
		runtime.EventsEmit(a.ctx, eventInPayloadReceived, txt)
	case *stats.InTrailer:
		runtime.EventsEmit(a.ctx, eventStatInTrailer, rpcStatInTrailer{s, fmt.Sprintf("%+v", s.Trailer)})
		runtime.EventsEmit(a.ctx, eventInTrailerReceived, s.Trailer)
	case *stats.End:

		errProtoStr := ""
		stus := status.Convert(s.Error)
		if stus != nil {
			var err error
			errProtoStr, err = formatPayload(stus.Proto())
			if err != nil {
				runtime.LogError(a.ctx, fmt.Errorf("failed to marshal in payload to proto text: %v", err).Error())
			}
			if errProtoStr != "" {
				runtime.EventsEmit(a.ctx, eventErrorReceived, errProtoStr)
			}
		}
		runtime.EventsEmit(a.ctx, eventStatEnd, rpcStatEnd{s, errProtoStr})

		var end rpcEnd
		end.StatusCode = int32(stus.Code())
		end.Status = stus.Code().String()
		end.Duration = s.EndTime.Sub(s.BeginTime).String()
		runtime.EventsEmit(a.ctx, eventRPCEnded, end)
	}
}

func formatPayload(payload interface{}) (string, error) {
	msg, ok := payload.(proto.Message)
	if !ok {
		// check to see if we are dealing with a APIv1 message
		//msgV1, ok := payload.(protoV1.Message)
		msgV1, ok := payload.(protoiface.MessageV1)
		if !ok {
			return "", fmt.Errorf("payload is not a proto message: %T", payload)
		}
		msg = protoadapt.MessageV2Of(msgV1)
	}

	marshaler := prototext.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}
	b, err := marshaler.Marshal(msg)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CloseSend will stop streaming client messages
func (a *api) CloseSend() {
	if a.streamReq != nil {
		runtime.LogDebug(a.ctx, "closing stream request channel")
		close(a.streamReq)
		a.streamReq = nil
	} else {
		runtime.LogDebug(a.ctx, "streamReq already closed or nil")
	}
}

// Cancel will attempt to cancel the current inflight request
func (a *api) Cancel() {
	if a.cancelInFlight != nil {
		runtime.LogDebug(a.ctx, "cancelling in-flight request")
		a.cancelInFlight()
		// Signal to frontend that the request was cancelled
		runtime.EventsEmit(a.ctx, eventRPCEnded, rpcEnd{
			StatusCode: int32(codes.Canceled),
			Status:     codes.Canceled.String(),
			Duration:   "0s",
		})
	} else {
		runtime.LogDebug(a.ctx, "no in-flight request to cancel")
	}
}

// Export commands for call
func (a *api) ExportCommands(method string, stringJSON string, rawHeaders interface{}) *commands {
	rawJSON := []byte(stringJSON)
	var sb strings.Builder
	sb.WriteString("grpcurl ")
	sb.WriteString("-d '")
	sb.Write(rawJSON)
	sb.WriteString("' \\\n")

	var hs headers
	if err := mapstructure.Decode(rawHeaders, &hs); err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to decode headers: %v", err))
		return &commands{
			Grpcurl: "Error: Failed to decode headers",
		}
	}

	for _, h := range hs {
		if len(h.Key) == 0 {
			continue
		}
		sb.WriteString("    -rpc-header '")
		sb.WriteString(h.Key)
		sb.WriteString(":")
		sb.WriteString(h.Val)
		sb.WriteString("' \\\n")
	}

	option, err := a.GetWorkspaceOptions()
	if err != nil {
		runtime.LogError(a.ctx, fmt.Sprintf("failed to get workspace options: %v", err))
		return &commands{
			Grpcurl: "Error: Failed to get workspace options",
		}
	}

	if option.Plaintext {
		sb.WriteString("    -plaintext \\\n")
	}
	if option.Insecure {
		sb.WriteString("    -insecure \\\n")
	}

	hds, err := a.GetReflectMetadata(option.Addr)
	if err != nil {
		runtime.LogWarning(a.ctx, fmt.Sprintf("failed to get reflection metadata: %v", err))
	} else {
		for _, h := range hds {
			if h.Key == "" {
				continue
			}
			sb.WriteString("    -reflect-header '")
			sb.WriteString(h.Key)
			sb.WriteString(":")
			sb.WriteString(h.Val)
			sb.WriteString("' \\\n")
		}
	}

	sb.WriteString("    ")
	sb.WriteString(option.Addr)
	sb.WriteString(" ")
	sb.WriteString(method[1:])

	return &commands{
		Grpcurl: sb.String(),
	}
}

func (a *api) ImportCommand(kind string, command string) (rerr error) {
	defer func() {
		if rerr != nil {
			const errTitle = "Failed to import command"
			runtime.LogError(a.ctx, rerr.Error())
			runtime.EventsEmit(a.ctx, eventError, errorMsg{errTitle, rerr.Error()})
		}
	}()

	switch strings.ToLower(kind) {
	case "grpcurl":
		args, err := parseGrpcurlCommand(command)
		if err != nil {
			return fmt.Errorf("error parsing grpcurl command: %v", err)
		}

		runtime.LogInfo(a.ctx, fmt.Sprintf("importing grpcurl command for method: %s", args.Method))
		return a.emitServicesSelect("/"+args.Method, args.Data, args.Metadata)
	default:
		return fmt.Errorf("unsupported command type: %s", kind)
	}
}

func (a *api) GetWindowInfo() map[string]interface{} {
	width, _ := runtime.WindowGetSize(a.ctx)
	height, _ := runtime.WindowGetSize(a.ctx)
	x, y := runtime.WindowGetPosition(a.ctx)

	// Detect the operating system using Go's standard library
	os := goruntime.GOOS // "windows", "darwin" (macOS), "linux", etc.
	isWindows := os == "windows"

	return map[string]interface{}{
		"width":     width,
		"height":    height,
		"x":         x,
		"y":         y,
		"isWindows": isWindows, // Add the platform information
	}
}
