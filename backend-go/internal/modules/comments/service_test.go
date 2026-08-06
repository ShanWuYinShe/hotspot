package comments

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/modules/notify"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
	"hotspot/backend-go/internal/platform/types"
)

type momentProviderStub struct {
	exists              bool
	commentableFound    bool
	commentable         bool
	commentableErr      error
	increment           int
	notificationContext MomentNotificationContext
	notificationOK      bool
	notificationErr     error
}

func (m *momentProviderStub) Exists(context.Context, string) (bool, error) { return m.exists, nil }
func (m *momentProviderStub) CheckCommentable(context.Context, string) (bool, bool, error) {
	return m.commentableFound, m.commentable, m.commentableErr
}
func (m *momentProviderStub) IncrementCommentCount(context.Context, string) error {
	m.increment++
	return nil
}
func (m *momentProviderStub) DecrementCommentCount(context.Context, string) error { return nil }
func (m *momentProviderStub) NotificationContext(context.Context, string) (MomentNotificationContext, bool, error) {
	return m.notificationContext, m.notificationOK, m.notificationErr
}

type sinkCapture struct{ events []realtime.Event }

func (s *sinkCapture) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *sinkCapture) Close() error                { return nil }

type failingNotificationWriter struct {
	calls int
}

func (w *failingNotificationWriter) AppendMomentCommented(context.Context, string, types.UserInfo, string, types.MomentSummary, string, string) (string, error) {
	w.calls++
	return "", errors.New("notification down")
}
func (w *failingNotificationWriter) AppendCommentReply(context.Context, string, types.UserInfo, string, types.MomentSummary, string, string, string, string) (string, error) {
	w.calls++
	return "", errors.New("notification down")
}
func (w *failingNotificationWriter) AppendMentioned(context.Context, string, types.UserInfo, string, string, types.MomentSummary, string, string) (string, error) {
	w.calls++
	return "", errors.New("notification down")
}

func testCommentContext(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID, SessionID: "sess_" + userID})
}

func TestCreateTopLevelCommentSendsRealtimeAndNotificationToMomentAuthor(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "moment summary"},
		},
	}
	notificationRepo := notify.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notify.NewServiceWithRepo(notificationRepo), nil, nil, nil)
	aliceSink := &sinkCapture{}
	bobSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)
	hub.RegisterUser("user_bob", "sess_bob", bobSink)

	comment, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_1", Content: "top level comment"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if len(aliceSink.events) != 1 {
		t.Fatalf("expected one event for moment author, got %#v", aliceSink.events)
	}
	if len(bobSink.events) != 0 {
		t.Fatalf("expected no event for commenter, got %#v", bobSink.events)
	}
	event := aliceSink.events[0]
	if event.Type != "MomentCommented" {
		t.Fatalf("unexpected event type: %#v", event)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", event.Payload)
	}
	if payload["moment_id"] != "moment_1" || payload["comment_id"] != comment.CommentID || payload["content"] != "top level comment" {
		t.Fatalf("unexpected event payload: %#v", payload)
	}
	notificationID, _ := payload["notify_id"].(string)
	if notificationID == "" {
		t.Fatalf("expected non-empty notification_id in event payload: %#v", payload)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notify.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "moment_commented" {
		t.Fatalf("unexpected notifications page: %#v", notificationsPage)
	}
	notifPayload := notificationsPage.Items[0].Payload
	if notifPayload["comment_id"] != comment.CommentID || notifPayload["comment_preview"] != "top level comment" {
		t.Fatalf("unexpected notification payload: %#v", notifPayload)
	}
	if notificationsPage.Items[0].NotifyID != notificationID {
		t.Fatalf("expected event notification_id to match stored item, got event=%q stored=%q", notificationID, notificationsPage.Items[0].NotifyID)
	}
}

