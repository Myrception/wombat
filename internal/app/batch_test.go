package app

import (
	"sync"
	"testing"
	"time"
)

// Many events in quick succession arrive in a few batches, in order.
func TestEventBatcher(t *testing.T) {
	var mu sync.Mutex
	var batches [][]rpcEvent
	b := newEventBatcher(20*time.Millisecond, func(evs []rpcEvent) {
		mu.Lock()
		defer mu.Unlock()
		batches = append(batches, evs)
	})

	const n = 5000
	for i := 0; i < n; i++ {
		b.emit("e", i)
	}
	waitFor(t, "a batch", func() bool { mu.Lock(); defer mu.Unlock(); return len(batches) > 0 })
	b.emit("e", n)
	b.flush() // sends at once

	mu.Lock()
	defer mu.Unlock()
	if len(batches) > 5 {
		t.Errorf("%d events came in %d batches, want a few", n+1, len(batches))
	}
	next := 0
	for _, batch := range batches {
		for _, e := range batch {
			if e.Data[0].(int) != next {
				t.Fatalf("event %v out of order, want %d", e.Data[0], next)
			}
			next++
		}
	}
	if next != n+1 {
		t.Fatalf("got %d events, want %d", next, n+1)
	}
}

// The events of a call reach the frontend in order, starting with
// rpc_started and ending with rpc_ended, and in fewer batches than events.
func TestCallEventsBatched(t *testing.T) {
	a, rec := newConnectedAPI(t)

	done := startStream(t, a, methodRouteChat, noteJSON)
	for i := 0; i < 50; i++ {
		if err := a.Send(methodRouteChat, noteJSON, headers{}); err != nil {
			t.Fatalf("push %d: %v", i, err)
		}
	}
	waitFor(t, "51 echoes", func() bool { return rec.count(eventInPayloadReceived) == 51 })
	a.CloseSend()
	if err := recvErr(t, done); err != nil {
		t.Fatalf("Send: %v", err)
	}
	waitFor(t, "rpc_ended", func() bool { return rec.count(eventRPCEnded) == 1 })

	rec.mu.Lock()
	defer rec.mu.Unlock()
	var names []string
	for _, e := range rec.events {
		switch e.name {
		case eventRPCStarted, eventRPCEnded, eventStatBegin, eventStatEnd, eventInPayloadReceived:
			names = append(names, e.name)
		}
	}
	if names[0] != eventRPCStarted || names[1] != eventStatBegin {
		t.Errorf("first events = %v", names[:2])
	}
	if l := len(names); names[l-2] != eventStatEnd || names[l-1] != eventRPCEnded {
		t.Errorf("last events = %v", names[l-2:])
	}
	total := 0
	for _, e := range rec.events {
		if e.name != eventClientStateChanged {
			total++
		}
	}
	if rec.batches >= total/2 {
		t.Errorf("%d batches for %d events", rec.batches, total)
	}
	t.Logf("%d events in %d batches", total, rec.batches)
}
