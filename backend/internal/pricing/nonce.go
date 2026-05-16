package pricing

import (
	"context"
	"math/big"
	"sync"
)

type NonceStore interface {
	Next(ctx context.Context) (*big.Int, error)
}

// MemoryNonceStore issues globally unique nonces for local MVP use.
type MemoryNonceStore struct {
	mu   sync.Mutex
	next *big.Int
}

func NewMemoryNonceStore(start *big.Int) *MemoryNonceStore {
	if start == nil || start.Sign() < 0 {
		start = big.NewInt(0)
	}
	return &MemoryNonceStore{next: new(big.Int).Set(start)}
}

func (s *MemoryNonceStore) Next(context.Context) (*big.Int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.next.Add(s.next, big.NewInt(1))
	return new(big.Int).Set(s.next), nil
}
