package privatechats

import (
	"context"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// chatStateStub 提供 QueryRow/Exec 的确定性结果。
type chatStateStub struct {
	rowValues []any
	rowErr    error
	execErr   error
	execs     int
}

func (s *chatStateStub) QueryRow(_ context.Context, _ string, _ ...any) persistence.Row {
	return chatStateRow{values: s.rowValues, err: s.rowErr}
}

func (s *chatStateStub) Query(_ context.Context, _ string, _ ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

func (s *chatStateStub) Exec(_ context.Context, _ string, _ ...any) error {
	s.execs++
	return s.execErr
}

type chatStateRow struct {
	values []any
	err    error
}

func (r chatStateRow) Scan(dest ...any) error {
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
			} else if v, ok := r.values[i].(time.Time); ok {
				*d = &v
			}
		case *bool:
			*d = r.values[i].(bool)
		}
	}
	return nil
}

func newChatFactory(backend persistence.Queryable) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresGetStateAndUpdateSettings(t *testing.T) {
	now := time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)
	backend := &chatStateStub{rowValues: []any{"chat_1", "u1", now, false, false, nil, nil}}
	repo := NewPostgresRepository(newChatFactory(backend))

	state, err := repo.GetState(context.Background(), "u1", "u2")
	if err != nil || state.ChatKey != "chat_1" || !state.CreatedAt.Equal(now) {
		t.Fatalf("get state: %#v err=%v", state, err)
	}

	muted := true
	if err := repo.UpdateSettings(context.Background(), "u1", "u2", UpdateChatSettings{IsMuted: &muted}); err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if backend.execs == 0 {
		t.Fatal("expected upsert exec")
	}

	if err := repo.RestoreForUser(context.Background(), "u1", "u2", now); err != nil {
		t.Fatalf("restore: %v", err)
	}
	if err := repo.DeleteForUser(context.Background(), "u1", "u2", &now); err != nil {
		t.Fatalf("delete for user: %v", err)
	}
}

func TestPostgresChatRepoNotConfigured(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	if _, err := repo.GetState(context.Background(), "u1", "u2"); err != persistence.ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
	if err := repo.UpdateSettings(context.Background(), "u1", "u2", UpdateChatSettings{}); err != persistence.ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
	if err := repo.RestoreForUser(context.Background(), "u1", "u2", time.Now()); err != persistence.ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
	if err := repo.DeleteForUser(context.Background(), "u1", "u2", nil); err != persistence.ErrNotImplemented {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}
