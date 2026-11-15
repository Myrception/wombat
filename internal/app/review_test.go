package app

import (
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
