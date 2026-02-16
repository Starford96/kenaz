// Package sse implements a Server-Sent Events broker for real-time updates.
package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

// Event represents an SSE event to broadcast.
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type noteEventReq struct {
	kind string
	path string
}

// Broker manages SSE client connections and broadcasts events.
//
// Concurrency model: a single internal event loop (goroutine) owns mutable state
// (clients + graph throttle timestamp). Public methods communicate with this loop
// through channels, so no mutexes are required.
type Broker struct {
	graphMin time.Duration

	subscribeCh   chan chan []byte
	unsubscribeCh chan chan []byte
	publishCh     chan Event
	noteEventCh   chan noteEventReq
	countReqCh    chan chan int

	stopCh  chan struct{}
	stopped chan struct{}
	closed  atomic.Bool
}

// NewBroker creates a new SSE broker with the given graph throttle interval.
func NewBroker(graphThrottle time.Duration) *Broker {
	if graphThrottle <= 0 {
		graphThrottle = 2 * time.Second
	}

	b := &Broker{
		graphMin:       graphThrottle,
		subscribeCh:   make(chan chan []byte),
		unsubscribeCh: make(chan chan []byte),
		publishCh:     make(chan Event, 256),
		noteEventCh:   make(chan noteEventReq, 256),
		countReqCh:    make(chan chan int),
		stopCh:        make(chan struct{}),
		stopped:       make(chan struct{}),
	}

	go b.run()
	return b
}

func (b *Broker) run() {
	defer close(b.stopped)

	clients := make(map[chan []byte]struct{})
	var lastGraph time.Time

	broadcast := func(event Event) {
		payload, err := json.Marshal(event.Data)
		if err != nil {
			return
		}
		msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, payload)
		raw := []byte(msg)

		for ch := range clients {
			select {
			case ch <- raw:
			default:
				// Client buffer full; skip to avoid blocking broker loop.
			}
		}
	}

	for {
		select {
		case <-b.stopCh:
			for ch := range clients {
				close(ch)
			}
			return

		case ch := <-b.subscribeCh:
			clients[ch] = struct{}{}

		case ch := <-b.unsubscribeCh:
			if _, ok := clients[ch]; ok {
				delete(clients, ch)
				close(ch)
			}

		case event := <-b.publishCh:
			broadcast(event)

		case req := <-b.noteEventCh:
			data := map[string]string{"path": req.path}
			switch req.kind {
			case "created":
				broadcast(Event{Type: "note.created", Data: data})
			case "updated":
				broadcast(Event{Type: "note.updated", Data: data})
			case "deleted":
				broadcast(Event{Type: "note.deleted", Data: data})
			}

			now := time.Now()
			if now.Sub(lastGraph) >= b.graphMin {
				lastGraph = now
				broadcast(Event{Type: "graph.updated", Data: map[string]string{}})
			}

		case resp := <-b.countReqCh:
			resp <- len(clients)
		}
	}
}

// Close gracefully stops broker loop and closes all client channels.
func (b *Broker) Close() {
	if b.closed.CompareAndSwap(false, true) {
		close(b.stopCh)
	}
	<-b.stopped
}

// Subscribe adds a new client and returns its channel.
func (b *Broker) Subscribe() chan []byte {
	ch := make(chan []byte, 64)
	if b.closed.Load() {
		close(ch)
		return ch
	}

	select {
	case b.subscribeCh <- ch:
	case <-b.stopped:
		close(ch)
	}

	return ch
}

// Unsubscribe removes a client and closes its channel.
func (b *Broker) Unsubscribe(ch chan []byte) {
	if b.closed.Load() {
		return
	}
	select {
	case b.unsubscribeCh <- ch:
	case <-b.stopped:
	}
}

// ClientCount returns the number of connected clients.
func (b *Broker) ClientCount() int {
	if b.closed.Load() {
		return 0
	}

	resp := make(chan int, 1)
	select {
	case b.countReqCh <- resp:
	case <-b.stopped:
		return 0
	}

	select {
	case n := <-resp:
		return n
	case <-b.stopped:
		return 0
	}
}

// Publish sends an event to all connected clients.
func (b *Broker) Publish(event Event) {
	if b.closed.Load() {
		return
	}
	select {
	case b.publishCh <- event:
	case <-b.stopped:
	}
}

// PublishNoteEvent publishes a note change and a throttled graph.updated event.
func (b *Broker) PublishNoteEvent(kind, path string) {
	if b.closed.Load() {
		return
	}
	select {
	case b.noteEventCh <- noteEventReq{kind: kind, path: path}:
	case <-b.stopped:
	}
}

// ServeHTTP is the SSE endpoint handler (GET /api/events).
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			_, _ = w.Write(msg)
			flusher.Flush()
		}
	}
}
