package notify

import (
	"context"
	"errors"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

type sinkStub struct{ events []realtime.Event }

type notificationListCaptureRepo struct {
	userID     string
	appendUser string
	appendMany []string
	appended   []NotifyItem
	req        NotificationListRequest
	list       PaginatedNotify
	listErr    error
}

func (s *sinkStub) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *sinkStub) Close() error                { return nil }

func (r *notificationListCaptureRepo) Append(_ context.Context, userID string, item NotifyItem) error {
	r.appendUser = userID
	r.appended = append(r.appended, item)
	return nil
}

func (r *notificationListCaptureRepo) AppendMany(_ context.Context, userIDs []string, item NotifyItem) error {
	r.appendMany = append([]string(nil), userIDs...)
	r.appended = append(r.appended, item)
	return nil
}

func (r *notificationListCaptureRepo) List(_ context.Context, userID string, req NotificationListRequest) (PaginatedNotify, error) {
	r.userID = userID
	r.req = req
	return r.list, r.listErr
}

func (r *notificationListCaptureRepo) UnreadCount(context.Context, string) (UnreadCountResult, error) {
	return UnreadCountResult{}, nil
}

func (r *notificationListCaptureRepo) MarkAsRead(context.Context, string, []string) error { return nil }
func (r *notificationListCaptureRepo) MarkAllRead(context.Context, string) error          { return nil }
func (r *notificationListCaptureRepo) MarkAllReadByType(context.Context, string, string) error {
	return nil
}

func TestAppendNewFollowerBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendNewFollower(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "", ""); err != nil {
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

func TestAppendNewFollowerIncludesSourceFields(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendNewFollower(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "moment", "moment_123"); err != nil {
		t.Fatalf("append new follower with source: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Payload["source"] != "moment" || item.Payload["source_moment_id"] != "moment_123" {
		t.Fatalf("unexpected source payload: %#v", item.Payload)
	}
}

func TestAppendMutualFollowBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendMutualFollow(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}); err != nil {
		t.Fatalf("append mutual follow: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "mutual_follow" || item.Payload["type"] != "mutual_follow" {
		t.Fatalf("unexpected notification item: %#v", item)
	}
}

func TestAppendMomentCommentedBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendMomentCommented(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "moment_1", types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "summary"}, "comment_1", "preview"); err != nil {
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

func TestAppendMentionedBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendMentioned(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "comment", "moment_1", types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "summary"}, "comment_1", "preview"); err != nil {
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

func TestAppendGroupInviteReceivedBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	message := "join us"
	id, err := service.AppendGroupInviteReceived(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "group_1", "Hot Group", "invite_1", &message)
	if err != nil {
		t.Fatalf("append group invite received: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty notification_id")
	}
	item := repo.appended[0]
	if item.NotifyID != id {
		t.Fatalf("expected item.NotifyID=%q, got %q", id, item.NotifyID)
	}
	if item.Type != "group_invite_received" || item.Payload["invite_id"] != "invite_1" || item.Payload["group_id"] != "group_1" || item.Payload["message"] != "join us" {
		t.Fatalf("unexpected notification item: %#v", item)
	}
}

