package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	usersmodule "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

func TestPostgresRepositoryFailsClosedWhenQueryableUnavailable(t *testing.T) {
	factory := &persistence.Factory{
		Config: persistence.Config{PostgresDSN: "postgres://example"},
		DB:     persistence.NewInMemoryDB(),
	}
	repo := NewPostgresRepository(factory)

	repo.SetProfileWriter(usersmodule.NewPostgresRepository(factory))
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
	repo := NewPostgresRepository(factory)

	repo.SetProfileWriter(usersmodule.NewPostgresRepository(factory))
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

// recordingProfileWriter 记录 auth 对 users.ProfileWriter 的调用（users/user_settings
// 归属 users 模块，auth 注册/游客/登录元数据经此写入，P0-4）。
type recordingProfileWriter struct {
	provisionInputs []usersmodule.ProvisionUserInput
	loginCalls      []struct{ userID, ip, location string }
	provisionErr    error
}

func (w *recordingProfileWriter) ProvisionUser(_ context.Context, input usersmodule.ProvisionUserInput) error {
	w.provisionInputs = append(w.provisionInputs, input)
	return w.provisionErr
}

func (w *recordingProfileWriter) RecordLogin(_ context.Context, userID, ip, location string) error {
	w.loginCalls = append(w.loginCalls, struct{ userID, ip, location string }{userID, ip, location})
	return nil
}

func TestPostgresRepositoryCreateGuestUsesUsersProfileWriter(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &authNoopBackend{}
	repo := NewPostgresRepository(factory)
	writer := &recordingProfileWriter{}
	repo.SetProfileWriter(writer)

	account, err := repo.CreateGuest(context.Background(), GuestParams{
		Nickname:         "Alice",
		Defaults:         ProfileDefaults{CreatedAt: time.Now().UTC(), LocationName: "Somewhere", Latitude: 1, Longitude: 2},
		RegisterIP:       "203.0.113.10",
		RegisterLocation: "US",
	})
	if err != nil {
		t.Fatalf("create guest: %v", err)
	}
	if account.UserID == "" || len(writer.provisionInputs) != 1 {
		t.Fatalf("unexpected guest result: account=%#v writerCalls=%#v", account, writer.provisionInputs)
	}
	input := writer.provisionInputs[0]
	if input.UserID != account.UserID || input.UserType != "temp" || input.Email != "" || input.Phone != "" {
		t.Fatalf("unexpected provision input: %#v", input)
	}
	if input.RegisterIP != "203.0.113.10" || input.RegisterLocation != "US" {
		t.Fatalf("unexpected register meta: %#v", input)
	}
}

func TestPostgresRepositoryCreateGuestPropagatesProfileWriterError(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &authNoopBackend{}
	repo := NewPostgresRepository(factory)
	writer := &recordingProfileWriter{provisionErr: errors.New("boom")}
	repo.SetProfileWriter(writer)

	if _, err := repo.CreateGuest(context.Background(), GuestParams{Nickname: "Alice", Defaults: ProfileDefaults{CreatedAt: time.Now().UTC()}}); err == nil {
		t.Fatal("expected create guest error")
	}
}

func TestPostgresRepositoryUpdateLoginMetaUsesUsersProfileWriter(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &authNoopBackend{}
	repo := NewPostgresRepository(factory)
	writer := &recordingProfileWriter{}
	repo.SetProfileWriter(writer)

	if err := repo.UpdateLoginMeta(context.Background(), "user_alice", "198.51.100.1", "JP"); err != nil {
		t.Fatalf("update login meta: %v", err)
	}
	if len(writer.loginCalls) != 1 {
		t.Fatalf("unexpected writer calls: %#v", writer.loginCalls)
	}
	call := writer.loginCalls[0]
	if call.userID != "user_alice" || call.ip != "198.51.100.1" || call.location != "JP" {
		t.Fatalf("unexpected login meta: %#v", call)
	}
}

func TestPostgresRepositoryProfileWriterNotWired(t *testing.T) {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = &authNoopBackend{}
	repo := NewPostgresRepository(factory)
	if _, err := repo.CreateGuest(context.Background(), GuestParams{Nickname: "Alice", Defaults: ProfileDefaults{CreatedAt: time.Now().UTC()}}); !errors.Is(err, errProfileWriterNotWired) {
		t.Fatalf("expected profile writer not wired, got %v", err)
	}
	if err := repo.UpdateLoginMeta(context.Background(), "u", "ip", ""); !errors.Is(err, errProfileWriterNotWired) {
		t.Fatalf("expected profile writer not wired, got %v", err)
	}
}

// authNoopBackend 提供无副作用的 Queryable 后端（auth 的凭证写入在测试中不关心）。
type authNoopBackend struct{}

func (authNoopBackend) Ping(context.Context) error                 { return nil }
func (authNoopBackend) Close() error                               { return nil }
func (authNoopBackend) Exec(context.Context, string, ...any) error { return nil }
func (authNoopBackend) QueryRow(context.Context, string, ...any) persistence.Row {
	return persistence.InMemoryRow{}
}
func (authNoopBackend) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}
