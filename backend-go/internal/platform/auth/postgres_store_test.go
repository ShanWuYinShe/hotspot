package auth

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type authRow struct {
	values []any
	err    error
}

func (r authRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *time.Time:
			*d = r.values[i].(time.Time)
		case **time.Time:
			if r.values[i] == nil {
				*d = nil
			} else {
				v := r.values[i].(time.Time)
				*d = &v
			}
		}
	}
	return nil
}

type authRows struct {
	rows [][]any
	idx  int
}

func (r *authRows) Next() bool { return r.idx < len(r.rows) }
func (r *authRows) Scan(dest ...any) error {
	row := authRow{values: r.rows[r.idx]}
	r.idx++
	return row.Scan(dest...)
}
func (r *authRows) Close() error { return nil }
func (r *authRows) Err() error   { return nil }

type authBackendStub struct{}

func (b *authBackendStub) Exec(context.Context, string, ...any) error { return nil }
func (b *authBackendStub) Query(_ context.Context, query string, _ ...any) (persistence.Rows, error) {
	if query == queryListActiveSessionsByUser {
		return &authRows{rows: [][]any{{"sess_1", "user_alice", time.Now().UTC(), time.Now().UTC(), nil}}}, nil
	}
	return &persistence.InMemoryRows{}, nil
}
func (b *authBackendStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	now := time.Now().UTC()
	switch query {
	case queryGetSession:
		return authRow{values: []any{"sess_1", "user_alice", now, now, nil}}
	case queryGetRefreshToken:
		return authRow{values: []any{"user_alice", "sess_1", "refresh_1", "", nil, now.Add(time.Hour)}}
	default:
		return authRow{err: persistence.ErrNotImplemented}
	}
}

func TestPostgresSessionStoreReadsQueryableDB(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &authBackendStub{}
	store := NewPostgresSessionStore(factory, nil)
	record, ok := store.Get("sess_1")
	if !ok || record.UserID != "user_alice" {
		t.Fatalf("unexpected record: %#v", record)
	}
	items := store.ListActiveByUser("user_alice")
	if len(items) != 1 || items[0].SessionID != "sess_1" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestPostgresRefreshStoreReadsQueryableDB(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &authBackendStub{}
	store := NewPostgresRefreshStore(factory, nil)
	session, ok := store.Get("refresh_1")
	if !ok || session.UserID != "user_alice" {
		t.Fatalf("unexpected session: %#v", session)
	}
}
