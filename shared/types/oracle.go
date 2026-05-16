package types

import "math/big"

// Outcome is the deterministic resolution result for a market.
type Outcome string

const (
	OutcomeYes Outcome = "YES"
	OutcomeNo  Outcome = "NO"
)

// OracleSubmission is a signed oracle payload for settlement or indexing.
type OracleSubmission struct {
	MarketID  *big.Int `json:"marketId"`
	Outcome   Outcome  `json:"outcome,omitempty"`
	PYes      *big.Int `json:"pYes,omitempty"`
	Nonce     *big.Int `json:"nonce"`
	Expiry    uint64   `json:"expiry"`
	Signature []byte   `json:"signature"`
}
