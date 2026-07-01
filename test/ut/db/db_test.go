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
	envVarNotSetMsg           = "DATABASE_URL environment variable not set"
	unableToCreateConnPoolMsg = "Unable to create connection pool"
	unableToPingMsg           = "Unable to ping database"

	invalidConnectionString         = "invalid"
	unreachableHostConnectionString = "postgres://test:test@127.0.0.1:1/food_stock"
)

func TestCreatePool_MissingEnvVar(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	pool, err := db.CreatePool(context.Background())

	if pool != nil {
		pool.Close()
		t.Fatal("Expected nil pool for an invalid connection string")
	}

	if err == nil || !strings.Contains(err.Error(), envVarNotSetMsg) {
		t.Fatalf("Expected missing env var error, got: %v", err)
	}
}

func TestCreatePool_InvalidConnectionString(t *testing.T) {
	t.Setenv("DATABASE_URL", invalidConnectionString)

	pool, err := db.CreatePool(context.Background())

	if pool != nil {
		pool.Close()
		t.Fatal("Expected nil pool for an invalid connection string")
	}

	if err == nil || !strings.Contains(err.Error(), unableToCreateConnPoolMsg) {
		t.Fatalf("Expected connection pool creation error, got: %v", err)
	}
}

func TestCreatePool_UnreachableHost(t *testing.T) {
	t.Setenv("DATABASE_URL", unreachableHostConnectionString)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := db.CreatePool(ctx)

	if pool != nil {
		pool.Close()
		t.Fatal("Expected nil pool when the database is unreachable")
	}

	if err == nil || !strings.Contains(err.Error(), unableToPingMsg) {
		t.Fatalf("expected ping error, got: %v", err)
	}
}
