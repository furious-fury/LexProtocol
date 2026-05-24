package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type Store struct {
	db *sql.DB
}

type PendingEvent struct {
	ID          int64
	EventType   string
	Address     string
	BlockNumber uint64
	BlockHash   string
	TxHash      string
	LogIndex    uint
	Payload     json.RawMessage
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) SaveBlock(ctx context.Context, number uint64, hash common.Hash, parent common.Hash, confirmed bool) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO blocks (block_number, block_hash, parent_hash, confirmed)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (block_number) DO UPDATE SET
			block_hash = EXCLUDED.block_hash,
			parent_hash = EXCLUDED.parent_hash,
			confirmed = blocks.confirmed OR EXCLUDED.confirmed
	`, int64(number), hash.Hex(), parent.Hex(), confirmed)
	return err
}

func (s *Store) InsertPendingEvent(ctx context.Context, event PendingEvent) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pending_events (
			event_type, address, block_number, block_hash, tx_hash, log_index, payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tx_hash, log_index) DO NOTHING
	`, event.EventType, event.Address, int64(event.BlockNumber), event.BlockHash, event.TxHash, int64(event.LogIndex), string(event.Payload))
	return err
}

func (s *Store) KnownMarkets(ctx context.Context) (map[common.Address]bool, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT market_address FROM markets
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	known := make(map[common.Address]bool)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if common.IsHexAddress(raw) {
			known[common.HexToAddress(raw)] = true
		}
	}
	return known, rows.Err()
}

func (s *Store) ConfirmableEvents(ctx context.Context, maxBlock uint64) ([]PendingEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, event_type, address, block_number, block_hash, tx_hash, log_index, payload
		FROM pending_events
		WHERE block_number <= $1
		ORDER BY block_number, tx_hash, log_index
	`, int64(maxBlock))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []PendingEvent
	for rows.Next() {
		var event PendingEvent
		var blockNumber int64
		var logIndex int64
		if err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.Address,
			&blockNumber,
			&event.BlockHash,
			&event.TxHash,
			&logIndex,
			&event.Payload,
		); err != nil {
			return nil, err
		}
		event.BlockNumber = uint64(blockNumber)
		event.LogIndex = uint(logIndex)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) PromoteEvent(ctx context.Context, event PendingEvent) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := writeNormalized(ctx, tx, event); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO confirmed_events (
			event_type, address, block_number, block_hash, tx_hash, log_index, payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tx_hash, log_index) DO NOTHING
	`, event.EventType, event.Address, int64(event.BlockNumber), event.BlockHash, event.TxHash, int64(event.LogIndex), string(event.Payload)); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM pending_events WHERE id = $1`, event.ID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE blocks SET confirmed = TRUE WHERE block_number = $1`, int64(event.BlockNumber)); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) RollbackFromBlock(ctx context.Context, blockNumber uint64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queries := []string{
		`DELETE FROM pending_events WHERE block_number >= $1`,
		`DELETE FROM confirmed_events WHERE block_number >= $1`,
		`DELETE FROM redemptions WHERE block_number >= $1`,
		`DELETE FROM settlements WHERE block_number >= $1`,
		`DELETE FROM oracle_submissions WHERE block_number >= $1`,
		`DELETE FROM trades WHERE block_number >= $1`,
		`DELETE FROM markets WHERE created_block_number >= $1`,
		`DELETE FROM blocks WHERE block_number >= $1`,
	}
	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query, int64(blockNumber)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func writeNormalized(ctx context.Context, tx *sql.Tx, event PendingEvent) error {
	switch event.EventType {
	case EventMarketCreated:
		var payload MarketCreatedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO markets (
				market_id, market_address, creator, status, lock_time, resolution_rule,
				created_block_number, created_tx_hash
			)
			VALUES ($1, $2, $3, 'OPEN', $4, $5, $6, $7)
			ON CONFLICT (market_id) DO UPDATE SET
				market_address = EXCLUDED.market_address,
				creator = EXCLUDED.creator,
				lock_time = EXCLUDED.lock_time,
				resolution_rule = EXCLUDED.resolution_rule
		`, payload.MarketID, payload.Market, payload.Creator, int64(payload.LockTime), payload.ResolutionRule, int64(event.BlockNumber), event.TxHash)
		return err
	case EventTradeExecuted:
		var payload TradeExecutedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO trades (market_id, user_address, side, amount, tx_hash, log_index, block_number)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (tx_hash, log_index) DO NOTHING
		`, payload.MarketID, payload.User, payload.Side, payload.Amount, event.TxHash, int64(event.LogIndex), int64(event.BlockNumber))
		return err
	case EventMarketLocked:
		var payload MarketLockedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			UPDATE markets SET status = 'LOCKED', lock_time = $2 WHERE market_id = $1
		`, payload.MarketID, int64(payload.LockTime))
		return err
	case EventOracleSubmitted:
		var payload OracleSubmittedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO oracle_submissions (
				market_id, outcome, nonce, expiry, tx_hash, log_index, block_number
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (tx_hash, log_index) DO NOTHING
		`, payload.MarketID, payload.Outcome, payload.Nonce, int64(payload.Expiry), event.TxHash, int64(event.LogIndex), int64(event.BlockNumber))
		return err
	case EventMarketResolved:
		var payload MarketResolvedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO settlements (market_id, outcome, tx_hash, log_index, block_number)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (tx_hash, log_index) DO NOTHING
		`, payload.MarketID, payload.Outcome, event.TxHash, int64(event.LogIndex), int64(event.BlockNumber)); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE markets SET status = 'RESOLVED' WHERE market_id = $1`, payload.MarketID)
		return err
	case EventRedeemed:
		var payload RedeemedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO redemptions (market_id, user_address, amount, tx_hash, log_index, block_number)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tx_hash, log_index) DO NOTHING
		`, payload.MarketID, payload.User, payload.Amount, event.TxHash, int64(event.LogIndex), int64(event.BlockNumber))
		return err
	case EventMarketFinalized:
		var payload MarketFinalizedPayload
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx, `UPDATE markets SET status = 'FINALIZED' WHERE market_id = $1`, payload.MarketID)
		return err
	default:
		return fmt.Errorf("unsupported event type %s", event.EventType)
	}
}

func confirmedSSE(event PendingEvent) SSEEvent {
	return SSEEvent{Type: event.EventType, Payload: event.Payload}
}
