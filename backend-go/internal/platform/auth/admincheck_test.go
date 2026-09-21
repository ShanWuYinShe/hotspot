package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"hotspot/backend-go/internal/platform/httpx"
)

type mockAdminChecker struct {
	isAdminFunc func(ctx context.Context, userID string) (bool, error)
}

func (m mockAdminChecker) IsAdmin(ctx context.Context, userID string) (bool, error) {
	if m.isAdminFunc != nil {
		return m.isAdminFunc(ctx, userID)
	}
	return false, nil
}

func TestIsAdminUser(t *testing.T) {
	// 保存原 adminChecker
	orig := adminChecker.Load()
	defer func() {
		if orig != nil {
			adminChecker.Store(orig)
		} else {
			adminChecker.Store((*AdminChecker)(nil))
		}
	}()

	ctx := context.Background()

	// 1. 未配置 checker
	adminChecker.Store((*AdminChecker)(nil))
	if IsAdminUser(ctx, "usr_1") {
		t.Fatal("expected IsAdminUser to return false when checker not set (fail-closed)")
	}

	// 2. checker 返回 error
	SetAdminChecker(mockAdminChecker{
		isAdminFunc: func(ctx context.Context, userID string) (bool, error) {
			return true, errors.New("db disconnect")
		},
	})
	if IsAdminUser(ctx, "usr_1") {
		t.Fatal("expected IsAdminUser to return false when checker returns error (fail-closed)")
	}

	// 3. checker 返回 false
	SetAdminChecker(mockAdminChecker{
		isAdminFunc: func(ctx context.Context, userID string) (bool, error) {
			return false, nil
		},
	})
	if IsAdminUser(ctx, "usr_normal") {
		t.Fatal("expected IsAdminUser to return false for non-admin")
	}

	// 4. checker 返回 true
	SetAdminChecker(mockAdminChecker{
		isAdminFunc: func(ctx context.Context, userID string) (bool, error) {
			return userID == "admin_1", nil
		},
	})
	if !IsAdminUser(ctx, "admin_1") {
		t.Fatal("expected IsAdminUser to return true for admin_1")
	}
	if IsAdminUser(ctx, "other_user") {
		t.Fatal("expected IsAdminUser to return false for other_user")
	}
}

func TestAdminAccessMiddleware(t *testing.T) {
	orig := adminChecker.Load()
	defer func() {
		if orig != nil {
			adminChecker.Store(orig)
		} else {
			adminChecker.Store((*AdminChecker)(nil))
		}
	}()

	SetAdminChecker(mockAdminChecker{
		isAdminFunc: func(ctx context.Context, userID string) (bool, error) {
			return userID == "real_admin", nil
		},
	})

	handler := AdminAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	// 1. 无 principal 上下文
	t.Run("No Principal Denied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/users", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 envelope, got status %d", rr.Code)
		}
		var resp httpx.ApiResponse[httpx.ForbiddenErrorData]
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Code != httpx.CodeForbidden || resp.Data == nil || resp.Data.ForbiddenReason != httpx.ReasonNotAdmin {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	// 2. 有 principal 但非管理员
	t.Run("Non Admin Denied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/users", nil)
		ctx := WithPrincipal(req.Context(), Principal{UserID: "normal_user"})
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 envelope, got status %d", rr.Code)
		}
		var resp httpx.ApiResponse[httpx.ForbiddenErrorData]
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resp.Code != httpx.CodeForbidden || resp.Data == nil || resp.Data.ForbiddenReason != httpx.ReasonNotAdmin {
			t.Fatalf("unexpected response: %+v", resp)
		}
	})

	// 3. 有 principal 且是真实管理员
	t.Run("Admin Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/users", nil)
		ctx := WithPrincipal(req.Context(), Principal{UserID: "real_admin"})
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content for authorized admin, got %d", rr.Code)
		}
	})
}
