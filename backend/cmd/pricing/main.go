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

	service := pricing.NewService(
		pricing.StaticEngine{},
		pricing.NewMemoryNonceStore(nil),
		signer,
	)

	server := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: api.NewServer(service),
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("lexprotocol pricing service listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("pricing server failed: %v", err)
	}
}
