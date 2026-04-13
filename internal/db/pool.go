package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool connects with retries until ctx is done or attempts exhausted.
func NewPool(ctx context.Context, databaseURL string, attempts int, wait time.Duration) (*pgxpool.Pool, error) {
	var lastErr error
	for i := 0; i < attempts; i++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		cfg, err := pgxpool.ParseConfig(databaseURL)
		if err != nil {
			return nil, fmt.Errorf("parse pool config: %w", err)
		}
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			lastErr = err
			time.Sleep(wait)
			continue
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			lastErr = err
			time.Sleep(wait)
			continue
		}
		return pool, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("unknown connection error")
	}
	return nil, fmt.Errorf("db connect after %d attempts: %w", attempts, lastErr)
}
