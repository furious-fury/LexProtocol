package pricing

import (
	"context"
	"math/big"
)

type Service struct {
	engine Engine
	nonces NonceStore
	signer *Signer
}

func NewService(engine Engine, nonces NonceStore, signer *Signer) *Service {
	return &Service{engine: engine, nonces: nonces, signer: signer}
}

func (s *Service) Price(ctx context.Context, marketID *big.Int) (PriceQuote, error) {
	return s.engine.Price(ctx, marketID)
}

func (s *Service) SignedSettlement(ctx context.Context, marketID *big.Int, outcomeID uint8) (SettlementSubmission, error) {
	nonce, err := s.nonces.Next(ctx)
	if err != nil {
		return SettlementSubmission{}, err
	}
	return s.signer.SignSettlement(marketID, outcomeID, nonce)
}
