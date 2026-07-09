package persistence

import "context"

// MemoryTx is an undo-log transaction used by in-memory repositories in tests.
// Repositories that opt in (e.g. comments.InMemoryRepository) record an undo
// action for each write they perform while a MemoryTx is active in the context.
// On rollback the undos run in reverse order; on commit they are discarded.
type MemoryTx struct {
	undos []func(ctx context.Context)
}

// NewMemoryTxRunner returns a TxRunner backed by an undo log. It lets tests
// exercise rollback semantics against in-memory repositories without a real
// database. Inside the runner, the derived context carries the MemoryTx so
// participating repositories can register undos via MemoryTxFromContext.
func NewMemoryTxRunner() TxRunner {
	return func(ctx context.Context, fn func(ctx context.Context) error) error {
		tx := &MemoryTx{}
		txCtx := WithMemoryTx(ctx, tx)
		if err := fn(txCtx); err != nil {
			tx.rollback(ctx)
			return err
		}
		return nil
	}
}

type memoryTxKey struct{}

// WithMemoryTx stores a MemoryTx into ctx so in-memory repositories can enroll
// their writes. Pair with WithTx so QueryableFromContext-based repos keep
// working too — the in-memory queryable is a no-op here, but presence in ctx
// is what some call paths check.
func WithMemoryTx(ctx context.Context, tx *MemoryTx) context.Context {
	// Also satisfy WithTx's carrier so a single QueryableFromContext lookup
	// path works uniformly; the memory queryable itself is never exercised by
	// in-memory repositories (they branch on MemoryTxFromContext instead).
	ctx = WithTx(ctx, &inMemoryNoopQueryable{}, nil)
	return context.WithValue(ctx, memoryTxKey{}, tx)
}

// MemoryTxFromContext returns the active MemoryTx, if any. In-memory
// repositories call this in their write methods to enroll undos.
func MemoryTxFromContext(ctx context.Context) (*MemoryTx, bool) {
	tx, ok := ctx.Value(memoryTxKey{}).(*MemoryTx)
	return tx, ok
}

// RecordUndo appends an undo action executed (in reverse order) on rollback.
// No-op when tx is nil — callers can pass a nil tx to keep call sites simple.
func (tx *MemoryTx) RecordUndo(fn func(ctx context.Context)) {
	if tx == nil || fn == nil {
		return
	}
	tx.undos = append(tx.undos, fn)
}

func (tx *MemoryTx) rollback(ctx context.Context) {
	for i := len(tx.undos) - 1; i >= 0; i-- {
		tx.undos[i](ctx)
	}
}

// inMemoryNoopQueryable satisfies Queryable so the WithTx carrier path used
// by pgx-aware repos is consistent. In-memory repos never call these methods.
type inMemoryNoopQueryable struct{}

func (inMemoryNoopQueryable) Exec(context.Context, string, ...any) error   { return ErrNotImplemented }
func (inMemoryNoopQueryable) QueryRow(context.Context, string, ...any) Row { return InMemoryRow{} }
func (inMemoryNoopQueryable) Query(context.Context, string, ...any) (Rows, error) {
	return &InMemoryRows{}, nil
}
