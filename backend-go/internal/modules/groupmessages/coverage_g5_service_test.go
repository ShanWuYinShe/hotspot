package groupmessages

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/hotcache"
	"hotspot/backend-go/internal/platform/types"
)

// gmCovStubRepo 在既有 stubRepo 基础上开放服务层分支所需的少量开关
// （成员/缓存/禁言/已读计数/@ 过滤/消息元数据），其余行为沿用 stubRepo。
type gmCovStubRepo struct {
	stubRepo

	isMember    *bool
	isMemberErr error

	muteMuted bool
	muteUntil string
	muteFound bool
	muteErr   error

	filtered  []string
	filterErr error
	filterSet bool

	latest    string
	latestErr error

	readCounts    map[string]int
	readCountsErr error

	markAllGroups    []string
	markAllGroupsErr error

	messageGroupID      string
	messageGroupIDFound bool
	messageGroupIDErr   error

	lookupMeta  MessageOwnerMeta
	lookupFound bool
	lookupErr   error
}

func (r *gmCovStubRepo) IsMember(context.Context, string, string) (bool, error) {
	if r.isMemberErr != nil {
		return false, r.isMemberErr
	}
	if r.isMember != nil {
		return *r.isMember, nil
	}
	return true, nil
}

func (r *gmCovStubRepo) MemberMuteStatus(context.Context, string, string) (bool, string, bool, error) {
	return r.muteMuted, r.muteUntil, r.muteFound, r.muteErr
}

func (r *gmCovStubRepo) FilterMemberIDs(_ context.Context, _ string, userIDs []string) ([]string, error) {
	if r.filterSet {
		return r.filtered, r.filterErr
	}
	return userIDs, nil
}

func (r *gmCovStubRepo) LatestMessageID(context.Context, string) (string, error) {
	return r.latest, r.latestErr
}

func (r *gmCovStubRepo) GetReadCounts(context.Context, string, []string) (map[string]int, error) {
	return r.readCounts, r.readCountsErr
}

func (r *gmCovStubRepo) MarkAllRead(context.Context, string) ([]string, error) {
	return r.markAllGroups, r.markAllGroupsErr
}

func (r *gmCovStubRepo) MessageGroupID(ctx context.Context, messageID string) (string, bool, error) {
	if r.messageGroupIDErr != nil || r.messageGroupIDFound {
		return r.messageGroupID, r.messageGroupIDFound, r.messageGroupIDErr
	}
	return r.stubRepo.MessageGroupID(ctx, messageID)
}

func (r *gmCovStubRepo) LookupMessageMeta(ctx context.Context, messageID string) (MessageOwnerMeta, bool, error) {
	if r.lookupErr != nil {
		return MessageOwnerMeta{}, false, r.lookupErr
	}
	if r.lookupFound {
		return r.lookupMeta, true, nil
	}
	return r.stubRepo.LookupMessageMeta(ctx, messageID)
}

// gmCovShareValidator 可编排的 share_code 校验器（nil 时 service 跳过校验）。
type gmCovShareValidator struct{ err error }

func (v gmCovShareValidator) ValidateShareCode(context.Context, string) error { return v.err }

// gmCovUserReader 可编排的 users.BatchGet 替身（resolveUserInfo 三分支）。
type gmCovUserReader struct {
	items map[string]users.UserInfo
	err   error
}

func (r gmCovUserReader) BatchGet(context.Context, []string) (map[string]users.UserInfo, error) {
	return r.items, r.err
}

func TestMutedErrorNilReceiver(t *testing.T) {
	var e *MutedError
	if got := e.Error(); got != "groupmessages: user is muted" {
		t.Fatalf("nil MutedError message = %q", got)
	}
}

// appconfig 撤回窗口 0/负值 = 撤回禁用（spec chat.md）。
func TestServiceRecallWindowDisabledByConfig(t *testing.T) {
	for _, seconds := range []int{0, -5} {
		svc := NewServiceWithRepo(&stubRepo{})
		svc.SetAppConfig(recallWindowSource(seconds))
		if got := svc.recallWindow(context.Background()); got != 0 {
			t.Fatalf("recallWindow(%d) = %v, want 0", seconds, got)
		}
	}
}