func TestCreateTopLevelCommentOnOwnMomentSkipsRealtimeAndNotification(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "moment summary"},
		},
	}
	notificationRepo := notify.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notify.NewServiceWithRepo(notificationRepo), nil, nil, nil)
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if _, err := service.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "moment_1", Content: "self comment"}); err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if len(aliceSink.events) != 0 {
		t.Fatalf("expected no self event, got %#v", aliceSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notify.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 0 {
		t.Fatalf("expected no self notifications, got %#v", notificationsPage)
	}
}

func TestCreateReplySendsRealtimeAndNotificationToParentAuthor(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "moment summary"},
		},
	}
	notificationRepo := notify.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notify.NewServiceWithRepo(notificationRepo), nil, nil, nil)
	if _, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "parent"}); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	parent, _, err := repo.Get(context.Background(), firstCommentID(t, repo, "moment_1"))
	if err != nil {
		t.Fatalf("get parent: %v", err)
	}
	aliceSink := &sinkCapture{}
	bobSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)
	hub.RegisterUser("user_bob", "sess_bob", bobSink)

	reply, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_1", ParentID: parent.CommentID, Content: "reply"})
	if err != nil {
		t.Fatalf("create reply: %v", err)
	}
	if len(aliceSink.events) != 1 {
		t.Fatalf("expected one event for parent author, got %#v", aliceSink.events)
	}
	if len(bobSink.events) != 0 {
		t.Fatalf("expected no event for replier, got %#v", bobSink.events)
	}
	event := aliceSink.events[0]
	if event.Type != "CommentReply" {
		t.Fatalf("unexpected event type: %#v", event)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", event.Payload)
	}
	if payload["comment_id"] != parent.CommentID || payload["reply_id"] != reply.CommentID || payload["replied_comment_preview"] != "parent" || payload["content"] != "reply" {
		t.Fatalf("unexpected reply payload: %#v", payload)
	}
	notificationID, _ := payload["notify_id"].(string)
	if notificationID == "" {
		t.Fatalf("expected non-empty notification_id in event payload: %#v", payload)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notify.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "comment_reply" {
		t.Fatalf("unexpected notifications page: %#v", notificationsPage)
	}
	notifPayload := notificationsPage.Items[0].Payload
	if notifPayload["comment_id"] != parent.CommentID || notifPayload["reply_id"] != reply.CommentID || notifPayload["reply_content_preview"] != "reply" {
		t.Fatalf("unexpected notification payload: %#v", notifPayload)
	}
	if notificationsPage.Items[0].NotifyID != notificationID {
		t.Fatalf("expected event notification_id to match stored item, got event=%q stored=%q", notificationID, notificationsPage.Items[0].NotifyID)
	}
}

func TestCreateReplyToOwnCommentSkipsRealtimeAndNotification(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "moment summary"},
		},
	}
	notificationRepo := notify.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notify.NewServiceWithRepo(notificationRepo), nil, nil, nil)
	if _, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "parent"}); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	parent, _, err := repo.Get(context.Background(), firstCommentID(t, repo, "moment_1"))
	if err != nil {
		t.Fatalf("get parent: %v", err)
	}
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if _, err := service.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "moment_1", ParentID: parent.CommentID, Content: "self reply"}); err != nil {
		t.Fatalf("create self reply: %v", err)
	}
	if len(aliceSink.events) != 0 {
		t.Fatalf("expected no self event, got %#v", aliceSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notify.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 0 {
		t.Fatalf("expected no self notifications, got %#v", notificationsPage)
	}
}

func TestCreateCommentMentionSendsRealtimeAndNotification(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "moment summary"},
		},
	}
	notificationRepo := notify.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notify.NewServiceWithRepo(notificationRepo), nil, nil, nil)
	carolSink := &sinkCapture{}
	hub.RegisterUser("user_carol", "sess_carol", carolSink)

	comment, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_1", Content: "hello @carol", Mentions: []string{"user_carol", "user_carol"}})
	if err != nil {
		t.Fatalf("create comment with mention: %v", err)
	}
	if len(carolSink.events) != 1 {
		t.Fatalf("expected one mention event, got %#v", carolSink.events)
	}
	event := carolSink.events[0]
	if event.Type != "MessageMentioned" {
		t.Fatalf("unexpected event type: %#v", event)
	}
	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", event.Payload)
	}
	if payload["comment_id"] != comment.CommentID || payload["target_type"] != "comment" || payload["comment_preview"] != "hello @carol" {
		t.Fatalf("unexpected mention payload: %#v", payload)
	}
	notificationID, _ := payload["notify_id"].(string)
	if notificationID == "" {
		t.Fatalf("expected non-empty notification_id in event payload: %#v", payload)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_carol", notify.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "mentioned" {
		t.Fatalf("unexpected mention notifications page: %#v", notificationsPage)
	}
	if notificationsPage.Items[0].NotifyID != notificationID {
		t.Fatalf("expected event notification_id to match stored item, got event=%q stored=%q", notificationID, notificationsPage.Items[0].NotifyID)
	}
}

