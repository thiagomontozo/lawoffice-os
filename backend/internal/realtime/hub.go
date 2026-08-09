package realtime

import (
	"context"
	"sync"
)

type Event struct {
	Type         string `json:"type"`
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
}
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[chan Event]struct{}
	closed  bool
}

func New() *Hub { return &Hub{clients: map[string]map[chan Event]struct{}{}} }
func (h *Hub) Subscribe(ctx context.Context, firmID string) <-chan Event {
	ch := make(chan Event, 16)
	h.mu.Lock()
	if h.closed {
		close(ch)
	} else {
		if h.clients[firmID] == nil {
			h.clients[firmID] = map[chan Event]struct{}{}
		}
		h.clients[firmID][ch] = struct{}{}
	}
	h.mu.Unlock()
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		if _, ok := h.clients[firmID][ch]; ok {
			delete(h.clients[firmID], ch)
			close(ch)
		}
		h.mu.Unlock()
	}()
	return ch
}
func (h *Hub) Publish(firmID string, e Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients[firmID] {
		select {
		case ch <- e:
		default:
		}
	}
}
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	for _, set := range h.clients {
		for ch := range set {
			close(ch)
		}
	}
	h.clients = map[string]map[chan Event]struct{}{}
}
