package app

import (
	"sync"
	"time"
)

// eventRPCEvents carries the events of a call (rpc_started, the stats and
// payload events, rpc_ended) in batches; the frontend dispatches each one
// as if it had been emitted on its own.
const eventRPCEvents = "wombat:rpc_events"

// batchInterval is how long events of a call are collected before they are
// sent. A fast stream then costs a few UI updates a second instead of one
// per message.
const batchInterval = 50 * time.Millisecond

type rpcEvent struct {
	Name string        `json:"name"`
	Data []interface{} `json:"data"`
}

// eventBatcher collects events and sends them, in order, as one
// eventRPCEvents event: after batchInterval, or at once on flush.
type eventBatcher struct {
	send     func([]rpcEvent)
	interval time.Duration

	mu      sync.Mutex
	pending []rpcEvent
	timer   *time.Timer
}

func newEventBatcher(interval time.Duration, send func([]rpcEvent)) *eventBatcher {
	return &eventBatcher{send: send, interval: interval}
}

func (b *eventBatcher) emit(name string, data ...interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.pending = append(b.pending, rpcEvent{Name: name, Data: data})
	if b.timer == nil {
		b.timer = time.AfterFunc(b.interval, b.flush)
	}
}

// flush sends what has been collected. Sending happens under the lock, so
// batches cannot overtake each other.
func (b *eventBatcher) flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.timer != nil {
		b.timer.Stop()
		b.timer = nil
	}
	if len(b.pending) == 0 {
		return
	}
	evs := b.pending
	b.pending = nil
	b.send(evs)
}
