package app

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"

	"wombat/internal/server"
)

const (
	methodGetFeature   = "/wombat.v1.RouteGuide/GetFeature"
	methodListFeatures = "/wombat.v1.RouteGuide/ListFeatures"
	methodRecordRoute  = "/wombat.v1.RouteGuide/RecordRoute"
	methodRouteChat    = "/wombat.v1.RouteGuide/RouteChat"

	pointJSON = `{"latitude":409146138,"longitude":-746188906}`
	rectJSON  = `{"lo":{"latitude":409000000,"longitude":-747000000},"hi":{"latitude":410000000,"longitude":-746000000}}`
	noteJSON  = `{"location":{"latitude":1,"longitude":2},"message":"hi"}`
)

// recorder is a fake wailsRuntime that records emitted events.
type recorder struct {
	mu     sync.Mutex
	events []recordedEvent
}

type recordedEvent struct {
	name string
	data []interface{}
}

func (r *recorder) Emit(name string, data ...interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, recordedEvent{name, data})
}
func (r *recorder) LogDebug(string)   {}
func (r *recorder) LogInfo(string)    {}
func (r *recorder) LogWarning(string) {}
func (r *recorder) LogError(string)   {}

func (r *recorder) all(name string) []recordedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []recordedEvent
	for _, e := range r.events {
		if e.name == name {
			out = append(out, e)
		}
	}
	return out
}

func (r *recorder) count(name string) int { return len(r.all(name)) }

func (r *recorder) rpcEnds() []rpcEnd {
	var out []rpcEnd
	for _, e := range r.all(eventRPCEnded) {
		out = append(out, e.data[0].(rpcEnd))
	}
	return out
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func recvErr(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for Send to return")
		return nil
	}
}

func startServer(t *testing.T, opts ...grpc.ServerOption) (string, *grpc.Server) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	gs := server.NewGRPCServer(opts...)
	go func() { _ = gs.Serve(lis) }()
	t.Cleanup(gs.Stop)
	return lis.Addr().String(), gs
}

func newTestAPI(t *testing.T) (*api, *recorder) {
	t.Helper()
	st, err := newStore(t.TempDir(), newStoreLogger(context.Background()))
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	a := NewApp()
	a.ctx = context.Background()
	a.rt = rec
	a.store = st
	t.Cleanup(func() { a.Shutdown(context.Background()) })
	return a, rec
}

func connectTo(t *testing.T, a *api, opts options) {
	t.Helper()
	if err := a.Connect(opts, headers{}, false); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	waitFor(t, "proto files", func() bool { return a.getProtofiles() != nil })
}

func newConnectedAPI(t *testing.T) (*api, *recorder) {
	addr, _ := startServer(t)
	a, rec := newTestAPI(t)
	connectTo(t, a, options{Addr: addr, Plaintext: true, Reflect: true})
	return a, rec
}

// startStream starts a streaming Send in the background and waits until the
// call is registered.
func startStream(t *testing.T, a *api, method, body string) <-chan error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- a.Send(method, body, headers{}) }()
	waitFor(t, "call to start", func() bool { return a.currentCall() != nil })
	return done
}

func TestUnary(t *testing.T) {
	a, rec := newConnectedAPI(t)

	if err := a.Send(methodGetFeature, pointJSON, headers{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, "rpc_ended", func() bool { return rec.count(eventRPCEnded) == 1 })
	if got := rec.rpcEnds()[0].StatusCode; got != int32(codes.OK) {
		t.Fatalf("status = %v, want OK", codes.Code(got))
	}
	if a.currentCall() != nil {
		t.Fatal("call still registered after Send returned")
	}
}

func TestServerStream(t *testing.T) {
	a, rec := newConnectedAPI(t)

	if err := a.Send(methodListFeatures, rectJSON, headers{}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, "rpc_ended", func() bool { return rec.count(eventRPCEnded) == 1 })
	if rec.count(eventInPayloadReceived) == 0 {
		t.Fatal("no responses received")
	}
	if got := rec.rpcEnds()[0].StatusCode; got != int32(codes.OK) {
		t.Fatalf("status = %v, want OK", codes.Code(got))
	}
}

func TestClientStream(t *testing.T) {
	a, rec := newConnectedAPI(t)

	done := startStream(t, a, methodRecordRoute, pointJSON)
	for i := 0; i < 3; i++ {
		if err := a.Send(methodRecordRoute, pointJSON, headers{}); err != nil {
			t.Fatalf("push %d: %v", i, err)
		}
	}
	a.CloseSend()
	a.CloseSend() // must be idempotent (used to panic: close of closed channel)

	if err := recvErr(t, done); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, "rpc_ended", func() bool { return rec.count(eventRPCEnded) == 1 })
	payloads := rec.all(eventInPayloadReceived)
	// prototext randomises whitespace, so normalise it before comparing.
	if len(payloads) != 1 || !strings.Contains(strings.Join(strings.Fields(payloads[0].data[0].(string)), " "), "point_count: 4") {
		t.Fatalf("unexpected summary: %+v", payloads)
	}
}

