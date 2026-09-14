package notify

import (
	"context"
	"errors"
	"testing"
	"time"

	systemmodule "hotspot/backend-go/internal/modules/system"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

type sinkStub struct{ events []realtime.Event }

type notifyListCaptureRepo struct {
	userID             string
	appendUser         string
	appendMany         []string
	appendBatchUserIDs []string
	appended           []NotifyItem
	req                NotifyListRequest
	list               PaginatedNotify
	listErr            error
}

func (s *sinkStub) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *sinkStub) Close() error                { return nil }

func (r *notifyListCaptureRepo) Append(_ context.Context, userID string, item NotifyItem) error {
	r.appendUser = userID
	r.appended = append(r.appended, item)
	return nil
}

func (r *notifyListCaptureRepo) AppendMany(_ context.Context, userIDs []string, item NotifyItem) error {
	r.appendMany = append([]string(nil), userIDs...)
	r.appended = append(r.appended, item)
	return nil
}

func (r *notifyListCaptureRepo) AppendBatchPerUser(_ context.Context, userIDs []string, items []NotifyItem) error {
	r.appendBatchUserIDs = append([]string(nil), userIDs...)
	r.appended = append(r.appended, items...)
	return nil
}

func (r *notifyListCaptureRepo) List(_ context.Context, userID string, req NotifyListRequest) (PaginatedNotify, error) {
	r.userID = userID
	r.req = req
	return r.list, r.listErr
}

func (r *notifyListCaptureRepo) UnreadCount(context.Context, string) (UnreadSummaryResult, error) {
	return UnreadSummaryResult{}, nil
}

func (r *notifyListCaptureRepo) MarkAsRead(context.Context, string, []string) error { return nil }
func (r *notifyListCaptureRepo) MarkAllRead(context.Context, string) error          { return nil }
func (r *notifyListCaptureRepo) PruneReadBefore(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *notifyListCaptureRepo) MarkAllReadByType(context.Context, string, string) error {
	return nil
}

func TestAppendNewFollowerBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendNewFollower(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "", ""); err != nil {
		t.Fatalf("append new follower: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "new_follower" || item.Payload["type"] != "new_follower" {
		t.Fatalf("unexpected notify item: %#v", item)
	}
}

func TestAppendNewFollowerIncludesSourceFields(t *testing.T) {
	repo := &notifyListCaptureRepo{}
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

func TestAppendMutualFollowBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendMutualFollow(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}); err != nil {
		t.Fatalf("append mutual follow: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "mutual_follow" || item.Payload["type"] != "mutual_follow" {
		t.Fatalf("unexpected notify item: %#v", item)
	}
}

func TestAppendMomentCommentedBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendMomentCommented(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "moment_1", types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "summary"}, "comment_1", "preview"); err != nil {
		t.Fatalf("append moment commented: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "moment_commented" || item.Payload["type"] != "moment_commented" {
		t.Fatalf("unexpected notify item: %#v", item)
	}
	if item.Payload["comment_id"] != "comment_1" || item.Payload["comment_preview"] != "preview" {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
}

func TestAppendMentionedBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	if _, err := service.AppendMentioned(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "comment", "moment_1", types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "summary"}, "comment_1", "preview"); err != nil {
		t.Fatalf("append mentioned: %v", err)
	}
	if repo.appendUser != "user_alice" || len(repo.appended) != 1 {
		t.Fatalf("unexpected append capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.Type != "mentioned" || item.Payload["type"] != "mentioned" {
		t.Fatalf("unexpected notify item: %#v", item)
	}
	if item.Payload["target_type"] != "comment" || item.Payload["comment_id"] != "comment_1" || item.Payload["comment_preview"] != "preview" {
		t.Fatalf("unexpected payload: %#v", item.Payload)
	}
}

func TestAppendGroupInviteReceivedBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	message := "join us"
	id, err := service.AppendGroupInviteReceived(context.Background(), "user_alice", types.UserInfo{UserID: "user_bob", Nickname: "Bob"}, "group_1", "Hot Group", "invite_1", &message)
	if err != nil {
		t.Fatalf("append group invite received: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty notify_id")
	}
	item := repo.appended[0]
	if item.NotifyID != id {
		t.Fatalf("expected item.NotifyID=%q, got %q", id, item.NotifyID)
	}
	if item.Type != "group_invite_received" || item.Payload["invite_id"] != "invite_1" || item.Payload["group_id"] != "group_1" || item.Payload["message"] != "join us" {
		t.Fatalf("unexpected notify item: %#v", item)
	}
}

