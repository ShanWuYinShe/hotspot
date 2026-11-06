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

// withAdminChecker 注入管理端判定依赖（生产由根中间件 WithDependencies 注入）。
func withAdminChecker(ctx context.Context, checker AdminChecker) context.Context {
	return context.WithValue(ctx, depsContextKey{}, requestDeps{adminChecker: checker})
}

func TestIsAdminUser(t *testing.T) {
	// 1. 未配置 checker：fail-closed
	if IsAdminUser(context.Background(), "usr_1") {
		t.Fatal("expected IsAdminUser to return false when checker not set (fail-closed)")
	}

	// 2. checker 返回 error：fail-closed
	ctxErr := withAdminChecker(context.Background(), mockAdminChecker{
		isAdminFunc: func(ctx context.Context, userID string) (bool, error) {
			return true, errors.New("db disconnect")
		},
	})
	if IsAdminUser(ctxErr, "usr_1") {
		t.Fatal("expected IsAdminUser to return false when checker returns error (fail-closed)")
	}

	// 3. checker 返回 false
	ctxNormal := withAdminChecker(context.Background(), mockAdminChecker{
		isAdminFunc: func(ctx context.Context, userID string) (bool, error) {
			return false, nil
		},
	})
	if IsAdminUser(ctxNormal, "usr_normal") {
		t.Fatal("expected IsAdminUser to return false for non-admin")
	}

	// 4. checker 返回 true
	ctxAdmin := withAdminChecker(context.Background(), mockAdminChecker{
		isAdminFunc: func(ctx context.Context, userID string) (bool, error) {
			return userID == "admin_1", nil
		},
	})
	if !IsAdminUser(ctxAdmin, "admin_1") {
		t.Fatal("expected IsAdminUser to return true for admin_1")
	}
	if IsAdminUser(ctxAdmin, "other_user") {
		t.Fatal("expected IsAdminUser to return false for other_user")
	}
}

func TestAdminAccessMiddleware(t *testing.T) {
	checker := mockAdminChecker{
		isAdminFunc: func(ctx context.Context, userID string) (bool, error) {
			return userID == "real_admin", nil
		},
	}

	handler := AdminAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	// 1. 无 principal 上下文
	t.Run("No Principal Denied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/users", nil)
		req = req.WithContext(withAdminChecker(req.Context(), checker))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assertForbiddenCode(t, rr.Body.Bytes())
	})

	// 2. principal 存在但非管理员
	t.Run("Non Admin Denied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/users", nil)
		req = req.WithContext(WithPrincipal(req.Context(), Principal{UserID: "user_bob"}))
		req = req.WithContext(withAdminChecker(req.Context(), checker))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assertForbiddenCode(t, rr.Body.Bytes())
	})

	// 3. 管理员放行
	t.Run("Admin Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/users", nil)
		req = req.WithContext(WithPrincipal(req.Context(), Principal{UserID: "real_admin"}))
		req = req.WithContext(withAdminChecker(req.Context(), checker))
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204, got %d", rr.Code)
		}
	})
}

func assertForbiddenCode(t *testing.T, body []byte) {
	t.Helper()
	var envelope struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatalf("decode forbidden envelope: %v body=%s", err, body)
	}
	if envelope.Code != httpx.CodeForbidden {
		t.Fatalf("expected 10200 forbidden, got %d", envelope.Code)
	}
}
