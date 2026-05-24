package indexer

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

const (
	defaultHTTPAddr          = ":8090"
	defaultConfirmations     = uint64(2)
	defaultBackfillChunkSize = uint64(2000)
)

type Config struct {
	DatabaseURL          string
	RPCWebSocketURL      string
	RPCHTTPURL           string
	MarketFactoryAddress common.Address
	StartBlock           uint64
	Confirmations        uint64
	BackfillChunkSize    uint64
	HTTPAddr             string
}

func LoadConfigFromEnv() (Config, error) {
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	wsURL := strings.TrimSpace(os.Getenv("RPC_WS_URL"))
	if wsURL == "" {
		return Config{}, errors.New("RPC_WS_URL is required")
	}

	httpURL := strings.TrimSpace(os.Getenv("RPC_HTTP_URL"))
	if httpURL == "" {
		httpURL = strings.TrimSpace(os.Getenv("RPC_URL"))
	}
	if httpURL == "" {
		return Config{}, errors.New("RPC_HTTP_URL or RPC_URL is required")
	}

	factoryRaw := strings.TrimSpace(os.Getenv("MARKET_FACTORY_ADDRESS"))
	if !common.IsHexAddress(factoryRaw) {
		return Config{}, errors.New("MARKET_FACTORY_ADDRESS must be a valid hex address")
	}

	startBlock, err := parseUintEnv("START_BLOCK", 0)
	if err != nil {
		return Config{}, err
	}
	confirmations, err := parseUintEnv("INDEXER_CONFIRMATIONS", defaultConfirmations)
	if err != nil {
		return Config{}, err
	}
	if confirmations == 0 {
		return Config{}, errors.New("INDEXER_CONFIRMATIONS must be greater than zero")
	}
	backfillChunkSize, err := parseUintEnv("INDEXER_BACKFILL_CHUNK_SIZE", defaultBackfillChunkSize)
	if err != nil {
		return Config{}, err
	}
	if backfillChunkSize == 0 {
		return Config{}, errors.New("INDEXER_BACKFILL_CHUNK_SIZE must be greater than zero")
	}

	httpAddr := strings.TrimSpace(os.Getenv("INDEXER_HTTP_ADDR"))
	if httpAddr == "" {
		httpAddr = defaultHTTPAddr
	}

	return Config{
		DatabaseURL:          databaseURL,
		RPCWebSocketURL:      wsURL,
		RPCHTTPURL:           httpURL,
		MarketFactoryAddress: common.HexToAddress(factoryRaw),
		StartBlock:           startBlock,
		Confirmations:        confirmations,
		BackfillChunkSize:    backfillChunkSize,
		HTTPAddr:             httpAddr,
	}, nil
}

func parseUintEnv(name string, fallback uint64) (uint64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an unsigned integer", name)
	}
	return value, nil
}

func bigBlock(number uint64) *big.Int {
	return new(big.Int).SetUint64(number)
}
