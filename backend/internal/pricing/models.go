package pricing

import (
	"math/big"
	"time"
)

const (
	OutcomeYesID uint8 = 1
	OutcomeNoID  uint8 = 2

	OutcomeYes = "YES"
	OutcomeNo  = "NO"

	ProbabilityScale int64 = 1_000_000
)

// PriceQuote is informational pricing data. It is not accepted by settlement contracts.
type PriceQuote struct {
	MarketID   *big.Int  `json:"marketId"`
	PYes       *big.Int  `json:"pYes"`
	PNo        *big.Int  `json:"pNo"`
	Confidence string    `json:"confidence"`
	Source     string    `json:"source"`
	AsOf       time.Time `json:"asOf"`
}

type EIP712Domain struct {
	Name              string   `json:"name"`
	Version           string   `json:"version"`
	ChainID           *big.Int `json:"chainId"`
	VerifyingContract string   `json:"verifyingContract"`
}

// SettlementSubmission is the contract-compatible signed payload for MarketContract.submitOutcome.
type SettlementSubmission struct {
	MarketID   *big.Int     `json:"marketId"`
	Outcome    string       `json:"outcome"`
	OutcomeID  uint8        `json:"outcomeId"`
	Nonce      *big.Int     `json:"nonce"`
	Expiry     uint64       `json:"expiry"`
	OracleData string       `json:"oracleData"`
	Signature  string       `json:"signature"`
	Signer     string       `json:"signer"`
	Domain     EIP712Domain `json:"domain"`
}
