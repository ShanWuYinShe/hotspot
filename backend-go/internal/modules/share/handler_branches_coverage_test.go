package share

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

// TestShareHandlersRejectMalformedJSON 覆盖四个 handler 的 DecodeStrictJSON
// 失败分支（请求体非法 JSON → 10000）。
func TestShareHandlersRejectMalformedJSON(t *testing.T) {
	handler := NewHandler(NewService())
	const badJSON = "{not-json"

	createReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(badJSON)))
	createReq.Header.Set("Idempotency-Key", "malformedcreatekey000000000000000001")
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	assertCode(t, createRR, httpx.CodeInvalidRequest)

	getRR := httptest.NewRecorder()
	handler.Get(getRR, httptest.NewRequest(http.MethodPost, "/get", strings.NewReader(badJSON)))
	assertCode(t, getRR, httpx.CodeInvalidRequest)

	listRR := httptest.NewRecorder()
	handler.List(listRR, withSharePrincipal(httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(badJSON))))
	assertCode(t, listRR, httpx.CodeInvalidRequest)

	revokeRR := httptest.NewRecorder()
	handler.Revoke(revokeRR, withSharePrincipal(httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader(badJSON))))
	assertCode(t, revokeRR, httpx.CodeInvalidRequest)
}

// TestShareRevokeHandlerBranches 覆盖 Revoke handler 的参数错误 / 不存在 / 成功三个分支。
func TestShareRevokeHandlerBranches(t *testing.T) {
	handler := NewHandler(NewService())

	// 既无 share_id 也无 share_code → 10000（service 层二选一校验）
	invalidRR := httptest.NewRecorder()
	handler.Revoke(invalidRR, withSharePrincipal(httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader("{}"))))
	assertCode(t, invalidRR, httpx.CodeInvalidRequest)

	// 分享不存在 → 10300（不泄露存在性）
	notFoundHandler := NewHandler(NewServiceWithRepo(&shareErrorRepo{revokeErr: ErrShareNotFound}))
	notFoundRR := httptest.NewRecorder()
	notFoundHandler.Revoke(notFoundRR, withSharePrincipal(httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader("{\"share_id\":\"share_missing\"}"))))
	assertCode(t, notFoundRR, httpx.CodeNotFound)

	// 成功撤销 → code 0
	svc := NewService()
	created, err := svc.Create(shareContextWithUser("user_alice"), CreateShareRequest{ShareType: ShareTypeUser, Payload: map[string]any{"shared_user_id": "user_bob"}})
	if err != nil {
		t.Fatalf("create share: %v", err)
	}
	okHandler := NewHandler(svc)
	okRR := httptest.NewRecorder()
	okHandler.Revoke(okRR, withSharePrincipal(httptest.NewRequest(http.MethodPost, "/revoke", strings.NewReader("{\"share_id\":\""+created.ShareID+"\"}"))))
	assertCode(t, okRR, 0)
}
