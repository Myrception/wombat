package app

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/go-viper/mapstructure/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"google.golang.org/grpc"
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
	ctx     context.Context
	rt      wailsRuntime
	appData string
	store   *store

	// startupErr is reported to the user once the frontend is ready.
	startupErr error
	readyOnce  sync.Once

	// connectMu serialises Connect so two connection attempts (e.g. startup
	// and a workspace switch) cannot interleave.
	connectMu sync.Mutex

	// mu guards the fields below. Wails invokes every bound method on its own
	// goroutine, so all of these are accessed concurrently.
	mu               sync.Mutex
	gen              uint64 // incremented on every (re)connect
	client           *client
	addr             string
	protofiles       *protoregistry.Files
	cancelMonitoring context.CancelFunc
	state            *workspaceState

	// callMu guards call, the single in-flight RPC.
	callMu sync.Mutex
	call   *rpcCall
}

type statsHandler struct {
	*api
}

type storeLogger struct {
	ctx context.Context
	log *slog.Logger
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
	s.log.Debug(strings.TrimSpace(fmt.Sprintf(format, args...)))
}

// Infof implements badger.Logger interface
func (s *storeLogger) Infof(format string, args ...interface{}) {
	// Use slog's Info level
	s.log.Info(strings.TrimSpace(fmt.Sprintf(format, args...)))
}

// Warningf implements badger.Logger interface
func (s *storeLogger) Warningf(format string, args ...interface{}) {
	// Use slog's Warn level and also send to Wails runtime
	s.log.Warn(strings.TrimSpace(fmt.Sprintf(format, args...)))
}

