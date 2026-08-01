package private_messages

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/idempotency"
)

// TestHandlerSendMapsMessageBlockedTo10200WithReason verifies the spec
// api_spec_chat.md contract: private_messages/send 被拒绝时返回 code=10200 且
// data 携带 { reason: BlockReason }。
func TestHandlerSendMapsMessageBlockedTo10200WithReason(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		reason string
	}{
		{name: "privacy", err: &MessageBlockedError{Reason: BlockReasonPrivacy}, reason: "privacy"},
		{name: "stranger quota", err: &MessageBlockedError{Reason: BlockReasonStrangerQuotaExceeded}, reason: "stranger_quota_exceeded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			idempotency.SetStore(idempotency.NewInMemoryStore())
			// ensureCanSend 必须通过,才能走到 repo.Send 返回 MessageBlockedError。
			userStub := &privateMessageUserStub{
				item:     users.UserItem{Relation: &users.Relationship{}},
				ok:       true,
				settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
			}
			handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{sendErr: tc.err}, userStub))

			req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`)))
			req.Header.Set("Idempotency-Key", "pm_blocked_"+strings.Repeat("0", 40))
			rr := httptest.NewRecorder()
			handler.Send(rr, req)

			var resp struct {
				Code int `json:"code"`
				Data struct {
					Reason string `json:"reason"`
				} `json:"data"`
			}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response: %v (body=%s)", err, rr.Body.String())
			}
			if resp.Code != 10200 {
				t.Fatalf("expected code 10200, got %d (body=%s)", resp.Code, rr.Body.String())
			}
			if resp.Data.Reason != tc.reason {
				t.Fatalf("expected data.reason=%q, got %q", tc.reason, resp.Data.Reason)
			}
		})
	}
}

// TestHandlerSendBlockedResponseNotCachedAsSuccess — MessageBlockedError 通过
// ReplayOrStore 的 producer 路径返回时同样映射为 10200(非 0 响应不缓存为成功)。
func TestHandlerSendBlockedResponseNotCachedAsSuccess(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	userStub := &privateMessageUserStub{
		item:     users.UserItem{Relation: &users.Relationship{}},
		ok:       true,
		settings: users.UserSettings{AllowPrivateChat: users.ChatPermEveryone},
	}
	handler := NewHandler(NewServiceWithDeps(&privateMessageErrorRepo{sendErr: &MessageBlockedError{Reason: BlockReasonPrivacy}}, userStub))

	req := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`)))
	req.Header.Set("Idempotency-Key", "pm_blocked_not_cached_"+strings.Repeat("0", 40))
	rr := httptest.NewRecorder()
	handler.Send(rr, req)

	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 10200 {
		t.Fatalf("expected code 10200, got %d", resp.Code)
	}
	// 非 0 响应按当前实现仍会缓存回放,但回放内容必须是 10200 而非成功。
	// 请求 body 已被首次读取消费,重建请求。
	req2 := withPrivateMessagePrincipal(httptest.NewRequest(http.MethodPost, "/send", strings.NewReader(`{"target_user_id":"user_bob","type":"text","content":"hello"}`)))
	req2.Header.Set("Idempotency-Key", "pm_blocked_not_cached_"+strings.Repeat("0", 40))
	rr2 := httptest.NewRecorder()
	handler.Send(rr2, req2)
	if err := json.NewDecoder(rr2.Body).Decode(&resp); err != nil {
		t.Fatalf("decode second response: %v", err)
	}
	if resp.Code != 10200 {
		t.Fatalf("expected replayed response code 10200, got %d", resp.Code)
	}
}
