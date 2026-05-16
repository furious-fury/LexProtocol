package pricing

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const testPrivateKey = "0x59c6995e998f97a5a0044966f0945389dc9e86dae547d3e236020d123b4d7bc5"

func TestSignerProducesRecoverableSettlementSubmission(t *testing.T) {
	cfg := Config{
		ChainID:               big.NewInt(10143),
		OracleRegistryAddress: common.HexToAddress("0x000000000000000000000000000000000000dEaD"),
		SignerPrivateKeyHex:   testPrivateKey,
		SignatureTTL:          5 * time.Minute,
	}

	signer, err := NewSigner(cfg)
	if err != nil {
		t.Fatalf("NewSigner() error = %v", err)
	}
	signer.now = func() time.Time { return time.Unix(1000, 0).UTC() }

	submission, err := signer.SignSettlement(big.NewInt(7), OutcomeYesID, big.NewInt(1))
	if err != nil {
		t.Fatalf("SignSettlement() error = %v", err)
	}

	if submission.Outcome != OutcomeYes || submission.OutcomeID != OutcomeYesID {
		t.Fatalf("unexpected outcome: %s/%d", submission.Outcome, submission.OutcomeID)
	}
	if submission.Expiry != 1300 {
		t.Fatalf("expected expiry 1300, got %d", submission.Expiry)
	}

	oracleData, err := hex.DecodeString(strings.TrimPrefix(submission.OracleData, "0x"))
	if err != nil {
		t.Fatalf("decode oracleData: %v", err)
	}
	if len(oracleData) != 128 {
		t.Fatalf("expected 128-byte ABI tuple, got %d", len(oracleData))
	}
	if got := new(big.Int).SetBytes(oracleData[0:32]); got.Cmp(big.NewInt(7)) != 0 {
		t.Fatalf("encoded marketId = %s", got)
	}
	if got := new(big.Int).SetBytes(oracleData[32:64]); got.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("encoded outcome = %s", got)
	}
	if got := new(big.Int).SetBytes(oracleData[64:96]); got.Cmp(big.NewInt(1)) != 0 {
		t.Fatalf("encoded nonce = %s", got)
	}
	if got := new(big.Int).SetBytes(oracleData[96:128]); got.Cmp(big.NewInt(1300)) != 0 {
		t.Fatalf("encoded expiry = %s", got)
	}

	signature, err := hex.DecodeString(strings.TrimPrefix(submission.Signature, "0x"))
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}
	digest := signer.Digest(big.NewInt(7), OutcomeYesID, big.NewInt(1), big.NewInt(1300))
	pubkey, err := crypto.SigToPub(digest.Bytes(), signature)
	if err != nil {
		t.Fatalf("recover signer: %v", err)
	}
	recovered := crypto.PubkeyToAddress(*pubkey)
	if recovered.Hex() != submission.Signer {
		t.Fatalf("recovered signer %s, response signer %s", recovered.Hex(), submission.Signer)
	}
}

func TestMemoryNonceStoreIncrementsGlobally(t *testing.T) {
	store := NewMemoryNonceStore(nil)

	first, err := store.Next(nil)
	if err != nil {
		t.Fatalf("first nonce: %v", err)
	}
	second, err := store.Next(nil)
	if err != nil {
		t.Fatalf("second nonce: %v", err)
	}

	if first.Cmp(big.NewInt(1)) != 0 || second.Cmp(big.NewInt(2)) != 0 {
		t.Fatalf("unexpected nonces: %s, %s", first, second)
	}
}

func TestOutcomeIDValidation(t *testing.T) {
	got, err := OutcomeID("yes")
	if err != nil {
		t.Fatalf("OutcomeID(yes): %v", err)
	}
	if got != OutcomeYesID {
		t.Fatalf("OutcomeID(yes) = %d", got)
	}

	if _, err := OutcomeID("MAYBE"); err == nil {
		t.Fatal("expected invalid outcome error")
	}
}
