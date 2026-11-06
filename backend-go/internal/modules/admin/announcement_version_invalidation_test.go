package admin

import (
	"context"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

// recordingAnnouncementStore 记录管理端公告写次数：公告归 system 模块所有，
// admin 只经 AnnouncementAdminStore 最小接口读写，不再持有 system.Repository
// 副本或跨实例缓存失效钩子（架构审计 P0-1 隐藏实例）。
type recordingAnnouncementStore struct {
	inner                     *InMemoryModerationStore
	creates, updates, deletes int
}

func (s *recordingAnnouncementStore) SearchAnnouncements(ctx context.Context, filter systemmodule.AnnouncementSearchFilter) (systemmodule.AnnouncementPage, error) {
	return s.inner.SearchAnnouncements(ctx, filter)
}

func (s *recordingAnnouncementStore) GetAnnouncement(ctx context.Context, announcementID string) (systemmodule.AnnouncementRecord, bool, error) {
	return s.inner.GetAnnouncement(ctx, announcementID)
}

func (s *recordingAnnouncementStore) CreateAnnouncement(ctx context.Context, input systemmodule.AnnouncementCreateInput) (systemmodule.AnnouncementRecord, error) {
	s.creates++
	return s.inner.CreateAnnouncement(ctx, input)
}

func (s *recordingAnnouncementStore) UpdateAnnouncement(ctx context.Context, input systemmodule.AnnouncementUpdateInput) (systemmodule.AnnouncementRecord, error) {
	s.updates++
	return s.inner.UpdateAnnouncement(ctx, input)
}

func (s *recordingAnnouncementStore) DeleteAnnouncement(ctx context.Context, announcementID string) error {
	s.deletes++
	return s.inner.DeleteAnnouncement(ctx, announcementID)
}

func TestAnnouncementWritesGoThroughSystemStore(t *testing.T) {
	announcements := &recordingAnnouncementStore{inner: NewInMemoryModerationStore()}
	store := NewPersistenceModerationStore(nil, announcements, nil)
	svc := NewService(NewInMemoryRepository(), &stubProvider{config: defaultConfig()}, store, &stubPushService{})
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
	if announcements.creates != 1 || announcements.updates != 1 || announcements.deletes != 1 {
		t.Fatalf("write delegation = create:%d update:%d delete:%d, want 1/1/1",
			announcements.creates, announcements.updates, announcements.deletes)
	}
}
