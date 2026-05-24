package testdb

import (
	"context"

	"github.com/jackc/pgx/v5"

	"hotspot/backend-go/internal/platform/persistence"
)

type txWrapper struct{ tx pgx.Tx }

var _ persistence.QueryableDB = (*txWrapper)(nil)

func (w *txWrapper) Ping(context.Context) error { return nil }
func (w *txWrapper) Close() error               { return nil }

func (w *txWrapper) Exec(ctx context.Context, query string, args ...any) error {
	_, err := w.tx.Exec(ctx, query, args...)
	return err
}

func (w *txWrapper) QueryRow(ctx context.Context, query string, args ...any) persistence.Row {
	return w.tx.QueryRow(ctx, query, args...)
}

func (w *txWrapper) Query(ctx context.Context, query string, args ...any) (persistence.Rows, error) {
	rows, err := w.tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &pgxRowsAdapter{rows: rows}, nil
}

type pgxRowsAdapter struct{ rows pgx.Rows }

func (a *pgxRowsAdapter) Next() bool             { return a.rows.Next() }
func (a *pgxRowsAdapter) Scan(dest ...any) error { return a.rows.Scan(dest...) }
func (a *pgxRowsAdapter) Close() error           { a.rows.Close(); return nil }
func (a *pgxRowsAdapter) Err() error             { return a.rows.Err() }
