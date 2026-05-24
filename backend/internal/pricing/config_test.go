package pricing

import (
	"math/big"
	"testing"
)

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("CHAIN_ID", "10143")
	t.Setenv("ORACLE_REGISTRY_ADDRESS", "0x000000000000000000000000000000000000dEaD")
	t.Setenv("PRICING_SIGNER_PRIVATE_KEY", testPrivateKey)
	t.Setenv("SIGNATURE_TTL_SECONDS", "600")
	t.Setenv("DATABASE_URL", "postgres://lexprotocol:lexprotocol@localhost:5432/lexprotocol?sslmode=disable")
	t.Setenv("PRICING_API_TOKEN", "secret-token")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}

	if cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
	}
	if cfg.ChainID.Cmp(big.NewInt(10143)) != 0 {
		t.Fatalf("ChainID = %s", cfg.ChainID)
	}
	if cfg.SignatureTTL.Seconds() != 600 {
		t.Fatalf("SignatureTTL = %s", cfg.SignatureTTL)
	}
	if cfg.DatabaseURL == "" || cfg.APIToken != "secret-token" {
		t.Fatalf("security config not loaded: %+v", cfg)
	}
}

func TestLoadConfigFromEnvRejectsMissingPrivateKey(t *testing.T) {
	t.Setenv("CHAIN_ID", "10143")
	t.Setenv("ORACLE_REGISTRY_ADDRESS", "0x000000000000000000000000000000000000dEaD")

	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("expected missing private key error")
	}
}
