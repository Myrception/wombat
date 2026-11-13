package app

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

var (
	errCallInProgress = errors.New("another request is still in progress")
	errStreamClosed   = errors.New("the request stream has already been closed")
	errStreamEnded    = errors.New("the request stream has already ended")
)

// ctxCallKey is used to attach the *rpcCall to the RPC context so the stats
// handler can mark the call as ended.
type ctxCallKey struct{}

// rpcCall holds the state of one in-flight RPC.
//
// Ownership rules (these are what make it safe to use from concurrent Wails
// binding calls):
//   - reqs is never closed. Senders use push, which also watches closeSend and
//     done, so a push can never block forever or panic.
//   - closeSend is closed exactly once, via closeSending.
//   - done is closed exactly once, by api.endCall, when Send returns.
type rpcCall struct {
	method    string
	cancel    context.CancelFunc
	reqs      chan proto.Message // nil unless the method is client streaming
	closeSend chan struct{}
	closeOnce sync.Once
	done      chan struct{}

	// ended is set once gRPC reports stats.End, which emits the rpc_ended
	// event to the frontend. If it is never set (e.g. the stream could not be
	// created) Send emits rpc_ended itself so the UI does not get stuck.
	ended atomic.Bool
}

func newRPCCall(method string, cancel context.CancelFunc, clientStream bool) *rpcCall {
	c := &rpcCall{
		method:    method,
		cancel:    cancel,
		closeSend: make(chan struct{}),
		done:      make(chan struct{}),
	}
	if clientStream {
		c.reqs = make(chan proto.Message)
	}
	return c
}

// closeSending half-closes the request stream. Safe to call multiple times.
func (c *rpcCall) closeSending() {
	c.closeOnce.Do(func() { close(c.closeSend) })
}

// push hands an additional request message to the stream pump.
func (c *rpcCall) push(m proto.Message) error {
	if c.reqs == nil {
		return errCallInProgress
	}
	select {
	case <-c.closeSend:
		return errStreamClosed
	case <-c.done:
		return errStreamEnded
	default:
	}
	select {
	case c.reqs <- m:
		return nil
	case <-c.closeSend:
		return errStreamClosed
	case <-c.done:
		return errStreamEnded
	}
}

// pump forwards pushed messages to the stream until sending is closed by the
// user, the call is cancelled, or a send fails. It always half-closes the
// stream before returning. It must be the only goroutine sending on stream.
func (c *rpcCall) pump(ctx context.Context, stream grpc.ClientStream) error {
	// The real outcome is reported by RecvMsg; CloseSend only fails if the
	// stream is already broken.
	defer func() { _ = stream.CloseSend() }()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-c.closeSend:
			return nil
		case m := <-c.reqs:
			if err := stream.SendMsg(m); err != nil {
				c.closeSending()
				if err == io.EOF {
					// The server ended the stream; the real status is
					// returned by RecvMsg and reported via stats.End.
					return nil
				}
				return err
			}
		}
	}
}

func (a *api) currentCall() *rpcCall {
	a.callMu.Lock()
	defer a.callMu.Unlock()
	return a.call
}

// beginCall registers c as the in-flight call. It fails if there already is one.
func (a *api) beginCall(c *rpcCall) error {
	a.callMu.Lock()
	defer a.callMu.Unlock()
	if a.call != nil {
		return errCallInProgress
	}
	a.call = c
	return nil
}

func (a *api) endCall(c *rpcCall) {
	a.callMu.Lock()
	if a.call == c {
		a.call = nil
	}
	a.callMu.Unlock()
	c.cancel()
	close(c.done)
}

// cancelCall cancels the in-flight call, if any.
func (a *api) cancelCall() bool {
	c := a.currentCall()
	if c == nil {
		return false
	}
	c.cancel()
	return true
}
