package realtime

import (
	"context"
	"strconv"
	"sync"
	"time"
)

type Event struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	ResourceType string    `json:"resourceType"`
	ResourceID   string    `json:"resourceId"`
	PublishedAt  time.Time `json:"publishedAt"`
}
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[chan Event]struct{}
	history map[string][]Event
	nextID  uint64
	done    chan struct{}
	publish func(string, Event) error
	closed  bool
}

const historyLimit = 100

func New() *Hub {
	return &Hub{clients: map[string]map[chan Event]struct{}{}, history: map[string][]Event{}, done: make(chan struct{})}
}
func (h *Hub) Subscribe(ctx context.Context, firmID, lastID string) (<-chan Event, []Event) {
	ch := make(chan Event, 16)
	replay := []Event{}
	h.mu.Lock()
	if h.closed {
		close(ch)
	} else {
		if h.clients[firmID] == nil {
			h.clients[firmID] = map[chan Event]struct{}{}
		}
		h.clients[firmID][ch] = struct{}{}
		if id, err := strconv.ParseUint(lastID, 10, 64); err == nil {
			for _, event := range h.history[firmID] {
				eventID, parseErr := strconv.ParseUint(event.ID, 10, 64)
				if parseErr == nil && eventID > id {
					replay = append(replay, event)
				}
			}
		}
	}
	h.mu.Unlock()
	go func() {
		select {
		case <-ctx.Done():
		case <-h.done:
		}
		h.mu.Lock()
		if _, ok := h.clients[firmID][ch]; ok {
			delete(h.clients[firmID], ch)
			close(ch)
		}
		h.mu.Unlock()
	}()
	return ch, replay
}
func (h *Hub) Publish(firmID string, e Event) {
	h.mu.RLock()
	publisher := h.publish
	h.mu.RUnlock()
	if publisher != nil {
		if err := publisher(firmID, e); err == nil {
			return
		}
	}
	h.Deliver(firmID, e)
}

func (h *Hub) Deliver(firmID string, e Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.nextID++
	e.ID = strconv.FormatUint(h.nextID, 10)
	e.PublishedAt = time.Now().UTC()
	h.history[firmID] = append(h.history[firmID], e)
	if len(h.history[firmID]) > historyLimit {
		h.history[firmID] = append([]Event(nil), h.history[firmID][len(h.history[firmID])-historyLimit:]...)
	}
	for ch := range h.clients[firmID] {
		select {
		case ch <- e:
		default:
		}
	}
}

func (h *Hub) SetPublisher(publisher func(string, Event) error) {
	h.mu.Lock()
	h.publish = publisher
	h.mu.Unlock()
}
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	close(h.done)
	for _, set := range h.clients {
		for ch := range set {
			close(ch)
		}
	}
	h.clients = map[string]map[chan Event]struct{}{}
	h.history = map[string][]Event{}
}
