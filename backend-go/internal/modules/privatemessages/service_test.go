package private_messages

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
)

type stubMsgRepo struct {
	sendResult MessageSent
	sendErr    error
	histResult PaginatedMessages
	histErr    error
	recentRes  PaginatedMessages
	recentErr  error
	lastResult *PrivateMessageItem
	lastErr    error
	unread     int
	unreadErr  error
	updateMeta MessageEventMeta
	updateErr  error
	recallMeta MessageEventMeta
	recallErr  error
	readMeta   MessageEventMeta
	readErr    error
	markAllErr error
}

func (r *stubMsgRepo) Send(_ context.Context, _, _ string, _ SendPrivateMessageRequest) (MessageSent, error) {
	return r.sendResult, r.sendErr
}
func (r *stubMsgRepo) History(_ context.Context, _ string, _ GetHistoryRequest) (PaginatedMessages, error) {
	return r.histResult, r.histErr
}
func (r *stubMsgRepo) Recent(_ context.Context, _, _ string, _ int, _ *time.Time) (PaginatedMessages, error) {
	return r.recentRes, r.recentErr
}
func (r *stubMsgRepo) Last(_ context.Context, _, _ string, _ *time.Time) (*PrivateMessageItem, error) {
	return r.lastResult, r.lastErr
}
func (r *stubMsgRepo) UnreadCount(_ context.Context, _, _ string, _ *time.Time) (int, error) {
	return r.unread, r.unreadErr
}
func (r *stubMsgRepo) Update(_ context.Context, _ string, _ UpdatePrivateMessageRequest) (MessageEventMeta, error) {
	return r.updateMeta, r.updateErr
}
func (r *stubMsgRepo) Recall(_ context.Context, _ string, _ RecallPrivateMessageRequest) (MessageEventMeta, error) {
	return r.recallMeta, r.recallErr
}
func (r *stubMsgRepo) MarkAsRead(_ context.Context, _ string, _ MarkPrivateMessagesReadRequest) (MessageEventMeta, error) {
	return r.readMeta, r.readErr
}
func (r *stubMsgRepo) MarkAllRead(_ context.Context, _ string) error {
	return r.markAllErr
}
func (r *stubMsgRepo) GetMessageForReply(_ context.Context, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}
func (r *stubMsgRepo) GetMediaForMessage(_ context.Context, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}

type stubUserReader struct {
	item        users.UserItem
	found       bool
	userErr     error
	settings    users.UserSettings
	settingsErr error
}

func (u *stubUserReader) OpenTheDoor(_ context.Context, _ string) (users.UserItem, bool, error) {
	return u.item, u.found, u.userErr
}
func (u *stubUserReader) GetMySettings(_ context.Context) (users.UserSettings, error) {
	return u.settings, u.settingsErr
}

func msgCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func TestServiceSendDelegatesToRepo(t *testing.T) {
	repo := &stubMsgRepo{sendResult: MessageSent{MessageID: "pm1", SentAt: "2025-01-01T00:00:00Z"}}
	svc := NewServiceWithDeps(repo, &stubUserReader{found: true, settings: users.UserSettings{AllowPrivateChat: users.PrivateChatEveryone}})
	result, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageID != "pm1" {
		t.Fatalf("expected pm1, got %s", result.MessageID)
	}
}

func TestServiceSendRejectsTargetNotFound(t *testing.T) {
	svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{found: false})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u_missing", Type: "text"})
	if err == nil {
		t.Fatal("expected error for missing target")
	}
}

func TestServiceSendRejectsBlockedUser(t *testing.T) {
	svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{
		found:    true,
		item:     users.UserItem{Relation: &users.Relationship{IsBlocked: true}},
		settings: users.UserSettings{AllowPrivateChat: users.PrivateChatEveryone},
	})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text"})
	if err == nil {
		t.Fatal("expected error for blocked user")
	}
}

func TestServiceSendRejectsNobodySetting(t *testing.T) {
	svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{
		found:    true,
		settings: users.UserSettings{AllowPrivateChat: users.PrivateChatNobody},
	})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text"})
	if err == nil {
		t.Fatal("expected error for nobody setting")
	}
}

