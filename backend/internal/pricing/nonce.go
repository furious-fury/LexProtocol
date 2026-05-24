package pricing

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"sync"
)

type NonceStore interface {
	Next(ctx context.Context) (*big.Int, error)
	Record(ctx context.Context, submission SettlementSubmission) error
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

func (s *MemoryNonceStore) Record(context.Context, SettlementSubmission) error {
	return nil
}

// PostgresNonceStore issues process-safe nonces from a database sequence and
// records signed submissions for replay/audit visibility.
type PostgresNonceStore struct {
	db *sql.DB
}

func NewPostgresNonceStore(db *sql.DB) *PostgresNonceStore {
	return &PostgresNonceStore{db: db}
}

func (s *PostgresNonceStore) Next(ctx context.Context) (*big.Int, error) {
	var nonceText string
	if err := s.db.QueryRowContext(ctx, `SELECT nextval('pricing_nonce_seq')::text`).Scan(&nonceText); err != nil {
		return nil, fmt.Errorf("allocate pricing nonce: %w", err)
	}
	nonce, ok := new(big.Int).SetString(nonceText, 10)
	if !ok || nonce.Sign() <= 0 {
		return nil, fmt.Errorf("database returned invalid nonce %q", nonceText)
	}
	return nonce, nil
}

func (s *PostgresNonceStore) Record(ctx context.Context, submission SettlementSubmission) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pricing_nonces (
			nonce, market_id, outcome, expiry, signature
		)
		VALUES ($1, $2, $3, $4, $5)
	`, submission.Nonce.String(), submission.MarketID.String(), submission.OutcomeID, int64(submission.Expiry), submission.Signature)
	if err != nil {
		return fmt.Errorf("record pricing nonce: %w", err)
	}
	return nil
}
