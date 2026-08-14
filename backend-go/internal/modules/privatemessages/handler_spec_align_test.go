package privatemessages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

// TestHandlerSendMapsForbiddenReasons verifies the spec api_spec_errors.md /
// api_spec_chat.md contract: private_messages/send 被拒绝时返回 code=httpx.CodeForbidden 且
// data 携带 { forbidden_reason: ForbiddenReason }（blocked_by_user / privacy_setting / stranger_quota_exceeded）。
func TestHandlerSendMapsForbiddenReasons(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		reason httpx.ForbiddenReason
	}{
		{name: "blocked by user", err: httpx.Forbidden(httpx.ReasonBlockedByUser, "已被对方拉黑"), reason: httpx.ReasonBlockedByUser},
		{name: "privacy", err: httpx.Forbidden(httpx.ReasonPrivacySetting, "对方隐私设置不允许"), reason: httpx.ReasonPrivacySetting},
		{name: "stranger quota", err: httpx.Forbidden(httpx.ReasonStrangerQuotaExceeded, "陌生人消息限额已用完"), reason: httpx.ReasonStrangerQuotaExceeded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idempotency.SetStore(idempotency.NewInMemoryStore())
			userStub := &privateMessageUserStub{
				item:     users.UserItem{Relation: &users.Relationship{}},
				ok:       true,
				settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
			}
			handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{sendErr: tc.err}, userStub))

			req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`)))
			req.Header.Set("Idempotency-Key", "pm_blocked_"+strings.Repeat("x", 40))
			rr := httptest.NewRecorder()
			handler.Send(rr, req)

			var resp struct {
				Code httpx.Code `json:"code"`
				Data struct {
					ForbiddenReason string `json:"forbidden_reason"`
				} `json:"data"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v (body=%s)", err, rr.Body.String())
			}
			if resp.Code != httpx.CodeForbidden {
				t.Fatalf("expected code httpx.CodeForbidden, got %d (body=%s)", resp.Code, rr.Body.String())
			}
			if resp.Data.ForbiddenReason != string(tc.reason) {
				t.Fatalf("expected data.forbidden_reason=%q, got %q", tc.reason, resp.Data.ForbiddenReason)
			}
		})
	}
}

// TestHandlerSendBlockedResponseNotCachedAsSuccess — httpx.APIError 通过
// ReplayOrStore 的 producer 路径返回时同样映射为 httpx.CodeForbidden(非 0 响应不缓存为成功)。
func TestHandlerSendBlockedResponseNotCachedAsSuccess(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	}
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{sendErr: httpx.Forbidden(httpx.ReasonPrivacySetting, "对方隐私设置不允许")}, userStub))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`)))
	req.Header.Set("Idempotency-Key", "pm_blocked_not_cached_"+strings.Repeat("y", 40))
	rr := httptest.NewRecorder()
	handler.Send(rr, req)

	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != httpx.CodeForbidden {
		t.Fatalf("expected code httpx.CodeForbidden, got %d", resp.Code)
	}
	req2 := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`)))
	req2.Header.Set("Idempotency-Key", "pm_blocked_not_cached_"+strings.Repeat("y", 40))
	rr2 := httptest.NewRecorder()
	handler.Send(rr2, req2)
	if err := json.NewDecoder(rr2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if resp.Code != httpx.CodeForbidden {
		t.Fatalf("expected replayed response code httpx.CodeForbidden, got %d", resp.Code)
	}
}

// TestHandlerUpdateRecallNonAuthorForbidden verifies the spec api_spec_chat.md
// contract: private_messages/update 与 private_messages/recall 非作者 → 10200 +
// data.forbidden_reason=not_resource_owner。
func TestHandlerUpdateRecallNonAuthorForbidden(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	}
	repo := &privateMessageErrorRepo{updateErr: ErrMessagePermissionDenied, recallErr: ErrMessagePermissionDenied}
	handler := NewHandler(NewServiceWithDeps(repo, userStub))

	t.Run("update non-author", func(t *testing.T) {
		req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(`{"message_id":"pm_1","content":"x"}`)))
		rr := httptest.NewRecorder()
		handler.Update(rr, req)
		assertForbiddenResponse(t, rr, "not_resource_owner")
	})
	t.Run("recall non-author", func(t *testing.T) {
		req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/recall", strings.NewReader(`{"message_id":"pm_1"}`)))
		rr := httptest.NewRecorder()
		handler.Recall(rr, req)
		assertForbiddenResponse(t, rr, "not_resource_owner")
	})
}

func assertForbiddenResponse(t *testing.T, rr *httptest.ResponseRecorder, reason string) {
	t.Helper()
	var resp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			ForbiddenReason string `json:"forbidden_reason"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, rr.Body.String())
	}
	if resp.Code != httpx.CodeForbidden {
		t.Fatalf("expected code httpx.CodeForbidden, got %d", resp.Code)
	}
	if resp.Data.ForbiddenReason != reason {
		t.Fatalf("expected forbidden_reason=%q, got %q", reason, resp.Data.ForbiddenReason)
	}
}
