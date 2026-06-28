package comments

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/modules/notifications"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/realtime"
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

func (w *failingNotificationWriter) AppendMomentCommented(context.Context, string, map[string]any, string, map[string]any, string, string) error {
	w.calls++
	return errors.New("notification down")
}
func (w *failingNotificationWriter) AppendCommentLiked(context.Context, string, map[string]any, string, string, string, map[string]any) error {
	w.calls++
	return errors.New("notification down")
}
func (w *failingNotificationWriter) AppendCommentReply(context.Context, string, map[string]any, string, map[string]any, string, string, string, string) error {
	w.calls++
	return errors.New("notification down")
}
func (w *failingNotificationWriter) AppendMentioned(context.Context, string, map[string]any, string, string, map[string]any, string, string) error {
	w.calls++
	return errors.New("notification down")
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
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo), nil)
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
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
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
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo), nil)
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if _, err := service.Create(testCommentContext("user_alice"), CreateCommentRequest{MomentID: "moment_1", Content: "self comment"}); err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if len(aliceSink.events) != 0 {
		t.Fatalf("expected no self event, got %#v", aliceSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
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
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo), nil)
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
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
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
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo), nil)
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
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
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
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo), nil)
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
	notificationsPage, err := notificationRepo.List(context.Background(), "user_carol", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "mentioned" {
		t.Fatalf("unexpected mention notifications page: %#v", notificationsPage)
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
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo), nil)
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
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
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
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo), nil)
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
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
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
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	writer := &failingNotificationWriter{}
	service := NewServiceWithDeps(repo, moments, writer, nil)

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

func TestLikeCommentSendsRealtimeAndNotificationToAuthor(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo), nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "comment body"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if err := service.Like(testCommentContext("user_bob"), LikeCommentRequest{CommentID: comment.CommentID}); err != nil {
		t.Fatalf("like comment: %v", err)
	}
	if len(aliceSink.events) != 1 {
		t.Fatalf("expected one event, got %#v", aliceSink.events)
	}
	if aliceSink.events[0].Type != "CommentLiked" {
		t.Fatalf("unexpected event type: %#v", aliceSink.events)
	}
	payload, ok := aliceSink.events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %#v", aliceSink.events[0].Payload)
	}
	if payload["comment_id"] != comment.CommentID || payload["comment_preview"] != "comment body" || payload["moment_id"] != "moment_1" {
		t.Fatalf("unexpected payload: %#v", payload)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "comment_liked" {
		t.Fatalf("unexpected notifications page: %#v", notificationsPage)
	}
}

func TestLikeCommentSkipsSelfNotification(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo), nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "comment body"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if err := service.Like(testCommentContext("user_alice"), LikeCommentRequest{CommentID: comment.CommentID}); err != nil {
		t.Fatalf("like comment: %v", err)
	}
	if len(aliceSink.events) != 0 {
		t.Fatalf("expected no self event, got %#v", aliceSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 0 {
		t.Fatalf("expected no self notifications, got %#v", notificationsPage)
	}
}

func TestLikeCommentDeduplicatesRepeatedLike(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{
		exists: true, commentableFound: true, commentable: true,
		notificationOK: true,
		notificationContext: MomentNotificationContext{
			UserID:  "user_alice",
			Summary: map[string]any{"moment_id": "moment_1", "description_preview": "moment summary"},
		},
	}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo), nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "comment body"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	aliceSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_alice", aliceSink)

	if err := service.Like(testCommentContext("user_bob"), LikeCommentRequest{CommentID: comment.CommentID}); err != nil {
		t.Fatalf("first like: %v", err)
	}
	if err := service.Like(testCommentContext("user_bob"), LikeCommentRequest{CommentID: comment.CommentID}); err != nil {
		t.Fatalf("second like: %v", err)
	}
	if len(aliceSink.events) != 1 {
		t.Fatalf("expected one event, got %#v", aliceSink.events)
	}
	notificationsPage, err := notificationRepo.List(context.Background(), "user_alice", notifications.NotificationListRequest{})
	if err != nil {
		t.Fatalf("list notifications: %v", err)
	}
	if len(notificationsPage.Items) != 1 || notificationsPage.Items[0].Type != "comment_liked" {
		t.Fatalf("expected one comment_liked notification, got %#v", notificationsPage)
	}
}

