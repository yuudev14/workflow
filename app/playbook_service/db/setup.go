package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool opens and pings a pgx connection pool.
func NewPool(ctx context.Context, dataSourceName string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	return pool, nil
}
