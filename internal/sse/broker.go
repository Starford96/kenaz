// Package sse implements a Server-Sent Events broker for real-time updates.
package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Event represents an SSE event to broadcast.
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Broker manages SSE client connections and broadcasts events.
type Broker struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}

	// Graph throttle.
	lastGraph time.Time
	graphMin  time.Duration
}

// NewBroker creates a new SSE broker with the given graph throttle interval.
func NewBroker(graphThrottle time.Duration) *Broker {
	if graphThrottle <= 0 {
		graphThrottle = 2 * time.Second
	}
	return &Broker{
		clients:  make(map[chan []byte]struct{}),
		graphMin: graphThrottle,
	}
}

// Subscribe adds a new client and returns its channel.
func (b *Broker) Subscribe() chan []byte {
	ch := make(chan []byte, 64)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a client and closes its channel.
func (b *Broker) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

// ClientCount returns the number of connected clients.
func (b *Broker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// Publish sends an event to all connected clients.
func (b *Broker) Publish(event Event) {
	payload, err := json.Marshal(event.Data)
	if err != nil {
		return
	}
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", event.Type, payload)
	raw := []byte(msg)

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.clients {
		select {
		case ch <- raw:
		default:
			// Client buffer full; skip to avoid blocking.
		}
	}
}

// PublishNoteEvent publishes a note change and a throttled graph.updated event.
func (b *Broker) PublishNoteEvent(kind, path string) {
	data := map[string]string{"path": path}

	switch kind {
	case "created":
		b.Publish(Event{Type: "note.created", Data: data})
	case "updated":
		b.Publish(Event{Type: "note.updated", Data: data})
	case "deleted":
		b.Publish(Event{Type: "note.deleted", Data: data})
	}

	// Throttle graph.updated.
	b.mu.Lock()
	now := time.Now()
	if now.Sub(b.lastGraph) >= b.graphMin {
		b.lastGraph = now
		b.mu.Unlock()
		b.Publish(Event{Type: "graph.updated", Data: map[string]string{}})
	} else {
		b.mu.Unlock()
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
