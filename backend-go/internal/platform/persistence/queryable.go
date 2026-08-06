package persistence

import (
	"context"
	"errors"
)

// Queryable is the read/write subset of a database connection: anything that
// can Exec/QueryRow/Query. Both *pgxpool.Pool and a pgx.Tx satisfy it, which is
// what lets repo methods target either transparently via the context.
type Queryable interface {
	Exec(context.Context, string, ...any) error
	QueryRow(context.Context, string, ...any) Row
	Query(context.Context, string, ...any) (Rows, error)
}

// QueryableDB is a pooled database handle: a DB that is also Queryable. Used by
// repositories to obtain the shared pool when no transaction is active.
type QueryableDB interface {
	DB
	Queryable
}

type Row interface {
	Scan(...any) error
}

type Rows interface {
	Next() bool
	Scan(...any) error
	Close() error
	Err() error
}

// QueryableForTx 优先返回 ctx 绑定的事务 queryable（RunInTransaction 注入），
// 否则回退到 factory 的共享连接池。各模块 repo 统一走此入口，
// 替代逐模块重复的 queryableForTx 实现。
func QueryableForTx(ctx context.Context, factory *Factory) (Queryable, bool) {
	if q, ok := QueryableFromContext(ctx); ok {
		return q, true
	}
	if factory == nil || factory.DB == nil {
		return nil, false
	}
	qdb, ok := factory.DB.(QueryableDB)
	return qdb, ok
}

// IsUniqueViolation 判断错误是否为 PostgreSQL 唯一约束冲突（SQLSTATE 23505）。
// 用 SQLState 接口而非 pgconn.PgError 具体类型，兼容包装错误与测试替身。
func IsUniqueViolation(err error) bool {
	var sqlState interface{ SQLState() string }
	return errors.As(err, &sqlState) && sqlState.SQLState() == "23505"
}
