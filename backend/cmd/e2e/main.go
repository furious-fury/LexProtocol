package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lexprotocol/lexprotocol/backend/internal/pricing"
)

const (
	defaultPricingURL = "http://localhost:8080"
	defaultIndexerURL = "http://localhost:8090"
)

var (
	factoryABI = mustABI(`[{"type":"function","name":"createMarket","inputs":[{"name":"resolutionRule","type":"string"},{"name":"lockTime","type":"uint256"}],"outputs":[{"name":"market","type":"address"}]},{"anonymous":false,"type":"event","name":"MarketCreated","inputs":[{"indexed":true,"name":"marketId","type":"uint256"},{"indexed":true,"name":"market","type":"address"},{"indexed":true,"name":"creator","type":"address"},{"indexed":false,"name":"lockTime","type":"uint256"},{"indexed":false,"name":"resolutionRule","type":"string"}]}]`)
	marketABI  = mustABI(`[{"type":"function","name":"buyYes","inputs":[],"outputs":[],"stateMutability":"payable"},{"type":"function","name":"buyNo","inputs":[],"outputs":[],"stateMutability":"payable"},{"type":"function","name":"lockMarket","inputs":[],"outputs":[]},{"type":"function","name":"submitOutcome","inputs":[{"name":"oracleData","type":"bytes"},{"name":"signature","type":"bytes"}],"outputs":[]},{"type":"function","name":"redeem","inputs":[],"outputs":[]}]`)
)

type config struct {
	RPCURL               string
	PricingURL           string
	IndexerURL           string
	MarketFactoryAddress common.Address
	DeployerKey          *ecdsa.PrivateKey
	YesTraderKey         *ecdsa.PrivateKey
	NoTraderKey          *ecdsa.PrivateKey
	ResolutionRule       string
	TradeAmount          *big.Int
	LockDelay            time.Duration
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("load e2e config: %v", err)
	}

	client, err := ethclient.DialContext(ctx, cfg.RPCURL)
	if err != nil {
		log.Fatalf("connect RPC: %v", err)
	}
	defer client.Close()

	if err := health(ctx, cfg.PricingURL); err != nil {
		log.Fatalf("pricing service health: %v", err)
	}
	if err := health(ctx, cfg.IndexerURL); err != nil {
		log.Printf("indexer health warning: %v", err)
	}

	chainID, err := client.NetworkID(ctx)
	if err != nil {
		log.Fatalf("load chain ID: %v", err)
	}

	factory := bind.NewBoundContract(cfg.MarketFactoryAddress, factoryABI, client, client, client)
	now := uint64(time.Now().Add(cfg.LockDelay).Unix())
	lockTime := new(big.Int).SetUint64(now)

	createTx, err := factory.Transact(newAuth(ctx, client, cfg.DeployerKey, chainID, nil), "createMarket", cfg.ResolutionRule, lockTime)
	if err != nil {
		log.Fatalf("create market tx: %v", err)
	}
	createReceipt := waitMined(ctx, client, createTx)
	marketID, marketAddress, err := parseMarketCreated(createReceipt)
	if err != nil {
		log.Fatalf("parse MarketCreated: %v", err)
	}
	log.Printf("created market id=%s address=%s tx=%s", marketID, marketAddress.Hex(), createTx.Hash().Hex())

	market := bind.NewBoundContract(marketAddress, marketABI, client, client, client)

	buyYesTx, err := market.Transact(newAuth(ctx, client, cfg.YesTraderKey, chainID, cfg.TradeAmount), "buyYes")
	if err != nil {
		log.Fatalf("buy YES: %v", err)
	}
	waitMined(ctx, client, buyYesTx)
	log.Printf("bought YES tx=%s", buyYesTx.Hash().Hex())

	buyNoTx, err := market.Transact(newAuth(ctx, client, cfg.NoTraderKey, chainID, cfg.TradeAmount), "buyNo")
	if err != nil {
		log.Fatalf("buy NO: %v", err)
	}
	waitMined(ctx, client, buyNoTx)
	log.Printf("bought NO tx=%s", buyNoTx.Hash().Hex())

	if sleepFor := time.Until(time.Unix(int64(now), 0)); sleepFor > 0 {
		log.Printf("waiting %s for lock time", sleepFor.Round(time.Second))
		time.Sleep(sleepFor)
	}

	lockTx, err := market.Transact(newAuth(ctx, client, cfg.DeployerKey, chainID, nil), "lockMarket")
	if err != nil {
		log.Fatalf("lock market: %v", err)
	}
	waitMined(ctx, client, lockTx)
	log.Printf("locked market tx=%s", lockTx.Hash().Hex())

	submission, err := signedOutcome(ctx, cfg.PricingURL, marketID.String(), "YES")
	if err != nil {
		log.Fatalf("get signed outcome: %v", err)
	}
	oracleData := mustHex(submission.OracleData)
	signature := mustHex(submission.Signature)

	resolveTx, err := market.Transact(newAuth(ctx, client, cfg.DeployerKey, chainID, nil), "submitOutcome", oracleData, signature)
	if err != nil {
		log.Fatalf("submit outcome: %v", err)
	}
	waitMined(ctx, client, resolveTx)
	log.Printf("resolved market tx=%s signer=%s nonce=%s", resolveTx.Hash().Hex(), submission.Signer, submission.Nonce)

	redeemTx, err := market.Transact(newAuth(ctx, client, cfg.YesTraderKey, chainID, nil), "redeem")
	if err != nil {
		log.Fatalf("redeem YES: %v", err)
	}
	waitMined(ctx, client, redeemTx)
	log.Printf("redeemed YES tx=%s", redeemTx.Hash().Hex())

	log.Printf("e2e complete: marketId=%s market=%s", marketID, marketAddress.Hex())
	log.Printf("check indexer SSE with: curl -N %s/events", cfg.IndexerURL)
}

