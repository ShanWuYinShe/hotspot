package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
)

func authReqWithPrincipal(t *testing.T, handler *Handler, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, method, strings.NewReader(body))
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: "u1", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RevokeSession(rr, req)
	return rr
}

func decodeAuthCode(t *testing.T, rr *httptest.ResponseRecorder) int {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.Code
}

func TestCheckStatusRejectsNonObjectBody(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/check_status", strings.NewReader(`[1]`))
	rr := httptest.NewRecorder()
	handler.CheckStatus(rr, req)
	if code := decodeAuthCode(t, rr); code != 10000 {
		t.Fatalf("expected 10000, got %d", code)
	}
}

func TestRevokeSessionErrorPaths(t *testing.T) {
	handler := newTestHandler()

	// 空 session_id → 10000
	req := httptest.NewRequest(http.MethodPost, "/revoke_session", strings.NewReader(`{"session_id":""}`))
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: "u1", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RevokeSession(rr, req)
	if code := decodeAuthCode(t, rr); code != 10000 {
		t.Fatalf("empty session id: expected 10000, got %d", code)
	}

	// 畸形 body → 10000
	req = httptest.NewRequest(http.MethodPost, "/revoke_session", strings.NewReader(`not-json`))
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: "u1", SessionID: "sess_1"}))
	rr = httptest.NewRecorder()
	handler.RevokeSession(rr, req)
	if code := decodeAuthCode(t, rr); code != 10000 {
		t.Fatalf("malformed body: expected 10000, got %d", code)
	}

	// 撤销当前会话 → 10200（禁止）
	req = httptest.NewRequest(http.MethodPost, "/revoke_session", strings.NewReader(`{"session_id":"sess_1"}`))
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: "u1", SessionID: "sess_1"}))
	rr = httptest.NewRecorder()
	handler.RevokeSession(rr, req)
	if code := decodeAuthCode(t, rr); code != 10200 {
		t.Fatalf("revoke current session: expected 10200, got %d", code)
	}

	// 撤销不存在的会话 → 10300（errors.As 解包 %w 包装的 APIError）
	req = httptest.NewRequest(http.MethodPost, "/revoke_session", strings.NewReader(`{"session_id":"no_such_session"}`))
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: "u1", SessionID: "sess_1"}))
	rr = httptest.NewRecorder()
	handler.RevokeSession(rr, req)
	if code := decodeAuthCode(t, rr); code != 10300 {
		t.Fatalf("revoke missing session: expected 10300, got %d", code)
	}
}

func TestRevokeSessionSuccessPath(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := newTestHandler()
	tokens := registerAppUser(t, handler, "revoke_user", "撤销用户", "password123")
	sessionID := handler.service.auth.UnsafeFirstSessionIDForUser(context.Background(), tokens.UserID)
	if sessionID == "" {
		t.Fatal("expected session id after register")
	}
	// 用真实会话身份列出并撤销另一个会话
	req := httptest.NewRequest(http.MethodPost, "/list_sessions", strings.NewReader(`{}`))
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: tokens.UserID, SessionID: sessionID}))
	rr := httptest.NewRecorder()
	handler.ListSessions(rr, req)
	var listResp struct {
		Code int         `json:"code"`
		Data SessionList `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&listResp); err != nil || listResp.Code != 0 {
		t.Fatalf("list sessions: %#v err=%v", listResp, err)
	}
	if len(listResp.Data.Items) != 1 {
		t.Fatalf("expected 1 session, got %d", len(listResp.Data.Items))
	}
	// 撤销当前会话以外的会话：先登录第二个设备再撤销它
	loginReq := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"login_id":"revoke_user","password":"password123","client_type":"app","device_id":"device-2"}`))
	loginRR := httptest.NewRecorder()
	handler.Login(loginRR, loginReq)
	var loginResp struct {
		Code int              `json:"code"`
		Data AuthTokenPairApp `json:"data"`
	}
	if err := json.NewDecoder(loginRR.Body).Decode(&loginResp); err != nil || loginResp.Code != 0 {
		t.Fatalf("login: %#v err=%v", loginResp, err)
	}
	secondSession := handler.service.auth.UnsafeFirstSessionIDForUser(context.Background(), tokens.UserID)
	if secondSession == "" || secondSession == sessionID {
		t.Fatalf("expected a second session, got %q", secondSession)
	}
	revokeReq := httptest.NewRequest(http.MethodPost, "/revoke_session", strings.NewReader(`{"session_id":"`+secondSession+`"}`))
	revokeReq = revokeReq.WithContext(auth.WithPrincipal(revokeReq.Context(), auth.Principal{UserID: tokens.UserID, SessionID: sessionID}))
	revokeRR := httptest.NewRecorder()
	handler.RevokeSession(revokeRR, revokeReq)
	if code := decodeAuthCode(t, revokeRR); code != 0 {
		t.Fatalf("revoke other session: expected 0, got %d", code)
	}
}

func TestRevokeAllSessionsRejectsNonEmptyBody(t *testing.T) {
	handler := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/revoke_all_sessions", strings.NewReader(`[1]`))
	req = req.WithContext(auth.WithPrincipal(req.Context(), auth.Principal{UserID: "u1", SessionID: "sess_1"}))
	rr := httptest.NewRecorder()
	handler.RevokeAllSessions(rr, req)
	if code := decodeAuthCode(t, rr); code != 10000 {
		t.Fatalf("expected 10000, got %d", code)
	}
}
