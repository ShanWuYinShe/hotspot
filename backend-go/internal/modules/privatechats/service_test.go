package privatechats

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/types"
)

type stubChatRepo struct {
	createdAt    time.Time
	createErr    error
	states       []ChatState
	listCursor   string
	listErr      error
	state        ChatState
	stateErr     error
	restoreErr   error
	updateSetErr error
	deleteErr    error
}

func (r *stubChatRepo) GetOrCreate(_ context.Context, _, _ string) (time.Time, error) {
	if r.createdAt.IsZero() && r.createErr == nil {
		return time.Now().UTC(), nil
	}
	return r.createdAt, r.createErr
}
func (r *stubChatRepo) List(_ context.Context, _ string, opts ListOptions) ([]ChatState, string, error) {
	if r.listErr != nil {
		return nil, "", r.listErr
	}
	// 服务层不再自行分页（审查 P2-7 下推后）：桩复用与内存实现相同的分页
	// helper，保证 service 级用例仍能覆盖 limit 裁剪/游标语义。
	if r.listCursor != "" {
		return r.states, r.listCursor, nil
	}
	items, next := paginateChatStates(r.states, opts)
	return items, next, nil
}
func (r *stubChatRepo) GetState(_ context.Context, _, _ string) (ChatState, error) {
	return r.state, r.stateErr
}
func (r *stubChatRepo) RestoreForUser(_ context.Context, _, _ string, _ time.Time) error {
	return r.restoreErr
}
func (r *stubChatRepo) UpdateSettings(_ context.Context, _, _ string, _ UpdateChatSettings) error {
	return r.updateSetErr
}
func (r *stubChatRepo) DeleteForUser(_ context.Context, _, _ string, _ *time.Time) error {
	return r.deleteErr
}

type stubMsgReadService struct {
	recentResult privatemessages.PaginatedMessages
	recentErr    error
	lastResult   *privatemessages.PrivateMessageItem
	lastErr      error
	unreadCount  int
	unreadErr    error
}

func (m *stubMsgReadService) Recent(_ context.Context, _ string, _ int, _ *time.Time) (privatemessages.PaginatedMessages, error) {
	return m.recentResult, m.recentErr
}
func (m *stubMsgReadService) Last(_ context.Context, _ string, _ *time.Time) (*privatemessages.PrivateMessageItem, error) {
	return m.lastResult, m.lastErr
}
func (m *stubMsgReadService) UnreadCount(_ context.Context, _ string, _ *time.Time) (int, error) {
	return m.unreadCount, m.unreadErr
}
func (m *stubMsgReadService) LastByChatKeys(_ context.Context, chatKeys []string, _ map[string]*time.Time) (map[string]privatemessages.PrivateMessageItem, error) {
	if m.lastErr != nil {
		return nil, m.lastErr
	}
	result := make(map[string]privatemessages.PrivateMessageItem, len(chatKeys))
	if m.lastResult != nil {
		for _, key := range chatKeys {
			result[key] = *m.lastResult
		}
	}
	return result, nil
}
func (m *stubMsgReadService) UnreadCountsByChatKeys(_ context.Context, chatKeys []string, _ map[string]*time.Time) (map[string]int, error) {
	if m.unreadErr != nil {
		return nil, m.unreadErr
	}
	result := make(map[string]int, len(chatKeys))
	if m.unreadCount > 0 {
		for _, key := range chatKeys {
			result[key] = m.unreadCount
		}
	}
	return result, nil
}
func (m *stubMsgReadService) IsReactionTargetVisible(_ context.Context, _ string) (bool, bool, error) {
	return false, false, nil
}

func (m *stubMsgReadService) LookupMessage(_ context.Context, _ string) (bool, string, error) {
	return false, "", nil
}

func (m *stubMsgReadService) GetMessageReportSnapshot(_ context.Context, _ string) (privatemessages.MessageReportSnapshot, bool, error) {
	return privatemessages.MessageReportSnapshot{}, false, nil
}

type stubChatUserReader struct {
	item        users.UserItem
	found       bool
	userErr     error
	settings    users.UserSettings
	settingsErr error
}

func (u *stubChatUserReader) OpenTheDoor(_ context.Context, _ string) (users.UserItem, bool, error) {
	return u.item, u.found, u.userErr
}
func (u *stubChatUserReader) GetMySettings(_ context.Context) (users.UserSettings, error) {
	return u.settings, u.settingsErr
}
func (u *stubChatUserReader) BlockStateBetween(_ context.Context, _ string, _ string) (callerBlocked, targetBlocked bool, err error) {
	return false, false, nil
}

