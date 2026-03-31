package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

func TestPersistenceRepositoryUsesMemoryOnlyWithoutPostgresDSN(t *testing.T) {
	repo := NewPersistenceRepository(nil)
	if _, err := repo.Register(context.Background(), RegisterParams{
		LoginID:  "alice",
		Nickname: "Alice",
		Password: "secret123",
		Defaults: ProfileDefaults{CreatedAt: time.Now().UTC()},
	}); err != nil {
		t.Fatalf("expected in-memory register without postgres dsn, got %v", err)
	}
}

func TestPostgresRepositoryFailsClosedWhenQueryableUnavailable(t *testing.T) {
	repo := NewPostgresRepository(&persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     persistence.NewInMemoryDB(),
	}, nil)

	if _, err := repo.Register(context.Background(), RegisterParams{
		LoginID:  "alice",
		Nickname: "Alice",
		Password: "secret123",
		Defaults: ProfileDefaults{CreatedAt: time.Now().UTC()},
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
		LoginID:  "alice",
		Nickname: "Alice",
		Password: "secret123",
		Defaults: ProfileDefaults{CreatedAt: time.Now().UTC()},
	}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented instead of in-memory fallback, got %v", err)
	}
}

type authExecRecorder struct {
	execs []string
}

func (r *authExecRecorder) Ping(context.Context) error { return nil }
func (r *authExecRecorder) Close() error               { return nil }
func (r *authExecRecorder) Exec(_ context.Context, query string, _ ...any) error {
	r.execs = append(r.execs, query)
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
	repo := NewPostgresRepository(factory, nil)

	_, err := repo.CreateGuest(context.Background(), GuestParams{
		Nickname: "Alice",
		Defaults: ProfileDefaults{CreatedAt: time.Now().UTC()},
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
}
