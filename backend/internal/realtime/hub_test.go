package realtime

import (
	"context"
	"testing"
	"time"
)

func TestHubIsolatesFirmsAndReplaysMissedEvents(t *testing.T) {
	hub := New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	alpha, _ := hub.Subscribe(ctx, "alpha", "")
	beta, _ := hub.Subscribe(ctx, "beta", "")
	hub.Publish("alpha", Event{Type: "matter.updated", ResourceType: "matter", ResourceID: "one"})
	var first Event
	select {
	case first = <-alpha:
	case <-time.After(time.Second):
		t.Fatal("alpha did not receive its event")
	}
	if first.ID == "" || first.PublishedAt.IsZero() {
		t.Fatalf("event metadata missing: %+v", first)
	}
	select {
	case event := <-beta:
		t.Fatalf("beta received alpha event: %+v", event)
	case <-time.After(20 * time.Millisecond):
	}
	hub.Publish("alpha", Event{Type: "task.updated", ResourceType: "task", ResourceID: "two"})
	reconnected, replay := hub.Subscribe(ctx, "alpha", first.ID)
	defer func() { _ = reconnected }()
	if len(replay) != 1 || replay[0].ResourceID != "two" {
		t.Fatalf("unexpected replay: %+v", replay)
	}
}

func TestHubCloseClosesSubscribers(t *testing.T) {
	hub := New()
	stream, _ := hub.Subscribe(context.Background(), "firm", "")
	hub.Close()
	if _, open := <-stream; open {
		t.Fatal("subscriber remained open")
	}
}
