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
	exists    bool
	increment int
}

func (m *momentProviderStub) Exists(context.Context, string) (bool, error) { return m.exists, nil }
func (m *momentProviderStub) IncrementCommentCount(context.Context, string) error {
	m.increment++
	return nil
}
func (m *momentProviderStub) DecrementCommentCount(context.Context, string) error { return nil }

type sinkCapture struct{ events []realtime.Event }

func (s *sinkCapture) Send(e realtime.Event) error { s.events = append(s.events, e); return nil }
func (s *sinkCapture) Close() error                { return nil }

func testCommentContext(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID, SessionID: "sess_" + userID})
}

func TestCreateReplySendsRealtimeAndNotificationToParentAuthor(t *testing.T) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo))
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
	moments := &momentProviderStub{exists: true}
	notificationRepo := notifications.NewInMemoryRepository()
	service := NewServiceWithDeps(repo, moments, notifications.NewServiceWithRepo(notificationRepo))
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

func TestCreateReplyToMissingParentReturnsCommentNotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	moments := &momentProviderStub{exists: true}
	service := NewServiceWithDeps(repo, moments, notifications.NewService())
	_, err := service.Create(testCommentContext("user_bob"), CreateCommentRequest{MomentID: "moment_1", ParentID: "missing_parent", Content: "reply"})
	if !errors.Is(err, ErrCommentNotFound) {
		t.Fatalf("expected ErrCommentNotFound, got %v", err)
	}
}

func firstCommentID(t *testing.T, repo *InMemoryRepository, momentID string) string {
	t.Helper()
	page, err := repo.List(context.Background(), momentID)
	if err != nil || len(page.Items) == 0 {
		t.Fatalf("list comments: %#v err=%v", page, err)
	}
	return page.Items[0].CommentID
}
