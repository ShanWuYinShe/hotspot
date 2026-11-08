package privatechats

import (
	"context"
	"errors"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/httpx"
)

// g5NonEnsurerRepo 具备 Repository 全部方法但不提供 EnsureChat 写能力，
// 用于覆盖 Service.EnsureChat 的 unsupported 分支。
type g5NonEnsurerRepo struct {
	*stubChatRepo
}

// g5EmptyTargetsUserReader 让 OpenTheDoor 判定目标存在、但批量目标上下文
// 为空（模拟目标在两次读取之间被注销），覆盖 buildChats 的跳过分支与
// OpenTheDoor 的「组装后为空 → 10300」分支。
type g5EmptyTargetsUserReader struct {
	found bool
}

func (u *g5EmptyTargetsUserReader) OpenTheDoor(context.Context, string) (users.UserItem, bool, error) {
	return users.UserItem{User: users.UserInfo{UserID: "u2", Nickname: "Bob"}}, u.found, nil
}

func (u *g5EmptyTargetsUserReader) ChatTargetsContext(context.Context, string, []string) (map[string]users.ChatTargetContext, error) {
	return map[string]users.ChatTargetContext{}, nil
}

func TestCoverageG5ServiceEnsureChatUnsupportedRepo(t *testing.T) {
	svc := NewServiceWithDeps(&g5NonEnsurerRepo{stubChatRepo: &stubChatRepo{}}, &stubMsgReadService{}, &stubChatUserReader{})
	err := svc.EnsureChat(context.Background(), "u1:u2", "u1", "u2", time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC))
	if !errors.Is(err, errChatEnsurerUnsupported) {
		t.Fatalf("expected errChatEnsurerUnsupported, got %v", err)
	}
}