// TestServiceSendErrorBranches 覆盖 Send 的禁言/share/附件错误分支与 share 载荷。
func TestServiceSendErrorBranches(t *testing.T) {
	boom := errors.New("boom")
	ctx := gmAuthCtx("u1")
	base := SendGroupMessageRequest{GroupID: "g1", Type: "text", Content: "hi"}

	t.Run("mute check error", func(t *testing.T) {
		svc := NewServiceWithDeps(&gmCovStubRepo{muteErr: boom}, nil, nil, nil)
		if _, err := svc.Send(ctx, base); !errors.Is(err, boom) {
			t.Fatalf("Send err = %v", err)
		}
	})

	t.Run("muted", func(t *testing.T) {
		repo := &gmCovStubRepo{muteFound: true, muteMuted: true, muteUntil: "2026-01-01T00:00:00Z"}
		svc := NewServiceWithDeps(repo, nil, nil, nil)
		_, err := svc.Send(ctx, base)
		if !errors.Is(err, ErrUserMuted) {
			t.Fatalf("Send err = %v, want ErrUserMuted", err)
		}
		var mutedErr *MutedError
		if !errors.As(err, &mutedErr) || mutedErr.MutedUntil != "2026-01-01T00:00:00Z" {
			t.Fatalf("muted error = %#v", err)
		}
	})

	t.Run("invalid share code", func(t *testing.T) {
		svc := NewServiceWithDeps(&gmCovStubRepo{}, nil, nil, gmCovShareValidator{err: boom})
		req := base
		req.ShareCode = "sc_invalid"
		if _, err := svc.Send(ctx, req); !errors.Is(err, ErrInvalidShareCode) {
			t.Fatalf("Send err = %v, want ErrInvalidShareCode", err)
		}
	})

	t.Run("media lookup error", func(t *testing.T) {
		repo := &gmCovStubRepo{stubRepo: stubRepo{mediaErr: boom}}
		svc := NewServiceWithDeps(repo, nil, nil, nil)
		req := base
		req.Type = "image"
		req.FileID = "file_1"
		if _, err := svc.Send(ctx, req); !errors.Is(err, boom) {
			t.Fatalf("Send err = %v", err)
		}
	})

	t.Run("file not owned", func(t *testing.T) {
		repo := &gmCovStubRepo{stubRepo: stubRepo{mediaOk: false}}
		svc := NewServiceWithDeps(repo, nil, nil, nil)
		req := base
		req.Type = "image"
		req.FileID = "file_1"
		if _, err := svc.Send(ctx, req); !errors.Is(err, ErrFileNotOwned) {
			t.Fatalf("Send err = %v, want ErrFileNotOwned", err)
		}
	})

	t.Run("share code payload", func(t *testing.T) {
		repo := &gmCovStubRepo{stubRepo: stubRepo{sendResult: MessageSentResult{MessageID: "gm_share", SentAt: "2026-01-01T00:00:00Z"}}}
		svc := NewServiceWithDeps(repo, nil, nil, nil)
		req := base
		req.Type = "share"
		req.Content = ""
		req.ShareCode = "sc_ok"
		if _, err := svc.Send(ctx, req); err != nil {
			t.Fatalf("Send err = %v", err)
		}
	})
}

// TestServiceNotifyMentionsErrorBranches 覆盖 @ 通知的成员过滤失败 / 无目标 /
// 批量落库失败降级（广播仍继续）。
func TestServiceNotifyMentionsErrorBranches(t *testing.T) {
	boom := errors.New("boom")
	ctx := gmAuthCtx("u1")
	req := SendGroupMessageRequest{GroupID: "g1", SenderUserID: "u1", Type: "text", Content: "hi", Mentions: []string{"u2"}}
	msg := MessageSentResult{MessageID: "m1", SentAt: "2026-01-01T00:00:00Z"}

	// 成员过滤失败：记日志并放弃通知，不返回错误。
	notifier := &stubMentionNotifier{}
	svc := NewServiceWithDeps(&gmCovStubRepo{filterSet: true, filterErr: boom}, nil, notifier, nil)
	svc.notifyMentions(ctx, "u1", req, msg)
	if notifier.appendedCount != 0 {
		t.Fatalf("filter failure must skip append, got %d", notifier.appendedCount)
	}

	// 过滤后没有真实成员目标：直接返回。
	notifier = &stubMentionNotifier{}
	svc = NewServiceWithDeps(&gmCovStubRepo{filterSet: true, filtered: nil}, nil, notifier, nil)
	svc.notifyMentions(ctx, "u1", req, msg)
	if notifier.appendedCount != 0 {
		t.Fatalf("no targets must skip append, got %d", notifier.appendedCount)
	}

	// 批量落库失败：仅降级 notify_id（广播照常），不返回错误。
	notifier = &stubMentionNotifier{returnErr: boom}
	svc = NewServiceWithDeps(&gmCovStubRepo{filterSet: true, filtered: []string{"u2"}}, nil, notifier, nil)
	svc.notifyMentions(ctx, "u1", req, msg)
	if notifier.appendedCount != 1 {
		t.Fatalf("batch append failure must still attempt append, got %d", notifier.appendedCount)
	}
}

