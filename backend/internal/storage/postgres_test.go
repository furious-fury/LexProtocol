//go:build integration

package storage

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOpenPing(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://lexprotocol:lexprotocol@localhost:5432/lexprotocol?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, url)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	defer db.Close()
}
