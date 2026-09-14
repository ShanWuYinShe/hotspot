package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// userRelStub 提供 QueryRow（关系查询）+ Exec（写入）的 stub。
type userRelStub struct {
	relationRow []any
	execErr     error
	// queryErr 注入 QueryRow 的扫描错误：Follow 现走 QueryRow + RETURNING
	// （审计 F-12 的 changed 语义），不再经 Exec 路径。
	queryErr error
	execs    int
}

func (s *userRelStub) QueryRow(_ context.Context, _ string, _ ...any) persistence.Row {
	return userRelRow{values: s.relationRow, err: s.queryErr}
}

func (s *userRelStub) Query(_ context.Context, _ string, _ ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}

func (s *userRelStub) Exec(_ context.Context, _ string, _ ...any) error {
	s.execs++
	return s.execErr
}

type userRelRow struct {
	values []any
	err    error
}

func (r userRelRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if r.values == nil {
		return errNoRows
	}
	for i := range dest {
		switch d := dest[i].(type) {
		case *bool:
			*d = r.values[i].(bool)
		case *string:
			*d = r.values[i].(string)
		case *int:
			*d = r.values[i].(int)
		case **time.Time:
			if r.values[i] == nil {
				*d = nil
			} else if v, ok := r.values[i].(time.Time); ok {
				*d = &v
			}
		}
	}
	return nil
}

func newUserFactory(backend persistence.Queryable) *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return factory
}

func TestPostgresFollowBlockUnblockLifecycle(t *testing.T) {
	backend := &userRelStub{relationRow: []any{false, false, "", nil, nil, ""}}
	repo := NewPostgresRepository(newUserFactory(backend))
	ctx := context.Background()

	if _, err := repo.Follow(ctx, "u1", "u2", "search", "m1"); err != nil {
		t.Fatalf("follow: %v", err)
	}
	if err := repo.Unfollow(ctx, "u1", "u2"); err != nil {
		t.Fatalf("unfollow: %v", err)
	}
	if err := repo.Block(ctx, "u1", "u3", "spam"); err != nil {
		t.Fatalf("block: %v", err)
	}
	if err := repo.Unblock(ctx, "u1", "u3"); err != nil {
		t.Fatalf("unblock: %v", err)
	}
	if err := repo.SetRemark(ctx, "u1", "u2", "备注"); err != nil {
		t.Fatalf("set remark: %v", err)
	}
	if backend.execs < 4 {
		t.Fatalf("expected writes, got %d", backend.execs)
	}

	// 关系查询出错 → 错误传播（Follow 走 QueryRow + RETURNING）
	backend.queryErr = errFakeDB
	if _, err := repo.Follow(ctx, "u1", "u2", "", ""); err == nil {
		t.Fatal("expected follow error propagation")
	}
}

func TestPostgresPutSettingsAndDeleteProfile(t *testing.T) {
	backend := &userRelStub{relationRow: []any{false, false, "", nil, nil, ""}}
	repo := NewPostgresRepository(newUserFactory(backend))
	ctx := context.Background()

	settings := UserSettings{PrivacyLevel: "public", AllowPrivateChat: ChatPerm("all")}
	if err := repo.PutSettings(ctx, "u1", settings); err != nil {
		t.Fatalf("put settings: %v", err)
	}
	if err := repo.DeleteProfile(ctx, "u1"); err != nil {
		t.Fatalf("delete profile: %v", err)
	}
}

func TestPostgresUsersNotConfigured(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	ctx := context.Background()
	if _, err := repo.Follow(ctx, "u1", "u2", "", ""); err != persistence.ErrNotImplemented {
		t.Fatalf("follow: %v", err)
	}
	if err := repo.PutSettings(ctx, "u1", UserSettings{}); err != persistence.ErrNotImplemented {
		t.Fatalf("put settings: %v", err)
	}
	if err := repo.DeleteProfile(ctx, "u1"); err != persistence.ErrNotImplemented {
		t.Fatalf("delete profile: %v", err)
	}
}

var errFakeDB = errors.New("fake db error")

var errNoRows = errors.New("no rows")