// TestServiceResolveUserInfoBranches 覆盖用户信息解析的命中/缺失/查询失败降级。
func TestServiceResolveUserInfoBranches(t *testing.T) {
	ctx := context.Background()
	avatar := "https://cdn.example/avatar.jpg"
	reader := gmCovUserReader{items: map[string]users.UserInfo{
		"u1": {UserID: "u1", Nickname: "昵称", AvatarURL: types.StrPtr(avatar)},
	}}
	svc := NewServiceWithRepoAndUsers(&stubRepo{}, reader)
	info := svc.resolveUserInfo(ctx, "u1")
	if info.Nickname != "昵称" || info.AvatarURL == nil || *info.AvatarURL != avatar {
		t.Fatalf("resolved info = %#v", info)
	}

	// 批量结果缺失该用户：昵称回退 user_id。
	info = svc.resolveUserInfo(ctx, "u_missing")
	if info.Nickname != "u_missing" || info.AvatarURL != nil {
		t.Fatalf("fallback info = %#v", info)
	}

	// 查询失败：同样回退 user_id（发送链路不因用户信息失败而中断）。
	svc = NewServiceWithRepoAndUsers(&stubRepo{}, gmCovUserReader{err: errors.New("boom")})
	info = svc.resolveUserInfo(ctx, "u_err")
	if info.Nickname != "u_err" {
		t.Fatalf("error fallback info = %#v", info)
	}
}

// TestServiceEnsureMemberCacheBranches 覆盖成员关系热缓存的命中/非成员命中/
// 回源写缓存（成员与非成员都缓存，防穿透）。
func TestServiceEnsureMemberCacheBranches(t *testing.T) {
	ctx := context.Background()
	cache := hotcache.New(nil)
	t.Cleanup(cache.Close)

	svc := NewServiceWithRepo(&gmCovStubRepo{})
	svc.SetMemberCache(cache)

	cache.Set(ctx, hotcache.GroupMemberKey("gc", "u_cached_member"), "1", hotcache.GroupMemberTTL)
	if err := svc.ensureMember(ctx, "gc", "u_cached_member"); err != nil {
		t.Fatalf("cached member err = %v", err)
	}

	cache.Set(ctx, hotcache.GroupMemberKey("gc", "u_cached_outsider"), "0", hotcache.GroupMemberTTL)
	if err := svc.ensureMember(ctx, "gc", "u_cached_outsider"); !errors.Is(err, ErrNotGroupMember) {
		t.Fatalf("cached outsider err = %v", err)
	}

	// 缓存未命中 + repo 判定成员：写入 "1"。
	if err := svc.ensureMember(ctx, "gc", "u_repo_member"); err != nil {
		t.Fatalf("repo member err = %v", err)
	}
	if v, ok := cache.Get(ctx, hotcache.GroupMemberKey("gc", "u_repo_member")); !ok || v != "1" {
		t.Fatalf("member cache value = %q ok=%v", v, ok)
	}

	// 缓存未命中 + repo 判定非成员：写入 "0" 并拒绝。
	outsider := false
	svc2 := NewServiceWithRepo(&gmCovStubRepo{isMember: &outsider})
	svc2.SetMemberCache(cache)
	if err := svc2.ensureMember(ctx, "gc", "u_repo_outsider"); !errors.Is(err, ErrNotGroupMember) {
		t.Fatalf("repo outsider err = %v", err)
	}
	if v, ok := cache.Get(ctx, hotcache.GroupMemberKey("gc", "u_repo_outsider")); !ok || v != "0" {
		t.Fatalf("outsider cache value = %q ok=%v", v, ok)
	}
}