func TestCreateReplyMentionDeduplicatesReplyTarget(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "moment summary"},
		},
	}
	notificationRepo := notify.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notify.NewServiceWithRepo(notificationRepo), nil, nil, nil)
	if _, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "parent"}); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	parent, _, err := repo.Get(context.Background(), firstCommentID(t, repo, "moment_1"))
	if err != nil {
		t.Fatalf("get parent: %v", err)
	}
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if _, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_1", ParentID: parent.CommentID, Content: "reply", Mentions: []string{"user_alice"}}); err != nil {
		t.Fatalf("create reply: %v", err)
	}
	if len(aliceSink.events) != 1 {
		t.Fatalf("expected only one event, got %#v", aliceSink.events)
	}
	if aliceSink.events[0].Type != "CommentReply" {
		t.Fatalf("unexpected event type: %#v", aliceSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notify.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "comment_reply" {
		t.Fatalf("expected only comment_reply notification, got %#v", notificationsPage)
	}
}

func TestCreateTopLevelCommentMentionDeduplicatesMomentAuthor(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "moment summary"},
		},
	}
	notificationRepo := notify.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notify.NewServiceWithRepo(notificationRepo), nil, nil, nil)
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if _, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_1", Content: "hello @alice", Mentions: []string{"user_alice"}}); err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if len(aliceSink.events) != 1 {
		t.Fatalf("expected one event, got %#v", aliceSink.events)
	}
	if aliceSink.events[0].Type != "MomentCommented" {
		t.Fatalf("unexpected event type: %#v", aliceSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notify.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "moment_commented" {
		t.Fatalf("expected only moment_commented notification, got %#v", notificationsPage)
	}
}

func TestCreateCommentIgnoresNotificationFailures(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: types.MomentSummary{MomentID: "moment_1", DescriptionPreview: "moment summary"},
		},
	}
	writer := &failingNotificationWriter{}
	service := NewServiceWithDeps(repo, moments, writer, nil, nil, nil)

	comment, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_1", Content: "still succeeds", Mentions: []string{"user_carol"}})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if comment.CommentID == "" {
		t.Fatalf("expected created comment, got %#v", comment)
	}
	if writer.calls != 2 {
		t.Fatalf("expected two notification attempts, got %d", writer.calls)
	}
	page, err := repo.List(context.Background(), "moment_1", 20, "")
	if err != nil {
		t.Fatalf("list comments: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected persisted comment, got %#v", page)
	}
}

func TestCreateReplyToMissingParentReturnsCommentNotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notify.NewService(), nil, nil, nil)
	_, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_1", ParentID: "missing_parent", Content: "reply"})
	if !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
}

func firstCommentID(t *testing.T, repo *InMemoryRepository, momentID string) string {
	t.Helper()
	page, err := repo.List(context.Background(), momentID, 20, "")
	if err != nil || len(page.Items) == 0 {
		t.Fatalf("list comments: %#v err=%v", page, err)
	}
	return page.Items[0].CommentID
}

