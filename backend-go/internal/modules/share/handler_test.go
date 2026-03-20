package share

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
)

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
	createReq2 := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"share_type":"user","payload":{"shared_user_id":"user_bob"}}`)))
	createReq2.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	createRR2 := httptest.NewRecorder()
	handler.Create(createRR2, createReq2)
	var createResp struct {
		Code int          `json:"code"`
		Data ShareCreated `json:"data"`
	}
	_ = json.NewDecoder(createRR.Body).Decode(&createResp)
	var createResp2 struct {
		Code int          `json:"code"`
		Data ShareCreated `json:"data"`
	}
	_ = json.NewDecoder(createRR2.Body).Decode(&createResp2)
	if createResp.Code != 0 || createResp.Data.ShareID == "" || createResp.Data.ShareID != createResp2.Data.ShareID {
		t.Fatalf("unexpected create response: %#v %#v", createResp, createResp2)
	}

	getReq := httptest.NewRequest(http.MethodPost, "/get", strings.NewReader(`{"share_code":"`+createResp.Data.ShareCode+`"}`))
	getRR := httptest.NewRecorder()
	handler.Get(getRR, getReq)
	var getResp struct {
		Code int         `json:"code"`
		Data ShareDetail `json:"data"`
	}
	_ = json.NewDecoder(getRR.Body).Decode(&getResp)
	if getResp.Code != 0 || getResp.Data.Share["share_code"] == nil {
		t.Fatalf("unexpected get response: %#v", getResp)
	}

	listReq := withSharePrincipal(httptest.NewRequest(http.MethodPost, "/me/list", strings.NewReader(`{}`)))
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	var listResp struct {
		Code int             `json:"code"`
		Data PaginatedShares `json:"data"`
	}
	_ = json.NewDecoder(listRR.Body).Decode(&listResp)
	if listResp.Code != 0 || len(listResp.Data.Items) == 0 {
		t.Fatalf("unexpected list response: %#v", listResp)
	}
}
