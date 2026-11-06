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

// WarmPlanner 在启动期把 PGroonga 等扩展的「首次查询规划」成本从请求路径
// 移到组合根：PGroonga 的关系/索引代价缓存按后端连接惰性初始化，每个新建
// 连接首次 &@ 查询的 Planning Time 可达 50-70ms（实测），表现为搜索端点
// 冷启动/扩容瞬间的 p99 尖刺。这里并发获取最多 maxConns 个池连接各执行一次
// 轻量查询，池连接随即归还；非 Postgres/stub 后端为 no-op。
func (db *PostgresDB) WarmPlanner(ctx context.Context, query string, maxConns int) error {
	if db.Backend != nil || query == "" || maxConns <= 0 {
		return nil
	}
	pool, err := db.ensurePool(ctx)
	if err != nil {
		return err
	}
	if configured := int(pool.Config().MaxConns); configured > 0 && configured < maxConns {
		maxConns = configured
	}
	var wg sync.WaitGroup
	errCh := make(chan error, maxConns)
	for i := 0; i < maxConns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := pool.Acquire(ctx)
			if err != nil {
				errCh <- err
				return
			}
			defer conn.Release()
			if _, err := conn.Exec(ctx, query); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		return err
	}
	return nil
}
