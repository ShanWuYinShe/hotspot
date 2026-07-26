package privatechats

import (
	"context"
	"errors"
	"testing"
	"time"

	private_messages "hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
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
func (r *stubChatRepo) List(_ context.Context, _ string, _ ListOptions) ([]ChatState, string, error) {
	return r.states, r.listCursor, r.listErr
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
	recentResult private_messages.PaginatedMessages
	recentErr    error
	lastResult   *private_messages.PrivateMessageItem
	lastErr      error
	unreadCount  int
	unreadErr    error
}

func (m *stubMsgReadService) Recent(_ context.Context, _ string, _ int, _ *time.Time) (private_messages.PaginatedMessages, error) {
	return m.recentResult, m.recentErr
}
func (m *stubMsgReadService) Last(_ context.Context, _ string, _ *time.Time) (*private_messages.PrivateMessageItem, error) {
	return m.lastResult, m.lastErr
}
func (m *stubMsgReadService) UnreadCount(_ context.Context, _ string, _ *time.Time) (int, error) {
	return m.unreadCount, m.unreadErr
}
func (m *stubMsgReadService) LookupMessage(_ context.Context, _ string) (bool, string, error) { return false, "", nil }

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
		recentResult: private_messages.PaginatedMessages{Items: []private_messages.PrivateMessageItem{{MessageID: "pm1"}}},
		lastResult:   &private_messages.PrivateMessageItem{MessageID: "pm1", SentAt: now.Format(time.RFC3339Nano)},
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
		recentResult: private_messages.PaginatedMessages{Items: []private_messages.PrivateMessageItem{{MessageID: "pm1"}}},
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
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &stubChatUserReader{})
	err := svc.UpdateSettings(chatCtx("u1"), UpdatePrivateChatSettingsRequest{TargetUserID: "u2", IsMuted: boolPtr(true)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServiceUpdateSettingsPropagatesError(t *testing.T) {
	repo := &stubChatRepo{updateSetErr: errors.New("boom")}
	svc := NewServiceWithDeps(repo, &stubMsgReadService{}, &stubChatUserReader{})
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
