package persistence

import "context"

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
