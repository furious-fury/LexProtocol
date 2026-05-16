package types

import "time"

// MarketStatus represents the on-chain / indexed market lifecycle.
type MarketStatus string

const (
	MarketCreated    MarketStatus = "CREATED"
	MarketOpen       MarketStatus = "OPEN"
	MarketLocked     MarketStatus = "LOCKED"
	MarketResolving  MarketStatus = "RESOLVING"
	MarketResolved   MarketStatus = "RESOLVED"
	MarketFinalized  MarketStatus = "FINALIZED"
	MarketInvalidated MarketStatus = "INVALIDATED"
)

// MarketState is the off-chain view of a market's current state.
type MarketState struct {
	ID             uint64       `json:"id"`
	Creator        string       `json:"creator"`
	Status         MarketStatus `json:"status"`
	LockTime       uint64       `json:"lockTime"`
	ResolutionRule string       `json:"resolutionRule"`
	CreatedAt      time.Time    `json:"createdAt"`
}
