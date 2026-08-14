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
	"hotspot/backend-go/internal/platform/httpx"
)

type reportErrorRepo struct {
	createErr error
	listErr   error
}

func (r *reportErrorRepo) Create(context.Context, string, CreateReportRequest) (ReportCreated, error) {
	return ReportCreated{}, r.createErr
}
func (r *reportErrorRepo) List(context.Context, string, ListReportsRequest) (PaginatedReports, error) {
	return PaginatedReports{}, r.listErr
}
func (r *reportErrorRepo) FindExistingReport(context.Context, string, ReportTargetType, string) (string, bool, error) {
	return "", false, r.createErr
}
func (r *reportErrorRepo) TargetOwner(context.Context, ReportTargetType, string) (string, bool, error) {
	return "", false, r.createErr
}

func TestCreateAndListReports(t *testing.T) {
	repo := NewInMemoryRepository()
	repo.SeedTargetOwner(ReportTargetTypeMoment, "moment_1", "user_bob")
	handler := NewHandler(NewServiceWithRepo(repo))
	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"target_type":"moment","target_id":"moment_1","reason":"spam"}`))
	ctx := platformauth.WithPrincipal(createReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	createRR2 := httptest.NewRecorder()
	createReq2 := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"target_type":"moment","target_id":"moment_1","reason":"spam"}`))
	createReq2 = createReq2.WithContext(ctx)
	handler.Create(createRR2, createReq2)

	listReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"target_type":"moment"}`))
	listReq = listReq.WithContext(ctx)
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	var replayResp struct {
		Code httpx.Code    `json:"code"`
		Data ReportCreated `json:"data"`
	}
	_ = json.NewDecoder(createRR.Body).Decode(&replayResp)
	var replayResp2 struct {
		Code httpx.Code    `json:"code"`
		Data ReportCreated `json:"data"`
	}
	_ = json.NewDecoder(createRR2.Body).Decode(&replayResp2)
	var listResp struct {
		Code httpx.Code       `json:"code"`
		Data PaginatedReports `json:"data"`
	}
	_ = json.NewDecoder(listRR.Body).Decode(&listResp)
	if listResp.Code != 0 || len(listResp.Data.Items) == 0 || replayResp.Data.IsDuplicate {
		t.Fatalf("unexpected list response: %#v %#v %#v", listResp, replayResp, replayResp2)
	}
	if !replayResp2.Data.IsDuplicate || replayResp2.Data.ReportID != replayResp.Data.ReportID {
		t.Fatalf("expected IsDuplicate on second identical report: %#v %#v", replayResp, replayResp2)
	}
}

func TestReportsHandlerReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&reportErrorRepo{createErr: errors.New("boom"), listErr: errors.New("boom")}))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"target_type":"moment","target_id":"moment_1","reason":"spam"}`))
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	assertReportCode(t, createRR, httpx.CodeInternalUnavailable)

	listReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{}`))
	listReq = listReq.WithContext(ctx)
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	assertReportCode(t, listRR, httpx.CodeInternalUnavailable)
}

func TestCreateReportRejectsSelfAndDuplicateReport(t *testing.T) {
	repo := NewInMemoryRepository()
	repo.SeedTargetOwner(ReportTargetTypeUser, "user_alice", "user_alice")
	repo.SeedTargetOwner(ReportTargetTypeMoment, "moment_1", "user_bob")
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	selfReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"target_type":"user","target_id":"user_alice","reason":"spam"}`))
	selfReq = selfReq.WithContext(ctx)
	selfRR := httptest.NewRecorder()
	handler.Create(selfRR, selfReq)
	assertReportCode(t, selfRR, httpx.CodeConflict)

	dupReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"target_type":"moment","target_id":"moment_1","reason":"spam"}`))
	dupReq = dupReq.WithContext(ctx)
	dupRR := httptest.NewRecorder()
	handler.Create(dupRR, dupReq)
	assertReportCode(t, dupRR, 0)

	dupReq2 := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"target_type":"moment","target_id":"moment_1","reason":"spam"}`))
	dupReq2 = dupReq2.WithContext(ctx)
	dupRR2 := httptest.NewRecorder()
	handler.Create(dupRR2, dupReq2)
	var dupResp struct {
		Code httpx.Code    `json:"code"`
		Data ReportCreated `json:"data"`
	}
	if err := json.NewDecoder(dupRR2.Body).Decode(&dupResp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if dupResp.Code != 0 {
		t.Fatalf("expected code 0 on duplicate, got %d", dupResp.Code)
	}
	if !dupResp.Data.IsDuplicate {
		t.Fatalf("expected IsDuplicate on duplicate report, got %#v", dupResp)
	}
}

func TestCreateReportReturns10300OnTargetNotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"target_type":"moment","target_id":"moment_nonexistent","reason":"spam"}`))
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.Create(rr, req)
	assertReportCode(t, rr, httpx.CodeNotFound)
}

func TestReportsMeListFiltersByStatus(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	repo.mu.Lock()
	repo.reports["user_alice"] = []ReportItem{
		{ReportID: "r1", TargetType: ReportTargetTypeMoment, TargetID: "m1", Reason: ReportReasonSpam, Status: ReportStatusPending},
		{ReportID: "r2", TargetType: ReportTargetTypeMoment, TargetID: "m2", Reason: ReportReasonSpam, Status: ReportStatusResolved},
	}
	repo.mu.Unlock()

	listReq := httptest.NewRequest(http.MethodPost, "/list", strings.NewReader(`{"status":"pending"}`))
	listReq = listReq.WithContext(ctx)
	listRR := httptest.NewRecorder()
	handler.List(listRR, listReq)
	var resp struct {
		Code httpx.Code       `json:"code"`
		Data PaginatedReports `json:"data"`
	}
	if err := json.NewDecoder(listRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Items) != 1 || resp.Data.Items[0].ReportID != "r1" {
		t.Fatalf("expected 1 pending report (r1), got %#v", resp)
	}
}

func assertReportCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
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