func TestBidiStream(t *testing.T) {
	a, rec := newConnectedAPI(t)

	done := startStream(t, a, methodRouteChat, noteJSON)
	for i := 0; i < 2; i++ {
		if err := a.Send(methodRouteChat, noteJSON, headers{}); err != nil {
			t.Fatalf("push %d: %v", i, err)
		}
	}
	waitFor(t, "3 echoes", func() bool { return rec.count(eventInPayloadReceived) == 3 })
	a.CloseSend()

	if err := recvErr(t, done); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, "rpc_ended", func() bool { return rec.count(eventRPCEnded) == 1 })
	if got := rec.rpcEnds()[0].StatusCode; got != int32(codes.OK) {
		t.Fatalf("status = %v, want OK", codes.Code(got))
	}
}

func TestCancelEmitsSingleRPCEnded(t *testing.T) {
	a, rec := newConnectedAPI(t)

	done := startStream(t, a, methodRecordRoute, pointJSON)
	a.Cancel()
	if err := recvErr(t, done); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, "rpc_ended", func() bool { return rec.count(eventRPCEnded) >= 1 })
	time.Sleep(200 * time.Millisecond)
	ends := rec.rpcEnds()
	if len(ends) != 1 {
		t.Fatalf("got %d rpc_ended events, want 1", len(ends))
	}
	if ends[0].StatusCode != int32(codes.Canceled) {
		t.Fatalf("status = %v, want Canceled", codes.Code(ends[0].StatusCode))
	}
}

func TestSendWhileBusy(t *testing.T) {
	a, _ := newConnectedAPI(t)

	done := startStream(t, a, methodRecordRoute, pointJSON)
	if err := a.Send(methodGetFeature, pointJSON, headers{}); err != errCallInProgress {
		t.Fatalf("Send while busy = %v, want errCallInProgress", err)
	}
	a.Cancel()
	_ = recvErr(t, done) // only that it ends
}

func TestSendWithoutConnection(t *testing.T) {
	a, _ := newTestAPI(t)
	// Must fail cleanly (no nil client dereference) when not connected.
	if err := a.Send(methodGetFeature, pointJSON, headers{}); err == nil {
		t.Fatal("expected an error")
	}
}

// TestStreamControlStress hammers a client stream with concurrent pushes,
// CloseSend and Cancel calls. Run with -race. Any panic or hang fails it.
func TestStreamControlStress(t *testing.T) {
	a, _ := newConnectedAPI(t)

	for i := 0; i < 25; i++ {
		done := startStream(t, a, methodRecordRoute, pointJSON)
		var wg sync.WaitGroup
		for j := 0; j < 5; j++ {
			wg.Add(3)
			// A push that loses the race against Cancel legitimately
			// starts a new stream; those are cancelled below.
			go func() { defer wg.Done(); _ = a.Send(methodRecordRoute, pointJSON, headers{}) }()
			go func() { defer wg.Done(); a.CloseSend() }()
			go func() { defer wg.Done(); a.Cancel() }()
		}
		_ = recvErr(t, done) // only that it ends

		allDone := make(chan struct{})
		go func() { wg.Wait(); close(allDone) }()
		deadline := time.After(10 * time.Second)
	drain:
		for {
			select {
			case <-allDone:
				break drain
			case <-deadline:
				t.Fatal("stream control calls did not return")
			case <-time.After(10 * time.Millisecond):
				a.Cancel()
			}
		}
		waitFor(t, "call to end", func() bool { return a.currentCall() == nil })
	}
}

func TestReconnectDuringStream(t *testing.T) {
	addr, _ := startServer(t)
	a, _ := newTestAPI(t)
	opts := options{Addr: addr, Plaintext: true, Reflect: true}
	connectTo(t, a, opts)

	done := startStream(t, a, methodRouteChat, noteJSON)
	connectTo(t, a, opts) // must cancel the in-flight stream
	_ = recvErr(t, done)  // only that it ends

	if err := a.Send(methodGetFeature, pointJSON, headers{}); err != nil {
		t.Fatalf("Send after reconnect: %v", err)
	}
}

// TestMonitorSurvivesIdle checks that connection state changes are still
// reported after the connection has been idle for a while. Previously the
// monitor gave up after 5s without a state change.
func TestMonitorSurvivesIdle(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	addr, gs := startServer(t)
	a, rec := newTestAPI(t)
	connectTo(t, a, options{Addr: addr, Plaintext: true, Reflect: true})

	time.Sleep(6 * time.Second)
	before := rec.count(eventClientStateChanged)
	gs.Stop()
	waitFor(t, "state change after server stop", func() bool {
		return rec.count(eventClientStateChanged) > before
	})
}

// TestTLSReconnectHandshakeFailure reproduces the "send on closed channel"
// crash: after a successful TLS connect, a failed handshake during reconnect
// used to panic inside a gRPC goroutine and take the whole app down.
func TestTLSReconnectHandshakeFailure(t *testing.T) {
	cert := selfSignedCert(t)
	addr, gs := startServer(t, grpc.Creds(credentials.NewTLS(&tls.Config{Certificates: []tls.Certificate{cert}})))

	a, _ := newTestAPI(t)
	connectTo(t, a, options{Addr: addr, Insecure: true, Reflect: true})

	// Replace the server with one that kills every connection, so all
	// further TLS handshakes fail.
	gs.Stop()
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lis.Close() }()
	go func() {
		for {
			c, err := lis.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()

	cl := a.getClient()
	deadline := time.Now().Add(5 * time.Second)
	for i := 0; time.Now().Before(deadline); i++ {
		a.RetryConnection()
		if i > 3 && cl.conn.GetState() == connectivity.TransientFailure {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	// Give gRPC a moment to run any further handshakes.
	time.Sleep(500 * time.Millisecond)
}

func selfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}
