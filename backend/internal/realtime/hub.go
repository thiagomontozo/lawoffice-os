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
	mu             sync.RWMutex
	clients        map[string]map[chan Event]struct{}
	history        map[string][]Event
	nextID         uint64
	done           chan struct{}
	publish        func(string, Event) error
	replay         func(context.Context, string, string) ([]Event, error)
	onPublishError func(error)
	closed         bool
}

const historyLimit = 100

func New() *Hub {
	return &Hub{clients: map[string]map[chan Event]struct{}{}, history: map[string][]Event{}, done: make(chan struct{})}
}
func (h *Hub) Subscribe(ctx context.Context, firmID, lastID string) (<-chan Event, []Event, error) {
	ch := make(chan Event, 16)
	replay := []Event{}
	h.mu.Lock()
	if h.closed {
		close(ch)
		h.mu.Unlock()
		return ch, replay, nil
	} else {
		if h.clients[firmID] == nil {
			h.clients[firmID] = map[chan Event]struct{}{}
		}
		h.clients[firmID][ch] = struct{}{}
		if h.replay == nil {
			if id, err := strconv.ParseUint(lastID, 10, 64); err == nil {
				for _, event := range h.history[firmID] {
					eventID, parseErr := strconv.ParseUint(event.ID, 10, 64)
					if parseErr == nil && eventID > id {
						replay = append(replay, event)
					}
				}
			}
		}
	}
	replayer := h.replay
	h.mu.Unlock()
	if replayer != nil {
		var err error
		replay, err = replayer(ctx, firmID, lastID)
		if err != nil {
			h.mu.Lock()
			if _, exists := h.clients[firmID][ch]; exists {
				delete(h.clients[firmID], ch)
				close(ch)
			}
			h.mu.Unlock()
			return ch, nil, err
		}
	}
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
	return ch, replay, nil
}
func (h *Hub) Publish(firmID string, e Event) {
	h.mu.RLock()
	publisher := h.publish
	h.mu.RUnlock()
	if publisher != nil {
		if err := publisher(firmID, e); err == nil {
			return
		} else {
			h.mu.RLock()
			handler := h.onPublishError
			h.mu.RUnlock()
			if handler != nil {
				handler(err)
			}
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
	if e.ID == "" {
		h.nextID++
		e.ID = strconv.FormatUint(h.nextID, 10)
	}
	if e.PublishedAt.IsZero() {
		e.PublishedAt = time.Now().UTC()
	}
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
func (h *Hub) SetReplay(replay func(context.Context, string, string) ([]Event, error)) {
	h.mu.Lock()
	h.replay = replay
	h.mu.Unlock()
}
func (h *Hub) SetPublishErrorHandler(handler func(error)) {
	h.mu.Lock()
	h.onPublishError = handler
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
