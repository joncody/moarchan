package frame

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type SSEEvent struct {
	Event string
	Data  []byte
}

type SSEHub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan SSEEvent]bool
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		subscribers: make(map[string]map[chan SSEEvent]bool),
	}
}

func (h *SSEHub) Subscribe(topic string) chan SSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan SSEEvent, 32)
	if _, exists := h.subscribers[topic]; !exists {
		h.subscribers[topic] = make(map[chan SSEEvent]bool)
	}
	h.subscribers[topic][ch] = true
	return ch
}

func (h *SSEHub) Unsubscribe(topic string, ch chan SSEEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if subs, exists := h.subscribers[topic]; exists {
		if _, present := subs[ch]; present {
			delete(subs, ch)
			close(ch)
		}
		if len(subs) == 0 {
			delete(h.subscribers, topic)
		}
	}
}

func (h *SSEHub) Broadcast(topic, event string, payload []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	subs, exists := h.subscribers[topic]
	if !exists {
		return
	}

	evt := SSEEvent{
		Event: event,
		Data:  payload,
	}

	for ch := range subs {
		select {
		case ch <- evt:
		default:
			// Non-blocking write: skip slow consumers to prevent blocking the publisher
		}
	}
}

func (app *App) SSEHandler(w http.ResponseWriter, r *http.Request) {
	topic := r.URL.Query().Get("topic")
	if topic == "" {
		http.Error(w, "Topic query parameter is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := app.Hub.Subscribe(topic)
	defer app.Hub.Unsubscribe(topic, ch)

	// Initial connection ack
	fmt.Fprintf(w, "event: open\ndata: connected\n\n")
	flusher.Flush()

	// 15-second Keep-Alive Heartbeat ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Event, string(msg.Data))
			flusher.Flush()
		}
	}
}
