package notifications

import (
	"context"
	"errors"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
)

type sinkStub struct{ events []realtime.Event }

type notificationListCaptureRepo struct {
	userID     string
	appendUser string
	appendMany []string
	appended   []NotificationItem
	req        NotificationListRequest
	list       PaginatedNotifications
	listErr    error
}

func (s *sinkStub) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *sinkStub) Close() error                { return nil }

func (r *notificationListCaptureRepo) Append(_ context.Context, userID string, item NotificationItem) error {
	r.appendUser = userID
	r.appended = append(r.appended, item)
	return nil
}

func (r *notificationListCaptureRepo) AppendMany(_ context.Context, userIDs []string, item NotificationItem) error {
	r.appendMany = append([]string(nil), userIDs...)
	r.appended = append(r.appended, item)
	return nil
}

func (r *notificationListCaptureRepo) List(_ context.Context, userID string, req NotificationListRequest) (PaginatedNotifications, error) {
	r.userID = userID
	r.req = req
	return r.list, r.listErr
}

func (r *notificationListCaptureRepo) UnreadCount(context.Context, string) (UnreadCountResult, error) {
	return UnreadCountResult{}, nil
}

func (r *notificationListCaptureRepo) MarkAsRead(context.Context, string, []string) error { return nil }
func (r *notificationListCaptureRepo) MarkAllRead(context.Context, string) error          { return nil }

func TestAppendNewFollowerBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if err := service.AppendNewFollower(context.Background(), "user_alice", map[string]any{"user_id": "user_bob", "nickname": "Bob"}); err != nil {
		t.Fatalf("append new follower: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "new_follower" || item.Payload["type"] != "new_follower" {
		t.Fatalf("unexpected notification item: %#v", item)
	}
}

func TestAppendMomentLikedBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if err := service.AppendMomentLiked(context.Background(), "user_alice", map[string]any{"user_id": "user_bob", "nickname": "Bob"}, "moment_1", map[string]any{"moment_id": "moment_1", "description_preview": "summary"}); err != nil {
		t.Fatalf("append moment liked: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "moment_liked" || item.Payload["type"] != "moment_liked" {
		t.Fatalf("unexpected notification item: %#v", item)
	}
	if item.Payload["moment_id"] != "moment_1" {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
}

func TestAppendMomentCommentedBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if err := service.AppendMomentCommented(context.Background(), "user_alice", map[string]any{"user_id": "user_bob", "nickname": "Bob"}, "moment_1", map[string]any{"moment_id": "moment_1", "description_preview": "summary"}, "comment_1", "preview"); err != nil {
		t.Fatalf("append moment commented: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "moment_commented" || item.Payload["type"] != "moment_commented" {
		t.Fatalf("unexpected notification item: %#v", item)
	}
	if item.Payload["comment_id"] != "comment_1" || item.Payload["comment_preview"] != "preview" {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
}

func TestAppendCommentLikedBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if err := service.AppendCommentLiked(context.Background(), "user_alice", map[string]any{"user_id": "user_bob", "nickname": "Bob"}, "comment_1", "preview", "moment_1", map[string]any{"moment_id": "moment_1", "description_preview": "summary"}); err != nil {
		t.Fatalf("append comment liked: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "comment_liked" || item.Payload["type"] != "comment_liked" {
		t.Fatalf("unexpected notification item: %#v", item)
	}
	if item.Payload["comment_id"] != "comment_1" || item.Payload["comment_preview"] != "preview" || item.Payload["moment_id"] != "moment_1" {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
}

func TestAppendMentionedBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if err := service.AppendMentioned(context.Background(), "user_alice", map[string]any{"user_id": "user_bob", "nickname": "Bob"}, "comment", "moment_1", map[string]any{"moment_id": "moment_1", "description_preview": "summary"}, "comment_1", "preview"); err != nil {
		t.Fatalf("append mentioned: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "mentioned" || item.Payload["type"] != "mentioned" {
		t.Fatalf("unexpected notification item: %#v", item)
	}
	if item.Payload["target_type"] != "comment" || item.Payload["comment_id"] != "comment_1" || item.Payload["comment_preview"] != "preview" {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
}

func TestPushSystemAnnouncementFansOut(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	aliceSink := &sinkStub{}
	bobSink := &sinkStub{}
	hub.RegisterUser("user_alice", "sess_1", aliceSink)
	hub.RegisterUser("user_bob", "sess_2", bobSink)
	service := NewService()
	payload := systemmodule.SystemAnnouncementItem{AnnouncementID: "ann_1", Title: "维护通知", Content: "今晚维护", Severity: systemmodule.AnnouncementSeverityInfo, CreatedAt: "2026-03-26T00:00:00Z"}
	service.PushSystemAnnouncement(payload)
	if len(aliceSink.events) != 1 || len(bobSink.events) != 1 {
		t.Fatalf("unexpected events: %#v %#v", aliceSink.events, bobSink.events)
	}
	if aliceSink.events[0].Type != "SystemAnnouncement" || bobSink.events[0].Type != "SystemAnnouncement" {
		t.Fatalf("unexpected event types: %#v %#v", aliceSink.events, bobSink.events)
	}
	alicePayload, ok := aliceSink.events[0].Payload.(systemmodule.SystemAnnouncementItem)
	if !ok || alicePayload.AnnouncementID != "ann_1" || alicePayload.Title != "维护通知" {
		t.Fatalf("unexpected alice payload: %#v", aliceSink.events[0].Payload)
	}
}

func TestAppendSystemAnnouncementBuildsNotificationsForUsers(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	payload := systemmodule.SystemAnnouncementItem{AnnouncementID: "ann_1", Title: "维护通知", Content: "今晚维护", Severity: systemmodule.AnnouncementSeverityInfo, CreatedAt: "2026-03-26T00:00:00Z"}
	if err := service.AppendSystemAnnouncement(context.Background(), []string{"user_alice", "user_bob"}, payload); err != nil {
		t.Fatalf("append system announcement: %v", err)
	}
	if len(repo.appendMany) != 2 || len(repo.appended) != 1 {
		t.Fatalf("unexpected append-many capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "system_announcement" || item.Payload["announcement_id"] != "ann_1" || item.Payload["type"] != "system_announcement" {
		t.Fatalf("unexpected system notification item: %#v", item)
	}
}

func TestServiceListNormalizesPaginationAndCursor(t *testing.T) {
	repo := &notificationListCaptureRepo{list: PaginatedNotifications{Items: []NotificationItem{{NotificationID: "notif_1"}}}}
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	cursor := encodeNotificationCursor(notificationCursor{CreatedAt: "2026-03-26T00:00:00Z", NotificationID: "notif_9"})

	list, err := service.List(ctx, NotificationListRequest{Type: "new_follower", Pagination: &PaginationParams{Limit: 999, Cursor: cursor}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].NotificationID != "notif_1" {
		t.Fatalf("unexpected list: %#v", list)
	}
	if repo.userID != "user_alice" {
		t.Fatalf("unexpected user id: %q", repo.userID)
	}
	if repo.req.Type != "new_follower" {
		t.Fatalf("unexpected type: %#v", repo.req)
	}
	if repo.req.Pagination == nil || repo.req.Pagination.Limit != maxNotificationListLimit {
		t.Fatalf("unexpected pagination limit: %#v", repo.req.Pagination)
	}
	if repo.req.Pagination.Cursor != cursor {
		t.Fatalf("unexpected cursor: got %q want %q", repo.req.Pagination.Cursor, cursor)
	}
}

func TestServiceListAppliesDefaultLimit(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	if _, err := service.List(ctx, NotificationListRequest{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.req.Pagination == nil || repo.req.Pagination.Limit != defaultNotificationListLimit {
		t.Fatalf("unexpected default pagination: %#v", repo.req.Pagination)
	}
	if repo.req.Pagination.Cursor != "" {
		t.Fatalf("unexpected cursor: %#v", repo.req.Pagination)
	}
}

func TestServiceListReturnsInvalidRequestError(t *testing.T) {
	service := NewServiceWithRepo(&notificationListCaptureRepo{})
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	if _, err := service.List(ctx, NotificationListRequest{Type: "bad_type"}); !errors.Is(err, ErrInvalidNotificationRequest) {
		t.Fatalf("expected invalid request error, got %v", err)
	}
	if _, err := service.List(ctx, NotificationListRequest{Pagination: &PaginationParams{Cursor: "%%%"}}); !errors.Is(err, ErrInvalidNotificationRequest) {
		t.Fatalf("expected invalid cursor error, got %v", err)
	}
}

func TestServiceListReturnsRepositoryError(t *testing.T) {
	repo := &notificationListCaptureRepo{listErr: errors.New("boom")}
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	if _, err := service.List(ctx, NotificationListRequest{}); !errors.Is(err, repo.listErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
}
