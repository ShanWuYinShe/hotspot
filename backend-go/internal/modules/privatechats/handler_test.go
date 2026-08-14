package privatechats

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

type privateChatErrorRepo struct {
	getOrCreateErr error
	listErr        error
	stateErr       error
	restoreErr     error
	updateErr      error
	deleteErr      error
}

func (r *privateChatErrorRepo) GetOrCreate(context.Context, string, string) (time.Time, error) {
	return time.Time{}, r.getOrCreateErr
}
func (r *privateChatErrorRepo) List(context.Context, string, ListOptions) ([]ChatState, string, error) {
	return nil, "", r.listErr
}
func (r *privateChatErrorRepo) GetState(context.Context, string, string) (ChatState, error) {
	return ChatState{}, r.stateErr
}
func (r *privateChatErrorRepo) RestoreForUser(context.Context, string, string, time.Time) error {
	return r.restoreErr
}
func (r *privateChatErrorRepo) UpdateSettings(context.Context, string, string, UpdateChatSettings) error {
	return r.updateErr
}
func (r *privateChatErrorRepo) DeleteForUser(context.Context, string, string, *time.Time) error {
	return r.deleteErr
}

type privateChatMessageReader struct {
	recentErr   error
	lastErr     error
	unreadErr   error
	recent      privatemessages.PaginatedMessages
	last        *privatemessages.PrivateMessageItem
	unreadCount int
}

func (r *privateChatMessageReader) Recent(context.Context, string, int, *time.Time) (privatemessages.PaginatedMessages, error) {
	if r.recentErr != nil {
		return privatemessages.PaginatedMessages{}, r.recentErr
	}
	return r.recent, nil
}

func (r *privateChatMessageReader) Last(context.Context, string, *time.Time) (*privatemessages.PrivateMessageItem, error) {
	if r.lastErr != nil {
		return nil, r.lastErr
	}
	return r.last, nil
}

func (r *privateChatMessageReader) UnreadCount(context.Context, string, *time.Time) (int, error) {
	if r.unreadErr != nil {
		return 0, r.unreadErr
	}
	return r.unreadCount, nil
}
func (r *privateChatMessageReader) IsReactionTargetVisible(_ context.Context, _ string) (bool, bool, error) {
	return false, false, nil
}

func (r *privateChatMessageReader) LookupMessage(_ context.Context, _ string) (bool, string, error) {
	return false, "", nil
}

type privateChatUserReader struct {
	item     users.UserItem
	ok       bool
	err      error
	settings users.UserSettings
}

func (r *privateChatUserReader) OpenTheDoor(context.Context, string) (users.UserItem, bool, error) {
	return r.item, r.ok, r.err
}

func (r *privateChatUserReader) GetMySettings(context.Context) (users.UserSettings, error) {
	return r.settings, nil
}

func newHandlers() (*Handler, *privatemessages.Handler) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	userRepo := users.NewInMemoryRepository()
	seedPrivateChatUsers(userRepo)
	return newHandlersWithUserRepo(userRepo)
}

func newHandlersWithUserRepo(userRepo *users.InMemoryRepository) (*Handler, *privatemessages.Handler) {
	messageRepo := privatemessages.NewInMemoryRepository()
	userService := users.NewServiceWithRepo(userRepo)
	messageService := privatemessages.NewServiceWithDeps(messageRepo, userService)
	messageHandler := privatemessages.NewHandler(messageService)
	chatHandler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), messageService, userService), appconfig.NewStaticSource(config.Load()))
	return chatHandler, messageHandler
}

