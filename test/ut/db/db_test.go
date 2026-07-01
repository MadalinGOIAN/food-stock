//go:build !integration

package db

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/MadalinGOIAN/food-stock/internal/db"
)

const (
	ENV_VAR_NOT_SET_MSG            = "DATABASE_URL environment variable not set"
	UNABLE_TO_CREATE_CONN_POOL_MSG = "Unable to create connection pool"
	UNABLE_TO_PING_MSG             = "Unable to ping database"

	InvalidConnectionString         = "invalid"
	UnreachableHostConnectionString = "postgres://test:test@127.0.0.1:1/food_stock"
)

func TestCreatePool_MissingEnvVar(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	pool, err := db.CreatePool(context.Background())

	if pool != nil {
		pool.Close()
		t.Fatal("Expected nil pool for an invalid connection string")
	}

	if err == nil || !strings.Contains(err.Error(), ENV_VAR_NOT_SET_MSG) {
		t.Fatalf("Expected missing env var error, got: %v", err)
	}
}

func TestCreatePool_InvalidConnectionString(t *testing.T) {
	t.Setenv("DATABASE_URL", InvalidConnectionString)

	pool, err := db.CreatePool(context.Background())

	if pool != nil {
		pool.Close()
		t.Fatal("Expected nil pool for an invalid connection string")
	}

	if err == nil || !strings.Contains(err.Error(), UNABLE_TO_CREATE_CONN_POOL_MSG) {
		t.Fatalf("Expected connection pool creation error, got: %v", err)
	}
}

func TestCreatePool_UnreachableHost(t *testing.T) {
	t.Setenv("DATABASE_URL", UnreachableHostConnectionString)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := db.CreatePool(ctx)

	if pool != nil {
		pool.Close()
		t.Fatal("Expected nil pool when the database is unreachable")
	}

	if err == nil || !strings.Contains(err.Error(), UNABLE_TO_PING_MSG) {
		t.Fatalf("expected ping error, got: %v", err)
	}
}
