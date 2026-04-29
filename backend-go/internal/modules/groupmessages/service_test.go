package groupmessages

import (
	"context"
	"errors"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

type stubRepo struct {
	sendResult     MessageSent
	sendErr        error
	historyResult  PaginatedGroupMessages
	historyErr     error
	updateMeta     MessageEventMeta
	updateErr      error
	recallMeta     MessageEventMeta
	recallErr      error
	markReadErr    error
	markAllReadErr error
}

func (r *stubRepo) Send(_ context.Context, _ SendGroupMessageRequest) (MessageSent, error) {
	return r.sendResult, r.sendErr
}
func (r *stubRepo) History(_ context.Context, _ string, _ *PaginationParams) (PaginatedGroupMessages, error) {
	return r.historyResult, r.historyErr
}
func (r *stubRepo) Update(_ context.Context, _ UpdateGroupMessageRequest) (MessageEventMeta, error) {
	return r.updateMeta, r.updateErr
}
func (r *stubRepo) Recall(_ context.Context, _ RecallGroupMessageRequest) (MessageEventMeta, error) {
	return r.recallMeta, r.recallErr
}
func (r *stubRepo) MarkAsRead(_ context.Context, _ string, _ MarkGroupMessagesReadRequest) error {
	return r.markReadErr
}
func (r *stubRepo) MarkAllRead(_ context.Context, _ string) error {
	return r.markAllReadErr
}
func (r *stubRepo) GetMessageForReply(_ context.Context, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}

func (r *stubRepo) GetMediaForMessage(_ context.Context, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}

func (r *stubRepo) IsMember(_ context.Context, _ string, _ string) (bool, error) {
	return true, nil
}

func gmAuthCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func TestServiceSendDelegatesToRepo(t *testing.T) {
	repo := &stubRepo{sendResult: MessageSent{MessageID: "gm1", SentAt: "2025-01-01T00:00:00Z"}}
	service := NewServiceWithRepo(repo)
	result, err := service.Send(gmAuthCtx("u1"), SendGroupMessageRequest{GroupID: "g1", Type: "text", Content: "hello", SenderUserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageID != "gm1" {
		t.Fatalf("expected gm1, got %s", result.MessageID)
	}
}

func TestServiceSendPropagatesError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{sendErr: errors.New("boom")})
	_, err := service.Send(context.Background(), SendGroupMessageRequest{GroupID: "g1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceHistoryDelegatesToRepo(t *testing.T) {
	repo := &stubRepo{historyResult: PaginatedGroupMessages{Items: []GroupMessageItem{{MessageID: "gm1"}}}}
	service := NewServiceWithRepo(repo)
	result, err := service.History(gmAuthCtx("u1"), "g1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Items))
	}
}

func TestServiceHistoryPropagatesError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{historyErr: errors.New("boom")})
	_, err := service.History(context.Background(), "g1", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceUpdateDelegatesToRepo(t *testing.T) {
	repo := &stubRepo{updateMeta: MessageEventMeta{GroupID: "g1", MessageID: "gm1"}}
	service := NewServiceWithRepo(repo)
	err := service.Update(context.Background(), UpdateGroupMessageRequest{MessageID: "gm1", Content: "edited"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUpdatePropagatesError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{updateErr: errors.New("boom")})
	err := service.Update(context.Background(), UpdateGroupMessageRequest{MessageID: "gm1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceRecallDelegatesToRepo(t *testing.T) {
	repo := &stubRepo{recallMeta: MessageEventMeta{GroupID: "g1", MessageID: "gm1"}}
	service := NewServiceWithRepo(repo)
	err := service.Recall(context.Background(), RecallGroupMessageRequest{MessageID: "gm1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceRecallPropagatesError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{recallErr: errors.New("boom")})
	err := service.Recall(context.Background(), RecallGroupMessageRequest{MessageID: "gm1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceMarkAsReadExtractsPrincipal(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{})
	err := service.MarkAsRead(gmAuthCtx("u1"), MarkGroupMessagesReadRequest{GroupID: "g1", UpToMessageID: "gm5"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceMarkAsReadPropagatesError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{markReadErr: errors.New("boom")})
	err := service.MarkAsRead(gmAuthCtx("u1"), MarkGroupMessagesReadRequest{GroupID: "g1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceMarkAllReadExtractsPrincipal(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{})
	err := service.MarkAllRead(gmAuthCtx("u1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceMarkAllReadPropagatesError(t *testing.T) {
	service := NewServiceWithRepo(&stubRepo{markAllReadErr: errors.New("boom")})
	err := service.MarkAllRead(gmAuthCtx("u1"))
	if err == nil {
		t.Fatal("expected error")
	}
}
