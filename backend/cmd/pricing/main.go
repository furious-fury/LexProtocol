package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/lexprotocol/lexprotocol/backend/internal/api"
	"github.com/lexprotocol/lexprotocol/backend/internal/pricing"
	"github.com/lexprotocol/lexprotocol/backend/internal/storage"
)

func main() {
	cfg, err := pricing.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("load pricing config: %v", err)
	}

	signer, err := pricing.NewSigner(cfg)
	if err != nil {
		log.Fatalf("initialize signer: %v", err)
	}

	var nonceStore pricing.NonceStore = pricing.NewMemoryNonceStore(nil)
	if cfg.DatabaseURL != "" {
		db, err := storage.Open(context.Background(), cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("connect pricing database: %v", err)
		}
		defer db.Close()
		if err := storage.Migrate(context.Background(), db); err != nil {
			log.Fatalf("run pricing migrations: %v", err)
		}
		nonceStore = pricing.NewPostgresNonceStore(db)
		log.Printf("pricing nonce store: postgres")
	} else {
		log.Printf("pricing nonce store: memory; set DATABASE_URL for restart-safe nonces")
	}

	service := pricing.NewService(
		pricing.StaticEngine{},
		nonceStore,
		signer,
	)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.NewServer(service, api.WithSignedAPIToken(cfg.APIToken)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if cfg.APIToken == "" {
		log.Printf("pricing signed endpoint is unprotected; set PRICING_API_TOKEN before exposing this service")
	}
	log.Printf(
		"lexprotocol pricing service listening on %s signer=%s chain_id=%s registry=%s",
		cfg.HTTPAddr,
		signer.Address().Hex(),
		cfg.ChainID.String(),
		cfg.OracleRegistryAddress.Hex(),
	)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("pricing server failed: %v", err)
	}
}