// TestCoverageG5ServiceOpenTheDoorErrorPaths 逐条覆盖 OpenTheDoor 的仓储/
// 消息读取错误路径。既有用例中的 user stub 默认 found=false，会在首步
// 提前返回，实际未触达这些分支。
func TestCoverageG5ServiceOpenTheDoorErrorPaths(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	boom := errors.New("boom")
	deletedAt := now.Add(-time.Hour)
	baseState := ChatState{TargetID: "u2", CreatedAt: now}
	foundUser := func() *stubChatUserReader {
		return &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	}

	cases := []struct {
		name string
		repo *stubChatRepo
		msgs *stubMsgReadService
	}{
		{
			name: "get_or_create",
			repo: &stubChatRepo{createdAt: now, state: baseState, createErr: boom},
			msgs: &stubMsgReadService{},
		},
		{
			name: "get_state",
			repo: &stubChatRepo{createdAt: now, state: baseState, stateErr: boom},
			msgs: &stubMsgReadService{},
		},
		{
			name: "recent",
			repo: &stubChatRepo{createdAt: now, state: baseState},
			msgs: &stubMsgReadService{recentErr: boom},
		},
		{
			// Recent 成功后 buildChats 的批量末条读取失败。
			name: "build_chats",
			repo: &stubChatRepo{createdAt: now, state: baseState},
			msgs: &stubMsgReadService{lastErr: boom},
		},
		{
			name: "restore",
			repo: &stubChatRepo{
				createdAt:  now,
				state:      ChatState{TargetID: "u2", CreatedAt: now, DeletedAt: &deletedAt},
				restoreErr: boom,
			},
			msgs: &stubMsgReadService{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewServiceWithDeps(tc.repo, tc.msgs, foundUser())
			if _, err := svc.OpenTheDoor(chatCtx("u1"), "u2", 20); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

// TestCoverageG5ServiceOpenTheDoorZeroCreatedAtFallback 覆盖 GetState 返回
// CreatedAt 零值时回退 GetOrCreate 时间戳的分支。
func TestCoverageG5ServiceOpenTheDoorZeroCreatedAtFallback(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	repo := &stubChatRepo{createdAt: now, state: ChatState{TargetID: "u2"}}
	usr := &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, usr)

	result, err := svc.OpenTheDoor(chatCtx("u1"), "u2", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := now.Format(time.RFC3339Nano)
	if result.Chat.CreatedAt == nil || *result.Chat.CreatedAt != want {
		t.Fatalf("created_at fallback: got %v want %q", result.Chat.CreatedAt, want)
	}
	if result.Chat.LastActivityAt != want {
		t.Fatalf("last_activity fallback: got %q want %q", result.Chat.LastActivityAt, want)
	}
}

// TestCoverageG5ServiceOpenTheDoorTargetDisappeared 覆盖批量组装结果为空时
// 按 10300（users.ErrUserNotFound）返回的分支。
func TestCoverageG5ServiceOpenTheDoorTargetDisappeared(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	repo := &stubChatRepo{createdAt: now, state: ChatState{TargetID: "u2", CreatedAt: now}}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &g5EmptyTargetsUserReader{found: true})

	if _, err := svc.OpenTheDoor(chatCtx("u1"), "u2", 20); !errors.Is(err, users.ErrUserNotFound) {
		t.Fatalf("expected users.ErrUserNotFound, got %v", err)
	}
}

// TestCoverageG5ServiceListBuildChatsErrors 覆盖 buildChats 三次批量读的
// 错误路径（任一失败列表请求都应返回错误而非部分结果）。
func TestCoverageG5ServiceListBuildChatsErrors(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	boom := errors.New("boom")
	states := []ChatState{{ChatKey: key("u1", "u2"), TargetID: "u2", CreatedAt: now}}
	okUser := func() *stubChatUserReader {
		return &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	}

	cases := []struct {
		name string
		msgs *stubMsgReadService
		usr  *stubChatUserReader
	}{
		{"last_batch", &stubMsgReadService{lastErr: boom}, okUser()},
		{"unread_batch", &stubMsgReadService{unreadErr: boom}, okUser()},
		{"targets_batch", &stubMsgReadService{}, &stubChatUserReader{userErr: boom}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewServiceWithDeps(&stubChatRepo{states: states}, tc.msgs, tc.usr)
			if _, err := svc.List(chatCtx("u1"), MyPrivateChatsRequest{}); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

// TestCoverageG5ServiceListEmptyStates 覆盖无会话时提前返回空列表的路径
// （next_cursor 必须保持 nil）。
func TestCoverageG5ServiceListEmptyStates(t *testing.T) {
	svc := NewServiceWithDeps(&stubChatRepo{}, &stubMsgReadService{}, &stubChatUserReader{})
	result, err := svc.List(chatCtx("u1"), MyPrivateChatsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 0 || result.NextCursor != nil {
		t.Fatalf("expected empty page, got items=%d next=%v", len(result.Items), result.NextCursor)
	}
}

// TestCoverageG5ServiceUpdateSettingsErrors 覆盖 UpdateSettings 的目标用户
// 读取错误与 GetOrCreate 错误分支。
func TestCoverageG5ServiceUpdateSettingsErrors(t *testing.T) {
	boom := errors.New("boom")

	svcUserErr := NewServiceWithDeps(&stubChatRepo{}, &stubMsgReadService{}, &stubChatUserReader{userErr: boom})
	if err := svcUserErr.UpdateSettings(chatCtx("u1"), UpdatePrivateChatSettingsRequest{TargetUserID: "u2", IsMuted: boolPtr(true)}); !errors.Is(err, boom) {
		t.Fatalf("expected user read error, got %v", err)
	}

	svcCreateErr := NewServiceWithDeps(&stubChatRepo{createErr: boom}, &stubMsgReadService{}, &stubChatUserReader{item: defaultUserItem(), found: true})
	if err := svcCreateErr.UpdateSettings(chatCtx("u1"), UpdatePrivateChatSettingsRequest{TargetUserID: "u2", IsMuted: boolPtr(true)}); !errors.Is(err, boom) {
		t.Fatalf("expected get_or_create error, got %v", err)
	}
}

// TestCoverageG5ServiceAssembleChatPermissionBranches 覆盖 assembleChat 的
// presence/双向拉黑/friends-only/自聊覆盖等权限分支。
func TestCoverageG5ServiceAssembleChatPermissionBranches(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	svc := NewServiceWithDeps(NewInMemoryRepository(), &stubMsgReadService{}, &stubChatUserReader{})
	state := ChatState{ChatKey: key("u1", "u2"), TargetID: "u2", CreatedAt: now, LastActivityAt: now}
	presence := users.UserPresenceOnline
	base := users.ChatTargetContext{
		Info:             users.UserInfo{UserID: "u2", Nickname: "Bob"},
		PresenceStatus:   &presence,
		AllowPrivateChat: users.ChatPermEveryone,
		SettingsFound:    true,
	}

	item := svc.assembleChat("u1", state, base, nil, 4)
	if item.PresenceStatus == nil || *item.PresenceStatus != string(users.UserPresenceOnline) {
		t.Fatalf("presence status: %#v", item.PresenceStatus)
	}
	if !item.CanMessage || item.ForbiddenReason != nil || item.UnreadCount != 4 {
		t.Fatalf("unexpected baseline item: %#v", item)
	}
	if item.CreatedAt == nil || *item.CreatedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("created_at: %v", item.CreatedAt)
	}

	viewerBlocked := base
	viewerBlocked.Relation = users.Relationship{IsBlocked: true}
	item = svc.assembleChat("u1", state, viewerBlocked, nil, 0)
	if item.CanMessage || item.ForbiddenReason == nil || *item.ForbiddenReason != httpx.ReasonBlockedByUser {
		t.Fatalf("viewer blocked: %#v", item)
	}

	targetBlocked := base
	targetBlocked.Relation = users.Relationship{TargetBlocked: true}
	item = svc.assembleChat("u1", state, targetBlocked, nil, 0)
	if item.CanMessage || item.ForbiddenReason == nil || *item.ForbiddenReason != httpx.ReasonBlockedByUser {
		t.Fatalf("target blocked: %#v", item)
	}

	friendsOnly := base
	friendsOnly.AllowPrivateChat = users.ChatPermFriends
	item = svc.assembleChat("u1", state, friendsOnly, nil, 0)
	if item.CanMessage || item.ForbiddenReason == nil || *item.ForbiddenReason != httpx.ReasonPrivacySetting {
		t.Fatalf("friends-only without mutual follow: %#v", item)
	}

	mutual := friendsOnly
	mutual.Relation = users.Relationship{IsFollowing: true, IsFollowedBy: true}
	item = svc.assembleChat("u1", state, mutual, nil, 0)
	if !item.CanMessage || item.ForbiddenReason != nil {
		t.Fatalf("friends-only with mutual follow: %#v", item)
	}

	// 自聊（userID == state.TargetID）必须覆盖黑名单判定。
	selfState := state
	selfState.TargetID = "u1"
	selfTarget := base
	selfTarget.Info.UserID = "u1"
	selfTarget.Relation = users.Relationship{IsBlocked: true, TargetBlocked: true}
	item = svc.assembleChat("u1", selfState, selfTarget, nil, 0)
	if !item.CanMessage || item.ForbiddenReason != nil {
		t.Fatalf("self chat must override block: %#v", item)
	}
}
