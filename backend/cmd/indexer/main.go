package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/lexprotocol/lexprotocol/backend/internal/indexer"
	"github.com/lexprotocol/lexprotocol/backend/internal/storage"
)

func main() {
	cfg, err := indexer.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("load indexer config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := storage.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	if err := storage.Migrate(ctx, db); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	httpRPC, err := ethclient.DialContext(ctx, cfg.RPCHTTPURL)
	if err != nil {
		log.Fatalf("connect HTTP RPC: %v", err)
	}
	defer httpRPC.Close()

	wsRPC, err := ethclient.DialContext(ctx, cfg.RPCWebSocketURL)
	if err != nil {
		log.Fatalf("connect WebSocket RPC: %v", err)
	}
	defer wsRPC.Close()

	broadcaster := indexer.NewBroadcaster()
	indexerService := indexer.NewService(cfg, db, broadcaster, httpRPC, wsRPC)

	httpServer := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: indexer.NewHTTPHandler(broadcaster),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	go func() {
		log.Printf("lexprotocol indexer HTTP listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("indexer HTTP server failed: %v", err)
			stop()
		}
	}()

	log.Printf("lexprotocol indexer starting at block %d with %d confirmations", cfg.StartBlock, cfg.Confirmations)
	if err := indexerService.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("indexer stopped: %v", err)
	}
}