func seedPrivateChatUsers(userRepo *users.InMemoryRepository) {
	now := time.Now().UTC().Format(time.RFC3339)
	_ = userRepo.PutProfile(context.Background(), users.MyProfile{UserID: "user_bob", Nickname: "Bob", UserType: users.UserTypeNormal, Location: users.Location{LocationName: "Somewhere"}, PresenceStatus: users.UserPresenceOnline, LastOnline: now, CreatedAt: now})
	_ = userRepo.PutSettings(context.Background(), "user_bob", users.UserSettings{AllowPrivateChat: users.ChatPermEveryone, PresenceVisible: true, LocationVisible: true, SocialListsVisible: true, GroupsVisible: true, ReactedMomentsVisible: true, StatsVisible: true, AllowFollow: true, Searchable: true, NicknameSearchable: true, LocationSearchable: true})
	_ = userRepo.PutProfile(context.Background(), users.MyProfile{UserID: "user_alice", Nickname: "Alice", UserType: users.UserTypeNormal, Location: users.Location{LocationName: "Somewhere"}, PresenceStatus: users.UserPresenceOnline, LastOnline: now, CreatedAt: now})
	_ = userRepo.PutSettings(context.Background(), "user_alice", users.UserSettings{AllowPrivateChat: users.ChatPermEveryone, PresenceVisible: true, LocationVisible: true, SocialListsVisible: true, GroupsVisible: true, ReactedMomentsVisible: true, StatsVisible: true, AllowFollow: true, Searchable: true, NicknameSearchable: true, LocationSearchable: true})
	_ = userRepo.PutProfile(context.Background(), users.MyProfile{UserID: "user_charlie", Nickname: "Charlie", UserType: users.UserTypeNormal, Location: users.Location{LocationName: "Somewhere"}, PresenceStatus: users.UserPresenceOnline, LastOnline: now, CreatedAt: now})
	_ = userRepo.PutSettings(context.Background(), "user_charlie", users.UserSettings{AllowPrivateChat: users.ChatPermEveryone, PresenceVisible: true, LocationVisible: true, SocialListsVisible: true, GroupsVisible: true, ReactedMomentsVisible: true, StatsVisible: true, AllowFollow: true, Searchable: true, NicknameSearchable: true, LocationSearchable: true})
}

