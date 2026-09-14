package groupmessages

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/types"
)

type stubRepo struct {
	sendResult     MessageSentResult
	sendErr        error
	historyResult  PaginatedGroupMessages
	historyErr     error
	updateMeta     MessageEventMeta
	updateErr      error
	recallMeta     MessageEventMeta
	recallErr      error
	recallWindow   time.Duration // 记录 service 传入的撤回窗口（窗口配置透传断言用）
	markReadErr    error
	markAllReadErr error

	// ownerMeta / ownerFound drive LookupMessageMeta, used by the service to
	// authorize Update/Recall (membership + ownership). Defaults to a message
	// owned by "u1" in "g1" so existing positive tests keep working.
	ownerMeta  MessageOwnerMeta
	ownerFound bool

	replyToResult *MessageBrief
	replyToOk     bool
	replyToErr    error

	mediaResult map[string]any
	mediaOk     bool
	mediaErr    error

	messageGroupIDErr error
}

func (r *stubRepo) Send(_ context.Context, _ SendGroupMessageRequest) (MessageSentResult, error) {
	return r.sendResult, r.sendErr
}
func (r *stubRepo) History(_ context.Context, _ string, _ *PaginationParams) (PaginatedGroupMessages, error) {
	return r.historyResult, r.historyErr
}
func (r *stubRepo) Update(_ context.Context, _ UpdateGroupMessageRequest) (MessageEventMeta, error) {
	return r.updateMeta, r.updateErr
}
func (r *stubRepo) Recall(_ context.Context, window time.Duration, _ RecallGroupMessageRequest) (MessageEventMeta, error) {
	r.recallWindow = window
	return r.recallMeta, r.recallErr
}
func (r *stubRepo) RecallAsAdmin(_ context.Context, _ string) (MessageEventMeta, error) {
	return r.recallMeta, r.recallErr
}
func (r *stubRepo) MarkAsRead(_ context.Context, _ string, _ MarkGroupMessagesReadRequest) error {
	return r.markReadErr
}
func (r *stubRepo) MarkAllRead(_ context.Context, _ string) ([]string, error) {
	return nil, r.markAllReadErr
}
func (r *stubRepo) GetReadCount(_ context.Context, _ string, _ string) (int, error) {
	return 0, nil
}
func (r *stubRepo) LatestMessageID(_ context.Context, _ string) (string, error) {
	return "", nil
}
func (r *stubRepo) GetMessageForReply(_ context.Context, _ string) (*MessageBrief, bool, error) {
	return r.replyToResult, r.replyToOk, r.replyToErr
}
func (r *stubRepo) MessageGroupID(_ context.Context, _ string) (string, bool, error) {
	return "", false, r.messageGroupIDErr
}

func (r *stubRepo) MessageExists(_ context.Context, _ string) (bool, string, error) {
	return false, "", nil
}

func (r *stubRepo) GetGroupName(_ context.Context, _ string) (string, bool, error) {
	return "test group", true, nil
}

func (r *stubRepo) LookupMessageMeta(_ context.Context, _ string) (MessageOwnerMeta, bool, error) {
	if !r.ownerFound {
		return MessageOwnerMeta{}, false, nil
	}
	return r.ownerMeta, true, nil
}

func (r *stubRepo) GetMessageReportSnapshot(_ context.Context, _ string) (MessageReportSnapshot, bool, error) {
	return MessageReportSnapshot{}, false, nil
}

func (r *stubRepo) GetMediaForMessage(_ context.Context, _ string, _ string) (map[string]any, bool, error) {
	return r.mediaResult, r.mediaOk, r.mediaErr
}

func (r *stubRepo) FilterMemberIDs(_ context.Context, _ string, userIDs []string) ([]string, error) {
	return userIDs, nil
}

func (r *stubRepo) IsMember(_ context.Context, _ string, _ string) (bool, error) {
	return true, nil
}

func (r *stubRepo) MemberMuteStatus(_ context.Context, _ string, _ string) (bool, string, bool, error) {
	return false, "", true, nil
}

func gmAuthCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func TestServiceSendDelegatesToRepo(t *testing.T) {
	repo := &stubRepo{sendResult: MessageSentResult{MessageID: "gm1", SentAt: "2025-01-01T00:00:00Z"}}
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
	repo := &stubRepo{
		ownerMeta:  MessageOwnerMeta{GroupID: "g1", SenderUserID: "u1"},
		ownerFound: true,
		updateMeta: MessageEventMeta{GroupID: "g1", MessageID: "gm1"},
	}
	service := NewServiceWithRepo(repo)
	err := service.Update(gmAuthCtx("u1"), UpdateGroupMessageRequest{MessageID: "gm1", UserID: "u1", Content: "edited"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUpdatePropagatesError(t *testing.T) {
	repo := &stubRepo{
		ownerMeta:  MessageOwnerMeta{GroupID: "g1", SenderUserID: "u1"},
		ownerFound: true,
		updateErr:  errors.New("boom"),
	}
	service := NewServiceWithRepo(repo)
	err := service.Update(gmAuthCtx("u1"), UpdateGroupMessageRequest{MessageID: "gm1", UserID: "u1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceRecallDelegatesToRepo(t *testing.T) {
	repo := &stubRepo{
		ownerMeta:  MessageOwnerMeta{GroupID: "g1", SenderUserID: "u1"},
		ownerFound: true,
		recallMeta: MessageEventMeta{GroupID: "g1", MessageID: "gm1"},
	}
	service := NewServiceWithRepo(repo)
	err := service.Recall(gmAuthCtx("u1"), RecallGroupMessageRequest{MessageID: "gm1", UserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// recallWindowSource 静态 appconfig.Source（撤回窗口配置透传断言用）。
type recallWindowSource int

func (w recallWindowSource) Get(context.Context) appconfig.AppConfig {
	return appconfig.AppConfig{Limits: appconfig.Limits{MessageRecallWindowSeconds: int(w)}}
}

// spec chat.md：撤回时间窗消费 AppConfig.limits.message_recall_window_seconds，
// service 取值后透传 repo。
func TestServiceRecallWindowFromAppConfig(t *testing.T) {
	repo := &stubRepo{ownerMeta: MessageOwnerMeta{GroupID: "g1", SenderUserID: "u1"}, ownerFound: true}
	service := NewServiceWithRepo(repo)
	service.SetAppConfig(recallWindowSource(90))
	err := service.Recall(gmAuthCtx("u1"), RecallGroupMessageRequest{MessageID: "gm1", UserID: "u1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.recallWindow != 90*time.Second {
		t.Fatalf("expected configured window 90s passed to repo, got %v", repo.recallWindow)
	}
}

// appConfig 未接线时回退默认 120s。
func TestServiceRecallWindowFallsBackToDefault(t *testing.T) {
	repo := &stubRepo{ownerMeta: MessageOwnerMeta{GroupID: "g1", SenderUserID: "u1"}, ownerFound: true}
	service := NewServiceWithRepo(repo)
	if err := service.Recall(gmAuthCtx("u1"), RecallGroupMessageRequest{MessageID: "gm1", UserID: "u1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.recallWindow != 120*time.Second {
		t.Fatalf("expected default window 120s, got %v", repo.recallWindow)
	}
}

func TestServiceRecallPropagatesError(t *testing.T) {
	repo := &stubRepo{
		ownerMeta:  MessageOwnerMeta{GroupID: "g1", SenderUserID: "u1"},
		ownerFound: true,
		recallErr:  errors.New("boom"),
	}
	service := NewServiceWithRepo(repo)
	err := service.Recall(gmAuthCtx("u1"), RecallGroupMessageRequest{MessageID: "gm1", UserID: "u1"})
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
	expected := &MessageBrief{MessageID: "m1", Content: types.StrPtr("hi")}
	repo := &stubRepo{replyToResult: expected, replyToOk: true}
	service := NewServiceWithRepo(repo)

	result := service.buildReplyTo(context.Background(), "m1")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.MessageID != "m1" {
		t.Fatalf("expected message_id=m1, got %v", result.MessageID)
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

	result := service.buildMedia(context.Background(), "f1", "u1")
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

	result := service.buildMedia(context.Background(), "missing", "u1")
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestBuildMediaReturnsNilOnError(t *testing.T) {
	repo := &stubRepo{mediaResult: nil, mediaOk: true, mediaErr: errors.New("db error")}
	service := NewServiceWithRepo(repo)

	result := service.buildMedia(context.Background(), "f1", "u1")
	if result != nil {
		t.Fatalf("expected nil on error, got %v", result)
	}
}

// ---------------------------------------------------------------------------
// Send 函数的 reply_to / media 分支
// ---------------------------------------------------------------------------

func TestServiceSendWithReplyToIncludesReplyInfo(t *testing.T) {
	replyInfo := &MessageBrief{MessageID: "orig1", Content: types.StrPtr("original")}
	repo := &stubRepo{
		sendResult:    MessageSentResult{MessageID: "gm-new", SentAt: "2025-06-01T00:00:00Z"},
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
		sendResult:  MessageSentResult{MessageID: "gm-media", SentAt: "2025-06-01T00:00:00Z"},
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

// ---------------------------------------------------------------------------
// Update / Recall 越权防护（成员校验 + 所有者校验 + NULL sender 拒绝）
// ---------------------------------------------------------------------------

// membershipStub 是一个 IsMember 可控的 repo，其余方法走零值默认。
// LookupMessageMeta 默认返回 group=g1 owner=u1，便于测试所有者分支。
type membershipStub struct {
	stubRepo
	members    map[string]bool // "groupID|userID" -> true
	ownerKey   string          // 当前 LookupMessageMeta 返回的 owner（用于 NULL sender 测试）
	ownerEmpty bool            // true => sender 为空（NULL）
}

func (m *membershipStub) IsMember(_ context.Context, groupID, userID string) (bool, error) {
	return m.members[groupID+"|"+userID], nil
}

func (m *membershipStub) LookupMessageMeta(_ context.Context, _ string) (MessageOwnerMeta, bool, error) {
	sender := m.ownerKey
	if m.ownerEmpty {
		sender = ""
	}
	return MessageOwnerMeta{GroupID: "g1", SenderUserID: sender}, true, nil
}

func TestServiceUpdate_RejectsNonMember(t *testing.T) {
	repo := &membershipStub{members: map[string]bool{"g1|u1": true}, ownerKey: "u1"} // u1 成员，u2 不是
	svc := NewServiceWithRepo(repo)
	err := svc.Update(gmAuthCtx("u2"), UpdateGroupMessageRequest{MessageID: "gm1", UserID: "u2", Content: "x"})
	if !errors.Is(err, ErrNotGroupMember) {
		t.Fatalf("expected ErrNotGroupMember, got %v", err)
	}
}

func TestServiceUpdate_RejectsNonOwner(t *testing.T) {
	repo := &membershipStub{members: map[string]bool{"g1|u2": true}, ownerKey: "u1"} // u2 是成员但非作者
	svc := NewServiceWithRepo(repo)
	err := svc.Update(gmAuthCtx("u2"), UpdateGroupMessageRequest{MessageID: "gm1", UserID: "u2", Content: "x"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestServiceUpdate_RejectsNullSender(t *testing.T) {
	repo := &membershipStub{members: map[string]bool{"g1|u1": true}, ownerEmpty: true} // NULL sender
	svc := NewServiceWithRepo(repo)
	err := svc.Update(gmAuthCtx("u1"), UpdateGroupMessageRequest{MessageID: "gm1", UserID: "u1", Content: "x"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied for NULL sender, got %v", err)
	}
}

func TestServiceUpdate_AllowsOwner(t *testing.T) {
	repo := &membershipStub{
		members:  map[string]bool{"g1|u1": true},
		ownerKey: "u1",
		stubRepo: stubRepo{updateMeta: MessageEventMeta{GroupID: "g1", MessageID: "gm1"}},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.Update(gmAuthCtx("u1"), UpdateGroupMessageRequest{MessageID: "gm1", UserID: "u1", Content: "ok"})
	if err != nil {
		t.Fatalf("expected success for owner, got %v", err)
	}
}

func TestServiceUpdate_NotFound(t *testing.T) {
	repo := &stubRepo{ownerFound: false} // 消息不存在
	svc := NewServiceWithRepo(repo)
	err := svc.Update(gmAuthCtx("u1"), UpdateGroupMessageRequest{MessageID: "missing", UserID: "u1"})
	if !errors.Is(err, ErrMessageNotFound) {
		t.Fatalf("expected ErrMessageNotFound, got %v", err)
	}
}

func TestServiceRecall_RejectsNonMember(t *testing.T) {
	repo := &membershipStub{members: map[string]bool{"g1|u1": true}, ownerKey: "u1"}
	svc := NewServiceWithRepo(repo)
	err := svc.Recall(gmAuthCtx("u2"), RecallGroupMessageRequest{MessageID: "gm1", UserID: "u2"})
	if !errors.Is(err, ErrNotGroupMember) {
		t.Fatalf("expected ErrNotGroupMember, got %v", err)
	}
}

func TestServiceRecall_RejectsNonOwner(t *testing.T) {
	repo := &membershipStub{members: map[string]bool{"g1|u2": true}, ownerKey: "u1"}
	svc := NewServiceWithRepo(repo)
	err := svc.Recall(gmAuthCtx("u2"), RecallGroupMessageRequest{MessageID: "gm1", UserID: "u2"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied, got %v", err)
	}
}

func TestServiceRecall_RejectsNullSender(t *testing.T) {
	repo := &membershipStub{members: map[string]bool{"g1|u1": true}, ownerEmpty: true}
	svc := NewServiceWithRepo(repo)
	err := svc.Recall(gmAuthCtx("u1"), RecallGroupMessageRequest{MessageID: "gm1", UserID: "u1"})
	if !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("expected ErrPermissionDenied for NULL sender, got %v", err)
	}
}

func TestServiceRecall_AllowsOwner(t *testing.T) {
	repo := &membershipStub{
		members:  map[string]bool{"g1|u1": true},
		ownerKey: "u1",
		stubRepo: stubRepo{recallMeta: MessageEventMeta{GroupID: "g1", MessageID: "gm1"}},
	}
	svc := NewServiceWithRepo(repo)
	err := svc.Recall(gmAuthCtx("u1"), RecallGroupMessageRequest{MessageID: "gm1", UserID: "u1"})
	if err != nil {
		t.Fatalf("expected success for owner, got %v", err)
	}
}

type stubUserReader struct{}

func (stubUserReader) OpenTheDoor(_ context.Context, _ string) (users.UserItem, bool, error) {
	return users.UserItem{}, false, nil
}
