package admin

import (
	"context"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

// TestAnnouncementWritesInvalidateBootstrapVersion：admin 用独立 systemRepo 实例写公告，
// 必须显式失效 system 模块的 bootstrap 版本缓存（否则 ETag 最长延迟 5s 才变化）。
func TestAnnouncementWritesInvalidateBootstrapVersion(t *testing.T) {
	svc, _, _, _ := newTestService()
	previousHub := realtime.DefaultHub()
	realtime.SetDefaultHub(realtime.NewHub())
	t.Cleanup(func() { realtime.SetDefaultHub(previousHub) })

	invalidations := 0
	svc.SetAnnouncementsVersionInvalidator(func() { invalidations++ })
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "admin", SessionID: "sess_admin"})

	created, apiErr, err := svc.CreateAnnouncement(ctx, CreateAnnouncementRequest{
		Title:    "维护通知",
		Content:  "今晚 22:00 进行例行维护，请提前保存。",
		Severity: AnnounceLevelInfo,
	})
	if err != nil || apiErr != nil {
		t.Fatalf("CreateAnnouncement err=%v apiErr=%+v", err, apiErr)
	}
	title := "维护通知（更新）"
	if _, apiErr, err = svc.UpdateAnnouncement(ctx, UpdateAnnouncementRequest{AnnouncementID: created.AnnouncementID, Title: &title}); err != nil || apiErr != nil {
		t.Fatalf("UpdateAnnouncement err=%v apiErr=%+v", err, apiErr)
	}
	if _, err = svc.DeleteAnnouncement(ctx, created.AnnouncementID); err != nil {
		t.Fatalf("DeleteAnnouncement: %v", err)
	}
	if invalidations != 3 {
		t.Fatalf("announcement version invalidations = %d, want 3 (create/update/delete)", invalidations)
	}
}