func TestServiceSendAllowsSelfMessage(t *testing.T) {
	repo := &stubMsgRepo{sendResult: MessageSent{MessageID: "pm1"}}
	svc := NewServiceWithDeps(repo, &stubUserReader{found: true, settings: users.UserSettings{AllowPrivateChat: users.PrivateChatNobody}})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u1", Type: "text"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSendPropagatesError(t *testing.T) {
	svc := NewServiceWithDeps(&stubMsgRepo{sendErr: errors.New("boom")}, &stubUserReader{found: true, settings: users.UserSettings{AllowPrivateChat: users.PrivateChatEveryone}})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceHistoryDelegates(t *testing.T) {
	repo := &stubMsgRepo{histResult: PaginatedMessages{Items: []PrivateMessageItem{{MessageID: "pm1"}}}}
	svc := NewServiceWithRepo(repo)
	result, err := svc.History(msgCtx("u1"), GetHistoryRequest{TargetUserID: "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestServiceRecentDelegates(t *testing.T) {
	repo := &stubMsgRepo{recentRes: PaginatedMessages{Items: []PrivateMessageItem{{MessageID: "pm1"}}}}
	svc := NewServiceWithRepo(repo)
	result, err := svc.Recent(msgCtx("u1"), "u2", 10, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestServiceLastDelegates(t *testing.T) {
	repo := &stubMsgRepo{lastResult: &PrivateMessageItem{MessageID: "pm1"}}
	svc := NewServiceWithRepo(repo)
	result, err := svc.Last(msgCtx("u1"), "u2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.MessageID != "pm1" {
		t.Fatal("expected pm1")
	}
}

func TestServiceUnreadCountDelegates(t *testing.T) {
	repo := &stubMsgRepo{unread: 5}
	svc := NewServiceWithRepo(repo)
	count, err := svc.UnreadCount(msgCtx("u1"), "u2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5, got %d", count)
	}
}

func TestServiceUpdateDelegates(t *testing.T) {
	repo := &stubMsgRepo{updateMeta: MessageEventMeta{TargetUserID: "u2", MessageID: "pm1"}}
	svc := NewServiceWithRepo(repo)
	err := svc.Update(msgCtx("u1"), UpdatePrivateMessageRequest{MessageID: "pm1", Content: "edited"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUpdatePropagatesError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMsgRepo{updateErr: errors.New("boom")})
	err := svc.Update(msgCtx("u1"), UpdatePrivateMessageRequest{MessageID: "pm1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceRecallDelegates(t *testing.T) {
	repo := &stubMsgRepo{recallMeta: MessageEventMeta{TargetUserID: "u2", MessageID: "pm1"}}
	svc := NewServiceWithRepo(repo)
	err := svc.Recall(msgCtx("u1"), RecallPrivateMessageRequest{MessageID: "pm1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceRecallPropagatesError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMsgRepo{recallErr: errors.New("boom")})
	err := svc.Recall(msgCtx("u1"), RecallPrivateMessageRequest{MessageID: "pm1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceMarkAsReadDelegates(t *testing.T) {
	repo := &stubMsgRepo{readMeta: MessageEventMeta{TargetUserID: "u2", MessageID: "pm1"}}
	svc := NewServiceWithRepo(repo)
	err := svc.MarkAsRead(msgCtx("u1"), MarkPrivateMessagesReadRequest{TargetUserID: "u2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceMarkAsReadPropagatesError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMsgRepo{readErr: errors.New("boom")})
	err := svc.MarkAsRead(msgCtx("u1"), MarkPrivateMessagesReadRequest{TargetUserID: "u2"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceMarkAllReadDelegates(t *testing.T) {
	svc := NewServiceWithRepo(&stubMsgRepo{})
	err := svc.MarkAllRead(msgCtx("u1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceMarkAllReadPropagatesError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMsgRepo{markAllErr: errors.New("boom")})
	err := svc.MarkAllRead(msgCtx("u1"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceRecentPropagatesError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMsgRepo{recentErr: errors.New("boom")})
	_, err := svc.Recent(msgCtx("u1"), "u2", 10, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceLastWithTimeBoundary(t *testing.T) {
	boundary := time.Now().UTC()
	repo := &stubMsgRepo{lastResult: &PrivateMessageItem{MessageID: "pm1"}}
	svc := NewServiceWithRepo(repo)
	result, err := svc.Last(msgCtx("u1"), "u2", &boundary)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.MessageID != "pm1" {
		t.Fatal("expected pm1")
	}
}

func TestServiceUnreadCountPropagatesError(t *testing.T) {
	svc := NewServiceWithRepo(&stubMsgRepo{unreadErr: errors.New("boom")})
	_, err := svc.UnreadCount(msgCtx("u1"), "u2", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
