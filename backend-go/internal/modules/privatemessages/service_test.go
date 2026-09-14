package privatemessages

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

type stubMsgRepo struct {
	sendResult MessageSentResult
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
	counts     map[string]int
	countErr   error
	ops        []string // 配额原子化断言：lock/count/send 的调用顺序记录
}

func (r *stubMsgRepo) Send(_ context.Context, _, _ string, _ SendPrivateMessageRequest) (MessageSentResult, error) {
	r.ops = append(r.ops, "send")
	return r.sendResult, r.sendErr
}
func (r *stubMsgRepo) History(_ context.Context, _ string, _ ListPrivateMessagesRequest) (PaginatedMessages, error) {
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
func (r *stubMsgRepo) LockStrangerQuota(_ context.Context, _, _ string) error {
	r.ops = append(r.ops, "lock")
	return nil
}

func (r *stubMsgRepo) Recall(_ context.Context, _ string, _ time.Duration, _ RecallPrivateMessageRequest) (MessageEventMeta, error) {
	return r.recallMeta, r.recallErr
}
func (r *stubMsgRepo) MarkAsRead(_ context.Context, _ string, _ MarkPrivateMessagesReadRequest) (MessageEventMeta, error) {
	return r.readMeta, r.readErr
}
func (r *stubMsgRepo) MarkAllRead(_ context.Context, _ string) error {
	return r.markAllErr
}
func (r *stubMsgRepo) GetMessageForReply(_ context.Context, _ string, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}
func (r *stubMsgRepo) GetMediaForMessage(_ context.Context, _ string, _ string) (map[string]any, bool, error) {
	return nil, false, nil
}
func (r *stubMsgRepo) MessageParticipants(_ context.Context, _ string) (string, string, bool, error) {
	return "", "", false, nil
}

func (r *stubMsgRepo) MessageExists(_ context.Context, _ string) (bool, string, error) {
	return false, "", nil
}
func (r *stubMsgRepo) CountMessagesByTarget(_ context.Context, _ string, _ time.Time) (map[string]int, error) {
	r.ops = append(r.ops, "count")
	return r.counts, r.countErr
}
func (r *stubMsgRepo) GetMessageReportSnapshot(_ context.Context, _ string) (MessageReportSnapshot, bool, error) {
	return MessageReportSnapshot{}, false, nil
}

type stubUserReader struct {
	item          users.UserItem
	found         bool
	userErr       error
	settings      users.UserSettings
	settingsErr   error
	relations     map[string]users.Relationship // per-target relation (stranger quota checks)
	relationErr   error
	callerBlocked bool // BlockStateBetween：发送者拉黑了对方
	targetBlocked bool // BlockStateBetween：对方拉黑了发送者
}

func (u *stubUserReader) OpenTheDoor(_ context.Context, userID string) (users.UserItem, bool, error) {
	if u.userErr != nil {
		return users.UserItem{}, false, u.userErr
	}
	return u.item, u.found, nil
}
func (u *stubUserReader) SettingsFor(_ context.Context, _ string) (users.UserSettings, bool, error) {
	return u.settings, true, u.settingsErr
}

func (u *stubUserReader) BlockStateBetween(_ context.Context, _ string, _ string) (callerBlocked, targetBlocked bool, err error) {
	return u.callerBlocked, u.targetBlocked, nil
}

func (u *stubUserReader) Relationship(_ context.Context, _ string, targetID string) (users.Relationship, error) {
	if u.relationErr != nil {
		return users.Relationship{}, u.relationErr
	}
	if u.relations != nil {
		if rel, ok := u.relations[targetID]; ok {
			return rel, nil
		}
	}
	return users.Relationship{}, nil
}

// quotaSource is a static appconfig.Source for the stranger quota tests.
type quotaSource struct {
	quota        int
	recallWindow int // 撤回窗口秒数；未设置时按默认 120
}

func (q quotaSource) Get(context.Context) appconfig.AppConfig {
	return appconfig.AppConfig{Limits: appconfig.Limits{StrangerMessageDailyQuota: q.quota, MessageRecallWindowSeconds: q.recallWindow}}
}

func msgCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func TestServiceSendDelegatesToRepo(t *testing.T) {
	repo := &stubMsgRepo{sendResult: MessageSentResult{MessageID: "pm1", SentAt: "2025-01-01T00:00:00Z"}}
	svc := NewServiceWithDeps(repo, &stubUserReader{found: true, settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone}})
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

func TestServiceSendRejectsBlockedByTarget(t *testing.T) {
	// 被对方拉黑：核心拦截场景（spec api_spec_chat.md：send 必须校验黑名单）
	svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{
		found:         true,
		targetBlocked: true,
		settings:      users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text"})
	if err == nil {
		t.Fatal("expected error for blocked-by-target user")
	}
}

func TestServiceSendRejectsCallerBlockedTarget(t *testing.T) {
	// 发送者拉黑了对方：同样禁止发送（双向黑名单口径）
	svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{
		found:         true,
		callerBlocked: true,
		settings:      users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text"})
	if err == nil {
		t.Fatal("expected error for caller-blocked user")
	}
}

func TestServiceSendRejectsNobodySetting(t *testing.T) {
	svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{
		found:    true,
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermNobody},
	})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text"})
	if err == nil {
		t.Fatal("expected error for nobody setting")
	}
}

func TestServiceSendAllowsSelfMessage(t *testing.T) {
	repo := &stubMsgRepo{sendResult: MessageSentResult{MessageID: "pm1"}}
	svc := NewServiceWithDeps(repo, &stubUserReader{found: true, settings: users.UserSettings{AllowPrivateChat: users.ChatPermNobody}})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u1", Type: "text"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceSendPropagatesError(t *testing.T) {
	svc := NewServiceWithDeps(&stubMsgRepo{sendErr: errors.New("boom")}, &stubUserReader{found: true, settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone}})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceHistoryDelegates(t *testing.T) {
	repo := &stubMsgRepo{histResult: PaginatedMessages{Items: []PrivateMessageItem{{MessageID: "pm1"}}}}
	svc := NewServiceWithRepo(repo)
	result, err := svc.History(msgCtx("u1"), ListPrivateMessagesRequest{TargetUserID: "u2"})
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

// spec chat.md：撤回时间窗消费 AppConfig.limits.message_recall_window_seconds。
func TestServiceRecallUsesConfiguredWindow(t *testing.T) {
	repo := NewInMemoryRepository()
	sent, err := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "hi"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	svc := NewServiceWithRepo(repo)
	svc.SetAppConfig(quotaSource{recallWindow: 90})
	if err := svc.Recall(msgCtx("alice"), RecallPrivateMessageRequest{MessageID: sent.MessageID}); err != nil {
		t.Fatalf("expected recall within configured window to succeed, got %v", err)
	}
}

// 窗口配置 0 = 撤回禁用（spec chat.md → message_recall_window_seconds 0=禁用）。
func TestServiceRecallDisabledWhenWindowZero(t *testing.T) {
	repo := NewInMemoryRepository()
	sent, err := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "hi"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	svc := NewServiceWithRepo(repo)
	svc.SetAppConfig(quotaSource{recallWindow: 0})
	err = svc.Recall(msgCtx("alice"), RecallPrivateMessageRequest{MessageID: sent.MessageID})
	if !errors.Is(err, ErrMessageNotRecallable) {
		t.Fatalf("expected ErrMessageNotRecallable when window disabled, got %v", err)
	}
}

// appConfig 未接线时回退默认 120s：新消息可撤回。
func TestServiceRecallFallsBackToDefaultWindow(t *testing.T) {
	repo := NewInMemoryRepository()
	sent, err := repo.Send(context.Background(), "alice", "bob", SendPrivateMessageRequest{Type: "text", Content: "hi"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	svc := NewServiceWithRepo(repo)
	if err := svc.Recall(msgCtx("alice"), RecallPrivateMessageRequest{MessageID: sent.MessageID}); err != nil {
		t.Fatalf("expected recall with default window to succeed, got %v", err)
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

// 配额原子化：lock 必须先于 count、count 先于 send（同事务内咨询锁串行化
// check-then-act，见 Send）。
func TestServiceSendStrangerQuotaLocksBeforeCounting(t *testing.T) {
	repo := &stubMsgRepo{
		sendResult: MessageSentResult{MessageID: "pm1"},
		counts:     map[string]int{"u2": 1},
	}
	svc := NewServiceWithDeps(repo, &stubUserReader{
		found:     true,
		item:      users.UserItem{Relation: &users.Relationship{}},
		relations: map[string]users.Relationship{"u2": {}},
		settings:  users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	})
	svc.SetAppConfig(quotaSource{quota: 5})
	if _, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"lock", "count", "send"}
	if len(repo.ops) != len(want) {
		t.Fatalf("expected ops %v, got %v", want, repo.ops)
	}
	for i, op := range want {
		if repo.ops[i] != op {
			t.Fatalf("expected ops %v, got %v", want, repo.ops)
		}
	}
}

// self-send 有意绕过配额：不取锁、不计数。
func TestServiceSendSelfTargetSkipsQuotaLock(t *testing.T) {
	repo := &stubMsgRepo{sendResult: MessageSentResult{MessageID: "pm1"}}
	svc := NewServiceWithDeps(repo, &stubUserReader{
		found:    true,
		item:     users.UserItem{Relation: &users.Relationship{}},
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	})
	svc.SetAppConfig(quotaSource{quota: 0}) // 0=禁陌生人消息；self 不受影响
	if _, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u1", Type: "text", Content: "note"}); err != nil {
		t.Fatalf("expected self-send to bypass quota, got %v", err)
	}
	for _, op := range repo.ops {
		if op == "lock" || op == "count" {
			t.Fatalf("self-send must not trigger quota ops, got %v", repo.ops)
		}
	}
}

func TestServiceSendStrangerQuotaAggregatesAcrossTargets(t *testing.T) {
	// 两个陌生人(u2/u3)各已发 3 条,quota=5:总量 6 >= 5,拒绝(per-target
	// 口径下各自 3<5 会放行,总量口径必须拒绝)。
	repo := &stubMsgRepo{counts: map[string]int{"u2": 3, "u3": 3}}
	svc := NewServiceWithDeps(repo, &stubUserReader{
		found: true,
		item:  users.UserItem{Relation: &users.Relationship{}},
		relations: map[string]users.Relationship{
			"u2": {}, // 陌生人(非互关)
			"u3": {}, // 陌生人(非互关)
		},
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	})
	svc.SetAppConfig(quotaSource{quota: 5})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"})
	if err == nil {
		t.Fatal("expected stranger quota exceeded when combined stranger volume >= quota")
	}
	assertForbiddenReason(t, err, httpx.ReasonStrangerQuotaExceeded)
}

func TestServiceSendStrangerQuotaAllowsUnderTotal(t *testing.T) {
	// 两个陌生人合计 4 < quota=5,允许发送。
	repo := &stubMsgRepo{
		sendResult: MessageSentResult{MessageID: "pm1"},
		counts:     map[string]int{"u2": 2, "u3": 2},
	}
	svc := NewServiceWithDeps(repo, &stubUserReader{
		found: true,
		item:  users.UserItem{Relation: &users.Relationship{}},
		relations: map[string]users.Relationship{
			"u2": {},
			"u3": {},
		},
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	})
	svc.SetAppConfig(quotaSource{quota: 5})
	if _, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"}); err != nil {
		t.Fatalf("expected send allowed under total quota, got %v", err)
	}
}

func TestServiceSendStrangerQuotaIgnoresMutualFollows(t *testing.T) {
	// 互关用户 u3 已发 100 条不计入陌生人总量;陌生人 u2 4 条 < quota=5,允许。
	repo := &stubMsgRepo{
		sendResult: MessageSentResult{MessageID: "pm1"},
		counts:     map[string]int{"u2": 4, "u3": 100},
	}
	svc := NewServiceWithDeps(repo, &stubUserReader{
		found: true,
		item:  users.UserItem{Relation: &users.Relationship{}},
		relations: map[string]users.Relationship{
			"u2": {},                                      // 陌生人
			"u3": {IsFollowing: true, IsFollowedBy: true}, // 互相关注
		},
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	})
	svc.SetAppConfig(quotaSource{quota: 5})
	if _, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"}); err != nil {
		t.Fatalf("mutual-follow volume must not count toward stranger quota, got %v", err)
	}
}

func TestServiceSendStrangerQuotaZeroRejects(t *testing.T) {
	// quota=0:禁止任何陌生人消息,直接拒绝(不查库)。
	svc := NewServiceWithDeps(&stubMsgRepo{}, &stubUserReader{
		found:    true,
		item:     users.UserItem{Relation: &users.Relationship{}},
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	})
	svc.SetAppConfig(quotaSource{quota: 0})
	_, err := svc.Send(msgCtx("u1"), SendPrivateMessageRequest{TargetUserID: "u2", Type: "text", Content: "hi"})
	if err == nil {
		t.Fatal("expected rejection when quota is 0")
	}
	assertForbiddenReason(t, err, httpx.ReasonStrangerQuotaExceeded)
}

// assertForbiddenReason 断言 err 为 httpx.APIError 且 forbidden_reason 匹配。
func assertForbiddenReason(t *testing.T, err error, reason httpx.ForbiddenReason) {
	t.Helper()
	var apiErr *httpx.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *httpx.APIError, got %v", err)
	}
	if apiErr.Code != httpx.CodeForbidden {
		t.Fatalf("expected code httpx.CodeForbidden, got %d", apiErr.Code)
	}
	data, ok := apiErr.Data.(httpx.ForbiddenErrorData)
	if !ok {
		t.Fatalf("expected ForbiddenErrorData data, got %#v", apiErr.Data)
	}
	if data.ForbiddenReason != reason {
		t.Fatalf("expected forbidden_reason=%q, got %q", reason, data.ForbiddenReason)
	}
}
