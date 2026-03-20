package persistence

import "context"

type PostgresQueryBackend interface {
	Exec(context.Context, string, ...any) error
	QueryRow(context.Context, string, ...any) Row
	Query(context.Context, string, ...any) (Rows, error)
}

type InMemoryQueryableBackend struct{}

func (b *InMemoryQueryableBackend) Exec(context.Context, string, ...any) error {
	return ErrNotImplemented
}
func (b *InMemoryQueryableBackend) QueryRow(context.Context, string, ...any) Row {
	return InMemoryRow{}
}
func (b *InMemoryQueryableBackend) Query(context.Context, string, ...any) (Rows, error) {
	return &InMemoryRows{}, nil
}
