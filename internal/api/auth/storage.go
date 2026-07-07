package auth

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage interface {
	IsActive(ctx context.Context, userID string) (bool, error)
	Deactivate(ctx context.Context, userID string) error
}

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(pool *pgxpool.Pool) *PostgresStorage {
	return &PostgresStorage{pool: pool}
}

func (s *PostgresStorage) IsActive(ctx context.Context, userID string) (bool, error) {
	var isActive bool

	err := s.pool.QueryRow(
		ctx,
		`SELECT is_active FROM public.users WHERE id = $1`,
		userID,
	).Scan(&isActive)

	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	return isActive, nil
}

func (s *PostgresStorage) Deactivate(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(
		ctx,
		`UPDATE public.users SET is_active = false WHERE id = $1`,
		userID,
	)
	return err
}
