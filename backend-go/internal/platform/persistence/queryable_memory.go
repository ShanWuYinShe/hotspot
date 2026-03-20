package persistence

import "context"

type InMemoryRow struct{}

func (r InMemoryRow) Scan(...any) error { return ErrNotImplemented }

type InMemoryRows struct{}

func (r *InMemoryRows) Next() bool        { return false }
func (r *InMemoryRows) Scan(...any) error { return ErrNotImplemented }
func (r *InMemoryRows) Close() error      { return nil }
func (r *InMemoryRows) Err() error        { return nil }

func (db *InMemoryDB) Exec(context.Context, string, ...any) error   { return ErrNotImplemented }
func (db *InMemoryDB) QueryRow(context.Context, string, ...any) Row { return InMemoryRow{} }
func (db *InMemoryDB) Query(context.Context, string, ...any) (Rows, error) {
	return &InMemoryRows{}, nil
}
