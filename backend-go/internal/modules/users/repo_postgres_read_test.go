package users

import (
	"context"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

type rowValues struct {
	values []any
	err    error
}

func (r rowValues) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *string:
			*d = r.values[i].(string)
		case *float64:
			*d = r.values[i].(float64)
		}
	}
	return nil
}

type backendStub struct{}

func (b *backendStub) Exec(context.Context, string, ...any) error { return nil }
func (b *backendStub) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}
func (b *backendStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	switch query {
	case queryGetUserProfile:
		return rowValues{values: []any{"user_alice", "Alice", "", "normal", "bio", "", "", "", "南天门", 35.0, 112.0, "online", "2026-03-20T00:00:00Z", "2026-03-19T00:00:00Z"}}
	case queryGetUserSettings:
		return rowValues{values: []any{"public", true, true, true, true, true, true, true, "everyone", true, false, true, true}}
	default:
		return rowValues{err: persistence.ErrNotImplemented}
	}
}

func TestPostgresRepositoryReadsFromQueryableDB(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &backendStub{}
	repo := NewPostgresRepository(factory, nil)
	profile, ok := repo.GetProfile(context.Background(), "user_alice")
	if !ok || profile.UserID != "user_alice" || profile.Nickname != "Alice" {
		t.Fatalf("unexpected profile: %#v, ok=%v", profile, ok)
	}
	settings, ok := repo.GetSettings(context.Background(), "user_alice")
	if !ok || settings.AllowPrivateChat != PrivateChatEveryone {
		t.Fatalf("unexpected settings: %#v, ok=%v", settings, ok)
	}
}