func TestOpenTheDoorReturnsChatAndRecentMessages(t *testing.T) {
	chatHandler, _ := newHandlers()
	req := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"user_bob"}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	chatHandler.OpenTheDoor(rr, req)

	var resp struct {
		Code httpx.Code            `json:"code"`
		Data PrivateChatOpenResult `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data.Chat.OtherUser.UserID != "user_bob" {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.Data.Chat.OtherUser.Nickname != "Bob" {
		t.Fatalf("expected real nickname, got %#v", resp.Data.Chat)
	}
}

func TestSendAndHistoryWorkTogether(t *testing.T) {
	chatHandler, messageHandler := newHandlers()
	sendReq := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`))
	sendReq.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	ctx := platformauth.WithPrincipal(sendReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	sendReq = sendReq.WithContext(ctx)
	sendRR := httptest.NewRecorder()
	messageHandler.Send(sendRR, sendReq)

	historyReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"target_user_id":"user_bob"}`))
	historyReq = historyReq.WithContext(ctx)
	historyRR := httptest.NewRecorder()
	messageHandler.List(historyRR, historyReq)

	var historyResp struct {
		Code httpx.Code                        `json:"code"`
		Data privatemessages.PaginatedMessages `json:"data"`
	}
	if err := json.NewDecoder(historyRR.Body).Decode(&historyResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if historyResp.Code != 0 || len(historyResp.Data.Items) != 1 || historyResp.Data.Items[0].Content != "hello" {
		t.Fatalf("unexpected history: %#v", historyResp)
	}

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"user_bob"}`))
	openReq = openReq.WithContext(ctx)
	openRR := httptest.NewRecorder()
	chatHandler.OpenTheDoor(openRR, openReq)
	var openResp struct {
		Code httpx.Code            `json:"code"`
		Data PrivateChatOpenResult `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if openResp.Data.Chat.LastMessage == nil || openResp.Data.Chat.LastMessage.Content != "hello" {
		t.Fatalf("unexpected chat open response: %#v", openResp)
	}
}

func TestPrivateChatsHandlerReturns10600OnServiceError(t *testing.T) {
	chatHandler := NewHandler(
		NewServiceWithDeps(
			&privateChatErrorRepo{getOrCreateErr: errors.New("boom"), listErr: errors.New("boom"), stateErr: errors.New("boom"), updateErr: errors.New("boom"), deleteErr: errors.New("boom")},
			&privateChatMessageReader{recentErr: errors.New("boom"), lastErr: errors.New("boom"), unreadErr: errors.New("boom")},
			&privateChatUserReader{err: errors.New("boom")},
		),
		appconfig.NewStaticSource(config.Load()),
	)

	listReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"pagination":{"limit":20}}`))
	listReq = listReq.WithContext(platformauth.WithPrincipal(listReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	listRR := httptest.NewRecorder()
	chatHandler.List(listRR, listReq)
	assertPrivateChatCode(t, listRR, httpx.CodeInternalUnavailable)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"user_bob"}`))
	openReq = openReq.WithContext(platformauth.WithPrincipal(openReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	openRR := httptest.NewRecorder()
	chatHandler.OpenTheDoor(openRR, openReq)
	assertPrivateChatCode(t, openRR, httpx.CodeInternalUnavailable)
}

func TestOpenTheDoorUsesSharedResponseDefaults(t *testing.T) {
	source := appconfig.NewStaticSource(config.Load())
	_, err := source.Update(context.Background(), func(current appconfig.AppConfig) appconfig.AppConfig {
		current.ResponseDefaults.ChatRecentMessagesCount = 1
		return current
	})
	if err != nil {
		t.Fatalf("update shared source: %v", err)
	}

	messageRepo := privatemessages.NewInMemoryRepository()
	userRepo := users.NewInMemoryRepository()
	_ = userRepo.PutProfile(context.Background(), users.MyProfile{UserID: "user_bob", Nickname: "Bob", UserType: users.UserTypeNormal, Location: users.Location{LocationName: "Somewhere"}, PresenceStatus: users.UserPresenceOnline, LastOnline: time.Now().UTC().Format(time.RFC3339), CreatedAt: time.Now().UTC().Format(time.RFC3339)})
	_ = userRepo.PutSettings(context.Background(), "user_bob", users.UserSettings{AllowPrivateChat: users.ChatPermEveryone, PresenceVisible: true, LocationVisible: true, SocialListsVisible: true, GroupsVisible: true, ReactedMomentsVisible: true, StatsVisible: true, AllowFollow: true, Searchable: true, NicknameSearchable: true, LocationSearchable: true})
	userService := users.NewServiceWithRepo(userRepo)
	messageService := privatemessages.NewServiceWithDeps(messageRepo, userService)
	chatHandler := NewHandler(NewServiceWithDeps(NewInMemoryRepository(), messageService, userService), source)
	messageHandler := privatemessages.NewHandler(messageService)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	for _, body := range []string{
		`{"target_user_id":"user_bob","type":"text","content":"one"}`,
		`{"target_user_id":"user_bob","type":"text","content":"two"}`,
	} {
		sendReq := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(body))
		sendReq = sendReq.WithContext(ctx)
		sendReq.Header.Set("Idempotency-Key", "shared1234abcd1234abcd1234abcd1234")
		if strings.Contains(body, "two") {
			sendReq.Header.Set("Idempotency-Key", "shared5678abcd1234abcd1234abcd1234")
		}
		sendRR := httptest.NewRecorder()
		messageHandler.Send(sendRR, sendReq)
		var sendResp struct {
			Code httpx.Code `json:"code"`
		}
		if err := json.NewDecoder(sendRR.Body).Decode(&sendResp); err != nil {
			t.Fatalf("decode send response: %v", err)
		}
		if sendResp.Code != 0 {
			t.Fatalf("unexpected send response: %#v", sendResp)
		}
	}

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"user_bob"}`))
	openReq = openReq.WithContext(ctx)
	openRR := httptest.NewRecorder()
	chatHandler.OpenTheDoor(openRR, openReq)

	var openResp struct {
		Code httpx.Code            `json:"code"`
		Data PrivateChatOpenResult `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResp.Code != 0 {
		t.Fatalf("unexpected open response: %#v", openResp)
	}
	if len(openResp.Data.RecentMessages.Items) != 1 {
		t.Fatalf("expected shared recent message limit to apply: %#v", openResp.Data.RecentMessages)
	}
}

func TestUpdateSettingsAndDeleteHandlers(t *testing.T) {
	chatHandler, messageHandler := newHandlers()
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	sendReq := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`))
	sendReq = sendReq.WithContext(ctx)
	sendReq.Header.Set("Idempotency-Key", "11111111111111111111111111111111")
	sendRR := httptest.NewRecorder()
	messageHandler.Send(sendRR, sendReq)
	assertPrivateChatCode(t, sendRR, 0)

	updateReq := httptest.NewRequest(http.MethodPost, "/update_preferences", strings.NewReader(`{"target_user_id":"user_bob","is_muted":true,"is_pinned":true}`))
	updateReq = updateReq.WithContext(ctx)
	updateRR := httptest.NewRecorder()
	chatHandler.UpdateSettings(updateRR, updateReq)
	assertPrivateChatCode(t, updateRR, 0)

	listReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"pagination":{"limit":20}}`))
	listReq = listReq.WithContext(ctx)
	listRR := httptest.NewRecorder()
	chatHandler.List(listRR, listReq)
	var listResp struct {
		Code httpx.Code            `json:"code"`
		Data PaginatedPrivateChats `json:"data"`
	}
	if err := json.NewDecoder(listRR.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listResp.Code != 0 || len(listResp.Data.Items) != 1 {
		t.Fatalf("unexpected list response: %#v", listResp)
	}
	if !listResp.Data.Items[0].IsMuted || !listResp.Data.Items[0].IsPinned {
		t.Fatalf("expected updated settings in list: %#v", listResp.Data.Items[0])
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"target_user_id":"user_bob"}`))
	deleteReq = deleteReq.WithContext(ctx)
	deleteRR := httptest.NewRecorder()
	chatHandler.Delete(deleteRR, deleteReq)
	assertPrivateChatCode(t, deleteRR, 0)

	listAfterDeleteReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"pagination":{"limit":20}}`))
	listAfterDeleteReq = listAfterDeleteReq.WithContext(ctx)
	listAfterDeleteRR := httptest.NewRecorder()
	chatHandler.List(listAfterDeleteRR, listAfterDeleteReq)
	var listAfterDeleteResp struct {
		Code httpx.Code            `json:"code"`
		Data PaginatedPrivateChats `json:"data"`
	}
	if err := json.NewDecoder(listAfterDeleteRR.Body).Decode(&listAfterDeleteResp); err != nil {
		t.Fatalf("decode list after delete response: %v", err)
	}
	if listAfterDeleteResp.Code != 0 {
		t.Fatalf("unexpected list after delete response: %#v", listAfterDeleteResp)
	}
	if len(listAfterDeleteResp.Data.Items) != 0 {
		t.Fatalf("expected deleted chat hidden from list: %#v", listAfterDeleteResp.Data.Items)
	}
}

func TestDeleteOnlyAffectsCurrentUserView(t *testing.T) {
	chatHandler, messageHandler := newHandlers()
	aliceCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	bobCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_bob"})

	sendReq := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`))
	sendReq = sendReq.WithContext(aliceCtx)
	sendReq.Header.Set("Idempotency-Key", "22222222222222222222222222222222")
	sendRR := httptest.NewRecorder()
	messageHandler.Send(sendRR, sendReq)
	assertPrivateChatCode(t, sendRR, 0)

	deleteReq := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"target_user_id":"user_bob"}`))
	deleteReq = deleteReq.WithContext(aliceCtx)
	deleteRR := httptest.NewRecorder()
	chatHandler.Delete(deleteRR, deleteReq)
	assertPrivateChatCode(t, deleteRR, 0)

	aliceListReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"pagination":{"limit":20}}`))
	aliceListReq = aliceListReq.WithContext(aliceCtx)
	aliceListRR := httptest.NewRecorder()
	chatHandler.List(aliceListRR, aliceListReq)
	var aliceResp struct {
		Code httpx.Code            `json:"code"`
		Data PaginatedPrivateChats `json:"data"`
	}
	if err := json.NewDecoder(aliceListRR.Body).Decode(&aliceResp); err != nil {
		t.Fatalf("decode alice list response: %v", err)
	}
	if len(aliceResp.Data.Items) != 0 {
		t.Fatalf("expected alice chat list empty after delete: %#v", aliceResp.Data.Items)
	}

	bobListReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"pagination":{"limit":20}}`))
	bobListReq = bobListReq.WithContext(bobCtx)
	bobListRR := httptest.NewRecorder()
	chatHandler.List(bobListRR, bobListReq)
	var bobResp struct {
		Code httpx.Code            `json:"code"`
		Data PaginatedPrivateChats `json:"data"`
	}
	if err := json.NewDecoder(bobListRR.Body).Decode(&bobResp); err != nil {
		t.Fatalf("decode bob list response: %v", err)
	}
	if bobResp.Code != 0 || len(bobResp.Data.Items) != 1 {
		t.Fatalf("expected bob still sees chat: %#v", bobResp)
	}
	if bobResp.Data.Items[0].LastMessage == nil || bobResp.Data.Items[0].LastMessage.Content != "hello" {
		t.Fatalf("expected bob still sees original message: %#v", bobResp.Data.Items[0])
	}
}

func TestDeleteThenNewMessageReappearsAsBlankConversation(t *testing.T) {
	chatHandler, messageHandler := newHandlers()
	aliceCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})
	bobCtx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_bob", SessionID: "sess_bob"})

	firstSendReq := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"old"}`))
	firstSendReq = firstSendReq.WithContext(aliceCtx)
	firstSendReq.Header.Set("Idempotency-Key", "33333333333333333333333333333333")
	firstSendRR := httptest.NewRecorder()
	messageHandler.Send(firstSendRR, firstSendReq)
	assertPrivateChatCode(t, firstSendRR, 0)

	deleteReq := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"target_user_id":"user_bob"}`))
	deleteReq = deleteReq.WithContext(aliceCtx)
	deleteRR := httptest.NewRecorder()
	chatHandler.Delete(deleteRR, deleteReq)
	assertPrivateChatCode(t, deleteRR, 0)

	secondSendReq := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_alice","type":"text","content":"new"}`))
	secondSendReq = secondSendReq.WithContext(bobCtx)
	secondSendReq.Header.Set("Idempotency-Key", "44444444444444444444444444444444")
	secondSendRR := httptest.NewRecorder()
	messageHandler.Send(secondSendRR, secondSendReq)
	assertPrivateChatCode(t, secondSendRR, 0)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"user_bob"}`))
	openReq = openReq.WithContext(aliceCtx)
	openRR := httptest.NewRecorder()
	chatHandler.OpenTheDoor(openRR, openReq)
	var openResp struct {
		Code httpx.Code            `json:"code"`
		Data PrivateChatOpenResult `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResp.Code != 0 {
		t.Fatalf("unexpected open response: %#v", openResp)
	}
	if openResp.Data.Chat.LastMessage == nil || openResp.Data.Chat.LastMessage.Content != "new" {
		t.Fatalf("expected only new last message visible: %#v", openResp.Data.Chat)
	}
	if len(openResp.Data.RecentMessages.Items) != 1 || openResp.Data.RecentMessages.Items[0].Content != "new" {
		t.Fatalf("expected only new recent message visible: %#v", openResp.Data.RecentMessages)
	}
}

