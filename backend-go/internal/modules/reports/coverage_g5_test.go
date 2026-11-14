package reports

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/testdb"
)

func g5DecodeCode(t *testing.T, rr *httptest.ResponseRecorder) httpx.Code {
	t.Helper()
	var env struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode response %q: %v", rr.Body.String(), err)
	}
	return env.Code
}

// TestServiceAdminReportStoreUnsupported 覆盖 Service 对不实现 adminReportStore
// 的仓储（内存实现）返回 errAdminReportStoreUnsupported 的降级分支。
func TestServiceAdminReportStoreUnsupported(t *testing.T) {
	svc := NewService()
	if _, err := svc.MarkHandledAsAdmin(context.Background(), "report_x", "resolved", time.Now(), nil, nil); !errors.Is(err, errAdminReportStoreUnsupported) {
		t.Fatalf("MarkHandledAsAdmin err = %v, want errAdminReportStoreUnsupported", err)
	}
	if _, err := svc.DeleteHandledBefore(context.Background(), time.Now(), 500); !errors.Is(err, errAdminReportStoreUnsupported) {
		t.Fatalf("DeleteHandledBefore err = %v, want errAdminReportStoreUnsupported", err)
	}
}

// TestPostgresIT_AdminReportStoreRoundTrip 真实库覆盖管理端裁决与保留期清理：
// pending 守卫（二次裁决返回 false、不存在返回 false）、终态删除只清 resolved/
// dismissed 且不动 pending 行。
func TestPostgresIT_AdminReportStoreRoundTrip(t *testing.T) {
	testdb.SkipIfShort(t)
	factory := testdb.Open(t)
	testdb.TxScope(t, factory, func(ctx context.Context, txFactory *persistence.Factory) {
		repo := NewPostgresRepository(txFactory)
		svc := NewServiceWithRepo(repo)
		reporter := testdb.InsertUser(t, txFactory, "u_g5_report_reporter", "Reporter")
		admin := testdb.InsertUser(t, txFactory, "u_g5_report_admin", "Admin")

		created, err := repo.Create(ctx, reporter, CreateReportRequest{
			TargetType: ReportTargetTypeUser,
			TargetID:   "u_g5_report_target_1",
			Reason:     ReportReasonSpam,
		}, nil)
		if err != nil {
			t.Fatalf("Create report: %v", err)
		}

		note := "processed by test"
		now := time.Now().UTC()
		handled, err := svc.MarkHandledAsAdmin(ctx, created.ReportID, "resolved", now, &admin, &note)
		if err != nil {
			t.Fatalf("MarkHandledAsAdmin: %v", err)
		}
		if !handled {
			t.Fatal("MarkHandledAsAdmin = false, want true for pending report")
		}

		// 二次裁决：status 已非 pending，RETURNING 无行 -> false。
		handled, err = svc.MarkHandledAsAdmin(ctx, created.ReportID, "dismissed", now, &admin, nil)
		if err != nil {
			t.Fatalf("MarkHandledAsAdmin second: %v", err)
		}
		if handled {
			t.Fatal("MarkHandledAsAdmin = true on already handled report, want false")
		}

		// 不存在的举报：同样 false 且不报错。
		handled, err = svc.MarkHandledAsAdmin(ctx, "report_g5_missing", "resolved", now, &admin, nil)
		if err != nil {
			t.Fatalf("MarkHandledAsAdmin missing: %v", err)
		}
		if handled {
			t.Fatal("MarkHandledAsAdmin = true for missing report, want false")
		}

		// 另一条 pending 举报：保留期清理不得删除 pending。
		if _, err := repo.Create(ctx, reporter, CreateReportRequest{
			TargetType: ReportTargetTypeUser,
			TargetID:   "u_g5_report_target_2",
			Reason:     ReportReasonHarassment,
		}, nil); err != nil {
			t.Fatalf("Create pending report: %v", err)
		}

		deleted, err := svc.DeleteHandledBefore(ctx, now.Add(time.Hour), 500)
		if err != nil {
			t.Fatalf("DeleteHandledBefore: %v", err)
		}
		if deleted != 1 {
			t.Fatalf("DeleteHandledBefore deleted = %d, want 1", deleted)
		}
		if _, duplicate, err := repo.FindExistingReport(ctx, reporter, ReportTargetTypeUser, "u_g5_report_target_2"); err != nil || !duplicate {
			t.Fatalf("pending report should survive prune: duplicate=%v err=%v", duplicate, err)
		}
		if _, duplicate, err := repo.FindExistingReport(ctx, reporter, ReportTargetTypeUser, "u_g5_report_target_1"); err != nil || duplicate {
			t.Fatalf("handled report should be pruned: duplicate=%v err=%v", duplicate, err)
		}

		// 分批语义：每次最多删除 limit 行，循环直到删空（避免无界 DELETE 长时间持锁）。
		for i, targetID := range []string{"u_g5_report_batch_1", "u_g5_report_batch_2"} {
			created, err := repo.Create(ctx, reporter, CreateReportRequest{
				TargetType: ReportTargetTypeUser,
				TargetID:   targetID,
				Reason:     ReportReasonSpam,
			}, nil)
			if err != nil {
				t.Fatalf("create batch report %d: %v", i, err)
			}
			if _, err := svc.MarkHandledAsAdmin(ctx, created.ReportID, "resolved", now, &admin, nil); err != nil {
				t.Fatalf("handle batch report %d: %v", i, err)
			}
		}
		for i, want := range []int64{1, 1, 0} {
			deleted, err := svc.DeleteHandledBefore(ctx, now.Add(time.Hour), 1)
			if err != nil {
				t.Fatalf("batch %d: %v", i, err)
			}
			if deleted != want {
				t.Fatalf("batch %d deleted = %d, want %d", i, deleted, want)
			}
		}
	})
}