// TestServiceReadCountBranches 覆盖已读计数推送的尽力而为分支：
// flusher 未接线 / 批量查询失败 / 目标消息不可得 / 单群失败不影响 MarkAllRead。
func TestServiceReadCountBranches(t *testing.T) {
	boom := errors.New("boom")
	ctx := gmAuthCtx("u1")

	// readFlusher 未接线（零值 Service）：直接返回。
	svc := &Service{repo: &gmCovStubRepo{}}
	svc.scheduleReadCountUpdate("g1", []string{"m1"})

	// 批量已读计数查询失败：记日志并返回，不 panic。
	NewServiceWithRepo(&gmCovStubRepo{readCountsErr: boom}).flushReadCounts("g1", []string{"m1"})

	// 无 up_to 且拿不到最新消息：已读标记成功（计数推送尽力而为）。
	svc2 := NewServiceWithRepo(&gmCovStubRepo{latestErr: boom})
	if err := svc2.MarkAsRead(ctx, MarkGroupMessagesReadRequest{GroupID: "g1"}); err != nil {
		t.Fatalf("MarkAsRead best-effort err = %v", err)
	}

	// MarkAllRead：受影响群拿不到最新消息时跳过该群，整体仍成功。
	svc3 := NewServiceWithRepo(&gmCovStubRepo{markAllGroups: []string{"g1"}, latestErr: boom})
	if err := svc3.MarkAllRead(ctx); err != nil {
		t.Fatalf("MarkAllRead best-effort err = %v", err)
	}
}

// TestServiceAdminAndVisibilityErrorBranches 覆盖管理撤回/反应可见性的错误上抛。
func TestServiceAdminAndVisibilityErrorBranches(t *testing.T) {
	boom := errors.New("boom")
	ctx := gmAuthCtx("u1")

	// 管理撤回：消息元数据查询失败直接上抛。
	svc := NewServiceWithRepo(&gmCovStubRepo{lookupErr: boom})
	if err := svc.RecallMessageAsAdmin(ctx, "admin", "g1", "m1"); !errors.Is(err, boom) {
		t.Fatalf("RecallMessageAsAdmin err = %v", err)
	}

	// 举报裁决按 id 撤回：撤回执行失败（非已撤回冲突）上抛。
	svc2 := NewServiceWithRepo(&gmCovStubRepo{
		lookupFound: true,
		lookupMeta:  MessageOwnerMeta{GroupID: "g1", SenderUserID: "u1"},
		stubRepo:    stubRepo{recallErr: boom},
	})
	if hit, err := svc2.RecallMessageByIDAsAdmin(ctx, "admin", "m1"); hit || !errors.Is(err, boom) {
		t.Fatalf("RecallMessageByIDAsAdmin = (%v, %v)", hit, err)
	}

	// Update 授权阶段元数据查询失败。
	if err := svc.Update(ctx, UpdateGroupMessageRequest{MessageID: "m1", UserID: "u1", Content: "x"}); !errors.Is(err, boom) {
		t.Fatalf("Update err = %v", err)
	}

	// 反应目标可见性：成员校验失败上抛。
	svc3 := NewServiceWithRepo(&gmCovStubRepo{
		messageGroupIDFound: true,
		messageGroupID:      "g1",
		isMemberErr:         boom,
	})
	found, visible, err := svc3.IsReactionTargetVisible(ctx, "m1")
	if found || visible || !errors.Is(err, boom) {
		t.Fatalf("IsReactionTargetVisible = (%v, %v, %v)", found, visible, err)
	}
}

// TestReadCountFlusherTimerFlushAndDedupe 覆盖节流窗口尾的批量 flush：
// 同群窗口内多次 schedule 合并为一次回调、回调清空状态并调用 flush；
// flush panic 被 recover（后台 goroutine 不得击穿进程）。
func TestReadCountFlusherTimerFlushAndDedupe(t *testing.T) {
	f := newReadCountFlusher()
	flushed := make(chan string, 1)
	f.schedule("gcov", []string{"m1", "m2"}, func(groupID string, ids []string) {
		flushed <- groupID + ":" + strings.Join(ids, ",")
		panic("boom")
	})
	// 窗口内再次 schedule：命中已有 timer 分支，仅合并待推送集合。
	f.schedule("gcov", []string{"m3"}, func(string, []string) {
		t.Error("deduped schedule must not register a second flush")
	})

	select {
	case got := <-flushed:
		for _, want := range []string{"gcov", "m1", "m2", "m3"} {
			if !strings.Contains(got, want) {
				t.Fatalf("flush payload %q missing %q", got, want)
			}
		}
	case <-time.After(6 * time.Second):
		t.Fatal("read count flush did not run within window")
	}

	// 等待 panic recover 分支执行完成，并确认回调已清空 pending/timers。
	time.Sleep(100 * time.Millisecond)
	f.mu.Lock()
	pending, timers := len(f.pending), len(f.timers)
	f.mu.Unlock()
	if pending != 0 || timers != 0 {
		t.Fatalf("flusher state not cleared: pending=%d timers=%d", pending, timers)
	}
}
