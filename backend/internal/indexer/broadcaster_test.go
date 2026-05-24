package indexer

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBroadcasterSubscribeBroadcastUnsubscribe(t *testing.T) {
	b := NewBroadcaster()
	id, ch := b.Subscribe()

	payload := json.RawMessage(`{"marketId":"1"}`)
	b.Broadcast(SSEEvent{Type: EventMarketCreated, Payload: payload})

	select {
	case event := <-ch:
		if event.Type != EventMarketCreated || string(event.Payload) != string(payload) {
			t.Fatalf("unexpected event: %+v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}

	b.Unsubscribe(id)
	if _, ok := <-ch; ok {
		t.Fatal("expected channel to close after unsubscribe")
	}
}