// TestPostgresRepositoryWithoutDatabaseReturnsNotImplemented 覆盖无 DB 时仓方法
// 统一返回 ErrNotImplemented（内存降级语义）。
func TestPostgresRepositoryWithoutDatabaseReturnsNotImplemented(t *testing.T) {
	repo := NewPostgresRepository(nil)
	ctx := context.Background()
	if _, err := repo.Create(ctx, "u1", CreateReportRequest{TargetType: ReportTargetTypeUser, TargetID: "u2", Reason: ReportReasonSpam}, nil); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("Create err = %v, want ErrNotImplemented", err)
	}
	if _, _, err := repo.FindExistingReport(ctx, "u1", ReportTargetTypeUser, "u2"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("FindExistingReport err = %v, want ErrNotImplemented", err)
	}
	if _, _, err := repo.TargetOwner(ctx, ReportTargetTypeUser, "u2"); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("TargetOwner err = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.List(ctx, "u1", ListReportsRequest{}); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("List err = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.MarkHandledAsAdmin(ctx, "report_x", "resolved", time.Now(), nil, nil); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("MarkHandledAsAdmin err = %v, want ErrNotImplemented", err)
	}
	if _, err := repo.DeleteHandledBefore(ctx, time.Now(), 500); !errors.Is(err, persistence.ErrNotImplemented) {
		t.Fatalf("DeleteHandledBefore err = %v, want ErrNotImplemented", err)
	}
}

// TestHandlerCreateRejectsInvalidBody 覆盖 Create 解码失败/字段缺失的 10000 分支。
func TestHandlerCreateRejectsInvalidBody(t *testing.T) {
	h := NewHandler(NewService())
	for _, body := range []string{"{", "{}", "{\"target_type\":\"user\",\"target_id\":\"u2\"}"} {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/reports/create", strings.NewReader(body))
		h.Create(rr, req)
		if code := g5DecodeCode(t, rr); code != httpx.CodeInvalidRequest {
			t.Fatalf("body %q: code = %d, want %d", body, code, httpx.CodeInvalidRequest)
		}
	}
}

// TestHandlerCreateMapsInvalidRequest 覆盖 service 返回 ErrInvalidRequest 时
// handler 映射为业务错误码 10000 的分支。
func TestHandlerCreateMapsInvalidRequest(t *testing.T) {
	h := NewHandler(NewService())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/reports/create", strings.NewReader("{\"target_type\":\"user\",\"target_id\":\"u2\",\"reason\":\"bogus\"}"))
	h.Create(rr, req)
	if code := g5DecodeCode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("code = %d, want %d", code, httpx.CodeInvalidRequest)
	}
}

// TestHandlerListRejectsInvalidBody 覆盖 List 解码失败分支。
func TestHandlerListRejectsInvalidBody(t *testing.T) {
	h := NewHandler(NewService())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/reports/fetch", strings.NewReader("{"))
	h.List(rr, req)
	if code := g5DecodeCode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("code = %d, want %d", code, httpx.CodeInvalidRequest)
	}
}

// TestServiceListNormalizesPaginationLimit 覆盖 service 层分页 limit 兜底裁剪。
func TestServiceListNormalizesPaginationLimit(t *testing.T) {
	svc := NewService()
	req := ListReportsRequest{Pagination: &PaginationParams{Limit: 10000}}
	if _, err := svc.List(context.Background(), req); err != nil {
		t.Fatalf("List with oversized limit: %v", err)
	}
}
