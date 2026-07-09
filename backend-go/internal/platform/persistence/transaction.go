package persistence

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// TxRunner runs fn inside a transaction (prod) or a memory-backed undo scope
// (tests). When nil, the caller behaves as if no transaction is in effect.
// Implementations must derive a child context carrying the tx and pass it to
// fn, so repo methods that read the tx off the context join automatically.
type TxRunner func(ctx context.Context, fn func(ctx context.Context) error) error

// TxContextKey is the context key under which an active transaction is stored.
type txContextKey struct{}

// queryableCarrier is the value we stash in the context: it satisfies
// Queryable so every repo method can reuse the same lookup helper.
type queryableCarrier struct {
	Queryable
	// rollback, if set, is invoked on transaction failure to undo writes done
	// against an in-memory store. pgx transactions handle rollback themselves;
	// this hook is only populated by NewMemoryTxRunner.
	rollback func(ctx context.Context) error
}

// WithTx stores tx (any Queryable) into ctx so downstream repo methods pick
// it up via QueryableFromContext.
func WithTx(ctx context.Context, q Queryable, rollback func(ctx context.Context) error) context.Context {
	return context.WithValue(ctx, txContextKey{}, &queryableCarrier{Queryable: q, rollback: rollback})
}

// QueryableFromContext returns the Queryable bound to ctx by WithTx / a
// transaction runner. Repo methods use it to join an in-flight transaction;
// when absent they fall back to the shared pool.
func QueryableFromContext(ctx context.Context) (Queryable, bool) {
	c, ok := ctx.Value(txContextKey{}).(*queryableCarrier)
	if !ok || c == nil {
		return nil, false
	}
	return c.Queryable, true
}

// RunInTransaction runs fn inside a real Postgres transaction when factory is
// backed by a pgxpool, otherwise (in-memory or nil factory) it invokes fn
// directly — in-memory tests that need rollback wire NewMemoryTxRunner in at
// the service layer instead.
func RunInTransaction(ctx context.Context, factory *Factory, fn func(ctx context.Context) error) (err error) {
	if factory == nil || factory.DB == nil {
		return fn(ctx)
	}
	pdb, ok := factory.DB.(*PostgresDB)
	if !ok {
		// Non-postgres backend (e.g. InMemoryDB): nothing to transact.
		return fn(ctx)
	}
	if pdb.Backend != nil || pdb.Pool == nil {
		// No live pgx pool available — fall back to no-op transaction so the
		// code path still works in environments without a database.
		return fn(ctx)
	}
	tx, beginErr := pdb.Pool.Begin(ctx)
	if beginErr != nil {
		return fmt.Errorf("begin transaction: %w", beginErr)
	}
	txCtx := WithTx(ctx, &pgxTxQueryable{tx: tx}, nil)
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()
	if err = fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	if commitErr := tx.Commit(ctx); commitErr != nil {
		return fmt.Errorf("commit transaction: %w", commitErr)
	}
	return nil
}

// pgxTxQueryable adapts a pgx.Tx to the QueryableDB interface so it can be
// threaded through the context the same way a pool is.
type pgxTxQueryable struct{ tx pgx.Tx }

func (q *pgxTxQueryable) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := q.tx.Exec(ctx, sql, args...)
	return err
}

func (q *pgxTxQueryable) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return q.tx.QueryRow(ctx, sql, args...)
}

func (q *pgxTxQueryable) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	rows, err := q.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRowsAdapter{rows: rows}, nil
}

// ErrNoTransaction is returned when a rollback is requested outside a tx.
var ErrNoTransaction = errors.New("persistence: no transaction in context")

// NewTxRunner returns a TxRunner bound to factory. Use it at module-assembly
// time to hand services a runner without exposing the factory to them.
func NewTxRunner(factory *Factory) TxRunner {
	return func(ctx context.Context, fn func(ctx context.Context) error) error {
		return RunInTransaction(ctx, factory, fn)
	}
}
