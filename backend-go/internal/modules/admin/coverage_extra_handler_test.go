package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/httpx"
)

// 本文件补齐 admin 全部 handler 的错误出口（参数解码失败 / 权限业务错误 /
// 服务层原始错误），以及此前完全零覆盖的 stats 两个 handler。
// handler 层只做「解码 → Service → 统一响应」，因此这里用真实 Service
// 配合可控 Repository 替身，避免伪造 handler 内部实现。

var errCovAdmin = errors.New("admin coverage extra: injected failure")

// covActorErrRepo 让取管理员资料（权限校验第一步）失败，用于触达 handler 的
// writeUnavailable（10600）出口。
type covActorErrRepo struct{ *InMemoryRepository }

func (r covActorErrRepo) GetActor(context.Context, string) (AdminActor, bool, error) {
	return AdminActor{}, false, errCovAdmin
}

// covStatsErrRepo 让统计聚合查询失败，用于触达 stats handler 的错误分支。
type covStatsErrRepo struct{ *InMemoryRepository }

func (r covStatsErrRepo) StatsGeo(context.Context, geoStatsQuery) (AdminGeoStatsResponse, error) {
	return AdminGeoStatsResponse{}, errCovAdmin
}

func (r covStatsErrRepo) StatsGeoDrilldown(context.Context, geoDrilldownQuery) (geoDrilldownPage, error) {
	return geoDrilldownPage{}, errCovAdmin
}

func covTestProvider() Provider {
	return NewAppConfigProvider(appconfig.NewStaticSource(config.Load()))
}

func newCovService(serviceRepo Repository) *Service {
	return NewService(serviceRepo, covTestProvider(), NewInMemoryModerationStore(), nil)
}

type covHandlerCall struct {
	name string
	call func(*Handler, http.ResponseWriter, *http.Request)
	body string
	key  string
}

func covStatsDrilldownBody(t *testing.T) string {
	t.Helper()
	token := geo.CellTokenAtLevel(geo.Point{Latitude: 31.2304, Longitude: 121.4737}, 8)
	if token == "" {
		t.Fatal("empty s2 cell token")
	}
	return `{"cell_token":"` + token + `","entity_type":"user"}`
}

func covHandlerCalls() []covHandlerCall {
	return []covHandlerCall{
		{name: "GetMe", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.GetMe(w, r) }, body: "{}"},
		{name: "ListPermissions", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.ListPermissions(w, r) }, body: "{}"},
		{name: "ListRoles", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.ListRoles(w, r) }, body: "{}"},
		{name: "CreateRole", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.CreateRole(w, r) }, body: `{"name":"role_cov"}`},
		{name: "UpdateRole", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.UpdateRole(w, r) }, body: `{"role_id":"role_cov"}`},
		{name: "DeleteRole", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.DeleteRole(w, r) }, body: `{"role_id":"role_cov"}`},
		{name: "SetRolePermissions", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.SetRolePermissions(w, r) }, body: `{"role_id":"role_cov","permission_keys":["admin.read"]}`},
		{name: "ListAdmins", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.ListAdmins(w, r) }, body: "{}"},
		{name: "GrantAdmin", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.GrantAdmin(w, r) }, body: `{"user_id":"user_bob","role_ids":["role_cov"]}`},
		{name: "RevokeAdmin", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.RevokeAdmin(w, r) }, body: `{"user_id":"user_bob"}`},
		{name: "ListAuditLogs", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.ListAuditLogs(w, r) }, body: "{}"},
		{name: "SearchUsers", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.SearchUsers(w, r) }, body: `{"nickname":"bob"}`},
		{name: "OpenAdminUser", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.OpenAdminUser(w, r) }, body: `{"user_id":"user_bob"}`},
		{name: "SetUserBan", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.SetUserBan(w, r) }, body: `{"user_id":"user_bob","active":true,"reason":"spam"}`},
		{name: "SearchGroups", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.SearchGroups(w, r) }, body: `{"name":"hot"}`},
		{name: "OpenAdminGroup", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.OpenAdminGroup(w, r) }, body: `{"group_id":"group_hotspot"}`},
		{name: "DismissGroup", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.DismissGroup(w, r) }, body: `{"group_id":"group_hotspot","reason":"policy"}`},
		{name: "SearchReports", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.SearchReports(w, r) }, body: "{}"},
		{name: "HandleReport", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.HandleReport(w, r) }, body: `{"report_id":"report_cov","action":"resolve"}`},
		{name: "GetAppConfig", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.GetAppConfig(w, r) }, body: "{}"},
		{name: "UpdateAppConfig", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.UpdateAppConfig(w, r) }, body: `{"limits":{"max_group_members":300}}`},
		{name: "SearchTags", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.SearchTags(w, r) }, body: `{"keyword":"hot"}`},
		{name: "SetTagBan", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.SetTagBan(w, r) }, body: `{"name":"tag_cov","active":true}`},
		{name: "ListAdminAnnouncements", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.ListAdminAnnouncements(w, r) }, body: "{}"},
		{name: "CreateAnnouncement", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.CreateAnnouncement(w, r) }, body: `{"title":"t","content":"c","severity":"info"}`, key: "cov1234abcd1234abcd1234abcd1234x"},
		{name: "UpdateAnnouncement", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.UpdateAnnouncement(w, r) }, body: `{"announcement_id":"ann_cov"}`},
		{name: "DeleteAnnouncement", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.DeleteAnnouncement(w, r) }, body: `{"announcement_id":"ann_cov"}`},
	}
}

