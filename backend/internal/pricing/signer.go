package pricing

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	eip712DomainTypeHash = crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	oracleSubmissionHash = crypto.Keccak256Hash([]byte("OracleSubmission(uint256 marketId,uint8 outcome,uint256 nonce,uint256 expiry)"))
	lexProtocolNameHash  = crypto.Keccak256Hash([]byte("LexProtocol"))
	lexProtocolVerHash   = crypto.Keccak256Hash([]byte("1"))
)

type Signer struct {
	privateKey      *ecdsa.PrivateKey
	signerAddress   common.Address
	chainID         *big.Int
	registryAddress common.Address
	ttl             time.Duration
	now             func() time.Time
}

func NewSigner(cfg Config) (*Signer, error) {
	keyHex := strings.TrimPrefix(strings.TrimSpace(cfg.SignerPrivateKeyHex), "0x")
	privateKey, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("parse pricing signer private key: %w", err)
	}
	if cfg.ChainID == nil || cfg.ChainID.Sign() <= 0 {
		return nil, fmt.Errorf("chain ID must be positive")
	}
	if cfg.SignatureTTL <= 0 {
		return nil, fmt.Errorf("signature TTL must be positive")
	}

	return &Signer{
		privateKey:      privateKey,
		signerAddress:   crypto.PubkeyToAddress(privateKey.PublicKey),
		chainID:         new(big.Int).Set(cfg.ChainID),
		registryAddress: cfg.OracleRegistryAddress,
		ttl:             cfg.SignatureTTL,
		now:             time.Now,
	}, nil
}

func (s *Signer) SignSettlement(marketID *big.Int, outcomeID uint8, nonce *big.Int) (SettlementSubmission, error) {
	if marketID == nil || marketID.Sign() <= 0 {
		return SettlementSubmission{}, fmt.Errorf("market ID must be positive")
	}
	if nonce == nil || nonce.Sign() <= 0 {
		return SettlementSubmission{}, fmt.Errorf("nonce must be positive")
	}
	outcome, err := OutcomeString(outcomeID)
	if err != nil {
		return SettlementSubmission{}, err
	}

	expiry := uint64(s.now().Add(s.ttl).Unix())
	oracleData := EncodeOracleData(marketID, outcomeID, nonce, new(big.Int).SetUint64(expiry))
	digest := s.Digest(marketID, outcomeID, nonce, new(big.Int).SetUint64(expiry))

	signature, err := crypto.Sign(digest.Bytes(), s.privateKey)
	if err != nil {
		return SettlementSubmission{}, fmt.Errorf("sign settlement payload: %w", err)
	}

	return SettlementSubmission{
		MarketID:   new(big.Int).Set(marketID),
		Outcome:    outcome,
		OutcomeID:  outcomeID,
		Nonce:      new(big.Int).Set(nonce),
		Expiry:     expiry,
		OracleData: "0x" + hex.EncodeToString(oracleData),
		Signature:  "0x" + hex.EncodeToString(signature),
		Signer:     s.signerAddress.Hex(),
		Domain: EIP712Domain{
			Name:              "LexProtocol",
			Version:           "1",
			ChainID:           new(big.Int).Set(s.chainID),
			VerifyingContract: s.registryAddress.Hex(),
		},
	}, nil
}

func (s *Signer) Digest(marketID *big.Int, outcomeID uint8, nonce *big.Int, expiry *big.Int) common.Hash {
	domain := s.DomainSeparator()
	structHash := OracleSubmissionStructHash(marketID, outcomeID, nonce, expiry)

	payload := make([]byte, 0, 66)
	payload = append(payload, 0x19, 0x01)
	payload = append(payload, domain.Bytes()...)
	payload = append(payload, structHash.Bytes()...)
	return crypto.Keccak256Hash(payload)
}

func (s *Signer) DomainSeparator() common.Hash {
	return crypto.Keccak256Hash(
		padHash(eip712DomainTypeHash),
		padHash(lexProtocolNameHash),
		padHash(lexProtocolVerHash),
		padBig(s.chainID),
		padAddress(s.registryAddress),
	)
}

func OracleSubmissionStructHash(marketID *big.Int, outcomeID uint8, nonce *big.Int, expiry *big.Int) common.Hash {
	return crypto.Keccak256Hash(
		padHash(oracleSubmissionHash),
		padBig(marketID),
		padBig(new(big.Int).SetUint64(uint64(outcomeID))),
		padBig(nonce),
		padBig(expiry),
	)
}

func EncodeOracleData(marketID *big.Int, outcomeID uint8, nonce *big.Int, expiry *big.Int) []byte {
	data := make([]byte, 0, 128)
	data = append(data, padBig(marketID)...)
	data = append(data, padBig(new(big.Int).SetUint64(uint64(outcomeID)))...)
	data = append(data, padBig(nonce)...)
	data = append(data, padBig(expiry)...)
	return data
}

func OutcomeID(raw string) (uint8, error) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case OutcomeYes:
		return OutcomeYesID, nil
	case OutcomeNo:
		return OutcomeNoID, nil
	default:
		return 0, fmt.Errorf("outcome must be YES or NO")
	}
}

func OutcomeString(outcomeID uint8) (string, error) {
	switch outcomeID {
	case OutcomeYesID:
		return OutcomeYes, nil
	case OutcomeNoID:
		return OutcomeNo, nil
	default:
		return "", fmt.Errorf("outcome ID must be 1 or 2")
	}
}

func padHash(hash common.Hash) []byte {
	return hash.Bytes()
}

func padAddress(addr common.Address) []byte {
	return common.LeftPadBytes(addr.Bytes(), 32)
}

func padBig(value *big.Int) []byte {
	if value == nil {
		value = big.NewInt(0)
	}
	return common.LeftPadBytes(value.Bytes(), 32)
}
