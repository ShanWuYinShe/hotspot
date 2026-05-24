package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

func TestPostgresRepositoryFailsClosedWhenQueryableUnavailable(t *testing.T) {
	repo := NewPostgresRepository(&persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     persistence.NewInMemoryDB(),
	})

	if _, err := repo.Register(context.Background(), RegisterParams{
		LoginID:          "alice",
		Nickname:         "Alice",
		Password:         "secret123",
		Defaults:         ProfileDefaults{CreatedAt: time.Now().UTC()},
		RegisterIP:       "203.0.113.10",
		RegisterLocation: "US",
	}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented when queryable db unavailable, got %v", err)
	}
}

func TestPersistenceRepositoryWithPostgresDSNDoesNotFallbackSilently(t *testing.T) {
	factory := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     persistence.NewInMemoryDB(),
	}
	repo := NewPersistenceRepository(factory)

	if _, err := repo.Register(context.Background(), RegisterParams{
		LoginID:          "alice",
		Nickname:         "Alice",
		Password:         "secret123",
		Defaults:         ProfileDefaults{CreatedAt: time.Now().UTC()},
		RegisterIP:       "203.0.113.10",
		RegisterLocation: "US",
	}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented instead of in-memory fallback, got %v", err)
	}
}

type authExecRecorder struct {
	execs    []string
	execArgs [][]any
}

func (r *authExecRecorder) Ping(context.Context) error { return nil }
func (r *authExecRecorder) Close() error               { return nil }
func (r *authExecRecorder) Exec(_ context.Context, query string, args ...any) error {
	r.execs = append(r.execs, query)
	r.execArgs = append(r.execArgs, args)
	if query == queryUpsertUserSettings {
		return errors.New("boom")
	}
	return nil
}
func (r *authExecRecorder) QueryRow(context.Context, string, ...any) persistence.Row {
	return persistence.InMemoryRow{}
}
func (r *authExecRecorder) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

func TestPostgresRepositoryCreateGuestRollsBackProfileWhenSettingsWriteFails(t *testing.T) {
	backend := &authExecRecorder{}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	_, err := repo.CreateGuest(context.Background(), GuestParams{
		Nickname:         "Alice",
		Defaults:         ProfileDefaults{CreatedAt: time.Now().UTC()},
		RegisterIP:       "203.0.113.10",
		RegisterLocation: "US",
	})
	if err == nil {
		t.Fatal("expected create guest error")
	}
	if len(backend.execs) != 3 {
		t.Fatalf("unexpected exec count: %#v", backend.execs)
	}
	if backend.execs[0] != queryInsertUserProfile || backend.execs[1] != queryUpsertUserSettings || backend.execs[2] != queryDeleteUserProfileByUserID {
		t.Fatalf("unexpected exec order: %#v", backend.execs)
	}
	if got := backend.execArgs[0][11]; got != "203.0.113.10" {
		t.Fatalf("unexpected register ip arg: %v", got)
	}
	if got := backend.execArgs[0][12]; got != "US" {
		t.Fatalf("unexpected register location arg: %v", got)
	}
}

func TestPostgresRepositoryUpdateLoginMetaWritesUserColumns(t *testing.T) {
	backend := &authExecRecorder{}
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	repo := NewPostgresRepository(factory)

	if err := repo.UpdateLoginMeta(context.Background(), "user_alice", "198.51.100.1", "JP"); err != nil {
		t.Fatalf("update login meta: %v", err)
	}
	if len(backend.execs) != 1 || backend.execs[0] != queryUpdateLoginMeta {
		t.Fatalf("unexpected execs: %#v", backend.execs)
	}
	if got := backend.execArgs[0][1]; got != "198.51.100.1" {
		t.Fatalf("unexpected login ip arg: %v", got)
	}
	if got := backend.execArgs[0][2]; got != "JP" {
		t.Fatalf("unexpected login location arg: %v", got)
	}
}