func TestAppendGroupApplicationAcceptedBuildsNotify(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	notifyID, err := service.AppendGroupApplicationAccepted(context.Background(), "user_alice", "group_1", "Hot Group", "app_1")
	if err != nil {
		t.Fatalf("append group application accepted: %v", err)
	}
	if notifyID == "" {
		t.Fatalf("expected generated notify_id, got empty")
	}
	item := repo.appended[0]
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "group_application_accepted" || item.Payload["application_id"] != "app_1" {
		t.Fatalf("unexpected notify item: %#v", item)
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
	actionURL := "https://example.com/maint"
	payload := systemmodule.SystemAnnouncementItem{AnnouncementID: "ann_1", Title: "维护通知", Content: "今晚维护", Severity: systemmodule.AnnounceLevelInfo, ActionURL: &actionURL, CreatedAt: "2026-03-26T00:00:00Z"}
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

func TestAppendSystemAnnouncementBuildsNotifysForUsers(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	payload := systemmodule.SystemAnnouncementItem{AnnouncementID: "ann_1", Title: "维护通知", Content: "今晚维护", Severity: systemmodule.AnnounceLevelInfo, CreatedAt: "2026-03-26T00:00:00Z"}
	notifyID, err := service.AppendSystemAnnouncement(context.Background(), []string{"user_alice", "user_bob"}, payload)
	if err != nil {
		t.Fatalf("append system announcement: %v", err)
	}
	if notifyID == "" {
		t.Fatal("expected generated notify_id, got empty")
	}
	if len(repo.appendMany) != 2 || len(repo.appended) != 1 {
		t.Fatalf("unexpected append-many capture: %#v", repo)
	}
	item := repo.appended[0]
	if item.NotifyID != notifyID {
		t.Fatalf("expected item.NotifyID=%q, got %q", notifyID, item.NotifyID)
	}
	if item.Type != "system_announcement" || item.Payload["announcement_id"] != "ann_1" || item.Payload["type"] != "system_announcement" {
		t.Fatalf("unexpected system notify item: %#v", item)
	}
}

func TestServiceListNormalizesPaginationAndCursor(t *testing.T) {
	repo := &notifyListCaptureRepo{list: PaginatedNotify{Items: []NotifyItem{{NotifyID: "notif_1"}}}}
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	cursor := encodeNotifyCursor(notifyCursor{CreatedAt: "2026-03-26T00:00:00Z", NotifyID: "notif_9"})

	list, err := service.List(ctx, NotifyListRequest{Type: "new_follower", Pagination: &PaginationParams{Limit: 999, Cursor: cursor}})
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
	if repo.req.Pagination == nil || repo.req.Pagination.Limit != maxNotifyListLimit {
		t.Fatalf("unexpected pagination limit: %#v", repo.req.Pagination)
	}
	if repo.req.Pagination.Cursor == "" {
		t.Fatal("expected a cursor to be passed to the repo")
	}
	// cursor 是 AES-GCM 加密 token（nonce 随机），断言字节相等无意义，
	// 改为解码后验证语义一致。
	decodedCursor, err := decodeNotifyCursor(repo.req.Pagination.Cursor)
	if err != nil || decodedCursor == nil || decodedCursor.CreatedAt != "2026-03-26T00:00:00Z" || decodedCursor.NotifyID != "notif_9" {
		t.Fatalf("unexpected cursor payload: %#v err=%v", decodedCursor, err)
	}
}

func TestServiceListAppliesDefaultLimit(t *testing.T) {
	repo := &notifyListCaptureRepo{}
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	if _, err := service.List(ctx, NotifyListRequest{}); err != nil {
		t.Fatalf("list: %v", err)
	}
	if repo.req.Pagination == nil || repo.req.Pagination.Limit != defaultNotifyListLimit {
		t.Fatalf("unexpected default pagination: %#v", repo.req.Pagination)
	}
	if repo.req.Pagination.Cursor != "" {
		t.Fatalf("unexpected cursor: %#v", repo.req.Pagination)
	}
}

func TestServiceListReturnsInvalidRequestError(t *testing.T) {
	service := NewServiceWithRepo(&notifyListCaptureRepo{})
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	if _, err := service.List(ctx, NotifyListRequest{Type: "bad_type"}); !errors.Is(err, ErrInvalidNotifyRequest) {
		t.Fatalf("expected invalid request error, got %v", err)
	}
	if _, err := service.List(ctx, NotifyListRequest{Pagination: &PaginationParams{Cursor: "%%%"}}); !errors.Is(err, ErrInvalidNotifyRequest) {
		t.Fatalf("expected invalid cursor error, got %v", err)
	}
}

func TestServiceListReturnsRepositoryError(t *testing.T) {
	repo := &notifyListCaptureRepo{listErr: errors.New("boom")}
	service := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	if _, err := service.List(ctx, NotifyListRequest{}); !errors.Is(err, repo.listErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
}

// pruneOnce 调用 repo.PruneReadBefore 并以 now-90d 为截止（保守口径）。
func TestServicePruneOnceDelegatesWithRetentionCutoff(t *testing.T) {
	var gotBefore time.Time
	called := false
	repo := &pruneCaptureRepo{prune: func(before time.Time) (int64, error) {
		called = true
		gotBefore = before
		return 3, nil
	}}
	svc := NewServiceWithRepo(repo)
	svc.pruneOnce(time.Now())
	if !called {
		t.Fatal("expected prune to be invoked")
	}
	age := time.Since(gotBefore)
	if age < 89*24*time.Hour || age > 91*24*time.Hour {
		t.Fatalf("expected cutoff ~90d ago, got %v", age)
	}
}

type pruneCaptureRepo struct {
	notifyListCaptureRepo
	prune func(before time.Time) (int64, error)
}

func (r *pruneCaptureRepo) PruneReadBefore(_ context.Context, before time.Time) (int64, error) {
	return r.prune(before)
}
