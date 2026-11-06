package privatemessages

import (
	"context"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// coverQueryBackend 提供空 Query/QueryRow/Exec：用于覆盖 PG 仓储剩余读/写路径
// 的骨架分支（命中/未命中由真实库 IT 覆盖），不引入假数据依赖。
type coverQueryBackend struct{}

func (coverQueryBackend) Exec(context.Context, string, ...any) error { return nil }
func (coverQueryBackend) Query(context.Context, string, ...any) (persistence.Rows, error) {
	return &persistence.InMemoryRows{}, nil
}
func (coverQueryBackend) QueryRow(context.Context, string, ...any) persistence.Row {
	return persistence.InMemoryRow{}
}

func newCoverFactory() *persistence.Factory {
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = coverQueryBackend{}
	return factory
}

// TestModuleWiringCoverage 覆盖一等模块装配/路由挂载/跨模块能力 accessor。
func TestModuleWiringCoverage(t *testing.T) {
	source := appconfig.NewStaticSource(config.Load())
	module := NewModule(persistence.NewFactory(config.Config{}), source, nil, &stubUserReader{})
	if module.MessageService() == nil || module.Service() == nil {
		t.Fatal("module must expose read/write services")
	}
	module.SetMessageReactionsSource(nil)
	r := chi.NewRouter()
	module.Mount(r)
}

// TestPostgresRemainingPathsCoverage 覆盖 PG 仓储剩余方法骨架（空结果/错误分支）；
// 真实语义由 IT 覆盖。
func TestPostgresRemainingPathsCoverage(t *testing.T) {
	ctx := context.Background()
	repo := NewPostgresRepository(newCoverFactory())
	userID := "user_cover"
	_, _, _ = repo.GetMessageForReply(ctx, "msg_1", userID)
	_, _, _ = repo.GetMediaForMessage(ctx, "file_1", userID)
	_, _ = repo.RecallAsAdmin(ctx, "msg_1")
	_, _, _, _ = repo.MessageParticipants(ctx, "msg_1")
	_, _, _ = repo.MessageExists(ctx, "msg_1")
	_, _, _ = repo.GetMessageReportSnapshot(ctx, "msg_1")
	_, _ = repo.CountMessagesByTarget(ctx, userID, time.Now().Add(-24*time.Hour))
	if err := repo.LockStrangerQuota(ctx, userID, "2026-01-01"); err != nil {
		t.Fatalf("lock quota: %v", err)
	}
	if _, err := repo.LastByChatKeys(ctx, userID, []string{"a:b"}, nil); err != nil {
		t.Fatalf("last by chat keys: %v", err)
	}
	if _, err := repo.UnreadCountsByChatKeys(ctx, userID, []string{"a:b"}, nil); err != nil {
		t.Fatalf("unread by chat keys: %v", err)
	}
	if err := repo.PurgeUserData(ctx, userID); err != nil {
		t.Fatalf("purge repo: %v", err)
	}
	svc := NewServiceWithDeps(repo, nil)
	if err := svc.PurgeUserData(ctx, userID); err != nil {
		t.Fatalf("purge service: %v", err)
	}
}