func TestCreateCommentReturnsNoPermissionWhenMomentExistsButNotCommentable(t *testing.T) {
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{commentableFound: true, commentable: false}
	service := NewServiceWithDeps(repo, moments, notify.NewService(), nil, nil, nil)
	_, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_private", Content: "hello"})
	if !errors.Is(err, ErrMomentNoPermission) {
		t.Fatalf("expected ErrMomentNoPermission, got %v", err)
	}
}

func TestCreateCommentReturnsNotFoundWhenMomentDoesNotExist(t *testing.T) {
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{commentableFound: false}
	service := NewServiceWithDeps(repo, moments, notify.NewService(), nil, nil, nil)
	_, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_missing", Content: "hello"})
	if !errors.Is(err, ErrMomentNotFound) {
		t.Fatalf("expected ErrMomentNotFound, got %v", err)
	}
}

// Delete service tests

func TestServiceDeleteRejectsNotOwner(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notify.NewService(), nil, nil, nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "alice comment"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	err = service.Delete(testCommentContext("user_bob"), DeleteCommentRequest{CommentID: comment.CommentID})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestServiceDeleteRejectsNotFound(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notify.NewService(), nil, nil, nil)
	err := service.Delete(testCommentContext("user_alice"), DeleteCommentRequest{CommentID: "nonexistent_comment"})
	if !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
}

func TestServiceDeleteHappyPath(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notify.NewService(), nil, nil, nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "to be deleted"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	err = service.Delete(testCommentContext("user_alice"), DeleteCommentRequest{CommentID: comment.CommentID})
	if err != nil {
		t.Fatalf("delete comment: %v", err)
	}
	updated, found, _ := repo.Get(context.Background(), comment.CommentID)
	if !found {
		t.Fatal("expected comment to still exist after soft delete")
	}
	if !updated.IsDeleted {
		t.Fatal("expected comment to be marked as deleted")
	}
	if updated.Content != "" {
		t.Fatal("expected comment content to be cleared after soft delete")
	}
}

func TestServiceDeleteIdempotentNoOp(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notify.NewService(), nil, nil, nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "to be deleted"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	// First delete
	err = service.Delete(testCommentContext("user_alice"), DeleteCommentRequest{CommentID: comment.CommentID})
	if err != nil {
		t.Fatalf("first delete: %v", err)
	}
	// Second delete — should be no-op (not return ErrCommentNotFound)
	err = service.Delete(testCommentContext("user_alice"), DeleteCommentRequest{CommentID: comment.CommentID})
	if err != nil {
		t.Fatalf("second delete should be no-op, got: %v", err)
	}
	// Still soft-deleted, not found
	updated, found, _ := repo.Get(context.Background(), comment.CommentID)
	if !found {
		t.Fatal("expected comment to still exist after second delete")
	}
	if !updated.IsDeleted {
		t.Fatal("expected comment to still be marked as deleted after second delete")
	}
}

func TestServiceDeleteAllowsAdmin(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notify.NewService(), nil, nil, nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "alice comment"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	// Admin deletes alice's comment — allowed（注入真实验证 stub）
	platformauth.SetAdminChecker(adminCheckStub{adminIDs: map[string]bool{"admin_1": true}})
	adminCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{
		UserID:    "admin_1",
		SessionID: "sess_admin",
	})
	err = service.Delete(adminCtx, DeleteCommentRequest{CommentID: comment.CommentID})
	if err != nil {
		t.Fatalf("admin delete: %v", err)
	}
	updated, found, _ := repo.Get(context.Background(), comment.CommentID)
	if !found {
		t.Fatal("expected comment to exist after soft delete by admin")
	}
	if !updated.IsDeleted {
		t.Fatal("expected comment to be marked as deleted after admin delete")
	}
}

// Like service tests

// (Like/Unlike/ListLikes tests removed — like endpoints removed in favour of
// the reactions module.)

// adminCheckStub 是 platformauth.AdminChecker 的测试替身。
type adminCheckStub struct{ adminIDs map[string]bool }

func (s adminCheckStub) IsAdmin(_ context.Context, userID string) (bool, error) {
	return s.adminIDs[userID], nil
}
