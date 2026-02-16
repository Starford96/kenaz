package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSubscribeUnsubscribe(t *testing.T) {
	b := NewBroker(100 * time.Millisecond)
	defer b.Close()
	if b.ClientCount() != 0 {
		t.Fatalf("expected 0 clients")
	}
	ch := b.Subscribe()
	if b.ClientCount() != 1 {
		t.Fatalf("expected 1 client")
	}
	b.Unsubscribe(ch)
	if b.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after unsub")
	}
}

func TestPublishDelivery(t *testing.T) {
	b := NewBroker(100 * time.Millisecond)
	defer b.Close()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Publish(Event{Type: "note.created", Data: map[string]string{"path": "a.md"}})

	select {
	case msg := <-ch:
		s := string(msg)
		if !strings.Contains(s, "event: note.created") {
			t.Errorf("missing event type in %q", s)
		}
		if !strings.Contains(s, `"path":"a.md"`) {
			t.Errorf("missing data in %q", s)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestPublishNoteEvent_GraphThrottle(t *testing.T) {
	b := NewBroker(500 * time.Millisecond)
	defer b.Close()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// First event should trigger graph.updated.
	b.PublishNoteEvent("created", "a.md")
	// Second event immediately should NOT trigger another graph.updated.
	b.PublishNoteEvent("updated", "b.md")

	// Drain and count events.
	time.Sleep(50 * time.Millisecond)
	graphCount := 0
	noteCount := 0
loop:
	for {
		select {
		case msg := <-ch:
			s := string(msg)
			if strings.Contains(s, "graph.updated") {
				graphCount++
			} else {
				noteCount++
			}
		default:
			break loop
		}
	}

	if noteCount != 2 {
		t.Errorf("note events = %d, want 2", noteCount)
	}
	if graphCount != 1 {
		t.Errorf("graph events = %d, want 1 (throttled)", graphCount)
	}
}

func TestSSEHandler(t *testing.T) {
	b := NewBroker(100 * time.Millisecond)
	defer b.Close()

	// Start handler in background.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/events", nil)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		b.ServeHTTP(w, req)
		close(done)
	}()

	// Give handler time to subscribe.
	time.Sleep(50 * time.Millisecond)
	if b.ClientCount() != 1 {
		t.Fatalf("expected 1 client from handler")
	}

	b.Publish(Event{Type: "note.updated", Data: map[string]string{"path": "x.md"}})
	time.Sleep(50 * time.Millisecond)

	// Cancel context to disconnect.
	cancel()
	<-done

	body := w.Body.String()
	if !strings.Contains(body, "event: note.updated") {
		t.Errorf("handler output missing event: %q", body)
	}

	// Client should be cleaned up.
	time.Sleep(50 * time.Millisecond)
	if b.ClientCount() != 0 {
		t.Errorf("client not cleaned up after disconnect")
	}
}

func TestPublishDropsOnFullBuffer(t *testing.T) {
	b := NewBroker(time.Second)
	defer b.Close()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	// Fill buffer (capacity 64) and then one more should not block.
	for i := 0; i < 70; i++ {
		b.Publish(Event{Type: "test", Data: map[string]string{"i": "x"}})
	}
	// If we reach here without deadlock, the test passes.
}

func TestCloseClosesSubscribersAndStopsOperations(t *testing.T) {
	b := NewBroker(100 * time.Millisecond)
	ch := b.Subscribe()
	if b.ClientCount() != 1 {
		t.Fatalf("expected 1 client")
	}

	b.Close()

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected subscriber channel to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for channel close")
	}

	if b.ClientCount() != 0 {
		t.Fatalf("expected 0 clients after close")
	}

	// Should be safe no-op after close.
	b.Publish(Event{Type: "note.updated", Data: map[string]string{"path": "x.md"}})
	b.PublishNoteEvent("updated", "x.md")
}
