package persistence

import "context"

type QueryableDB interface {
	DB
	Exec(context.Context, string, ...any) error
	QueryRow(context.Context, string, ...any) Row
	Query(context.Context, string, ...any) (Rows, error)
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
