//go:build integration

package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/MadalinGOIAN/food-stock/internal/db"
)

func TestCreatePool_Success(t *testing.T) {
    url := os.Getenv("DATABASE_URL")
    if url == "" {
        t.Skip("DATABASE_URL not set; skipping live connection test")
    }
  
    ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancel()
    
    pool, err := db.CreatePool(ctx)
    if err != nil {
        t.Fatalf("Expected successful connection, got error: %v", err)
    }
    defer pool.Close()

    if err := pool.Ping(ctx); err != nil {
        t.Fatalf("Expected pool to be reachable, got error: %v", err)
    }
}
