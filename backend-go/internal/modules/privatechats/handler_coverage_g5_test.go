package privatechats

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
)

// TestCoverageG5HandlerValidation 覆盖四个 handler 的参数校验/解码错误分支：
// 统一返回 10000（CodeInvalidRequest）。
func TestCoverageG5HandlerValidation(t *testing.T) {
	h, _ := newHandlers()
	ctx := chatCtx("user_alice")
	cases := []struct {
		name string
		path string
		body string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"list_invalid_json", "/api/v1/private_chats/fetch", "not-json", h.List},
		{"list_invalid_cursor", "/api/v1/private_chats/fetch", `{"pagination":{"cursor":"garbage"}}`, h.List},
		{"open_the_door_invalid_json", "/api/v1/private_chats/open_the_door", "[]", h.OpenTheDoor},
		{"open_the_door_missing_target", "/api/v1/private_chats/open_the_door", "{}", h.OpenTheDoor},
		{"update_settings_invalid_json", "/api/v1/private_chats/update_preferences", `{"target_user_id":`, h.UpdateSettings},
		{"update_settings_missing_target", "/api/v1/private_chats/update_preferences", `{"is_muted":true}`, h.UpdateSettings},
		{"update_settings_missing_flags", "/api/v1/private_chats/update_preferences", `{"target_user_id":"user_bob"}`, h.UpdateSettings},
		{"delete_invalid_json", "/api/v1/private_chats/delete", `"x"`, h.Delete},
		{"delete_missing_target", "/api/v1/private_chats/delete", "{}", h.Delete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			tc.call(rr, req)
			assertPrivateChatCode(t, rr, httpx.CodeInvalidRequest)
		})
	}
}

// TestCoverageG5HandlerGenericServiceErrors 覆盖 handleServiceError 的非
// ErrUserNotFound 分支（10600），避免把内部错误映射成 10300。
func TestCoverageG5HandlerGenericServiceErrors(t *testing.T) {
	h := NewHandler(
		NewServiceWithDeps(
			NewInMemoryRepository(),
			&privateChatMessageReader{},
			&privateChatUserReader{err: errors.New("boom")},
		),
		appconfig.NewStaticSource(config.Load()),
	)
	ctx := chatCtx("user_alice")

	cases := []struct {
		name string
		path string
		body string
		call func(http.ResponseWriter, *http.Request)
	}{
		{"open_the_door", "/api/v1/private_chats/open_the_door", `{"target_user_id":"user_bob"}`, h.OpenTheDoor},
		{"update_settings", "/api/v1/private_chats/update_preferences", `{"target_user_id":"user_bob","is_muted":true}`, h.UpdateSettings},
		{"delete", "/api/v1/private_chats/delete", `{"target_user_id":"user_bob"}`, h.Delete},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()
			tc.call(rr, req)
			assertPrivateChatCode(t, rr, httpx.CodeInternalUnavailable)
		})
	}
}