func (u *stubChatUserReader) ChatTargetsContext(_ context.Context, _ string, targetUserIDs []string) (map[string]users.ChatTargetContext, error) {
	if u.userErr != nil {
		return nil, u.userErr
	}
	result := make(map[string]users.ChatTargetContext, len(targetUserIDs))
	for _, id := range targetUserIDs {
		if u.found {
			result[id] = users.ChatTargetContext{Info: u.item.User, PresenceStatus: u.item.PresenceStatus, Relation: users.Relationship{}, AllowPrivateChat: u.settings.AllowPrivateChat, SettingsFound: true}
		}
	}
	return result, nil
}

func chatCtx(userID string) context.Context {
	return platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: userID})
}

func defaultUserItem() users.UserItem {
	return users.UserItem{User: users.UserInfo{UserID: "u2", Nickname: "bob"}}
}

func defaultSettings() users.UserSettings {
	return users.UserSettings{AllowPrivateChat: users.ChatPermEveryone}
}

func TestServiceOpenTheDoorCreatesChat(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubChatRepo{createdAt: now, state: ChatState{TargetID: "u2", CreatedAt: now}}
	msgs := &stubMsgReadService{}
	usr := &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	svc := NewServiceWithDeps(repo, msgs, usr)
	result, err := svc.OpenTheDoor(chatCtx("u1"), "u2", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Chat.OtherUser.UserID != "u2" {
		t.Fatalf("expected u2, got %s", result.Chat.OtherUser.UserID)
	}
}

func TestServiceOpenTheDoorRejectsMissingTarget(t *testing.T) {
	repo := &stubChatRepo{}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &stubChatUserReader{found: false})
	_, err := svc.OpenTheDoor(chatCtx("u1"), "missing", 20)
	if !errors.Is(err, users.ErrUserNotFound) {
		t.Fatalf("expected users.ErrUserNotFound, got %v", err)
	}
}

