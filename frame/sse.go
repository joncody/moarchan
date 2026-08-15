package frame

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/lib/pq"
)

type SSEEvent struct {
	Topic string          `json:"topic"`
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

type SSEHub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan SSEEvent]bool
	db          *sql.DB
	listener    *pq.Listener
	stopChan    chan struct{}
	closeOnce   sync.Once
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		subscribers: make(map[string]map[chan SSEEvent]bool),
		stopChan:    make(chan struct{}),
	}
}

// InitDBListener starts the PostgreSQL LISTEN worker on channel "moarchan_events"
func (h *SSEHub) InitDBListener(db *sql.DB, connStr string) {
	h.db = db
	if connStr == "" {
		return
	}

	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("[SSE Hub] Postgres listener warning: %v", err)
		}
	}

	listener := pq.NewListener(connStr, 10*time.Second, time.Minute, reportProblem)
	if err := listener.Listen("moarchan_events"); err != nil {
		log.Printf("[SSE Hub] Failed to start Postgres LISTEN on moarchan_events: %v", err)
		return
	}
	h.listener = listener

	go func() {
		for {
			select {
			case <-h.stopChan:
				return
			case notification, ok := <-h.listener.Notify:
				if !ok {
					return
				}
				if notification == nil {
					continue
				}

				var evt SSEEvent
				if err := json.Unmarshal([]byte(notification.Extra), &evt); err == nil {
					h.broadcastLocal(evt.Topic, evt.Event, evt.Data)
				}
			}
		}
	}()
}

func (h *SSEHub) Close() {
	h.closeOnce.Do(func() {
		close(h.stopChan)
		if h.listener != nil {
			h.listener.Close()
		}
	})
}

func (h *SSEHub) Subscribe(topic string) chan SSEEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan SSEEvent, 64)
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

func (h *SSEHub) broadcastLocal(topic, event string, payload json.RawMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	subs, exists := h.subscribers[topic]
	if !exists {
		return
	}

	evt := SSEEvent{
		Topic: topic,
		Event: event,
		Data:  payload,
	}

	for ch := range subs {
		select {
		case ch <- evt:
		default:
			// Non-blocking send: drop slow consumer to avoid blocking publisher
		}
	}
}

func (h *SSEHub) Broadcast(topic, event string, payload []byte) {
	evt := SSEEvent{
		Topic: topic,
		Event: event,
		Data:  json.RawMessage(payload),
	}

	notificationBytes, err := json.Marshal(evt)
	if err == nil && h.db != nil {
		// Broadcast across all cluster nodes using PostgreSQL pg_notify
		_, notifyErr := h.db.Exec(`SELECT pg_notify('moarchan_events', $1)`, string(notificationBytes))
		if notifyErr == nil {
			return
		}
		log.Printf("[SSE Hub] Postgres NOTIFY error, falling back to local broadcast: %v", notifyErr)
	}

	// Fallback to local in-process broadcast
	h.broadcastLocal(topic, event, evt.Data)
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

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ch := app.Hub.Subscribe(topic)
	defer app.Hub.Unsubscribe(topic, ch)

	// Handshake
	if _, err := fmt.Fprintf(w, "event: open\ndata: connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	// 15-second heartbeat ping
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", msg.Event, string(msg.Data)); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
