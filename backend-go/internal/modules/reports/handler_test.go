package reports

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
)

func TestCreateAndListReports(t *testing.T) {
	idempotency.SetStore(idempotency.NewInMemoryStore())
	handler := NewHandler(NewService())
	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"target_type":"moment","target_id":"moment_1","reason":"spam"}`))
	createReq.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	ctx := platformauth.WithPrincipal(createReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	createRR2 := httptest.NewRecorder()
	createReq2 := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"target_type":"moment","target_id":"moment_1","reason":"spam"}`))
	createReq2.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	createReq2 = createReq2.WithContext(ctx)
	handler.Create(createRR2, createReq2)

	listReq := httptest.NewRequest(http.MethodPost, "/me/list", strings.NewReader(`{}`))
	listReq = listReq.WithContext(ctx)
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	var replayResp struct {
		Code int           `json:"code"`
		Data ReportCreated `json:"data"`
	}
	_ = json.NewDecoder(createRR.Body).Decode(&replayResp)
	var replayResp2 struct {
		Code int           `json:"code"`
		Data ReportCreated `json:"data"`
	}
	_ = json.NewDecoder(createRR2.Body).Decode(&replayResp2)
	var listResp struct {
		Code int              `json:"code"`
		Data PaginatedReports `json:"data"`
	}
	_ = json.NewDecoder(listRR.Body).Decode(&listResp)
	if listResp.Code != 0 || len(listResp.Data.Items) == 0 || replayResp.Data.ReportID != replayResp2.Data.ReportID {
		t.Fatalf("unexpected list response: %#v %#v %#v", listResp, replayResp, replayResp2)
	}
}