func TestServiceOpenTheDoorPropagatesCreateError(t *testing.T) {
	repo := &stubChatRepo{createErr: errors.New("boom")}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &stubChatUserReader{})
	_, err := svc.OpenTheDoor(chatCtx("u1"), "u2", 20)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceOpenTheDoorPropagatesStateError(t *testing.T) {
	repo := &stubChatRepo{stateErr: errors.New("boom")}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &stubChatUserReader{})
	_, err := svc.OpenTheDoor(chatCtx("u1"), "u2", 20)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceOpenTheDoorPropagatesRecentError(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubChatRepo{createdAt: now, state: ChatState{TargetID: "u2", CreatedAt: now}}
	msgs := &stubMsgReadService{recentErr: errors.New("boom")}
	svc := NewServiceWithDeps(repo, msgs, &stubChatUserReader{})
	_, err := svc.OpenTheDoor(chatCtx("u1"), "u2", 20)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceOpenTheDoorRestoresDeletedChatWithNewMessages(t *testing.T) {
	now := time.Now().UTC()
	deletedAt := now.Add(-1 * time.Hour)
	repo := &stubChatRepo{
		createdAt: now,
		state:     ChatState{TargetID: "u2", CreatedAt: now, DeletedAt: &deletedAt},
	}
	msgs := &stubMsgReadService{
		recentResult: privatemessages.PaginatedMessages{Items: []privatemessages.PrivateMessageItem{{MessageID: "pm1"}}},
		lastResult:   &privatemessages.PrivateMessageItem{MessageID: "pm1", SentAt: now.Format(time.RFC3339Nano)},
	}
	usr := &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	svc := NewServiceWithDeps(repo, msgs, usr)
	result, err := svc.OpenTheDoor(chatCtx("u1"), "u2", 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Chat.LastMessage == nil {
		t.Fatal("expected last message after restore")
	}
}

func TestServiceOpenTheDoorRestoreError(t *testing.T) {
	now := time.Now().UTC()
	deletedAt := now.Add(-1 * time.Hour)
	repo := &stubChatRepo{
		createdAt:  now,
		state:      ChatState{TargetID: "u2", CreatedAt: now, DeletedAt: &deletedAt},
		restoreErr: errors.New("boom"),
	}
	msgs := &stubMsgReadService{
		recentResult: privatemessages.PaginatedMessages{Items: []privatemessages.PrivateMessageItem{{MessageID: "pm1"}}},
	}
	svc := NewServiceWithDeps(repo, msgs, &stubChatUserReader{})
	_, err := svc.OpenTheDoor(chatCtx("u1"), "u2", 20)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceListReturnsChats(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubChatRepo{states: []ChatState{
		{TargetID: "u2", CreatedAt: now},
		{TargetID: "u3", CreatedAt: now.Add(-1 * time.Hour)},
	}}
	msgs := &stubMsgReadService{}
	usr := &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	svc := NewServiceWithDeps(repo, msgs, usr)
	result, err := svc.List(chatCtx("u1"), MyPrivateChatsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
}

// spec api_spec_base.md：limit default=20、max=100。此前 List 直接采纳客户端 limit，
// limit=100000 会一次性返回整份会话列表（越过分页语义 + 超规格外泄）。
func TestServiceListClampsLimitToMaxPageSize(t *testing.T) {
	now := time.Now().UTC()
	states := make([]ChatState, 0, 150)
	for i := 0; i < 150; i++ {
		states = append(states, ChatState{TargetID: "u" + strconv.Itoa(i+2), CreatedAt: now.Add(-time.Duration(i) * time.Minute)})
	}
	repo := &stubChatRepo{states: states}
	msgs := &stubMsgReadService{}
	usr := &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	svc := NewServiceWithDeps(repo, msgs, usr)

	result, err := svc.List(chatCtx("u1"), MyPrivateChatsRequest{Pagination: &types.PaginationParams{Limit: 100000}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != types.MaxPageSize {
		t.Fatalf("expected limit clamped to %d, got %d", types.MaxPageSize, len(result.Items))
	}
	if result.NextCursor == nil {
		t.Fatal("expected next cursor after clamping")
	}

	// ≤0 走默认页
	def, err := svc.List(chatCtx("u1"), MyPrivateChatsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(def.Items) != types.DefaultPageSize {
		t.Fatalf("expected default page size %d, got %d", types.DefaultPageSize, len(def.Items))
	}
}

func TestServiceListFiltersEmptyDeleted(t *testing.T) {
	now := time.Now().UTC()
	deletedAt := now.Add(-1 * time.Hour)
	repo := &stubChatRepo{states: []ChatState{
		{TargetID: "u2", CreatedAt: now, DeletedAt: &deletedAt},
		{TargetID: "u3", CreatedAt: now},
	}}
	msgs := &stubMsgReadService{}
	usr := &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	svc := NewServiceWithDeps(repo, msgs, usr)
	result, err := svc.List(chatCtx("u1"), MyPrivateChatsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item (deleted with no messages filtered), got %d", len(result.Items))
	}
}

func TestServiceListPropagatesError(t *testing.T) {
	repo := &stubChatRepo{listErr: errors.New("boom")}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &stubChatUserReader{})
	_, err := svc.List(chatCtx("u1"), MyPrivateChatsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceUpdateSettingsHappyPath(t *testing.T) {
	repo := &stubChatRepo{}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &stubChatUserReader{found: true})
	err := svc.UpdateSettings(chatCtx("u1"), UpdatePrivateChatSettingsRequest{TargetUserID: "u2", IsMuted: boolPtr(true)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUpdateSettingsRejectsMissingTarget(t *testing.T) {
	repo := &stubChatRepo{}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &stubChatUserReader{found: false})
	err := svc.UpdateSettings(chatCtx("u1"), UpdatePrivateChatSettingsRequest{TargetUserID: "missing", IsMuted: boolPtr(true)})
	if !errors.Is(err, users.ErrUserNotFound) {
		t.Fatalf("expected users.ErrUserNotFound, got %v", err)
	}
}

func TestServiceUpdateSettingsPropagatesError(t *testing.T) {
	repo := &stubChatRepo{updateSetErr: errors.New("boom")}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &stubChatUserReader{found: true})
	err := svc.UpdateSettings(chatCtx("u1"), UpdatePrivateChatSettingsRequest{TargetUserID: "u2"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceDeleteHappyPath(t *testing.T) {
	repo := &stubChatRepo{}
	usr := &stubChatUserReader{item: defaultUserItem(), found: true}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, usr)
	err := svc.Delete(chatCtx("u1"), "u2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceDeletePropagatesError(t *testing.T) {
	repo := &stubChatRepo{deleteErr: errors.New("boom")}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &stubChatUserReader{})
	err := svc.Delete(chatCtx("u1"), "u2")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestServiceBuildChatRejectsMissingUser(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubChatRepo{createdAt: now, state: ChatState{TargetID: "missing", CreatedAt: now}}
	msgs := &stubMsgReadService{}
	usr := &stubChatUserReader{found: false}
	svc := NewServiceWithDeps(repo, msgs, usr)
	_, err := svc.OpenTheDoor(chatCtx("u1"), "missing", 20)
	if err == nil {
		t.Fatal("expected error for missing user")
	}
}

func boolPtr(b bool) *bool { return &b }

func TestServiceWithInMemoryChatRepo(t *testing.T) {
	memoryChat := NewInMemoryRepository()
	msgs := &stubMsgReadService{}
	usr := &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	svc := NewServiceWithDeps(memoryChat, msgs, usr)

	_, err := svc.OpenTheDoor(chatCtx("u1"), "u2", 20)
	if err != nil {
		t.Fatalf("OpenTheDoor with in-memory repo failed: %v", err)
	}
}

func TestServiceDeleteWithInMemoryRepo(t *testing.T) {
	memoryChat := NewInMemoryRepository()
	msgs := &stubMsgReadService{}
	usr := &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	svc := NewServiceWithDeps(memoryChat, msgs, usr)

	err := svc.Delete(chatCtx("u1"), "u2")
	if err != nil {
		t.Fatalf("Delete with in-memory repo failed: %v", err)
	}
}

func TestServiceUpdateSettingsWithInMemoryRepo(t *testing.T) {
	memoryChat := NewInMemoryRepository()
	msgs := &stubMsgReadService{}
	usr := &stubChatUserReader{item: defaultUserItem(), found: true, settings: defaultSettings()}
	svc := NewServiceWithDeps(memoryChat, msgs, usr)

	err := svc.UpdateSettings(chatCtx("u1"), UpdatePrivateChatSettingsRequest{TargetUserID: "u2", IsMuted: boolPtr(true)})
	if err != nil {
		t.Fatalf("UpdateSettings with in-memory repo failed: %v", err)
	}
}

func (u *stubChatUserReader) ChatSettingsFor(_ context.Context, _ string) (string, bool, error) {
	return string(u.settings.AllowPrivateChat), u.found, u.userErr
}

// echoChatTargets 让每个会话目标解析为自身（stubChatUserReader 对全部目标
// 返回同一用户，无法构造多用户分页场景）。
type echoChatTargets struct{ stubChatUserReader }

func (u *echoChatTargets) ChatTargetsContext(_ context.Context, _ string, targetUserIDs []string) (map[string]users.ChatTargetContext, error) {
	result := make(map[string]users.ChatTargetContext, len(targetUserIDs))
	for _, id := range targetUserIDs {
		result[id] = users.ChatTargetContext{
			Info:             users.UserInfo{UserID: id, Nickname: "用户" + id},
			AllowPrivateChat: users.ChatPermEveryone,
			SettingsFound:    true,
		}
	}
	return result, nil
}

// 回归：排序为「置顶优先 → 活跃时间倒序」，游标必须携带置顶维度。
// 旧实现页边界落在置顶会话（活跃时间早于非置顶会话）时，下一页把更新的
// 非置顶会话整体过滤掉，永远不可达。
func TestServiceListPaginationAcrossPinnedBoundary(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubChatRepo{states: []ChatState{
		{TargetID: "u2", CreatedAt: now, IsPinned: true},
		{TargetID: "u3", CreatedAt: now.Add(-2 * time.Hour), IsPinned: true},
		{TargetID: "u4", CreatedAt: now.Add(-1 * time.Hour)},
	}}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &echoChatTargets{stubChatUserReader{found: true}})

	seen := make([]string, 0, 3)
	visited := make(map[string]bool, 3)
	cursor := ""
	for page := 0; page < 10; page++ {
		result, err := svc.List(chatCtx("u1"), MyPrivateChatsRequest{Pagination: &PaginationParams{Limit: 1, Cursor: cursor}})
		if err != nil {
			t.Fatalf("page %d: unexpected error: %v", page, err)
		}
		if len(result.Items) != 1 {
			t.Fatalf("page %d: expected 1 item, got %d", page, len(result.Items))
		}
		id := result.Items[0].OtherUser.UserID
		if visited[id] {
			t.Fatalf("page %d: user %s visited twice (infinite loop)", page, id)
		}
		visited[id] = true
		seen = append(seen, id)
		if result.NextCursor == nil {
			break
		}
		cursor = *result.NextCursor
	}
	if len(seen) != 3 {
		t.Fatalf("expected all 3 chats reachable across pages, got %v", seen)
	}
	if seen[0] != "u2" || seen[1] != "u3" || seen[2] != "u4" {
		t.Fatalf("expected pinned-first order [u2 u3 u4], got %v", seen)
	}
}
