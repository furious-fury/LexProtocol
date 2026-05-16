package pricing

import (
	"context"
	"math/big"
	"time"
)

type Engine interface {
	Price(ctx context.Context, marketID *big.Int) (PriceQuote, error)
}

// StaticEngine is the deterministic MVP engine. External fetchers can replace it later.
type StaticEngine struct {
	Now func() time.Time
}

func (e StaticEngine) Price(_ context.Context, marketID *big.Int) (PriceQuote, error) {
	now := time.Now().UTC()
	if e.Now != nil {
		now = e.Now().UTC()
	}

	pYes := big.NewInt(500_000)
	pNo := new(big.Int).Sub(big.NewInt(ProbabilityScale), pYes)

	return PriceQuote{
		MarketID:   new(big.Int).Set(marketID),
		PYes:       pYes,
		PNo:        pNo,
		Confidence: "stub",
		Source:     "static",
		AsOf:       now,
	}, nil
}