func covRequestBody(t *testing.T, tc covHandlerCall) string {
	if tc.name == "StatsGeoDrilldown" {
		return covStatsDrilldownBody(t)
	}
	return tc.body
}

func TestHandlerDecodeErrorBranches(t *testing.T) {
	handler := newTestHandler()
	for _, tc := range covHandlerCalls() {
		t.Run(tc.name, func(t *testing.T) {
			req := adminContext(httptest.NewRequest(http.MethodPost, "/admin/api/v1/test", strings.NewReader("{")), "admin", true)
			if tc.key != "" {
				req.Header.Set("Idempotency-Key", tc.key)
			}
			rr := httptest.NewRecorder()
			tc.call(handler, rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
			}
			resp := decodeResponse[struct{}](t, rr)
			if resp.Code != httpx.CodeInvalidRequest {
				t.Fatalf("decode error should map to %d, got %#v", httpx.CodeInvalidRequest, resp)
			}
		})
	}
}

func TestHandlerPermissionErrorBranches(t *testing.T) {
	// user_bob 不是管理员：Service 的权限校验返回 10503，handler 走 writeAPIError。
	handler := newTestHandler()
	for _, tc := range covHandlerCalls() {
		t.Run(tc.name, func(t *testing.T) {
			req := adminContext(httptest.NewRequest(http.MethodPost, "/admin/api/v1/test", strings.NewReader(covRequestBody(t, tc))), "user_bob", false)
			if tc.key != "" {
				req.Header.Set("Idempotency-Key", tc.key)
			}
			rr := httptest.NewRecorder()
			tc.call(handler, rr, req)
			resp := decodeResponse[struct{}](t, rr)
			if resp.Code != httpx.CodeForbidden {
				t.Fatalf("expected %d, got %#v", httpx.CodeForbidden, resp)
			}
		})
	}
}

func TestHandlerServiceErrorBranches(t *testing.T) {
	// 管理员资料查询直接失败：所有 handler 走 writeUnavailable（10600）出口。
	handler := NewHandler(newCovService(covActorErrRepo{NewInMemoryRepository()}))
	for _, tc := range covHandlerCalls() {
		t.Run(tc.name, func(t *testing.T) {
			req := adminContext(httptest.NewRequest(http.MethodPost, "/admin/api/v1/test", strings.NewReader(covRequestBody(t, tc))), "admin", true)
			if tc.key != "" {
				req.Header.Set("Idempotency-Key", tc.key)
			}
			rr := httptest.NewRecorder()
			tc.call(handler, rr, req)
			resp := decodeResponse[struct{}](t, rr)
			if resp.Code != httpx.CodeInternalUnavailable {
				t.Fatalf("expected %d, got %#v", httpx.CodeInternalUnavailable, resp)
			}
		})
	}
}

func TestHandlerStatsGeoSuccessAndValidation(t *testing.T) {
	handler := newTestHandler()
	drilldownBody := covStatsDrilldownBody(t)

	t.Run("stats success", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler.StatsGeo(rr, adminContext(httptest.NewRequest(http.MethodPost, "/stats/geo", strings.NewReader("{}")), "admin", true))
		resp := decodeResponse[AdminGeoStatsResponse](t, rr)
		if resp.Code != 0 {
			t.Fatalf("unexpected response: %#v", resp)
		}
	})

	t.Run("stats invalid level", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler.StatsGeo(rr, adminContext(httptest.NewRequest(http.MethodPost, "/stats/geo", strings.NewReader(`{"level":99}`)), "admin", true))
		resp := decodeResponse[struct{}](t, rr)
		if resp.Code != httpx.CodeInvalidRequest {
			t.Fatalf("unexpected response: %#v", resp)
		}
	})

	t.Run("drilldown success", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler.StatsGeoDrilldown(rr, adminContext(httptest.NewRequest(http.MethodPost, "/stats/geo/drilldown", strings.NewReader(drilldownBody)), "admin", true))
		resp := decodeResponse[AdminGeoDrilldownPage](t, rr)
		if resp.Code != 0 {
			t.Fatalf("unexpected response: %#v", resp)
		}
	})

	t.Run("drilldown invalid entity type", func(t *testing.T) {
		rr := httptest.NewRecorder()
		handler.StatsGeoDrilldown(rr, adminContext(httptest.NewRequest(http.MethodPost, "/stats/geo/drilldown", strings.NewReader(`{"cell_token":"abc","entity_type":"bogus"}`)), "admin", true))
		resp := decodeResponse[struct{}](t, rr)
		if resp.Code != httpx.CodeInvalidRequest {
			t.Fatalf("unexpected response: %#v", resp)
		}
	})
}