func TestCreateReplyToMissingParentReturnsCommentNotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
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
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
	_, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_private", Content: "hello"})
	if !errors.Is(err, ErrMomentNoPermission) {
		t.Fatalf("expected ErrMomentNoPermission, got %v", err)
	}
}

func TestCreateCommentReturnsNotFoundWhenMomentDoesNotExist(t *testing.T) {
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{commentableFound: false}
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
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
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
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
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
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
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
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
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
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
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "alice comment"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	// Admin deletes alice's comment — allowed
	adminCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{
		UserID:          "admin_1",
		SessionID:       "sess_admin",
		AdminMembership: &platformauth.AdminMembership{},
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

func TestServiceLikeRejectsMomentNotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: false}
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
	// Like uses repo.Get to check comment existence; if comment doesn't exist, ErrCommentNotFound is returned.
	err := service.Like(testCommentContext("user_bob"), LikeCommentRequest{CommentID: "missing_comment"})
	if !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
}

func TestServiceLikeHappyPath(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "likeable"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	err = service.Like(testCommentContext("user_bob"), LikeCommentRequest{CommentID: comment.CommentID})
	if err != nil {
		t.Fatalf("like comment: %v", err)
	}
	updated, found, _ := repo.Get(context.Background(), comment.CommentID)
	if !found {
		t.Fatal("expected comment to exist")
	}
	if updated.LikesCount != 1 {
		t.Fatalf("expected likes_count=1, got %d", updated.LikesCount)
	}
}

// Unlike service tests

func TestServiceUnlikeHappyPath(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "unlikeable"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	// Like first
	if err := service.Like(testCommentContext("user_bob"), LikeCommentRequest{CommentID: comment.CommentID}); err != nil {
		t.Fatalf("like comment: %v", err)
	}
	// Then unlike
	err = service.Unlike(testCommentContext("user_bob"), UnlikeCommentRequest{CommentID: comment.CommentID})
	if err != nil {
		t.Fatalf("unlike comment: %v", err)
	}
	updated, found, _ := repo.Get(context.Background(), comment.CommentID)
	if !found {
		t.Fatal("expected comment to exist")
	}
	if updated.LikesCount != 0 {
		t.Fatalf("expected likes_count=0 after unlike, got %d", updated.LikesCount)
	}
}

// ListLikes service tests

func TestServiceListLikesDelegates(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "liked"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	// Like from two users
	if _, err := repo.Like(context.Background(), comment.CommentID, "user_bob"); err != nil {
		t.Fatalf("bob like: %v", err)
	}
	if _, err := repo.Like(context.Background(), comment.CommentID, "user_carol"); err != nil {
		t.Fatalf("carol like: %v", err)
	}
	result, err := service.ListLikes(context.Background(), CommentLikesRequest{CommentID: comment.CommentID})
	if err != nil {
		t.Fatalf("list likes: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 likes, got %d", len(result.Items))
	}
}

func TestServiceListLikesNormalizesLimit(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true, commentableFound: true, commentable: true}
	service := NewServiceWithDeps(repo, moments, notifications.NewService(), nil)
	comment, err := repo.Create(testCommentContext("user_alice"), "user_alice", CreateCommentRequest{MomentID: "moment_1", Content: "liked"})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	// Like from 3 users
	for _, uid := range []string{"user_bob", "user_carol", "user_dave"} {
		if _, err := repo.Like(context.Background(), comment.CommentID, uid); err != nil {
			t.Fatalf("like from %s: %v", uid, err)
		}
	}
	// Request with limit=2
	result, err := service.ListLikes(context.Background(), CommentLikesRequest{
		CommentID:  comment.CommentID,
		Pagination: &PaginationParams{Limit: 2},
	})
	if err != nil {
		t.Fatalf("list likes: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items with limit=2, got %d", len(result.Items))
	}
	if result.NextCursor == "" {
		t.Fatal("expected next_cursor when results are truncated")
	}
}
