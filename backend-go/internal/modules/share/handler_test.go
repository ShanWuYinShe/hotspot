package share

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

type shareErrorRepo struct {
	createErr error
	getErr    error
	listErr   error
	revokeErr error
}

func (r *shareErrorRepo) Create(context.Context, createShareCommand) (ShareCreated, error) {
	return ShareCreated{}, r.createErr
}
func (r *shareErrorRepo) Get(context.Context, getShareQuery) (ShareDetail, error) {
	return ShareDetail{}, r.getErr
}
func (r *shareErrorRepo) List(context.Context, listSharesQuery) (PaginatedShares, error) {
	return PaginatedShares{}, r.listErr
}
func (r *shareErrorRepo) Revoke(context.Context, revokeShareCommand) error {
	return r.revokeErr
}

func withSharePrincipal(req *http.Request) *http.Request {
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	return req.WithContext(ctx)
}

func TestCreateGetAndListShare(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewService())
	createReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"share_type":"user","payload":{"shared_user_id":"user_bob"}}`)))
	createReq.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	var createResp struct {
		Code httpx.Code   `json:"code"`
		Data ShareCreated `json:"data"`
	}
	_ = json.NewDecoder(createRR.Body).Decode(&createResp)
	if createResp.Code != 0 || createResp.Data.ShareID == "" || !validShareCode(createResp.Data.ShareCode) {
		t.Fatalf("unexpected create response: %#v", createResp)
	}

	getReq := httptest.NewRequest(http.MethodPost, "/get", strings.NewReader(`{"share_code":"`+strings.ToLower(createResp.Data.ShareCode)+`"}`))
	getRR := httptest.NewRecorder()
	handler.Get(getRR, getReq)
	var getResp struct {
		Code httpx.Code  `json:"code"`
		Data ShareDetail `json:"data"`
	}
	_ = json.NewDecoder(getRR.Body).Decode(&getResp)
	if getResp.Code != 0 || getResp.Data.Share.ShareCode != createResp.Data.ShareCode {
		t.Fatalf("unexpected get response: %#v", getResp)
	}

	listReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/list_my", strings.NewReader(`{"share_type":"user"}`)))
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	var listResp struct {
		Code httpx.Code      `json:"code"`
		Data PaginatedShares `json:"data"`
	}
	_ = json.NewDecoder(listRR.Body).Decode(&listResp)
	if listResp.Code != 0 || len(listResp.Data.Items) == 0 || listResp.Data.Items[0].ShareType != ShareTypeUser {
		t.Fatalf("unexpected list response: %#v", listResp)
	}
}

func TestShareHandlerValidatesRequest(t *testing.T) {
	handler := NewHandler(NewService())

	badCreate := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"share_type":"bad","payload":{}}`)))
	badCreate.Header.Set("Idempotency-Key", "bbbb1234bbbb1234bbbb1234bbbb1234")
	badCreateRR := httptest.NewRecorder()
	handler.Create(badCreateRR, badCreate)
	assertCode(t, badCreateRR, httpx.CodeInvalidRequest)

	badGet := httptest.NewRequest(http.MethodPost, "/get", strings.NewReader(`{"share_code":"bad"}`))
	badGetRR := httptest.NewRecorder()
	handler.Get(badGetRR, badGet)
	assertCode(t, badGetRR, httpx.CodeInvalidRequest)

	badList := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/list_my", strings.NewReader(`{"share_type":"bad"}`)))
	badListRR := httptest.NewRecorder()
	handler.List(badListRR, badList)
	assertCode(t, badListRR, httpx.CodeInvalidRequest)
}

func TestShareHandlerReturnsBusinessErrors(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&shareErrorRepo{createErr: ErrShareNotFound, getErr: ErrShareNotFound, revokeErr: ErrSharePermission}))

	createReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"share_type":"user","payload":{"shared_user_id":"user_bob"}}`)))
	createReq.Header.Set("Idempotency-Key", "efgh1234efgh1234efgh1234efgh1234")
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	assertCode(t, createRR, httpx.CodeNotFound)

	getReq := httptest.NewRequest(http.MethodPost, "/get", strings.NewReader(`{"share_code":"ABCDEFGHJKLM"}`))
	getRR := httptest.NewRecorder()
	handler.Get(getRR, getReq)
	assertCode(t, getRR, httpx.CodeNotFound)

	revokeReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader(`{"share_id":"share_1"}`)))
	revokeRR := httptest.NewRecorder()
	handler.Revoke(revokeRR, revokeReq)
	// 隐私一致性（spec api_spec_share.md）：share 存在但无权限时统一 httpx.CodeNotFound，
	// 不通过 httpx.CodeForbidden 泄露 share 的存在性。
	assertCode(t, revokeRR, httpx.CodeNotFound)
}

func TestShareHandlerReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&shareErrorRepo{createErr: errors.New("boom"), getErr: errors.New("boom"), listErr: errors.New("boom"), revokeErr: errors.New("boom")}))

	createReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"share_type":"user","payload":{"shared_user_id":"user_bob"}}`)))
	createReq.Header.Set("Idempotency-Key", "ijkl1234ijkl1234ijkl1234ijkl1234")
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	assertCode(t, createRR, httpx.CodeInternalUnavailable)

	getReq := httptest.NewRequest(http.MethodPost, "/get", strings.NewReader(`{"share_code":"ABCDEFGHJKLM"}`))
	getRR := httptest.NewRecorder()
	handler.Get(getRR, getReq)
	assertCode(t, getRR, httpx.CodeInternalUnavailable)

	listReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/list_my", strings.NewReader(`{}`)))
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	assertCode(t, listRR, httpx.CodeInternalUnavailable)

	revokeReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader(`{"share_id":"share_1"}`)))
	revokeRR := httptest.NewRecorder()
	handler.Revoke(revokeRR, revokeReq)
	assertCode(t, revokeRR, httpx.CodeInternalUnavailable)
}

