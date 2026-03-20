package persistence

import "context"

type PostgresDB struct {
	DSN     string
	Backend PostgresQueryBackend
}

func NewPostgresDB(dsn string) *PostgresDB {
	return &PostgresDB{DSN: dsn, Backend: &InMemoryQueryableBackend{}}
}
func (db *PostgresDB) Ping(context.Context) error { return nil }
func (db *PostgresDB) Close() error               { return nil }
func (db *PostgresDB) Exec(ctx context.Context, query string, args ...any) error {
	return db.Backend.Exec(ctx, query, args...)
}
func (db *PostgresDB) QueryRow(ctx context.Context, query string, args ...any) Row {
	return db.Backend.QueryRow(ctx, query, args...)
}
func (db *PostgresDB) Query(ctx context.Context, query string, args ...any) (Rows, error) {
	return db.Backend.Query(ctx, query, args...)
}
