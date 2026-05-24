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

// TestDigestDeterminism verifies that the Go EIP-712 signer produces deterministic,
// internally consistent digests. Run alongside Foundry's SignatureCompatTest and
// compare the logged intermediate values to confirm Go ↔ Solidity byte-compatibility.
func TestDigestDeterminism(t *testing.T) {
	// Use a fixed registry address for deterministic digest computation.
	// The Solidity test deploys fresh and uses its own address, but both sides
	// use the same algorithm — if the algorithm matches, they're compatible.
	registryAddr := common.HexToAddress("0x000000000000000000000000000000000000dEaD")

	cfg := Config{
		ChainID:               big.NewInt(10143),
		OracleRegistryAddress: registryAddr,
		SignerPrivateKeyHex:   testPrivateKey,
		SignatureTTL:          5 * time.Minute,
	}

	signer, err := NewSigner(cfg)
	if err != nil {
		t.Fatalf("NewSigner() error = %v", err)
	}

	marketID := big.NewInt(7)
	outcomeID := OutcomeYesID
	nonce := big.NewInt(1)
	expiry := big.NewInt(1300)

	// Compute intermediate values
	domainSep := signer.DomainSeparator()
	structHash := OracleSubmissionStructHash(marketID, outcomeID, nonce, expiry)
	digest := signer.Digest(marketID, outcomeID, nonce, expiry)

	t.Logf("=== Go EIP-712 Intermediate Values ===")
	t.Logf("Registry address: %s", registryAddr.Hex())
	t.Logf("Chain ID:         10143")
	t.Logf("Domain separator: 0x%s", hex.EncodeToString(domainSep.Bytes()))
	t.Logf("Struct hash:      0x%s", hex.EncodeToString(structHash.Bytes()))
	t.Logf("Digest:           0x%s", hex.EncodeToString(digest.Bytes()))

	// Verify the digest is deterministic across calls
	digest2 := signer.Digest(marketID, outcomeID, nonce, expiry)
	if digest != digest2 {
		t.Fatal("digest is not deterministic across calls")
	}

	// Verify the signature is recoverable with the correct address
	keyHex := strings.TrimPrefix(strings.TrimSpace(testPrivateKey), "0x")
	privateKey, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}

	signature, err := crypto.Sign(digest.Bytes(), privateKey)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	pubkey, err := crypto.SigToPub(digest.Bytes(), signature)
	if err != nil {
		t.Fatalf("recover: %v", err)
	}

	recovered := crypto.PubkeyToAddress(*pubkey)
	expected := crypto.PubkeyToAddress(privateKey.PublicKey)
	if recovered != expected {
		t.Fatalf("recovered %s != expected %s", recovered.Hex(), expected.Hex())
	}

	// Verify struct hash matches known construction
	expectedStructHash := crypto.Keccak256Hash(
		padHash(oracleSubmissionHash),
		padBig(marketID),
		padBig(new(big.Int).SetUint64(uint64(outcomeID))),
		padBig(nonce),
		padBig(expiry),
	)
	if structHash != expectedStructHash {
		t.Fatal("struct hash construction mismatch")
	}
}

// TestOracleDataABIEncoding verifies the ABI encoding matches what abi.encode() produces in Solidity.
func TestOracleDataABIEncoding(t *testing.T) {
	marketID := big.NewInt(7)
	outcomeID := OutcomeYesID
	nonce := big.NewInt(1)
	expiry := big.NewInt(1300)

	data := EncodeOracleData(marketID, outcomeID, nonce, expiry)
	if len(data) != 128 {
		t.Fatalf("expected 128-byte ABI tuple, got %d", len(data))
	}

	// Verify each 32-byte slot
	if got := new(big.Int).SetBytes(data[0:32]); got.Cmp(marketID) != 0 {
		t.Fatalf("slot 0 (marketId): got %s", got)
	}
	if got := new(big.Int).SetBytes(data[32:64]); got.Cmp(big.NewInt(int64(outcomeID))) != 0 {
		t.Fatalf("slot 1 (outcome): got %s", got)
	}
	if got := new(big.Int).SetBytes(data[64:96]); got.Cmp(nonce) != 0 {
		t.Fatalf("slot 2 (nonce): got %s", got)
	}
	if got := new(big.Int).SetBytes(data[96:128]); got.Cmp(expiry) != 0 {
		t.Fatalf("slot 3 (expiry): got %s", got)
	}
}