func TestOpenTheDoorRespectsTargetPrivacy(t *testing.T) {
	userRepo := users.NewInMemoryRepository()
	seedPrivateChatUsers(userRepo)
	_ = userRepo.PutSettings(context.Background(), "user_bob", users.UserSettings{AllowPrivateChat: users.ChatPermNobody, PresenceVisible: true, LocationVisible: true, SocialListsVisible: true, GroupsVisible: true, ReactedMomentsVisible: true, StatsVisible: true, AllowFollow: true, Searchable: true, NicknameSearchable: true, LocationSearchable: true})
	chatHandler, _ := newHandlersWithUserRepo(userRepo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"user_bob"}`))
	openReq = openReq.WithContext(ctx)
	openRR := httptest.NewRecorder()
	chatHandler.OpenTheDoor(openRR, openReq)

	var openResp struct {
		Code httpx.Code            `json:"code"`
		Data PrivateChatOpenResult `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResp.Code != 0 {
		t.Fatalf("unexpected open response: %#v", openResp)
	}
	if openResp.Data.Chat.CanMessage {
		t.Fatalf("expected can_message=false: %#v", openResp.Data.Chat)
	}
	if openResp.Data.Chat.ForbiddenReason == nil || *openResp.Data.Chat.ForbiddenReason != "privacy_setting" {
		t.Fatalf("expected privacy_setting blocked reason: %#v", openResp.Data.Chat)
	}
}

