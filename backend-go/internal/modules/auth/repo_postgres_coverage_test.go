package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// authRowStub 按查询串返回预设行；QueryRow 支持 *string/*time.Time/*bool。
type authRowStub struct {
	rowValues map[string][]any
	execErr   map[string]error
	execs     int
}

func (s *authRowStub) QueryRow(_ context.Context, query string, _ ...any) persistence.Row {
	return authSingleRow{values: s.rowValues[query], query: query}
}

func (s *authRowStub) Query(_ context.Context, _ string, _ ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

func (s *authRowStub) Exec(_ context.Context, query string, _ ...any) error {
	s.execs++
	return s.execErr[query]
}

type authSingleRow struct {
	values []any
	query  string
}

func (r authSingleRow) Scan(dest ...any) error {
	if r.values == nil {
		return pgx.ErrNoRows
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

func newAuthFactory(backend persistence.Queryable) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresAuthenticateAndUpdateLoginMeta(t *testing.T) {
	hash, _ := hashPassword("secret123")
	backend := &authRowStub{
		rowValues: map[string][]any{
			queryGetAccountByLoginID: {"alice", "user_1", hash},
		},
	}
	repo := NewPostgresRepository(newAuthFactory(backend))

	account, err := repo.Authenticate(context.Background(), "alice", "secret123")
	if err != nil || account.UserID != "user_1" {
		t.Fatalf("authenticate: %#v err=%v", account, err)
	}
	// 密码错误 → 统一 httpx.CodeUnauthorized 语义错误
	if _, err := repo.Authenticate(context.Background(), "alice", "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("wrong password must be invalid credentials, got %v", err)
	}
	// 用户不存在 → 同样视为凭据错误（防枚举）
	backend.rowValues[queryGetAccountByLoginID] = nil
	if _, err := repo.Authenticate(context.Background(), "nobody", "secret123"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("missing user must be invalid credentials, got %v", err)
	}

	if err := repo.UpdateLoginMeta(context.Background(), "user_1", "1.2.3.4", "US"); err != nil {
		t.Fatalf("update login meta: %v", err)
	}
}

func TestPostgresCreateAndResetPassword(t *testing.T) {
	hash, _ := hashPassword("secret123")
	now := time.Now().UTC()
	backend := &authRowStub{
		rowValues: map[string][]any{
			queryGetAccountByLoginID:          {"alice", "user_1", hash},
			queryLatestPasswordResetByLoginID: {"reset_1", "alice", "123456", now.Add(-time.Hour), now.Add(-time.Hour), nil},
			queryGetPasswordReset:             {"reset_1", "alice", "123456", now.Add(time.Hour), now, nil},
		},
	}
	repo := NewPostgresRepository(newAuthFactory(backend))

	code, err := repo.CreatePasswordReset(context.Background(), "alice", now.Add(time.Hour))
	if err != nil || code == "" {
		t.Fatalf("create reset: code=%q err=%v", code, err)
	}
	// 用户不存在 → 返回空 code（不暴露存在性）
	backend.rowValues[queryGetAccountByLoginID] = nil
	code, err = repo.CreatePasswordReset(context.Background(), "nobody", now.Add(time.Hour))
	if err != nil || code != "" {
		t.Fatalf("missing user reset: code=%q err=%v", code, err)
	}
	// 节流：最近一次重置在窗口内
	backend.rowValues[queryGetAccountByLoginID] = []any{"alice", "user_1", hash}
	backend.rowValues[queryLatestPasswordResetByLoginID] = []any{"reset_1", "alice", "123456", now.Add(time.Hour), now, nil}
	if _, err := repo.CreatePasswordReset(context.Background(), "alice", now.Add(time.Hour)); !errors.Is(err, ErrPasswordResetThrottled) {
		t.Fatalf("expected throttled, got %v", err)
	}

	account, err := repo.ResetPassword(context.Background(), "alice", "123456", "newpass123")
	if err != nil || account.UserID != "user_1" {
		t.Fatalf("reset password: %#v err=%v", account, err)
	}
	// 错误验证码 → not found
	backend.rowValues[queryGetPasswordReset] = nil
	if _, err := repo.ResetPassword(context.Background(), "alice", "000000", "newpass123"); !errors.Is(err, ErrPasswordResetNotFound) {
		t.Fatalf("bad code must be not found, got %v", err)
	}
}

func TestPostgresChangePassword(t *testing.T) {
	hash, _ := hashPassword("oldpass123")
	backend := &authRowStub{
		rowValues: map[string][]any{
			queryGetAccountByUserID: {"alice", "user_1", hash},
		},
	}
	repo := NewPostgresRepository(newAuthFactory(backend))

	if err := repo.ChangePassword(context.Background(), "user_1", "oldpass123", "newpass123"); err != nil {
		t.Fatalf("change password: %v", err)
	}
	if err := repo.ChangePassword(context.Background(), "user_1", "wrong", "newpass123"); !errors.Is(err, ErrInvalidOldPassword) {
		t.Fatalf("wrong old password must be rejected, got %v", err)
	}
	backend.rowValues[queryGetAccountByUserID] = nil
	if err := repo.ChangePassword(context.Background(), "missing", "oldpass123", "newpass123"); !errors.Is(err, ErrInvalidOldPassword) {
		t.Fatalf("missing user must be rejected, got %v", err)
	}
}