func loadConfig() (config, error) {
	rpcURL := firstNonEmpty(os.Getenv("RPC_HTTP_URL"), os.Getenv("RPC_URL"))
	if rpcURL == "" {
		return config{}, errors.New("RPC_HTTP_URL or RPC_URL is required")
	}
	factory := strings.TrimSpace(os.Getenv("MARKET_FACTORY_ADDRESS"))
	if !common.IsHexAddress(factory) {
		return config{}, errors.New("MARKET_FACTORY_ADDRESS must be a valid address")
	}
	deployerKey, err := keyFromEnv("E2E_DEPLOYER_PRIVATE_KEY", "PRIVATE_KEY")
	if err != nil {
		return config{}, err
	}
	yesKey, err := keyFromEnv("E2E_YES_TRADER_PRIVATE_KEY")
	if err != nil {
		return config{}, err
	}
	noKey, err := keyFromEnv("E2E_NO_TRADER_PRIVATE_KEY")
	if err != nil {
		return config{}, err
	}

	amount := big.NewInt(100_000_000_000_000_000) // 0.1 native token
	if raw := strings.TrimSpace(os.Getenv("E2E_TRADE_AMOUNT_WEI")); raw != "" {
		parsed, ok := new(big.Int).SetString(raw, 10)
		if !ok || parsed.Sign() <= 0 {
			return config{}, errors.New("E2E_TRADE_AMOUNT_WEI must be a positive integer")
		}
		amount = parsed
	}

	lockDelay := 3 * time.Second
	if raw := strings.TrimSpace(os.Getenv("E2E_LOCK_DELAY_SECONDS")); raw != "" {
		seconds, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || seconds == 0 {
			return config{}, errors.New("E2E_LOCK_DELAY_SECONDS must be a positive integer")
		}
		lockDelay = time.Duration(seconds) * time.Second
	}

	return config{
		RPCURL:               rpcURL,
		PricingURL:           strings.TrimRight(firstNonEmpty(os.Getenv("PRICING_URL"), defaultPricingURL), "/"),
		IndexerURL:           strings.TrimRight(firstNonEmpty(os.Getenv("INDEXER_URL"), defaultIndexerURL), "/"),
		MarketFactoryAddress: common.HexToAddress(factory),
		DeployerKey:          deployerKey,
		YesTraderKey:         yesKey,
		NoTraderKey:          noKey,
		ResolutionRule:       firstNonEmpty(os.Getenv("E2E_MARKET_RULE"), "LexProtocol local E2E resolves YES"),
		TradeAmount:          amount,
		LockDelay:            lockDelay,
	}, nil
}

func keyFromEnv(names ...string) (*ecdsa.PrivateKey, error) {
	for _, name := range names {
		raw := strings.TrimSpace(os.Getenv(name))
		if raw == "" {
			continue
		}
		key, err := crypto.HexToECDSA(strings.TrimPrefix(raw, "0x"))
		if err != nil {
			return nil, fmt.Errorf("%s must be a valid private key: %w", name, err)
		}
		return key, nil
	}
	return nil, fmt.Errorf("%s is required", strings.Join(names, " or "))
}

func newAuth(ctx context.Context, client *ethclient.Client, key *ecdsa.PrivateKey, chainID *big.Int, value *big.Int) *bind.TransactOpts {
	auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		log.Fatalf("create transactor: %v", err)
	}
	nonce, err := client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		log.Fatalf("load nonce for %s: %v", auth.From.Hex(), err)
	}
	auth.Nonce = new(big.Int).SetUint64(nonce)
	auth.Context = ctx
	auth.Value = value
	return auth
}

func waitMined(ctx context.Context, client *ethclient.Client, tx *types.Transaction) *types.Receipt {
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		log.Fatalf("wait mined %s: %v", tx.Hash().Hex(), err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatalf("transaction reverted %s", tx.Hash().Hex())
	}
	return receipt
}

func parseMarketCreated(receipt *types.Receipt) (*big.Int, common.Address, error) {
	event := factoryABI.Events["MarketCreated"]
	for _, logEvent := range receipt.Logs {
		if len(logEvent.Topics) == 0 || logEvent.Topics[0] != event.ID {
			continue
		}
		marketID := new(big.Int).SetBytes(logEvent.Topics[1].Bytes())
		market := common.BytesToAddress(logEvent.Topics[2].Bytes()[12:])
		return marketID, market, nil
	}
	return nil, common.Address{}, errors.New("MarketCreated not found in receipt")
}

func signedOutcome(ctx context.Context, baseURL string, marketID string, outcome string) (pricing.SettlementSubmission, error) {
	url := fmt.Sprintf("%s/signed/%s?outcome=%s", baseURL, marketID, outcome)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return pricing.SettlementSubmission{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return pricing.SettlementSubmission{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return pricing.SettlementSubmission{}, fmt.Errorf("pricing returned %s: %s", resp.Status, bytes.TrimSpace(body))
	}
	var submission pricing.SettlementSubmission
	if err := json.NewDecoder(resp.Body).Decode(&submission); err != nil {
		return pricing.SettlementSubmission{}, err
	}
	return submission, nil
}

func health(ctx context.Context, baseURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health returned %s", resp.Status)
	}
	return nil
}

func mustHex(value string) []byte {
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
	if err != nil {
		log.Fatalf("decode hex value: %v", err)
	}
	return decoded
}

func mustABI(raw string) abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(raw))
	if err != nil {
		panic(err)
	}
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