func TestShareCreateRequiresIdempotencyKey(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewService())
	req := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"share_type":"user","payload":{"shared_user_id":"user_bob"}}`)))
	// No Idempotency-Key header set
	rr := httptest.NewRecorder()
	handler.Create(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestShareGetExpiredReturns10300(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	svc := NewService()
	handler := NewHandler(svc)

	// Create a share with expires_in_seconds = 1 (will expire in 1 second)
	createReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"share_type":"user","payload":{"shared_user_id":"user_bob"},"expires_in_seconds":1}`)))
	createReq.Header.Set("Idempotency-Key", "expi_test_create_once_00000000000")
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	var createResp struct {
		Code httpx.Code   `json:"code"`
		Data ShareCreated `json:"data"`
	}
	_ = json.NewDecoder(createRR.Body).Decode(&createResp)
	if createResp.Code != 0 {
		t.Fatalf("create failed: %#v", createResp)
	}

	// Poll until the share expires (expires_in_seconds=1) instead of a fixed sleep.
	// This avoids flaky timing in CI and is still bounded by a hard timeout.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		getReq := httptest.NewRequest(http.MethodPost, "/get", strings.NewReader(`{"share_code":"`+createResp.Data.ShareCode+`"}`))
		getRR := httptest.NewRecorder()
		handler.Get(getRR, getReq)
		var checkResp struct{ Code httpx.Code }
		_ = json.NewDecoder(getRR.Body).Decode(&checkResp)
		if checkResp.Code == httpx.CodeNotFound {
			return // expired as expected
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("share did not expire within timeout")
}

func TestShareGetReturnsFullShareDetailShape(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewService())

	// Create a share
	createReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"share_type":"user","payload":{"shared_user_id":"user_bob"}}`)))
	createReq.Header.Set("Idempotency-Key", "detail_shape_create_once_00000000")
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	var createResp struct {
		Code httpx.Code   `json:"code"`
		Data ShareCreated `json:"data"`
	}
	_ = json.NewDecoder(createRR.Body).Decode(&createResp)
	if createResp.Code != 0 {
		t.Fatalf("create failed: %#v", createResp)
	}

	// Get the share and verify all required fields are present
	getReq := httptest.NewRequest(http.MethodPost, "/get", strings.NewReader(`{"share_code":"`+createResp.Data.ShareCode+`"}`))
	getRR := httptest.NewRecorder()
	handler.Get(getRR, getReq)
	var getResp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Share struct {
				ShareID   string `json:"share_id"`
				ShareCode string `json:"share_code"`
				ShareType string `json:"share_type"`
				Creator   struct {
					UserID string `json:"user_id"`
				} `json:"creator"`
				CreatedAt string `json:"created_at"`
			} `json:"share"`
			Payload map[string]any `json:"payload"`
		} `json:"data"`
	}
	_ = json.NewDecoder(getRR.Body).Decode(&getResp)
	if getResp.Code != 0 {
		t.Fatalf("get failed: %#v", getResp)
	}
	if getResp.Data.Share.ShareID == "" || getResp.Data.Share.ShareCode == "" || getResp.Data.Share.ShareType == "" || getResp.Data.Share.Creator.UserID == "" || getResp.Data.Share.CreatedAt == "" {
		t.Fatalf("missing required fields in share detail: %#v", getResp.Data.Share)
	}
	if getResp.Data.Payload == nil {
		t.Fatalf("payload is missing in share detail response")
	}
}

func TestShareMeListRequiresAuth(t *testing.T) {
	handler := NewHandler(NewService())
	// Request without principal — List handler delegates to service which checks for auth
	req := httptest.NewRequest(http.MethodPost, "/list_my", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.List(rr, req)
	assertCode(t, rr, httpx.CodeInvalidRequest)
}

func TestShareCodeValidationRejectsInvalidChars(t *testing.T) {
	handler := NewHandler(NewService())

	for _, code := range []string{"abc", "ABCDEFGHJKLMN", "ABC0EFGHJKLM", "ABCOEFGHJKLM"} {
		getRR := httptest.NewRecorder()
		handler.Get(getRR, httptest.NewRequest(http.MethodPost, "/get", strings.NewReader(`{"share_code":"`+code+`"}`)))
		var resp struct{ Code httpx.Code }
		_ = json.NewDecoder(getRR.Body).Decode(&resp)
		if resp.Code != httpx.CodeInvalidRequest {
			t.Fatalf("code %q: expected httpx.CodeInvalidRequest, got %d", code, resp.Code)
		}
	}

	// "abcdefghjklm" is lowercase 12-char — it passes length check (12 chars)
	// and normalizeShareCode uppercases it, so validShareCode() sees "ABCDEFGHJKLM"
	// which is a valid code. We verify this by checking it does NOT return httpx.CodeInvalidRequest.
	getRR := httptest.NewRecorder()
	handler.Get(getRR, httptest.NewRequest(http.MethodPost, "/get", strings.NewReader(`{"share_code":"abcdefghjklm"}`)))
	var resp struct{ Code httpx.Code }
	_ = json.NewDecoder(getRR.Body).Decode(&resp)
	if resp.Code == httpx.CodeInvalidRequest {
		t.Fatalf("lowercase 12-char code got httpx.CodeInvalidRequest — normalizeShareCode should have uppercased it")
	}
}

func assertCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
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
