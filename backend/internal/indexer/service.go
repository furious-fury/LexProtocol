package indexer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	cfg          Config
	store        *Store
	broadcaster  *Broadcaster
	httpClient   *ethclient.Client
	wsClient     *ethclient.Client
	knownMu      sync.RWMutex
	knownMarkets map[common.Address]bool
}

func NewService(cfg Config, db *sql.DB, broadcaster *Broadcaster, httpClient *ethclient.Client, wsClient *ethclient.Client) *Service {
	return &Service{
		cfg:          cfg,
		store:        NewStore(db),
		broadcaster:  broadcaster,
		httpClient:   httpClient,
		wsClient:     wsClient,
		knownMarkets: make(map[common.Address]bool),
	}
}

func (s *Service) Run(ctx context.Context) error {
	if err := s.loadKnownMarkets(ctx); err != nil {
		return err
	}
	if err := s.backfill(ctx); err != nil {
		return err
	}

	errCh := make(chan error, 2)
	go func() { errCh <- s.listen(ctx) }()
	go func() { errCh <- s.confirmLoop(ctx) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (s *Service) backfill(ctx context.Context) error {
	latest, err := s.httpClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("load latest header for backfill: %w", err)
	}

	for from := s.cfg.StartBlock; from <= latest.Number.Uint64(); {
		to := from + s.cfg.BackfillChunkSize - 1
		if to > latest.Number.Uint64() {
			to = latest.Number.Uint64()
		}
		query := ethereum.FilterQuery{
			FromBlock: bigBlock(from),
			ToBlock:   bigBlock(to),
			Topics:    [][]common.Hash{EventTopics()},
		}

		logs, err := s.httpClient.FilterLogs(ctx, query)
		if err != nil {
			return fmt.Errorf("backfill logs %d-%d: %w", from, to, err)
		}
		for _, logEvent := range logs {
			if err := s.handleLog(ctx, logEvent); err != nil {
				log.Printf("indexer: skip backfill log %s/%d: %v", logEvent.TxHash.Hex(), logEvent.Index, err)
			}
		}
		if to == latest.Number.Uint64() {
			break
		}
		from = to + 1
	}
	return nil
}

func (s *Service) listen(ctx context.Context) error {
	query := ethereum.FilterQuery{
		FromBlock: bigBlock(s.cfg.StartBlock),
		Topics:    [][]common.Hash{EventTopics()},
	}

	logs := make(chan types.Log, 128)
	sub, err := s.wsClient.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		return fmt.Errorf("subscribe logs: %w", err)
	}
	defer sub.Unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-sub.Err():
			return fmt.Errorf("log subscription: %w", err)
		case logEvent := <-logs:
			if err := s.handleLog(ctx, logEvent); err != nil {
				log.Printf("indexer: skip log %s/%d: %v", logEvent.TxHash.Hex(), logEvent.Index, err)
			}
		}
	}
}

func (s *Service) handleLog(ctx context.Context, logEvent types.Log) error {
	if logEvent.Removed {
		return s.store.RollbackFromBlock(ctx, logEvent.BlockNumber)
	}

	decoded, err := DecodeTrustedLog(logEvent, s.cfg.MarketFactoryAddress, s.isKnownMarket)
	if err != nil {
		return err
	}

	header, err := s.httpClient.HeaderByNumber(ctx, bigBlock(logEvent.BlockNumber))
	if err != nil {
		return fmt.Errorf("load block header: %w", err)
	}
	if header.Hash() != logEvent.BlockHash {
		if err := s.store.RollbackFromBlock(ctx, logEvent.BlockNumber); err != nil {
			return err
		}
		return fmt.Errorf("canonical hash mismatch at block %d", logEvent.BlockNumber)
	}
	if err := s.store.SaveBlock(ctx, logEvent.BlockNumber, header.Hash(), header.ParentHash, false); err != nil {
		return err
	}

	if err := s.store.InsertPendingEvent(ctx, PendingEvent{
		EventType:   decoded.Type,
		Address:     logEvent.Address.Hex(),
		BlockNumber: logEvent.BlockNumber,
		BlockHash:   logEvent.BlockHash.Hex(),
		TxHash:      logEvent.TxHash.Hex(),
		LogIndex:    logEvent.Index,
		Payload:     decoded.Payload,
	}); err != nil {
		return err
	}
	if decoded.Type == EventMarketCreated {
		var payload MarketCreatedPayload
		if err := json.Unmarshal(decoded.Payload, &payload); err != nil {
			return err
		}
		if common.IsHexAddress(payload.Market) {
			s.rememberMarket(common.HexToAddress(payload.Market))
		}
	}
	return nil
}

func (s *Service) confirmLoop(ctx context.Context) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.confirm(ctx); err != nil {
				log.Printf("indexer: confirmation pass failed: %v", err)
			}
		}
	}
}

func (s *Service) loadKnownMarkets(ctx context.Context) error {
	markets, err := s.store.KnownMarkets(ctx)
	if err != nil {
		return fmt.Errorf("load known markets: %w", err)
	}
	s.knownMu.Lock()
	defer s.knownMu.Unlock()
	for market := range markets {
		s.knownMarkets[market] = true
	}
	return nil
}

func (s *Service) isKnownMarket(market common.Address) bool {
	s.knownMu.RLock()
	defer s.knownMu.RUnlock()
	return s.knownMarkets[market]
}

func (s *Service) rememberMarket(market common.Address) {
	s.knownMu.Lock()
	defer s.knownMu.Unlock()
	s.knownMarkets[market] = true
}

func (s *Service) confirm(ctx context.Context) error {
	header, err := s.httpClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return fmt.Errorf("load latest header: %w", err)
	}
	if header.Number.Uint64() < s.cfg.Confirmations {
		return nil
	}

	maxBlock := header.Number.Uint64() - s.cfg.Confirmations
	events, err := s.store.ConfirmableEvents(ctx, maxBlock)
	if err != nil {
		return err
	}

	for _, event := range events {
		canonical, err := s.httpClient.HeaderByNumber(ctx, bigBlock(event.BlockNumber))
		if err != nil {
			return err
		}
		if canonical.Hash().Hex() != event.BlockHash {
			return s.store.RollbackFromBlock(ctx, event.BlockNumber)
		}
		if err := s.store.PromoteEvent(ctx, event); err != nil {
			return err
		}
		s.broadcaster.Broadcast(confirmedSSE(event))
	}
	return nil
}
