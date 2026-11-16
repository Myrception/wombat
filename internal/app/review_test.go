package app

import (
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"wombat/internal/server"
)

// A client stream the server ends before the user closes it must end the
// call with the server's status, instead of waiting for "Close & Receive".
func TestClientStreamServerEndsEarly(t *testing.T) {
	reject := grpc.StreamInterceptor(func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, h grpc.StreamHandler) error {
		if info.FullMethod == methodRecordRoute {
			return status.Error(codes.PermissionDenied, "not allowed")
		}
		return h(srv, ss)
	})
	addr, _ := startServer(t, reject)
	a, rec := newTestAPI(t)
	connectTo(t, a, options{Addr: addr, Plaintext: true, Reflect: true})

	// The call may be over before it could be seen in progress, so
	// startStream (which waits for that) is not used.
	done := make(chan error, 1)
	go func() { done <- a.Send(methodRecordRoute, pointJSON, headers{}) }()
	if err := recvErr(t, done); err != nil { // no CloseSend
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, "rpc_ended", func() bool { return rec.count(eventRPCEnded) == 1 })
	if got := codes.Code(rec.rpcEnds()[0].StatusCode); got != codes.PermissionDenied {
		t.Fatalf("status = %v, want PermissionDenied", got)
	}
}

// A unary call with a non-OK status shows the status in the response pane
// (via rpc_ended), not an extra error dialog.
func TestUnaryErrorStatusNoDialog(t *testing.T) {
	a, rec := newConnectedAPI(t)

	// The test server does not implement Foobar.Foo.
	if err := a.Send("/wombat.v1.Foobar/Foo", "{}", headers{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, "rpc_ended", func() bool { return rec.count(eventRPCEnded) == 1 })
	if got := codes.Code(rec.rpcEnds()[0].StatusCode); got != codes.Unimplemented {
		t.Fatalf("status = %v, want Unimplemented", got)
	}
	if n := rec.count(eventError); n != 0 {
		t.Fatalf("got %d error dialogs, want 0: %+v", n, rec.all(eventError))
	}
}

func closedAddr(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := lis.Addr().String()
	_ = lis.Close()
	return addr
}

// The workspace dialog's settings are saved even when the server cannot be
// reached, including for a new workspace.
func TestConnectFailureKeepsSettings(t *testing.T) {
	a, _ := newTestAPI(t)
	addr := closedAddr(t)
	md := headers{{Key: "authorization", Val: "Bearer x"}}

	if err := a.Connect(options{Addr: addr, Plaintext: true, Reflect: true}, md, true); err == nil {
		t.Fatal("expected a connection error")
	}

	opts, err := a.GetWorkspaceOptions()
	if err != nil {
		t.Fatal(err)
	}
	if opts.Addr != addr || !strings.HasPrefix(opts.ID, workspacePrefix) || opts.ID == defaultWorkspaceKey {
		t.Fatalf("current workspace = %+v, want a new one for %s", opts, addr)
	}
	list, err := a.ListWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range list {
		found = found || w.ID == opts.ID
	}
	if !found {
		t.Fatalf("new workspace %s not listed: %+v", opts.ID, list)
	}
	got, err := a.GetReflectMetadata(addr)
	if err != nil || len(got) != 1 || got[0] != md[0] {
		t.Fatalf("reflection metadata = %+v, %v; want %+v", got, err, md)
	}
}

// Switching to a workspace without an address disconnects quietly.
func TestSelectUnconfiguredWorkspace(t *testing.T) {
	a, rec := newConnectedAPI(t)
	a.setWorkspaceOptions(options{ID: workspacePrefix + "other", Addr: "localhost:1"})
	a.changeWorkspace(workspacePrefix + "other")

	if err := a.SelectWorkspace(defaultWorkspaceKey); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond) // anything reported asynchronously
	if n := rec.count(eventError); n != 0 {
		t.Fatalf("got %d error dialogs, want 0: %+v", n, rec.all(eventError))
	}
	if a.getClient() != nil {
		t.Fatal("still connected")
	}
	if a.currentWorkspaceID() != defaultWorkspaceKey {
		t.Fatalf("current workspace = %s", a.currentWorkspaceID())
	}
}

func TestProtocNotFound(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := protoFilesFromDisk(nil, []string{"x.proto"})
	if err == nil || !strings.Contains(err.Error(), "protoc was not found") {
		t.Fatalf("got %v, want a 'protoc was not found' error", err)
	}
}

// Connect without an address fails before anything is saved.
func TestConnectEmptyAddressNotSaved(t *testing.T) {
	a, _ := newTestAPI(t)
	if err := a.Connect(options{Addr: "  "}, headers{}, true); err == nil {
		t.Fatal("expected an error")
	}
	list, err := a.ListWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range list {
		if w.ID != defaultWorkspaceKey {
			t.Fatalf("a workspace was saved: %+v", list)
		}
	}
	if a.currentWorkspaceID() != defaultWorkspaceKey && a.currentWorkspaceID() != "" {
		t.Fatalf("current workspace = %s", a.currentWorkspaceID())
	}
}

// The frontend checks JSON-edited field names against the JSON name the
// backend accepts, including a custom json_name.
func TestFieldJSONName(t *testing.T) {
	m, err := messageViewFromDesc((&server.Tree{}).ProtoReflect().Descriptor(), &cyclicDetector{})
	if err != nil {
		t.Fatal(err)
	}
	if got := fieldByName(t, m, "display_name").JSONName; got != "label" {
		t.Errorf("display_name json name = %q, want label", got)
	}
	if got := fieldByName(t, m, "children").JSONName; got != "children" {
		t.Errorf("children json name = %q", got)
	}
}