func TestDeleteThenOpenTheDoorCreatesBlankConversation(t *testing.T) {
	chatHandler, messageHandler := newHandlers()
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})

	sendReq := httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"old"}`))
	sendReq = sendReq.WithContext(ctx)
	sendReq.Header.Set("Idempotency-Key", "55555555555555555555555555555555")
	sendRR := httptest.NewRecorder()
	messageHandler.Send(sendRR, sendReq)
	assertPrivateChatCode(t, sendRR, 0)

	deleteReq := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"target_user_id":"user_bob"}`))
	deleteReq = deleteReq.WithContext(ctx)
	deleteRR := httptest.NewRecorder()
	chatHandler.Delete(deleteRR, deleteReq)
	assertPrivateChatCode(t, deleteRR, 0)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"user_bob"}`))
	openReq = openReq.WithContext(ctx)
	openRR := httptest.NewRecorder()
	chatHandler.OpenTheDoor(openRR, openReq)

	var openResp struct {
		Code httpx.Code            `json:"code"`
		Data PrivateChatOpenResult `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResp.Code != 0 {
		t.Fatalf("unexpected open response: %#v", openResp)
	}
	if openResp.Data.Chat.LastMessage != nil {
		t.Fatalf("expected blank conversation after reopen: %#v", openResp.Data.Chat)
	}
	if len(openResp.Data.RecentMessages.Items) != 0 {
		t.Fatalf("expected no historical messages after reopen: %#v", openResp.Data.RecentMessages)
	}
	if openResp.Data.Chat.CreatedAt == "" {
		t.Fatalf("expected created_at on reopened blank conversation: %#v", openResp.Data.Chat)
	}
	listReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"pagination":{"limit":20}}`))
	listReq = listReq.WithContext(ctx)
	listRR := httptest.NewRecorder()
	chatHandler.List(listRR, listReq)
	var listResp struct {
		Code httpx.Code            `json:"code"`
		Data PaginatedPrivateChats `json:"data"`
	}
	if err := json.NewDecoder(listRR.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listResp.Code != 0 || len(listResp.Data.Items) != 1 {
		t.Fatalf("expected reopened conversation visible in list: %#v", listResp)
	}
	if listResp.Data.Items[0].LastMessage != nil {
		t.Fatalf("expected reopened list item without last_message: %#v", listResp.Data.Items[0])
	}
}

func TestListSupportsPaginationCursor(t *testing.T) {
	chatHandler, _ := newHandlers()
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_alice"})

	for _, target := range []string{"user_bob", "user_charlie"} {
		openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"`+target+`"}`))
		openReq = openReq.WithContext(ctx)
		openRR := httptest.NewRecorder()
		chatHandler.OpenTheDoor(openRR, openReq)
		assertPrivateChatCode(t, openRR, 0)
	}

	firstReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"pagination":{"limit":1}}`))
	firstReq = firstReq.WithContext(ctx)
	firstRR := httptest.NewRecorder()
	chatHandler.List(firstRR, firstReq)
	var firstResp struct {
		Code httpx.Code            `json:"code"`
		Data PaginatedPrivateChats `json:"data"`
	}
	if err := json.NewDecoder(firstRR.Body).Decode(&firstResp); err != nil {
		t.Fatalf("decode first list response: %v", err)
	}
	if firstResp.Code != 0 || len(firstResp.Data.Items) != 1 {
		t.Fatalf("unexpected first list response: %#v", firstResp)
	}
	if firstResp.Data.NextCursor == "" {
		t.Fatalf("expected next_cursor on first page: %#v", firstResp)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"pagination":{"limit":1,"cursor":"`+firstResp.Data.NextCursor+`"}}`))
	secondReq = secondReq.WithContext(ctx)
	secondRR := httptest.NewRecorder()
	chatHandler.List(secondRR, secondReq)
	var secondResp struct {
		Code httpx.Code            `json:"code"`
		Data PaginatedPrivateChats `json:"data"`
	}
	if err := json.NewDecoder(secondRR.Body).Decode(&secondResp); err != nil {
		t.Fatalf("decode second list response: %v", err)
	}
	if secondResp.Code != 0 || len(secondResp.Data.Items) != 1 {
		t.Fatalf("unexpected second list response: %#v", secondResp)
	}
	if secondResp.Data.Items[0].OtherUser.UserID == firstResp.Data.Items[0].OtherUser.UserID {
		t.Fatalf("expected second page to advance cursor: first=%#v second=%#v", firstResp.Data.Items[0], secondResp.Data.Items[0])
	}
}

