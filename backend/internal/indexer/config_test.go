package indexer

import "testing"

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://lexprotocol:lexprotocol@localhost:5432/lexprotocol?sslmode=disable")
	t.Setenv("RPC_WS_URL", "ws://127.0.0.1:8545")
	t.Setenv("RPC_HTTP_URL", "http://127.0.0.1:8545")
	t.Setenv("MARKET_FACTORY_ADDRESS", "0x1000000000000000000000000000000000000001")
	t.Setenv("START_BLOCK", "10")
	t.Setenv("INDEXER_CONFIRMATIONS", "2")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv() error = %v", err)
	}
	if cfg.StartBlock != 10 || cfg.Confirmations != 2 || cfg.HTTPAddr != defaultHTTPAddr {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigFromEnvRejectsMissingFactory(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://lexprotocol:lexprotocol@localhost:5432/lexprotocol?sslmode=disable")
	t.Setenv("RPC_WS_URL", "ws://127.0.0.1:8545")
	t.Setenv("RPC_HTTP_URL", "http://127.0.0.1:8545")

	if _, err := LoadConfigFromEnv(); err == nil {
		t.Fatal("expected factory address error")
	}
}
