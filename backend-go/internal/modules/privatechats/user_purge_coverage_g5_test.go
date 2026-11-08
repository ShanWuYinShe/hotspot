package privatechats

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// TestCoverageG5ServicePurgeUserDataBranches 覆盖账户注销级联的两种仓储形态：
// 内存/测试仓储无跨模块行可清时返回 nil；PG 仓储未接库时返回 ErrNotImplemented。
func TestCoverageG5ServicePurgeUserDataBranches(t *testing.T) {
	ctx := context.Background()

	memorySvc := NewServiceWithDeps(NewInMemoryRepository(), &stubMsgReadService{}, &stubChatUserReader{})
	if err := memorySvc.PurgeUserData(ctx, "u1"); err != nil {
		t.Fatalf("memory repo purge must be a no-op, got %v", err)
	}

	pgSvc := NewServiceWithDeps(
		NewPostgresRepository(persistence.NewFactory(config.Config{})),
		&stubMsgReadService{},
		&stubChatUserReader{},
	)
	if err := pgSvc.PurgeUserData(ctx, "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented from PG repo, got %v", err)
	}
}

// TestCoverageG5PostgresPurgeUserDataNotConfigured 覆盖 PostgresRepository
// 未接库时的前置检查分支。
func TestCoverageG5PostgresPurgeUserDataNotConfigured(t *testing.T) {
	repo := NewPostgresRepository(persistence.NewFactory(config.Config{}))
	if err := repo.PurgeUserData(context.Background(), "u1"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("expected ErrNotImplemented, got %v", err)
	}
}