func TestHandlerStatsGeoRepositoryError(t *testing.T) {
	handler := NewHandler(newCovService(covStatsErrRepo{NewInMemoryRepository()}))
	for _, tc := range []struct {
		name string
		call func(*Handler, http.ResponseWriter, *http.Request)
		body string
	}{
		{name: "stats", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.StatsGeo(w, r) }, body: "{}"},
		{name: "drilldown", call: func(h *Handler, w http.ResponseWriter, r *http.Request) { h.StatsGeoDrilldown(w, r) }, body: covStatsDrilldownBody(t)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.call(handler, rr, adminContext(httptest.NewRequest(http.MethodPost, "/stats", strings.NewReader(tc.body)), "admin", true))
			resp := decodeResponse[struct{}](t, rr)
			if resp.Code != httpx.CodeInternalUnavailable {
				t.Fatalf("unexpected response: %#v", resp)
			}
		})
	}
}

func TestHandlerCreateAnnouncementInnerErrorBranches(t *testing.T) {
	body := `{"title":"t","content":"c","severity":"info"}`
	t.Run("service error", func(t *testing.T) {
		handler := NewHandler(newCovService(covActorErrRepo{NewInMemoryRepository()}))
		req := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(body)), "admin", true)
		req.Header.Set("Idempotency-Key", "covann1234abcd1234abcd1234abcd12x")
		rr := httptest.NewRecorder()
		handler.CreateAnnouncement(rr, req)
		resp := decodeResponse[struct{}](t, rr)
		if resp.Code != httpx.CodeInternalUnavailable {
			t.Fatalf("unexpected response: %#v", resp)
		}
	})

	t.Run("business error", func(t *testing.T) {
		handler := newTestHandler()
		req := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(body)), "user_bob", false)
		req.Header.Set("Idempotency-Key", "covann5678abcd1234abcd1234abcd12x")
		rr := httptest.NewRecorder()
		handler.CreateAnnouncement(rr, req)
		resp := decodeResponse[struct{}](t, rr)
		if resp.Code != httpx.CodeForbidden {
			t.Fatalf("unexpected response: %#v", resp)
		}
	})
}

func TestHandlerMethodNameFromChiRouteContext(t *testing.T) {
	handler := newTestHandler()
	invalidJSON := func() *http.Request {
		req := adminContext(httptest.NewRequest(http.MethodPost, "/admin/api/v1/rbac/roles/fetch", strings.NewReader("{")), "admin", true)
		return req
	}

	t.Run("route pattern present", func(t *testing.T) {
		req := invalidJSON()
		rctx := chi.NewRouteContext()
		rctx.RoutePatterns = []string{"/admin/api/v1/rbac/roles/fetch"}
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()
		handler.ListRoles(rr, req)
		resp := decodeResponse[struct{}](t, rr)
		if resp.Code != httpx.CodeInvalidRequest {
			t.Fatalf("unexpected response: %#v", resp)
		}
	})

	t.Run("route context without pattern", func(t *testing.T) {
		req := invalidJSON()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, chi.NewRouteContext()))
		rr := httptest.NewRecorder()
		handler.ListRoles(rr, req)
		resp := decodeResponse[struct{}](t, rr)
		if resp.Code != httpx.CodeInvalidRequest {
			t.Fatalf("unexpected response: %#v", resp)
		}
	})
}

// 断言 handler 错误响应仍是统一 envelope（HTTP 200 + 业务错误码）。
func TestHandlerErrorResponsesKeepEnvelope(t *testing.T) {
	handler := newTestHandler()
	rr := httptest.NewRecorder()
	handler.ListRoles(rr, adminContext(httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("{")), "admin", true))
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
	var envelope map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if _, ok := envelope["code"]; !ok {
		t.Fatalf("missing code in envelope: %#v", envelope)
	}
}

var _ = platformauth.Principal{}
