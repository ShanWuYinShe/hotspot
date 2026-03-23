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

	private_messages "hotspot/backend-go/internal/modules/privatemessages"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
)

type privateChatErrorRepo struct {
	getOrCreateErr error
	listErr        error
}

func (r *privateChatErrorRepo) GetOrCreate(context.Context, string, string) (time.Time, error) {
	return time.Time{}, r.getOrCreateErr
}
func (r *privateChatErrorRepo) List(context.Context, string) (map[string]time.Time, error) {
	return nil, r.listErr
}

type privateChatMessageReader struct {
	recentErr error
	lastErr   error
}

func (r *privateChatMessageReader) Recent(context.Context, string, int) (private_messages.PaginatedMessages, error) {
	return private_messages.PaginatedMessages{}, r.recentErr
}

func (r *privateChatMessageReader) Last(context.Context, string) (*private_messages.PrivateMessageItem, error) {
	return nil, r.lastErr
}

func newHandlers() (*Handler, *private_messages.Handler) {
	messageRuntime := private_messages.NewRuntimeWithRepo(private_messages.NewInMemoryRepository())
	chatHandler := NewHandler(NewServiceWithRepos(NewInMemoryRepository(), messageRuntime.ReadService()), appconfig.NewStaticSource(config.Load()))
	return chatHandler, messageRuntime.Handler()
}

func TestOpenTheDoorReturnsChatAndRecentMessages(t *testing.T) {
	chatHandler, _ := newHandlers()
	req := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"user_bob"}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	chatHandler.OpenTheDoor(rr, req)

	var resp struct {
		Code int                   `json:"code"`
		Data PrivateChatOpenResult `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data.Chat.OtherUser.UserID != "user_bob" {
		t.Fatalf("unexpected response: %#v", resp)
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

	historyReq := httptest.NewRequest(http.MethodPost, "/get_history", strings.NewReader(`{"target_user_id":"user_bob"}`))
	historyReq = historyReq.WithContext(ctx)
	historyRR := httptest.NewRecorder()
	messageHandler.GetHistory(historyRR, historyReq)

	var historyResp struct {
		Code int                                `json:"code"`
		Data private_messages.PaginatedMessages `json:"data"`
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
		Code int                   `json:"code"`
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
		NewServiceWithRepos(
			&privateChatErrorRepo{getOrCreateErr: errors.New("boom"), listErr: errors.New("boom")},
			&privateChatMessageReader{recentErr: errors.New("boom"), lastErr: errors.New("boom")},
		),
		appconfig.NewStaticSource(config.Load()),
	)

	listReq := httptest.NewRequest(http.MethodPost, "/me/list", strings.NewReader(`{}`))
	listReq = listReq.WithContext(platformauth.WithPrincipal(listReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	listRR := httptest.NewRecorder()
	chatHandler.List(listRR, listReq)
	assertPrivateChatCode(t, listRR, 10600)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"target_user_id":"user_bob"}`))
	openReq = openReq.WithContext(platformauth.WithPrincipal(openReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	openRR := httptest.NewRecorder()
	chatHandler.OpenTheDoor(openRR, openReq)
	assertPrivateChatCode(t, openRR, 10600)
}

func assertPrivateChatCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