func TestOpenTheDoorReturns10300OnNotFound(t *testing.T) {
	notFoundUserReader := &privateChatUserReader{item: users.UserItem{}, ok: false, err: users.ErrUserNotFound}
	chatHandler := NewHandler(
		NewServiceWithDeps(
			NewInMemoryRepository(),
			&privateChatMessageReader{},
			notFoundUserReader,
		),
		appconfig.NewStaticSource(config.Load()),
	)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"nonexistent_user"}`))
	openReq = openReq.WithContext(ctx)
	openRR := httptest.NewRecorder()
	chatHandler.OpenTheDoor(openRR, openReq)
	assertPrivateChatCode(t, openRR, httpx.CodeNotFound)
}

func TestPrivateChatDeleteReturns10300OnNotFound(t *testing.T) {
	notFoundUserReader := &privateChatUserReader{item: users.UserItem{}, ok: false, err: users.ErrUserNotFound}
	chatHandler := NewHandler(
		NewServiceWithDeps(
			NewInMemoryRepository(),
			&privateChatMessageReader{},
			notFoundUserReader,
		),
		appconfig.NewStaticSource(config.Load()),
	)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	deleteReq := httptest.NewRequest(http.MethodPost, "/delete", strings.NewReader(`{"target_user_id":"nonexistent_user"}`))
	deleteReq = deleteReq.WithContext(ctx)
	deleteRR := httptest.NewRecorder()
	chatHandler.Delete(deleteRR, deleteReq)
	assertPrivateChatCode(t, deleteRR, httpx.CodeNotFound)
}

func assertPrivateChatCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
	t.Helper()
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
