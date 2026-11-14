package system

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/persistence"
)

// 本文件锁定第 81 轮审计的「公告软删同事务清理 announcement_reads」口径：
// 公告删除是软删（deleted_at），067 迁移的 ON DELETE CASCADE 不会触发，
// 必须由 DeleteAnnouncement 显式清理，且与软删同事务（真实库 IT 见
// announcement_notify_it_test.go 的同事务清理用例）。
func newAnnouncementCleanupRepo(t *testing.T) (*PostgresRepository, *announcementBackendStub) {
	t.Helper()
	backend := newAnnouncementBackendStub()
	factory := persistence.NewFactory(config.Config{PostgresDSN: "postgres://example"})
	factory.DB.(*persistence.PostgresDB).Backend = backend
	return NewPostgresRepository(factory), backend
}

func seedAnnouncementForCleanup(backend *announcementBackendStub, announcementID string) {
	backend.announcements[announcementID] = AnnouncementRecord{
		AnnouncementID: announcementID,
		Title:          announcementID,
		Content:        "内容",
		Severity:       AnnounceLevelInfo,
		CreatedAt:      time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC),
		CreatedBy:      AnnouncementActor{UserID: "admin", Nickname: "Admin"},
	}
}

func TestPostgresRepositoryDeleteAnnouncementCleansAnnouncementReads(t *testing.T) {
	repo, backend := newAnnouncementCleanupRepo(t)
	ctx := context.Background()
	seedAnnouncementForCleanup(backend, "ann_del")
	seedAnnouncementForCleanup(backend, "ann_keep")
	backend.announcementReads["ann_del"] = map[string]bool{"user_1": true, "user_2": true}
	backend.announcementReads["ann_keep"] = map[string]bool{"user_1": true}

	if err := repo.DeleteAnnouncement(ctx, "ann_del"); err != nil {
		t.Fatalf("delete announcement: %v", err)
	}
	if !backend.announcements["ann_del"].IsDeleted() {
		t.Fatal("公告应软删（deleted_at 置位）")
	}
	if got := len(backend.announcementReads["ann_del"]); got != 0 {
		t.Fatalf("被删公告的 announcement_reads 行 = %d，期望 0（必须同事务清理）", got)
	}
	if got := len(backend.announcementReads["ann_keep"]); got != 1 {
		t.Fatalf("其它公告的 announcement_reads 行 = %d，期望 1（不得受影响）", got)
	}

	// 重复删除幂等：不报错、不再清理其它公告。
	if err := repo.DeleteAnnouncement(ctx, "ann_del"); err != nil {
		t.Fatalf("delete announcement again must be idempotent: %v", err)
	}
	if got := len(backend.announcementReads["ann_keep"]); got != 1 {
		t.Fatalf("幂等删除后其它公告 reads = %d，期望 1", got)
	}
}

func TestPostgresRepositoryDeleteAnnouncementCleanupErrorPropagates(t *testing.T) {
	repo, backend := newAnnouncementCleanupRepo(t)
	seedAnnouncementForCleanup(backend, "ann_err")
	cleanupErr := errors.New("cleanup reads failed")
	backend.execErrs[queryDeleteAnnouncementReads] = cleanupErr

	if err := repo.DeleteAnnouncement(context.Background(), "ann_err"); !errors.Is(err, cleanupErr) {
		t.Fatalf("cleaning announcement_reads failure must propagate, got %v", err)
	}
}