func TestAppendGroupApplicationAcceptedBuildsNotification(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	notificationID, err := service.AppendGroupApplicationAccepted(context.Background(), "user_alice", "group_1", "Hot Group", "app_1")
	if err != nil {
		t.Fatalf("append group application accepted: %v", err)
	}
	if notificationID == "" {
		t.Fatalf("expected generated notification_id, got empty")
	}
	item := repo.appended[0]
	if item.NotifyID != notificationID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notificationID, item.NotifyID)
	}
	if item.Type != "group_application_accepted" || item.Payload["application_id"] != "app_1" {
		t.Fatalf("unexpected notification item: %#v", item)
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
	payload := systemmodule.SystemAnnouncementItem{AnnouncementID: "ann_1", Title: "维护通知", Content: "今晚维护", Severity: systemmodule.AnnounceLevelInfo, ActionURL: "https://example.com/maint", CreatedAt: "2026-03-26T00:00:00Z"}
	service.PushSystemAnnouncement(payload, "notif_1")
	if len(aliceSink.events) != 1 || len(bobSink.events) != 1 {
		t.Fatalf("unexpected events: %#v %#v", aliceSink.events, bobSink.events)
	}
	if aliceSink.events[0].Type != "SystemAnnouncement" || bobSink.events[0].Type != "SystemAnnouncement" {
		t.Fatalf("unexpected event types: %#v %#v", aliceSink.events, bobSink.events)
	}
	alicePayload, ok := aliceSink.events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected alice payload type: %#v", aliceSink.events[0].Payload)
	}
	// spec websocket.md → SystemAnnouncement：payload 结构为 {notify_id, announcement}
	aliceAnnouncement, ok := alicePayload["announcement"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected alice announcement payload: %#v", alicePayload)
	}
	if alicePayload["notify_id"] != "notif_1" || aliceAnnouncement["announcement_id"] != "ann_1" || aliceAnnouncement["title"] != "维护通知" || aliceAnnouncement["content"] != "今晚维护" || aliceAnnouncement["action_url"] != "https://example.com/maint" || aliceAnnouncement["severity"] != systemmodule.AnnounceLevelInfo {
		t.Fatalf("unexpected alice payload: %#v", alicePayload)
	}
	if bobPayload, ok := bobSink.events[0].Payload.(map[string]any); !ok || bobPayload["notify_id"] != "notif_1" {
		t.Fatalf("unexpected bob payload: %#v", bobSink.events[0].Payload)
	}
}

func TestAppendSystemAnnouncementBuildsNotificationsForUsers(t *testing.T) {
	repo := &notificationListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	payload := systemmodule.SystemAnnouncementItem{AnnouncementID: "ann_1", Title: "维护通知", Content: "今晚维护", Severity: systemmodule.AnnounceLevelInfo, CreatedAt: "2026-03-26T00:00:00Z"}
	notificationID, err := service.AppendSystemAnnouncement(context.Background(), []string{"user_alice", "user_bob"}, payload)
	if err != nil {
		t.Fatalf("append system announcement: %v", err)
	}
	if notificationID == "" {
		t.Fatal("expected generated notification_id, got empty")
	}
	if len(repo.appendMany) != 2 || len(repo.appended) != 1 {
		t.Fatalf("unexpected append-many capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.NotifyID != notificationID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notificationID, item.NotifyID)
	}
	if item.Type != "system_announcement" || item.Payload["announcement_id"] != "ann_1" || item.Payload["type"] != "system_announcement" {
		t.Fatalf("unexpected system notification item: %#v", item)
	}
}

func TestServiceListNormalizesPaginationAndCursor(t *testing.T) {
	repo := &notificationListCaptureRepo{list: PaginatedNotify{Items: []NotifyItem{{NotifyID: "notif_1"}}}}
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	cursor := encodeNotificationCursor(notificationCursor{CreatedAt: "2026-03-26T00:00:00Z", NotifyID: "notif_9"})

	list, err := service.List(ctx, NotificationListRequest{Type: "new_follower", Pagination: &PaginationParams{Limit: 999, Cursor: cursor}})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].NotifyID != "notif_1" {
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
	if repo.req.Pagination.Cursor == "" {
		t.Fatal("expected a cursor to be passed to the repo")
	}
	// cursor 是 AES-GCM 加密 token（nonce 随机），断言字节相等无意义，
	// 改为解码后验证语义一致。
	decodedCursor, err := decodeNotificationCursor(repo.req.Pagination.Cursor)
	if err != nil || decodedCursor == nil || decodedCursor.CreatedAt != "2026-03-26T00:00:00Z" || decodedCursor.NotifyID != "notif_9" {
		t.Fatalf("unexpected cursor payload: %#v err=%v", decodedCursor, err)
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
