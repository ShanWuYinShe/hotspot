package persistence

import "testing"

func TestNewPostgresDB(t *testing.T) {
	db := NewPostgresDB("postgres://example")
	if db.DSN != "postgres://example" {
		t.Fatalf("unexpected dsn: %s", db.DSN)
	}
}
