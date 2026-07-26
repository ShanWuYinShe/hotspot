package resourceguard

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubChecker struct {
	ready   bool
	results map[ResourceKind]map[string]bool
	err     error
}

func (s stubChecker) Ready(context.Context) bool { return s.ready }
func (s stubChecker) MayExist(_ context.Context, kind ResourceKind, id string) (bool, error) {
	if s.err != nil {
		return true, s.err
	}
	if s.results[kind] == nil {
		return true, nil
	}
	allowed, ok := s.results[kind][id]
	if !ok {
		return true, nil
	}
	return allowed, nil
}
func (s stubChecker) Record(context.Context, ResourceKind, string) error { return nil }

func TestMiddlewareRejectsKnownMissingUser(t *testing.T) {
	SetDefaultChecker(stubChecker{ready: true, results: map[ResourceKind]map[string]bool{ResourceUser: {"user_missing": false}}})
	t.Cleanup(func() { SetDefaultChecker(NoopChecker{}) })
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/get_detail", strings.NewReader(`{"user_id":"user_missing"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if body := rr.Body.String(); !strings.Contains(body, "用户不存在") {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestMiddlewareBypassesWhenCheckerNotReady(t *testing.T) {
	SetDefaultChecker(stubChecker{ready: false})
	t.Cleanup(func() { SetDefaultChecker(NoopChecker{}) })
	called := false
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/groups/get_detail", strings.NewReader(`{"group_id":"group_1"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected middleware to bypass request")
	}
}

func TestMiddlewareBypassesOnCheckerError(t *testing.T) {
	SetDefaultChecker(stubChecker{ready: true, err: context.DeadlineExceeded})
	t.Cleanup(func() { SetDefaultChecker(NoopChecker{}) })
	called := false
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/moments/get_detail", strings.NewReader(`{"moment_id":"moment_1"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected middleware to bypass request")
	}
}

func TestMiddlewareBypassesNonWhitelistedRoute(t *testing.T) {
	SetDefaultChecker(stubChecker{ready: true, results: map[ResourceKind]map[string]bool{ResourceUser: {"user_missing": false}}})
	t.Cleanup(func() { SetDefaultChecker(NoopChecker{}) })
	called := false
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/search", strings.NewReader(`{"user_id":"user_missing"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatal("expected middleware to bypass non-whitelisted route")
	}
}

func TestMiddlewareRestoresRequestBody(t *testing.T) {
	SetDefaultChecker(stubChecker{ready: true, results: map[ResourceKind]map[string]bool{ResourceUser: {"user_1": true}}})
	t.Cleanup(func() { SetDefaultChecker(NoopChecker{}) })
	var body string
	handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body = string(data)
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/get_detail", strings.NewReader(`{"user_id":"user_1"}`))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	if body != `{"user_id":"user_1"}` {
		t.Fatalf("unexpected restored body: %s", body)
	}
}