// Errorf implements badger.Logger interface
func (s *storeLogger) Errorf(format string, args ...interface{}) {
	// Use slog's Error level and also send to Wails runtime
	s.log.Error(strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func NewApp() *api {
	return &api{state: &workspaceState{CurrentID: defaultWorkspaceKey}}
}

// Startup is the initialization function for the Wails v2 runtime.
//
// Note: on Linux this runs concurrently with the page load, so nothing here may
// emit events the frontend depends on; those would be lost. The initial
// connection is made in FrontendReady instead.
func (a *api) Startup(ctx context.Context) {
	a.ctx = ctx
	a.rt = ctxRuntime{ctx}

	var err error
	a.store, err = newStore(a.appData, newStoreLogger(ctx))
	if err != nil {
		a.startupErr = fmt.Errorf("failed to open database in %s (is another Wombat instance running?): %v", a.appData, err)
		a.rt.LogError(a.startupErr.Error())
	}

	st := a.getCurrentState()
	a.mu.Lock()
	a.state = st
	a.mu.Unlock()
}

// FrontendReady must be called by the frontend once all components are mounted
// and subscribed to events. The first call makes the initial connection; later
// calls (e.g. after a webview reload) re-emit the current state.
func (a *api) FrontendReady() {
	first := false
	a.readyOnce.Do(func() { first = true })
	if !first {
		a.emitCurrentState()
		return
	}

	if a.startupErr != nil {
		a.emitError("Database Error", a.startupErr.Error())
	}
	a.connectCurrentWorkspace()
	a.goSafe("update check", a.checkForUpdate)
}

func (a *api) connectCurrentWorkspace() {
	opts, err := a.GetWorkspaceOptions()
	if err != nil {
		a.rt.LogError(fmt.Sprintf("failed to load workspace options: %v", err))
		return
	}
	if opts.Addr == "" {
		// Fresh workspace, nothing to connect to yet.
		return
	}
	hds, err := a.GetReflectMetadata(opts.Addr)
	if err != nil {
		a.rt.LogWarning(fmt.Sprintf("failed to get reflection metadata: %v", err))
	}
	// Connect reports its own errors to the frontend.
	_ = a.Connect(opts, hds, false)
}

// emitCurrentState re-sends the connection and services state to the frontend.
func (a *api) emitCurrentState() {
	a.mu.Lock()
	cl, addr := a.client, a.addr
	a.mu.Unlock()

	if cl == nil {
		a.rt.Emit(eventClientStateChanged, connectivity.Shutdown.String())
		return
	}
	a.rt.Emit(eventClientConnectStarted, addr)
	a.rt.Emit(eventClientConnected, addr)
	a.rt.Emit(eventClientStateChanged, cl.conn.GetState().String())
	if err := a.emitServicesSelect("", "", nil); err != nil {
		a.rt.LogError(err.Error())
	}
}

func (a *api) checkForUpdate() {
	r, err := checkForUpdate()
	if err != nil {
		if err == errNoUpdate {
			a.rt.LogInfo(err.Error())
			return
		}
		a.rt.LogWarning(fmt.Sprintf("failed to check for updates: %v", err))
		return
	}
	a.rt.Emit(eventUpdateAvailable, r)
}

// Shutdown is called when the application is closing
func (a *api) Shutdown(ctx context.Context) {
	a.resetConnection()
	if err := a.store.close(); err != nil && a.rt != nil {
		a.rt.LogWarning(fmt.Sprintf("failed to close the database: %v", err))
	}
}

func (a *api) getClient() *client {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.client
}

func (a *api) getProtofiles() *protoregistry.Files {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.protofiles
}

func (a *api) currentWorkspaceID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state.CurrentID
}

func (a *api) emitError(title, msg string) {
	a.rt.Emit(eventError, errorMsg{title, msg})
}

func (a *api) getCurrentState() *workspaceState {
	rtn := &workspaceState{
		CurrentID: defaultWorkspaceKey,
	}
	val, err := a.store.get([]byte(defaultStateKey))
	if err != nil && err != errKeyNotFound {
		a.rt.LogError(fmt.Sprintf("failed to get current state from store: %v", err))
	}
	if len(val) == 0 {
		return rtn
	}
	dec := gob.NewDecoder(bytes.NewBuffer(val))
	if err := dec.Decode(rtn); err != nil {
		a.rt.LogError(fmt.Sprintf("failed to decode state: %v", err))
	}
	return rtn
}

// GetWorkspaceOptions gets the workspace options from the store
func (a *api) GetWorkspaceOptions() (*options, error) {
	wo := &options{
		ID: a.currentWorkspaceID(),
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

// GetReflectMetadata gets the reflection metadata from the store by addr
func (a *api) GetReflectMetadata(addr string) (headers, error) {
	val, err := a.store.get([]byte(reflectMetadataKeyPrefix + hash(addr)))
	if err == errKeyNotFound {
		return headers{}, nil
	}
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
	if err == errKeyNotFound {
		return headers{}, nil
	}
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
	if a.currentWorkspaceID() == id {
		return nil
	}

	defer func() {
		if rerr != nil {
			a.rt.LogError(rerr.Error())
			a.emitError("Workspace Error", rerr.Error())
		}
	}()

	// Connect tears down the previous connection.
	a.changeWorkspace(id)
	opts, err := a.GetWorkspaceOptions()
	if err != nil {
		return err
	}

	hds, err := a.GetReflectMetadata(opts.Addr)
	if err != nil {
		a.rt.LogWarning(fmt.Sprintf("failed to get reflection metadata: %v", err))
	}

	// Connect reports its own errors to the frontend.
	_ = a.Connect(opts, hds, false)

	return nil
}

// DeleteWorkspace will remove a workspace from the store and switch to
// the default workspace, if the deleted workspace is current.
func (a *api) DeleteWorkspace(id string) error {
	if err := a.store.del([]byte(id)); err != nil {
		return err
	}
	if a.currentWorkspaceID() == id {
		// SelectWorkspace reports its own errors to the frontend.
		_ = a.SelectWorkspace(defaultWorkspaceKey)
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
	if err == errKeyNotFound {
		// Nothing saved yet for this method.
		return "", nil
	}
	return string(val), err
}

// FindProtoFiles opens a directory dialog to search for proto files
func (a *api) FindProtoFiles() (files []string, rerr error) {
	defer func() {
		if rerr != nil {
			const errTitle = "Not found"
			a.rt.LogError(rerr.Error())
			a.emitError(errTitle, rerr.Error())
		}
	}()

	dir, err := a.SelectDirectory()
	if err != nil {
		return nil, fmt.Errorf("failed to open the directory dialog: %v", err)
	}
	if dir == "" {
		return nil, nil // dialog cancelled
	}
	return findProtoFiles(dir)
}

// findProtoFiles returns all *.proto files below dir. Subdirectories that
// cannot be read are skipped.
func findProtoFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path != dir && d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".proto" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search %s: %v", dir, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no *.proto files found in %s", dir)
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
	a.connectMu.Lock()
	defer a.connectMu.Unlock()

	defer func() {
		if rerr != nil {
			const errTitle = "Connection error"
			a.rt.LogError(rerr.Error())
			a.rt.Emit(eventClientStateChanged, connectivity.Shutdown.String())
			a.emitError(errTitle, rerr.Error())
		}
	}()

	var opts options
	if err := mapstructure.Decode(data, &opts); err != nil {
		return err
	}

	// reset all things
	a.rt.Emit(eventClientConnectStarted, opts.Addr)
	a.rt.Emit(eventServicesSelectChanged)
	a.rt.Emit(eventMethodInputChanged)

	gen := a.resetConnection()

	var hds headers
	if err := mapstructure.Decode(rawHeaders, &hds); err != nil {
		a.rt.LogError(fmt.Sprintf("unable to decode reflection metadata headers: %v", err))
	}

	cl := &client{}
	if err := cl.connect(opts, statsHandler{a}); err != nil {
		// Still try to parse proto definitions. Will fail silently
		// if using reflection services as there is no connection
		// to a valid server.
		a.goSafe("loading proto files", func() { _ = a.loadProtoFiles(gen, nil, opts, hds, true) }) // reports its own errors
		return fmt.Errorf("failed to connect to server: %v", err)
	}

	monCtx, cancelMon := context.WithCancel(context.Background())
	a.mu.Lock()
	a.client = cl
	a.addr = opts.Addr
	a.cancelMonitoring = cancelMon
	a.mu.Unlock()
	a.goSafe("connection monitor", func() { a.monitorStateChanges(monCtx, cl) })

	a.rt.Emit(eventClientConnected, opts.Addr)

	a.goSafe("loading proto files", func() { _ = a.loadProtoFiles(gen, cl, opts, hds, false) }) // reports its own errors

	if save {
		if opts.ID == "" {
			opts.ID = workspacePrefix + uuid.Must(uuid.NewV4()).String()
			a.changeWorkspace(opts.ID)
		}
		a.setWorkspaceOptions(opts)
		a.setMetadata(reflectMetadataKeyPrefix+hash(opts.Addr), hds)
	}
	return nil
}

// resetConnection cancels any in-flight call, stops state monitoring and closes
// the current connection. It returns the new connection generation.
func (a *api) resetConnection() uint64 {
	a.cancelCall()

	a.mu.Lock()
	a.gen++
	gen := a.gen
	old, cancelMon := a.client, a.cancelMonitoring
	a.client, a.cancelMonitoring, a.protofiles, a.addr = nil, nil, nil, ""
	a.mu.Unlock()

	if cancelMon != nil {
		cancelMon()
	}
	if err := old.close(); err != nil {
		a.rt.LogWarning(fmt.Sprintf("failed to close previous connection: %v", err))
	}
	return gen
}

func (a *api) changeWorkspace(id string) {
	a.mu.Lock()
	a.state.CurrentID = id
	st := *a.state
	a.mu.Unlock()

	var val bytes.Buffer
	if err := gob.NewEncoder(&val).Encode(st); err != nil {
		a.rt.LogError(fmt.Sprintf("failed to encode state: %v", err))
		return
	}
	if err := a.store.set([]byte(defaultStateKey), val.Bytes()); err != nil {
		a.rt.LogError(fmt.Sprintf("failed to store state: %v", err))
	}
}

// loadProtoFiles loads the schema for connection generation gen. The result is
// dropped if a newer connection was made in the meantime.
func (a *api) loadProtoFiles(gen uint64, cl *client, opts options, reflectHeaders headers, silent bool) (rerr error) {
	defer func() {
		if rerr != nil {
			const errTitle = "Failed to load RPC schema"
			a.rt.LogError(rerr.Error())
			if !silent {
				a.rt.Emit(eventError, errorMsg{errTitle, rerr.Error()})
			}
		}
	}()

	var files *protoregistry.Files
	var err error
	switch {
	case opts.Reflect:
		if cl == nil {
			return errors.New("unable to load proto files via reflection: not connected")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		ctx = metadata.NewOutgoingContext(ctx, metadata.New(nil))
		for _, h := range reflectHeaders {
			if h.Key == "" {
				continue
			}
			ctx = metadata.AppendToOutgoingContext(ctx, h.Key, h.Val)
		}
		ctx = context.WithValue(ctx, ctxInternalKey{}, struct{}{})
		if files, err = protoFilesFromReflectionAPI(ctx, cl.conn); err != nil {
			return fmt.Errorf("error getting proto files from reflection API: %v", err)
		}
	case len(opts.Protos.Files) > 0:
		if files, err = protoFilesFromDisk(opts.Protos.Roots, opts.Protos.Files); err != nil {
			return fmt.Errorf("error parsing proto files from disk: %v", err)
		}
	}

	a.mu.Lock()
	if a.gen != gen {
		a.mu.Unlock()
		return nil
	}
	a.protofiles = files
	a.mu.Unlock()

	return a.emitServicesSelect("", "", nil)
}

func isReflectionService(name string) bool {
	return strings.HasPrefix(name, "grpc.reflection.")
}

func (a *api) emitServicesSelect(method string, data string, metadata headers) error {
	files := a.getProtofiles()
	if files == nil {
		return nil
	}

	var targetMd protoreflect.MethodDescriptor
	var ss servicesSelect
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
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

	// Sorted by name, with the reflection services last: they are rarely
	// what the user wants to call, and the first service is selected
	// automatically.
	sort.SliceStable(ss, func(i, j int) bool {
		ri, rj := isReflectionService(ss[i].FullName), isReflectionService(ss[j].FullName)
		if ri != rj {
			return rj
		}
		return ss[i].FullName < ss[j].FullName
	})

	if method != "" && targetMd == nil {
		return fmt.Errorf("method %q not found", method)
	}

	// Use Wails v2 EventsEmit to send the services select data to the frontend
	a.rt.Emit(eventServicesSelectChanged, ss, method, data, metadata)
	return nil
}

func (a *api) setWorkspaceOptions(opts options) {
	if opts.ID == "" {
		opts.ID = defaultWorkspaceKey
	}

	var val bytes.Buffer
	enc := gob.NewEncoder(&val)
	if err := enc.Encode(opts); err != nil {
		a.rt.LogError(fmt.Sprintf("failed to encode workspace options: %v", err))
		return
	}

	if err := a.store.set([]byte(opts.ID), val.Bytes()); err != nil {
		a.rt.LogError(fmt.Sprintf("failed to store workspace options: %v", err))
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
		a.rt.LogError(fmt.Sprintf("failed to encode metadata: %v", err))
		return
	}

	if err := a.store.set([]byte(key), val.Bytes()); err != nil {
		a.rt.LogError(fmt.Sprintf("failed to store metadata: %v", err))
	}
}

func (a *api) setMessage(method string, rawJSON []byte) {
	opts, err := a.GetWorkspaceOptions()
	if err != nil {
		a.rt.LogError(fmt.Sprintf("failed to set message, no workspace options: %v", err))
		return
	}

	if err := a.store.set([]byte(messageKeyPrefix+hash(opts.Addr, method)), rawJSON); err != nil {
		a.rt.LogError(fmt.Sprintf("failed to store message: %v", err))
	}
}

// monitorStateChanges forwards connectivity changes of cl to the frontend until
// ctx is cancelled.
func (a *api) monitorStateChanges(ctx context.Context, cl *client) {
	for {
		state := cl.conn.GetState()
		a.rt.Emit(eventClientStateChanged, state.String())
		// Only returns false once ctx is done.
		if !cl.conn.WaitForStateChange(ctx, state) || ctx.Err() != nil {
			a.rt.LogDebug("ending monitoring of state changes")
			return
		}
	}
}

func (a *api) getMethodDesc(fullname string) (protoreflect.MethodDescriptor, error) {
	files := a.getProtofiles()
	if files == nil {
		return nil, fmt.Errorf("no proto files loaded")
	}
	if len(fullname) < 2 || fullname[0] != '/' {
		return nil, fmt.Errorf("invalid method name %q", fullname)
	}

	name := strings.Replace(fullname[1:], "/", ".", 1)
	desc, err := files.FindDescriptorByName(protoreflect.FullName(name))
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
			a.rt.LogError(rerr.Error())
			a.rt.Emit(eventError, errorMsg{errTitle, rerr.Error()})
			a.rt.Emit(eventMethodInputChanged)
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
		a.rt.LogDebug(fmt.Sprintf("failed to decode metadata: %v", err))
		a.rt.Emit(eventMethodInputChanged, m, initState)
	} else {
		a.rt.Emit(eventMethodInputChanged, m, initState, hs)
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

// RetryConnection resets the connection backoff if the connection is in a
// failed state and waits briefly for it to recover.
func (a *api) RetryConnection() {
	if cl := a.getClient(); cl != nil {
		a.retryConnection(cl)
	}
}

func (a *api) retryConnection(cl *client) {
	state := cl.conn.GetState()
	switch state {
	case connectivity.Idle:
		cl.conn.Connect()
		return
	case connectivity.TransientFailure:
	default:
		return
	}

	// The server may have restarted recently; retry now instead of waiting
	// for the backoff to expire.
	a.rt.LogInfo("connection in failed state, resetting connection backoff")
	cl.conn.ResetConnectBackoff()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		if !cl.conn.WaitForStateChange(ctx, state) {
			a.rt.LogWarning("retry connection timed out waiting for state change")
			return
		}
		state = cl.conn.GetState()
		switch state {
		case connectivity.Idle:
			cl.conn.Connect()
		case connectivity.Connecting:
		default:
			return
		}
	}
}

// Send invokes method with the given JSON request. If a client stream for the
// same method is already open, the message is sent on that stream instead.
func (a *api) Send(method string, stringJSON string, rawHeaders interface{}) (rerr error) {
	defer func() {
		if rerr != nil {
			const errTitle = "Unable to send request"
			a.rt.LogError(rerr.Error())
			a.rt.Emit(eventError, errorMsg{errTitle, rerr.Error()})
		}
	}()

	rawJSON := []byte(stringJSON)
	md, err := a.getMethodDesc(method)
	if err != nil {
		return err
	}

	req := dynamicpb.NewMessage(md.Input())
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(rawJSON, req); err != nil {
		return fmt.Errorf("failed to unmarshal request: %v", err)
	}

	// Store message for later use
	a.goSafe("saving message", func() { a.setMessage(method, rawJSON) })

	// Additional message for an already open client stream.
	if cur := a.currentCall(); cur != nil {
		if cur.reqs != nil && cur.method == method {
			return cur.push(req)
		}
		return errCallInProgress
	}

	var hs headers
	if err := mapstructure.Decode(rawHeaders, &hs); err != nil {
		return fmt.Errorf("failed to decode headers: %v", err)
	}

	opts, err := a.GetWorkspaceOptions()
	if err != nil {
		return err
	}
	a.goSafe("saving metadata", func() { a.setMetadata(metadataKeyPrefix+hash(opts.Addr), hs) })

	cl := a.getClient()
	if cl == nil {
		return errNoConn
	}
	a.retryConnection(cl)

	ctx := metadata.NewOutgoingContext(context.Background(), metadata.New(nil))
	for _, h := range hs {
		if h.Key == "" {
			continue
		}
		ctx = metadata.AppendToOutgoingContext(ctx, h.Key, h.Val)
	}
	ctx, cancel := context.WithCancel(ctx)
	call := newRPCCall(method, cancel, md.IsStreamingClient())
	ctx = context.WithValue(ctx, ctxCallKey{}, call)

	if err := a.beginCall(call); err != nil {
		cancel()
		return err
	}
	defer func() {
		a.endCall(call)
		if !call.ended.Load() {
			// gRPC never reported the end of the RPC (e.g. the stream
			// could not be created), so tell the frontend ourselves.
			st := status.Convert(rerr)
			a.rt.Emit(eventRPCEnded, rpcEnd{
				StatusCode: int32(st.Code()),
				Status:     st.Code().String(),
				Duration:   "0s",
			})
		}
	}()

	a.rt.Emit(eventRPCStarted, rpcStart{
		ClientStream: md.IsStreamingClient(),
		ServerStream: md.IsStreamingServer(),
	})

	// Responses and status are delivered to the frontend by the stats
	// handler, so receive errors are only logged here.
	recvAll := func(stream grpc.ClientStream) {
		for {
			if err := stream.RecvMsg(dynamicpb.NewMessage(md.Output())); err != nil {
				if err != io.EOF {
					a.rt.LogDebug(fmt.Sprintf("stream receive ended with: %v", err))
				}
				return
			}
		}
	}

	switch {
	case md.IsStreamingClient() && md.IsStreamingServer():
		stream, err := cl.invokeBidiStream(ctx, method)
		if err != nil {
			return fmt.Errorf("failed to invoke bidirectional stream: %v", err)
		}
		if err := stream.SendMsg(req); err != nil && err != io.EOF {
			return fmt.Errorf("failed to send message to stream: %v", err)
		}
		a.goSafe("request stream", func() {
			if err := call.pump(ctx, stream); err != nil {
				a.rt.LogError(fmt.Sprintf("failed to send message to stream: %v", err))
			}
		})
		recvAll(stream)
		return nil

	case md.IsStreamingClient():
		stream, err := cl.invokeClientStream(ctx, method)
		if err != nil {
			return fmt.Errorf("failed to invoke client stream: %v", err)
		}
		if err := stream.SendMsg(req); err != nil && err != io.EOF {
			return fmt.Errorf("failed to send message to stream: %v", err)
		}
		// Blocks until the user clicks "close" (CloseSend) or cancels.
		if err := call.pump(ctx, stream); err != nil {
			a.rt.LogError(fmt.Sprintf("failed to send message to stream: %v", err))
		}
		recvAll(stream)
		return nil

	case md.IsStreamingServer():
		stream, err := cl.invokeServerStream(ctx, method, req)
		if err != nil {
			return fmt.Errorf("failed to invoke server stream: %v", err)
		}
		recvAll(stream)
		return nil

	default:
		resp := dynamicpb.NewMessage(md.Output())
		if err := cl.invoke(ctx, method, req, resp); err != nil {
			return fmt.Errorf("failed to invoke RPC: %v", err)
		}
		return nil
	}
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
		a.rt.Emit(eventStatBegin, s)
	case *stats.OutHeader:
		a.rt.Emit(eventStatOutHeader, rpcStatOutHeader{s, fmt.Sprintf("%+v", s.Header)})
	case *stats.OutPayload:
		if p, err := formatPayload(s.Payload); err == nil {
			s.Payload = p
		}
		a.rt.Emit(eventStatOutPayload, rpcStatOutPayload{s, fmt.Sprintf("%+v", s.Payload)})
		a.rt.Emit(eventOutPayloadReceived, s.Payload)
	case *stats.OutTrailer:
		a.rt.Emit(eventStatOutTrailer, rpcStatOutTrailer{s, fmt.Sprintf("%+v", s.Trailer)})
	case *stats.InHeader:
		a.rt.Emit(eventStatInHeader, rpcStatInHeader{s, fmt.Sprintf("%+v", s.Header)})
		a.rt.Emit(eventInHeaderReceived, s.Header)
	case *stats.InPayload:
		txt, err := formatPayload(s.Payload)
		if err != nil {
			a.rt.LogError(fmt.Errorf("failed to marshal in payload to proto text: %v", err).Error())
			return
		}
		s.Payload = txt
		a.rt.Emit(eventStatInPayload, rpcStatInPayload{s, fmt.Sprintf("%+v", s.Payload)})
		a.rt.Emit(eventInPayloadReceived, txt)
	case *stats.InTrailer:
		a.rt.Emit(eventStatInTrailer, rpcStatInTrailer{s, fmt.Sprintf("%+v", s.Trailer)})
		a.rt.Emit(eventInTrailerReceived, s.Trailer)
	case *stats.End:
		if c, ok := ctx.Value(ctxCallKey{}).(*rpcCall); ok {
			c.ended.Store(true)
		}

		errProtoStr := ""
		stus := status.Convert(s.Error)
		if stus != nil {
			var err error
			errProtoStr, err = formatPayload(stus.Proto())
			if err != nil {
				a.rt.LogError(fmt.Errorf("failed to marshal in payload to proto text: %v", err).Error())
			}
			if errProtoStr != "" {
				a.rt.Emit(eventErrorReceived, errProtoStr)
			}
		}
		a.rt.Emit(eventStatEnd, rpcStatEnd{s, errProtoStr})

		var end rpcEnd
		end.StatusCode = int32(stus.Code())
		end.Status = stus.Code().String()
		end.Duration = s.EndTime.Sub(s.BeginTime).String()
		a.rt.Emit(eventRPCEnded, end)
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
	if c := a.currentCall(); c != nil {
		a.rt.LogDebug("closing request stream")
		c.closeSending()
	}
}

// Cancel will attempt to cancel the current inflight request. The rpc_ended
// event is emitted by the stats handler once gRPC has torn the call down.
func (a *api) Cancel() {
	if !a.cancelCall() {
		a.rt.LogDebug("no in-flight request to cancel")
	}
}

// Export commands for call
func (a *api) ExportCommands(method string, stringJSON string, rawHeaders interface{}) *commands {
	rawJSON := []byte(stringJSON)
	var sb strings.Builder
	sb.WriteString("grpcurl ")
	sb.WriteString("-d ")
	sb.WriteString(shellQuote(string(rawJSON)))
	sb.WriteString(" \\\n")

	var hs headers
	if err := mapstructure.Decode(rawHeaders, &hs); err != nil {
		a.rt.LogError(fmt.Sprintf("failed to decode headers: %v", err))
		return &commands{
			Grpcurl: "Error: Failed to decode headers",
		}
	}

	for _, h := range hs {
		if len(h.Key) == 0 {
			continue
		}
		sb.WriteString("    -rpc-header ")
		sb.WriteString(shellQuote(h.Key + ": " + h.Val))
		sb.WriteString(" \\\n")
	}

	option, err := a.GetWorkspaceOptions()
	if err != nil {
		a.rt.LogError(fmt.Sprintf("failed to get workspace options: %v", err))
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
		a.rt.LogWarning(fmt.Sprintf("failed to get reflection metadata: %v", err))
	} else {
		for _, h := range hds {
			if h.Key == "" {
				continue
			}
			sb.WriteString("    -reflect-header ")
			sb.WriteString(shellQuote(h.Key + ": " + h.Val))
			sb.WriteString(" \\\n")
		}
	}

	sb.WriteString("    ")
	sb.WriteString(option.Addr)
	sb.WriteString(" ")
	sb.WriteString(strings.TrimPrefix(method, "/"))

	return &commands{
		Grpcurl: sb.String(),
	}
}

func (a *api) ImportCommand(kind string, command string) (rerr error) {
	defer func() {
		if rerr != nil {
			const errTitle = "Failed to import command"
			a.rt.LogError(rerr.Error())
			a.rt.Emit(eventError, errorMsg{errTitle, rerr.Error()})
		}
	}()

	switch strings.ToLower(kind) {
	case "grpcurl":
		args, err := parseGrpcurlCommand(command)
		if err != nil {
			return fmt.Errorf("error parsing grpcurl command: %v", err)
		}

		a.rt.LogInfo(fmt.Sprintf("importing grpcurl command for method: %s", args.Method))
		return a.emitServicesSelect("/"+args.Method, args.Data, args.Metadata)
	default:
		return fmt.Errorf("unsupported command type: %s", kind)
	}
}
