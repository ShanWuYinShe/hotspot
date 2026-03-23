package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

func newTestHandler() *Handler {
	store := NewInMemoryRepository()
	provider := NewAppConfigProvider(appconfig.NewStaticSource(config.Load()))
	return NewHandler(NewService(store, provider))
}

func adminContext(req *http.Request, userID string, admin bool) *http.Request {
	principal := platformauth.Principal{UserID: userID, SessionID: "sess_" + userID}
	if admin {
		principal.AdminMembership = &platformauth.AdminMembership{}
	}
	return req.WithContext(platformauth.WithPrincipal(req.Context(), principal))
}

func decodeResponse[T any](t *testing.T, rr *httptest.ResponseRecorder) httpx.ApiResponse[T] {
	t.Helper()
	var resp httpx.ApiResponse[T]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func TestGetMeReturnsPermissions(t *testing.T) {
	handler := newTestHandler()
	req := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/get_me", strings.NewReader(`{}`)), "admin", true)
	rr := httptest.NewRecorder()

	handler.GetMe(rr, req)

	resp := decodeResponse[AdminPermissions](t, rr)
	if resp.Code != 0 || resp.Data == nil || len(resp.Data.Permissions) == 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestGetMeReturnsNonAdminMessage(t *testing.T) {
	handler := newTestHandler()
	req := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/get_me", strings.NewReader(`{}`)), "user_alice", true)
	rr := httptest.NewRecorder()

	handler.GetMe(rr, req)

	resp := decodeResponse[struct{}](t, rr)
	if resp.Code != 10200 || resp.Message != "非系统管理员" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestListPermissionsRequiresRbacPermission(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())

	createRoleReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/roles/create", strings.NewReader(`{"name":"Viewer"}`)), "admin", true)
	createRoleReq.Header.Set("Idempotency-Key", "zzzz1234abcd1234abcd1234abcd1234")
	createRoleRR := httptest.NewRecorder()
	handler.CreateRole(createRoleRR, createRoleReq)
	createdRole := decodeResponse[AdminRoleItem](t, createRoleRR)
	if createdRole.Code != 0 || createdRole.Data == nil {
		t.Fatalf("unexpected create role response: %#v", createdRole)
	}

	grantReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/admins/grant", strings.NewReader(`{"user_id":"user_alice","role_ids":["`+createdRole.Data.RoleID+`"]}`)), "admin", true)
	grantReq.Header.Set("Idempotency-Key", "yyyy1234abcd1234abcd1234abcd1234")
	grantRR := httptest.NewRecorder()
	handler.GrantAdmin(grantRR, grantReq)
	grantResp := decodeResponse[struct{}](t, grantRR)
	if grantResp.Code != 0 {
		t.Fatalf("unexpected grant response: %#v", grantResp)
	}

	body := bytes.NewBufferString(`{}`)
	req := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/list_permissions", body), "user_alice", true)
	rr := httptest.NewRecorder()

	handler.ListPermissions(rr, req)

	resp := decodeResponse[struct{}](t, rr)
	if resp.Code != 10200 || resp.Message != "系统管理员权限不足: admin.rbac.manage" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestCreateRoleRequiresIdempotencyKey(t *testing.T) {
	handler := newTestHandler()
	req := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/roles/create", strings.NewReader(`{"name":"Ops"}`)), "admin", true)
	rr := httptest.NewRecorder()

	handler.CreateRole(rr, req)

	resp := decodeResponse[AdminRoleItem](t, rr)
	if resp.Code != 22000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestCreateRoleReplaysSameIdempotencyKey(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())

	makeReq := func() *http.Request {
		req := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/roles/create", strings.NewReader(`{"name":"Ops","description":"Operators"}`)), "admin", true)
		req.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
		return req
	}

	first := httptest.NewRecorder()
	handler.CreateRole(first, makeReq())
	firstResp := decodeResponse[AdminRoleItem](t, first)
	if firstResp.Code != 0 || firstResp.Data == nil {
		t.Fatalf("unexpected first response: %#v", firstResp)
	}

	second := httptest.NewRecorder()
	handler.CreateRole(second, makeReq())
	secondResp := decodeResponse[AdminRoleItem](t, second)
	if secondResp.Code != 0 || secondResp.Data == nil || secondResp.Data.RoleID != firstResp.Data.RoleID {
		t.Fatalf("unexpected second response: %#v", secondResp)
	}
}

func TestGrantAdminRejectsMutuallyExclusiveFields(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	req := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/admins/grant", strings.NewReader(`{"user_id":"user_bob","grant_all_permissions":true,"role_ids":["role_x"]}`)), "admin", true)
	req.Header.Set("Idempotency-Key", "bbbb1234abcd1234abcd1234abcd1234")
	rr := httptest.NewRecorder()

	handler.GrantAdmin(rr, req)

	resp := decodeResponse[struct{}](t, rr)
	if resp.Code != 10000 || resp.Message != "grant_all_permissions 与 role_ids 互斥" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestGrantAdminAndListAdmins(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())

	createRoleReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/roles/create", strings.NewReader(`{"name":"Reviewer"}`)), "admin", true)
	createRoleReq.Header.Set("Idempotency-Key", "cccc1234abcd1234abcd1234abcd1234")
	createRoleRR := httptest.NewRecorder()
	handler.CreateRole(createRoleRR, createRoleReq)
	createdRole := decodeResponse[AdminRoleItem](t, createRoleRR)
	if createdRole.Code != 0 || createdRole.Data == nil {
		t.Fatalf("unexpected create role response: %#v", createdRole)
	}

	grantReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/admins/grant", strings.NewReader(`{"user_id":"user_bob","role_ids":["`+createdRole.Data.RoleID+`"]}`)), "admin", true)
	grantReq.Header.Set("Idempotency-Key", "dddd1234abcd1234abcd1234abcd1234")
	grantRR := httptest.NewRecorder()
	handler.GrantAdmin(grantRR, grantReq)
	grantResp := decodeResponse[struct{}](t, grantRR)
	if grantResp.Code != 0 {
		t.Fatalf("unexpected grant response: %#v", grantResp)
	}

	listReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/admins/list", strings.NewReader(`{"pagination":{"limit":10}}`)), "admin", true)
	listRR := httptest.NewRecorder()
	handler.ListAdmins(listRR, listReq)
	listResp := decodeResponse[PaginatedAdminOperators](t, listRR)
	if listResp.Code != 0 || listResp.Data == nil || len(listResp.Data.Items) != 1 {
		t.Fatalf("unexpected list response: %#v", listResp)
	}
	item := listResp.Data.Items[0]
	if item.User.UserID != "user_bob" || len(item.Roles) != 1 || item.Roles[0].RoleID != createdRole.Data.RoleID {
		t.Fatalf("unexpected admin item: %#v", item)
	}
}

func TestGetAppConfigRequiresPermission(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())

	createRoleReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/roles/create", strings.NewReader(`{"name":"Operator"}`)), "admin", true)
	createRoleReq.Header.Set("Idempotency-Key", "xxxx1234abcd1234abcd1234abcd1234")
	createRoleRR := httptest.NewRecorder()
	handler.CreateRole(createRoleRR, createRoleReq)
	createdRole := decodeResponse[AdminRoleItem](t, createRoleRR)
	if createdRole.Code != 0 || createdRole.Data == nil {
		t.Fatalf("unexpected create role response: %#v", createdRole)
	}

	grantReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/admins/grant", strings.NewReader(`{"user_id":"user_bob","role_ids":["`+createdRole.Data.RoleID+`"]}`)), "admin", true)
	grantReq.Header.Set("Idempotency-Key", "wwww1234abcd1234abcd1234abcd1234")
	grantRR := httptest.NewRecorder()
	handler.GrantAdmin(grantRR, grantReq)
	grantResp := decodeResponse[struct{}](t, grantRR)
	if grantResp.Code != 0 {
		t.Fatalf("unexpected grant response: %#v", grantResp)
	}

	req := adminContext(httptest.NewRequest(http.MethodPost, "/app_config/get", strings.NewReader(`{}`)), "user_bob", true)
	rr := httptest.NewRecorder()

	handler.GetAppConfig(rr, req)

	resp := decodeResponse[AppConfig](t, rr)
	if resp.Code != 10200 || resp.Message != "系统管理员权限不足: admin.app_config.read" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
