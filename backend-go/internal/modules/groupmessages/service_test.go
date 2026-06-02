package groupmessages

import (
	"context"
	"errors"
	"testing"

	"hotspot/backend-go/internal/modules/users"
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

	replyToResult map[string]any
	replyToOk     bool
	replyToErr    error

	mediaResult map[string]any
	mediaOk     bool
	mediaErr    error
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
	return r.replyToResult, r.replyToOk, r.replyToErr
}

func (r *stubRepo) GetMediaForMessage(_ context.Context, _ string) (map[string]any, bool, error) {
	return r.mediaResult, r.mediaOk, r.mediaErr
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

// ---------------------------------------------------------------------------
// buildReplyTo / buildMedia 单元测试
// ---------------------------------------------------------------------------

func TestBuildReplyToReturnsInfo(t *testing.T) {
	expected := map[string]any{"message_id": "m1", "content": "hi"}
	repo := &stubRepo{replyToResult: expected, replyToOk: true}
	service := NewServiceWithRepo(repo)

	result := service.buildReplyTo(context.Background(), "m1")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result["message_id"] != "m1" {
		t.Fatalf("expected message_id=m1, got %v", result["message_id"])
	}
}

func TestBuildReplyToReturnsNilWhenNotFound(t *testing.T) {
	repo := &stubRepo{replyToResult: nil, replyToOk: false}
	service := NewServiceWithRepo(repo)

	result := service.buildReplyTo(context.Background(), "missing")
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestBuildReplyToReturnsNilOnError(t *testing.T) {
	repo := &stubRepo{replyToResult: nil, replyToOk: true, replyToErr: errors.New("db error")}
	service := NewServiceWithRepo(repo)

	result := service.buildReplyTo(context.Background(), "m1")
	if result != nil {
		t.Fatalf("expected nil on error, got %v", result)
	}
}

func TestBuildMediaReturnsInfo(t *testing.T) {
	expected := map[string]any{"file_id": "f1", "url": "https://example.com/img.jpg"}
	repo := &stubRepo{mediaResult: expected, mediaOk: true}
	service := NewServiceWithRepo(repo)

	result := service.buildMedia(context.Background(), "f1")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result["file_id"] != "f1" {
		t.Fatalf("expected file_id=f1, got %v", result["file_id"])
	}
}

func TestBuildMediaReturnsNilWhenNotFound(t *testing.T) {
	repo := &stubRepo{mediaResult: nil, mediaOk: false}
	service := NewServiceWithRepo(repo)

	result := service.buildMedia(context.Background(), "missing")
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestBuildMediaReturnsNilOnError(t *testing.T) {
	repo := &stubRepo{mediaResult: nil, mediaOk: true, mediaErr: errors.New("db error")}
	service := NewServiceWithRepo(repo)

	result := service.buildMedia(context.Background(), "f1")
	if result != nil {
		t.Fatalf("expected nil on error, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// Send 函数的 reply_to / media 分支
// ---------------------------------------------------------------------------

func TestServiceSendWithReplyToIncludesReplyInfo(t *testing.T) {
	replyInfo := map[string]any{"message_id": "orig1", "content": "original"}
	repo := &stubRepo{
		sendResult:    MessageSent{MessageID: "gm-new", SentAt: "2025-06-01T00:00:00Z"},
		replyToResult: replyInfo,
		replyToOk:     true,
	}
	service := NewServiceWithRepo(repo)

	_, err := service.Send(gmAuthCtx("u1"), SendGroupMessageRequest{
		GroupID:          "g1",
		SenderUserID:     "u1",
		Type:             "text",
		Content:          "reply msg",
		ReplyToMessageID: "orig1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSendWithMediaIncludesMediaInfo(t *testing.T) {
	mediaInfo := map[string]any{"file_id": "f1", "url": "https://example.com/img.jpg"}
	repo := &stubRepo{
		sendResult:  MessageSent{MessageID: "gm-media", SentAt: "2025-06-01T00:00:00Z"},
		mediaResult: mediaInfo,
		mediaOk:     true,
	}
	service := NewServiceWithRepo(repo)

	_, err := service.Send(gmAuthCtx("u1"), SendGroupMessageRequest{
		GroupID:      "g1",
		SenderUserID: "u1",
		Type:         "image",
		Content:      "check this out",
		FileID:       "f1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 构造函数测试
// ---------------------------------------------------------------------------

func TestServiceNewServiceConstructors(t *testing.T) {
	repo := NewInMemoryRepository()
	svc1 := NewService()
	if svc1 == nil {
		t.Fatal("NewService returned nil")
	}

	svc2 := NewServiceWithRepo(repo)
	if svc2 == nil {
		t.Fatal("NewServiceWithRepo returned nil")
	}
	if svc2.repo == nil {
		t.Fatal("expected repo to be set")
	}

	svc3 := NewServiceWithRepoAndUsers(repo, &stubUserReader{})
	if svc3 == nil {
		t.Fatal("NewServiceWithRepoAndUsers returned nil")
	}
	if svc3.repo == nil {
		t.Fatal("expected repo to be set")
	}
	if svc3.users == nil {
		t.Fatal("expected users to be set")
	}
}

type stubUserReader struct{}

func (stubUserReader) OpenTheDoor(_ context.Context, _ string) (users.UserItem, bool, error) {
	return users.UserItem{}, false, nil
}
