package persistence

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDB struct {
	DSN     string
	Pool    *pgxpool.Pool
	Backend PostgresQueryBackend
	mu      sync.Mutex
}

func NewPostgresDB(dsn string) *PostgresDB {
	return &PostgresDB{DSN: dsn}
}

func (db *PostgresDB) ensurePool(ctx context.Context) (*pgxpool.Pool, error) {
	if db.Backend != nil {
		return nil, nil
	}
	if db.Pool != nil {
		return db.Pool, nil
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.Pool != nil {
		return db.Pool, nil
	}
	pool, err := pgxpool.New(ctx, db.DSN)
	if err != nil {
		return nil, err
	}
	db.Pool = pool
	return db.Pool, nil
}

func (db *PostgresDB) Ping(ctx context.Context) error {
	if db.Backend != nil {
		return nil
	}
	pool, err := db.ensurePool(ctx)
	if err != nil {
		return err
	}
	return pool.Ping(ctx)
}

func (db *PostgresDB) Close() error {
	if db.Pool != nil {
		db.Pool.Close()
	}
	return nil
}

func (db *PostgresDB) Exec(ctx context.Context, query string, args ...any) error {
	if db.Backend != nil {
		return db.Backend.Exec(ctx, query, args...)
	}
	pool, err := db.ensurePool(ctx)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, query, args...)
	return err
}

func (db *PostgresDB) QueryRow(ctx context.Context, query string, args ...any) Row {
	if db.Backend != nil {
		return db.Backend.QueryRow(ctx, query, args...)
	}
	pool, err := db.ensurePool(ctx)
	if err != nil {
		return errorRow{err: err}
	}
	return pool.QueryRow(ctx, query, args...)
}

func (db *PostgresDB) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	if db.Backend != nil {
		return db.Backend.Query(ctx, query, args...)
	}
	pool, err := db.ensurePool(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRowsAdapter{rows: rows}, nil
}
