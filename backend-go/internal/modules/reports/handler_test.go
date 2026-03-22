package reports

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/idempotency"
)

type reportErrorRepo struct {
	createErr error
	listErr   error
}

func (r *reportErrorRepo) Create(context.Context, string, CreateReportRequest) (ReportCreated, error) {
	return ReportCreated{}, r.createErr
}
func (r *reportErrorRepo) List(context.Context, string) (PaginatedReports, error) {
	return PaginatedReports{}, r.listErr
}

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

func TestReportsHandlerReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&reportErrorRepo{createErr: errors.New("boom"), listErr: errors.New("boom")}))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"target_type":"moment","target_id":"moment_1","reason":"spam"}`))
	createReq.Header.Set("Idempotency-Key", "ijkl1234ijkl1234ijkl1234ijkl1234")
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	assertReportCode(t, createRR, 10600)

	listReq := httptest.NewRequest(http.MethodPost, "/me/list", strings.NewReader(`{}`))
	listReq = listReq.WithContext(ctx)
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	assertReportCode(t, listRR, 10600)
}

func assertReportCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
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
