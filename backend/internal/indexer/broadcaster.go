package indexer

import (
	"encoding/json"
	"sync"
)

type SSEEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Broadcaster struct {
	mu      sync.RWMutex
	nextID  uint64
	clients map[uint64]chan SSEEvent
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{clients: make(map[uint64]chan SSEEvent)}
}

func (b *Broadcaster) Subscribe() (uint64, <-chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := b.nextID
	ch := make(chan SSEEvent, 32)
	b.clients[id] = ch
	return id, ch
}

func (b *Broadcaster) Unsubscribe(id uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.clients[id]; ok {
		delete(b.clients, id)
		close(ch)
	}
}

func (b *Broadcaster) Broadcast(event SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.clients {
		select {
		case ch <- event:
		default:
		}
	}
}
