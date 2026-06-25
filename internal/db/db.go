package db

import (
    "context"
    "fmt"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
)

func CreatePool(ctx context.Context) (*pgxpool.Pool, error) {
    dbUrl := os.Getenv("DATABASE_URL")
    if dbUrl == "" {
        return nil, fmt.Errorf("DATABASE_URL environment variable not set")
    }

    pool, err := pgxpool.New(ctx, dbUrl)
    if err != nil {
        return nil, fmt.Errorf("Unable to create connection pool: %w", err)
    }

    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, fmt.Errorf("Unable to ping database: %w", err)
    }

    return pool, nil
}
