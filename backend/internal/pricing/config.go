package pricing

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const (
	defaultHTTPAddr            = ":8080"
	defaultSignatureTTLSeconds = 300
)

// Config contains runtime settings for the pricing service.
type Config struct {
	HTTPAddr              string
	DatabaseURL           string
	APIToken              string
	ChainID               *big.Int
	OracleRegistryAddress common.Address
	SignerPrivateKeyHex   string
	SignatureTTL          time.Duration
}

// LoadConfigFromEnv reads pricing configuration from environment variables.
func LoadConfigFromEnv() (Config, error) {
	httpAddr := strings.TrimSpace(os.Getenv("PRICING_HTTP_ADDR"))
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	apiToken := strings.TrimSpace(os.Getenv("PRICING_API_TOKEN"))

	chainIDRaw := firstNonEmpty(os.Getenv("CHAIN_ID"), os.Getenv("MONAD_TESTNET_CHAIN_ID"))
	if strings.TrimSpace(chainIDRaw) == "" {
		return Config{}, errors.New("CHAIN_ID is required")
	}
	chainID, ok := new(big.Int).SetString(strings.TrimSpace(chainIDRaw), 10)
	if !ok || chainID.Sign() <= 0 {
		return Config{}, fmt.Errorf("invalid CHAIN_ID %q", chainIDRaw)
	}

	registryRaw := strings.TrimSpace(firstNonEmpty(
		os.Getenv("ORACLE_REGISTRY_ADDRESS"),
		os.Getenv("ORACLE_ADDRESS"),
	))
	if !common.IsHexAddress(registryRaw) {
		return Config{}, fmt.Errorf("ORACLE_REGISTRY_ADDRESS must be a valid hex address")
	}

	privateKey := strings.TrimSpace(os.Getenv("PRICING_SIGNER_PRIVATE_KEY"))
	if privateKey == "" {
		return Config{}, errors.New("PRICING_SIGNER_PRIVATE_KEY is required; export it in the terminal, do not store it in .env")
	}

	ttlSeconds := defaultSignatureTTLSeconds
	if raw := strings.TrimSpace(os.Getenv("SIGNATURE_TTL_SECONDS")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return Config{}, fmt.Errorf("SIGNATURE_TTL_SECONDS must be a positive integer")
		}
		ttlSeconds = parsed
	}

	return Config{
		HTTPAddr:              httpAddr,
		DatabaseURL:           databaseURL,
		APIToken:              apiToken,
		ChainID:               chainID,
		OracleRegistryAddress: common.HexToAddress(registryRaw),
		SignerPrivateKeyHex:   privateKey,
		SignatureTTL:          time.Duration(ttlSeconds) * time.Second,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
