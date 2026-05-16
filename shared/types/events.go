package types

import (
	"encoding/json"
	"math/big"
)

// Event type constants for indexer and SSE streams.
const (
	EventTradeExecuted  = "TRADE_EXECUTED"
	EventMarketLocked   = "MARKET_LOCKED"
	EventOracleSubmitted = "ORACLE_SUBMITTED"
	EventMarketResolved = "MARKET_RESOLVED"
)

// TradeSide is YES or NO for a trade.
type TradeSide string

const (
	TradeSideYes TradeSide = "YES"
	TradeSideNo  TradeSide = "NO"
)

// TradeEvent matches the PRD TRADE_EXECUTED schema.
type TradeEvent struct {
	Type        string    `json:"type"`
	MarketID    *big.Int  `json:"marketId"`
	User        string    `json:"user"`
	Side        TradeSide `json:"side"`
	Amount      *big.Int  `json:"amount"`
	BlockNumber uint64    `json:"blockNumber"`
	TxHash      []byte    `json:"txHash"`
}

// Event is the generic SSE / indexer envelope (PRD §9).
type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