// Importing a command whose data has a field the method does not have
// imports it, and says that the field is ignored.
func TestImportUnknownFieldNotice(t *testing.T) {
	a, rec := newConnectedAPI(t)

	ok := `grpcurl -d '{"latitude":1}' -plaintext localhost:1 wombat.v1.RouteGuide/GetFeature`
	if err := a.ImportCommand("grpcurl", ok); err != nil {
		t.Fatal(err)
	}
	if n := rec.count(eventError); n != 0 {
		t.Fatalf("got %d dialogs for valid data: %+v", n, rec.all(eventError))
	}

	typo := `grpcurl -d '{"latitude":1,"lattitude":2}' -plaintext localhost:1 wombat.v1.RouteGuide/GetFeature`
	if err := a.ImportCommand("grpcurl", typo); err != nil {
		t.Fatal(err)
	}
	errs := rec.all(eventError)
	if len(errs) != 1 || !strings.Contains(errs[0].data[0].(errorMsg).Message, "lattitude") {
		t.Fatalf("want one notice naming lattitude, got %+v", errs)
	}
}

// Deleting a workspace deletes its metadata unless another workspace has the
// same address, and only workspaces can be deleted.
func TestDeleteWorkspaceMetadata(t *testing.T) {
	a, _ := newTestAPI(t)
	md := headers{{Key: "authorization", Val: "Bearer secret"}}
	for id, addr := range map[string]string{"a": "only:1", "b": "shared:1", "c": "shared:1"} {
		a.setWorkspaceOptions(options{ID: workspacePrefix + id, Addr: addr})
		a.setMetadata(metadataKeyPrefix+hash(addr), md)
		a.setMetadata(reflectMetadataKeyPrefix+hash(addr), md)
	}

	if err := a.DeleteWorkspace(workspacePrefix + "a"); err != nil {
		t.Fatal(err)
	}
	if err := a.DeleteWorkspace(workspacePrefix + "b"); err != nil {
		t.Fatal(err)
	}
	if got, _ := a.GetMetadata("only:1"); len(got) != 0 {
		t.Errorf("metadata of the deleted workspace kept: %+v", got)
	}
	if got, _ := a.GetReflectMetadata("only:1"); len(got) != 0 {
		t.Errorf("reflection metadata of the deleted workspace kept: %+v", got)
	}
	if got, _ := a.GetMetadata("shared:1"); len(got) != 1 {
		t.Errorf("metadata still used by workspace c was deleted: %+v", got)
	}

	if err := a.DeleteWorkspace(metadataKeyPrefix + hash("shared:1")); err == nil {
		t.Error("deleted a key that is not a workspace")
	}
	if got, _ := a.GetMetadata("shared:1"); len(got) != 1 {
		t.Errorf("metadata deleted through DeleteWorkspace: %+v", got)
	}
}

func TestIsNewerVersion(t *testing.T) {
	for _, c := range []struct {
		latest, current string
		want            bool
	}{
		{"v0.6.1", "v0.6.0", true},
		{"v0.6.0", "v0.6.0", false},
		{"v0.6.0", "v0.6.0-rc3", true},
		{"v0.5.0", "v0.6.0", false},
		{"not-a-version", "v0.6.0", false}, // used to show the update banner
		{"v0.7.0", "garbage", false},
	} {
		if got := isNewerVersion(c.latest, c.current); got != c.want {
			t.Errorf("isNewerVersion(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

// A failed connection reports why, not only the connectivity state.
func TestConnectErrorHasCause(t *testing.T) {
	a, _ := newTestAPI(t)
	err := a.Connect(options{Addr: closedAddr(t), Plaintext: true}, headers{}, false)
	if err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("got %v, want the dial error (connection refused)", err)
	}
	err = a.Connect(options{Addr: "localhost:1", Rootca: "not a certificate"}, headers{}, false)
	if err == nil || !strings.Contains(err.Error(), "root CA") {
		t.Fatalf("got %v, want an invalid root CA error", err)
	}
}

func TestDialCheckTargets(t *testing.T) {
	addr, _ := startServer(t)
	for _, target := range []string{addr, "dns:///" + addr, "passthrough:///" + addr} {
		if err := dialCheck(target); err != nil {
			t.Errorf("dialCheck(%q) = %v", target, err)
		}
	}
	if err := dialCheck("unix:///no/such/socket"); err == nil {
		t.Error("dialCheck of a missing unix socket succeeded")
	}
	if err := dialCheck("xds:///whatever"); err != nil {
		t.Errorf("unknown scheme: %v, want nil", err)
	}
}

// An imported command without -d is an empty request with the command's
// (possibly no) headers; invalid -d data is reported.
func TestImportDataForms(t *testing.T) {
	a, rec := newConnectedAPI(t)
	last := func() recordedEvent {
		evs := rec.all(eventServicesSelectChanged)
		return evs[len(evs)-1]
	}

	if err := a.ImportCommand("grpcurl", `grpcurl -plaintext localhost:1 wombat.v1.RouteGuide.GetFeature`); err != nil {
		t.Fatal(err)
	}
	ev := last()
	if ev.data[1] != methodGetFeature || ev.data[2] != "{}" || ev.data[3] == nil {
		t.Fatalf("import without -d emitted %v, want %s, {} and empty headers", ev.data[1:], methodGetFeature)
	}

	if err := a.ImportCommand("grpcurl", `grpcurl -d '{"latitude":' localhost:1 wombat.v1.RouteGuide/GetFeature`); err == nil || !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("got %v, want an invalid JSON error", err)
	}
}
