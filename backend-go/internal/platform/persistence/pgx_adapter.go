package persistence

import "github.com/jackc/pgx/v5"

type pgxRowsAdapter struct{ rows pgx.Rows }

func (a *pgxRowsAdapter) Next() bool             { return a.rows.Next() }
func (a *pgxRowsAdapter) Scan(dest ...any) error { return a.rows.Scan(dest...) }
func (a *pgxRowsAdapter) Close() error           { a.rows.Close(); return nil }
func (a *pgxRowsAdapter) Err() error             { return a.rows.Err() }
