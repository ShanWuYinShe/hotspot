package persistence

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
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
	// 全程持锁检查：此前的无锁快速路径构成不合法的双重检查锁定（无锁读与
	// 持锁写并发是数据竞争）。NewFactory 构造时即预热 Pool，这里只兜底直构
	// PostgresDB 的路径，锁开销可忽略。
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

// ExecSimple 以简单协议执行 SQL（允许一次发送多条语句）。仅用于迁移/DDL 等
// 必须支持多语句文本的路径：运行期池默认使用预编译语句（extended protocol
// 不支持一条消息里多条语句）。非 Postgres 实现（stub）回退普通 Exec。
func (db *PostgresDB) ExecSimple(ctx context.Context, query string) error {
	if db.Backend != nil {
		return db.Backend.Exec(ctx, query)
	}
	pool, err := db.ensurePool(ctx)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, query, pgx.QueryExecModeSimpleProtocol)
	return err
}

// EnsurePool 是 ensurePool 的公共入口，供需要直接持有 pool 的调用方使用
// （如迁移 advisory lock 的专用连接获取）。
func (db *PostgresDB) EnsurePool(ctx context.Context) (*pgxpool.Pool, error) {
	return db.ensurePool(ctx)
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
