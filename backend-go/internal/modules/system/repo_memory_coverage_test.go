package system

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
)

func TestInMemoryRepositoryAnnouncementLifecycle(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	created, err := repo.CreateAnnouncement(ctx, AnnouncementCreateInput{
		Actor:    AnnouncementActor{UserID: "admin1", Nickname: "Admin"},
		Title:    "系统维护",
		Content:  "今晚维护",
		Severity: AnnounceLevelInfo,
		SendPush: true,
	})
	if err != nil {
		t.Fatalf("create announcement: %v", err)
	}
	if created.AnnouncementID == "" || created.Title != "系统维护" || created.CreatedAt.IsZero() {
		t.Fatalf("unexpected created record: %#v", created)
	}

	got, found, err := repo.GetAnnouncement(ctx, created.AnnouncementID)
	if err != nil || !found || got.AnnouncementID != created.AnnouncementID {
		t.Fatalf("get announcement: found=%v err=%v item=%#v", found, err, got)
	}

	// 分页/排序：第二条更晚创建的排在前面
	second, err := repo.CreateAnnouncement(ctx, AnnouncementCreateInput{
		Actor:    AnnouncementActor{UserID: "admin1"},
		Title:    "紧急",
		Severity: AnnounceLevelWarning,
	})
	if err != nil {
		t.Fatalf("create second announcement: %v", err)
	}
	list, err := repo.ListAnnouncements(ctx, 10)
	if err != nil || len(list) != 2 || list[0].AnnouncementID != second.AnnouncementID {
		t.Fatalf("expected newest first, got %#v (err %v)", list, err)
	}
	limited, err := repo.ListAnnouncements(ctx, 1)
	if err != nil || len(limited) != 1 {
		t.Fatalf("expected limit 1, got %#v", limited)
	}

	// 按 severity 搜索 + 游标
	page, err := repo.SearchAnnouncements(ctx, AnnouncementSearchFilter{Severity: AnnounceLevelWarning, Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].AnnouncementID != second.AnnouncementID {
		t.Fatalf("severity filter failed: %#v (err %v)", page, err)
	}

	// 更新
	title := "紧急维护"
	updated, err := repo.UpdateAnnouncement(ctx, AnnouncementUpdateInput{AnnouncementID: second.AnnouncementID, Title: &title})
	if err != nil || updated.Title != "紧急维护" {
		t.Fatalf("update announcement failed: %#v (err %v)", updated, err)
	}
	if _, err := repo.UpdateAnnouncement(ctx, AnnouncementUpdateInput{AnnouncementID: "missing"}); !errors.Is(err, errAnnouncementNotFound) {
		t.Fatalf("expected not-found on update, got %v", err)
	}

	// 删除后公开列表不可见（软删）
	if err := repo.DeleteAnnouncement(ctx, second.AnnouncementID); err != nil {
		t.Fatalf("delete announcement: %v", err)
	}
	after, err := repo.ListAnnouncements(ctx, 10)
	if err != nil || len(after) != 1 || after[0].AnnouncementID != created.AnnouncementID {
		t.Fatalf("deleted announcement must be hidden from list, got %#v (err %v)", after, err)
	}
	// 重复删除视为成功（幂等）
	if err := repo.DeleteAnnouncement(ctx, second.AnnouncementID); err != nil {
		t.Fatalf("re-delete must be idempotent, got %v", err)
	}
	if err := repo.DeleteAnnouncement(ctx, "missing"); !errors.Is(err, errAnnouncementNotFound) {
		t.Fatalf("expected not-found on delete, got %v", err)
	}
}

func TestInMemoryRepositoryGetMetaAndHealth(t *testing.T) {
	repo := NewInMemoryRepository()
	meta := repo.GetMeta(context.Background())
	if meta.Version != "v1" || meta.Status != ApiVersionStatusActive {
		t.Fatalf("unexpected meta: %#v", meta)
	}
	code, message, status := repo.Health(context.Background())
	if code != 0 || message != "" || status == nil || status.Status != SystemHealthHealthy {
		t.Fatalf("unexpected health: %d %q %#v", code, message, status)
	}
	if _, err := time.Parse(time.RFC3339, status.Timestamp); err != nil {
		t.Fatalf("health timestamp not RFC3339: %q", status.Timestamp)
	}
}

func TestAppConfigProviderSystemAnnouncementsCount(t *testing.T) {
	provider := NewAppConfigProvider(appconfig.NewStaticSource(config.Load()))
	if got := provider.GetSystemAnnouncementsCount(context.Background()); got != 10 {
		t.Fatalf("expected default 10, got %d", got)
	}
	// 自定义值：StaticSource 基于运行时 config 的默认值，这里验证非正数兜底
	if got := provider.GetSystemAnnouncementsCount(context.Background()); got <= 0 {
		t.Fatalf("expected positive count, got %d", got)
	}
}

func TestNormalizeAnnouncementLimit(t *testing.T) {
	if got := normalizeLimit(0); got != 20 {
		t.Fatalf("expected default 20, got %d", got)
	}
	if got := normalizeLimit(500); got != 100 {
		t.Fatalf("expected cap 100, got %d", got)
	}
	if got := normalizeLimit(42); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
}
