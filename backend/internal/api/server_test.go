package api

import (
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lexprotocol/lexprotocol/backend/internal/pricing"
)

func TestPricingAPI(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/price/1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var quote pricing.PriceQuote
	if err := json.NewDecoder(rec.Body).Decode(&quote); err != nil {
		t.Fatalf("decode quote: %v", err)
	}
	if quote.MarketID.Cmp(big.NewInt(1)) != 0 || quote.PYes.Cmp(big.NewInt(500000)) != 0 {
		t.Fatalf("unexpected quote: %+v", quote)
	}
}

func TestSignedAPI(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/signed/1?outcome=YES", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var submission pricing.SettlementSubmission
	if err := json.NewDecoder(rec.Body).Decode(&submission); err != nil {
		t.Fatalf("decode signed response: %v", err)
	}
	if submission.Outcome != pricing.OutcomeYes {
		t.Fatalf("outcome = %s", submission.Outcome)
	}
	if submission.OracleData == "" || submission.Signature == "" {
		t.Fatalf("expected oracleData and signature: %+v", submission)
	}
}

func TestSignedAPIRejectsInvalidOutcome(t *testing.T) {
	handler := newTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/signed/1?outcome=MAYBE", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()

	cfg := pricing.Config{
		ChainID:               big.NewInt(10143),
		OracleRegistryAddress: common.HexToAddress("0x000000000000000000000000000000000000dEaD"),
		SignerPrivateKeyHex:   pricingTestPrivateKey,
		SignatureTTL:          5 * time.Minute,
	}
	signer, err := pricing.NewSigner(cfg)
	if err != nil {
		t.Fatalf("NewSigner() error = %v", err)
	}

	service := pricing.NewService(
		pricing.StaticEngine{Now: func() time.Time { return time.Unix(1000, 0).UTC() }},
		pricing.NewMemoryNonceStore(nil),
		signer,
	)
	return NewServer(service)
}

const pricingTestPrivateKey = "0x59c6995e998f97a5a0044966f0945389dc9e86dae547d3e236020d123b4d7bc5"
