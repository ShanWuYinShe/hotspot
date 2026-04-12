package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"hotspot/backend-go/internal/modules/notifications"
	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
	"hotspot/backend-go/internal/platform/realtime"
)

const (
	testAuthTokenSecret    = "test-secret"
	defaultAuthTokenSecret = "dev-secret-change-me"
)

func newTestHandler() *Handler {
	handler, _ := newTestHandlerWithStore()
	return handler
}

func newTestModuleWithStore() (*Module, *InMemoryRepository) {
	store := NewInMemoryRepository()
	provider := NewAppConfigProvider(appconfig.NewStaticSource(config.Load()))
	return NewModuleWithDeps(store, provider, nil, nil), store
}

func newTestModuleWithPushStore() (*Module, *InMemoryRepository) {
	store := NewInMemoryRepository()
	provider := NewAppConfigProvider(appconfig.NewStaticSource(config.Load()))
	pushService := notifications.NewService()
	return NewModuleWithDeps(store, provider, nil, pushService), store
}

func newTestHandlerWithStore() (*Handler, *InMemoryRepository) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	module, store := newTestModuleWithStore()
	return module.handler, store
}

func newTestHandlerWithPushStore() (*Handler, *InMemoryRepository) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	module, store := newTestModuleWithPushStore()
	return module.handler, store
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

func newMountedAdminRouter(t *testing.T) (http.Handler, func(string) string) {
	t.Helper()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	previousHub := realtime.DefaultHub()
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	t.Cleanup(func() {
		realtime.SetDefaultHub(previousHub)
	})
	module, _ := newTestModuleWithStore()
	authService := platformauth.NewService(config.Config{AuthTokenSecret: testAuthTokenSecret})
	platformauth.SetDefaultService(authService)
	t.Cleanup(func() {
		platformauth.SetDefaultService(platformauth.NewService(config.Config{AuthTokenSecret: defaultAuthTokenSecret}))
	})
	router := chi.NewRouter()
	module.Mount(router)
	issueToken := func(userID string) string {
		result, err := authService.IssueSession(userID)
		if err != nil {
			t.Fatalf("issue session: %v", err)
		}
		return result.AccessToken
	}
	return router, issueToken
}

func TestAdminModerationRoutesAreMounted(t *testing.T) {
	router, issueToken := newMountedAdminRouter(t)
	adminToken := issueToken("admin")

	tests := []struct {
		name string
		path string
		body string
	}{
		{name: "users search", path: "/api/v1/admin/users/search", body: `{"nickname":"bob","pagination":{"limit":10}}`},
		{name: "users open", path: "/api/v1/admin/users/open_the_door", body: `{"user_id":"user_bob"}`},
		{name: "groups search", path: "/api/v1/admin/groups/search", body: `{"name":"hot","pagination":{"limit":10}}`},
		{name: "groups open", path: "/api/v1/admin/groups/open_the_door", body: `{"group_id":"group_hotspot"}`},
		{name: "reports search", path: "/api/v1/admin/reports/search", body: `{"status":"pending","pagination":{"limit":10}}`},
		{name: "tags search", path: "/api/v1/admin/tags/search", body: `{"keyword":"hot","pagination":{"limit":10}}`},
		{name: "announcements search", path: "/api/v1/admin/announcements/search", body: `{"pagination":{"limit":10}}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(tc.body))
			req.Header.Set("Authorization", "Bearer "+adminToken)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
			}
			resp := decodeResponse[struct{}](t, rr)
			if resp.Code != 0 {
				t.Fatalf("unexpected response: %#v", resp)
			}
		})
	}
}

func TestAdminModerationRoutesRequireAuth(t *testing.T) {
	router, _ := newMountedAdminRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/search", strings.NewReader(`{"nickname":"bob"}`))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse[struct{}](t, rr)
	if resp.Code != 10100 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestAdminModerationRoutesRequireAdminPermission(t *testing.T) {
	router, issueToken := newMountedAdminRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/search", strings.NewReader(`{"nickname":"bob"}`))
	req.Header.Set("Authorization", "Bearer "+issueToken("user_bob"))
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", rr.Code, rr.Body.String())
	}
	resp := decodeResponse[struct{}](t, rr)
	if resp.Code != 10200 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestBanUserAuditUsesCloudflareCountry(t *testing.T) {
	handler, store := newTestHandlerWithStore()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	req := adminContext(httptest.NewRequest(http.MethodPost, "/users/ban", strings.NewReader(`{"user_id":"user_bob","reason":"spam"}`)), "admin", true)
	req.Header.Set("Idempotency-Key", "banusercountry1234def05678ghij9012")
	req.RemoteAddr = "173.245.48.5:443"
	req.Header.Set("CF-Connecting-IP", "8.8.8.8")
	req.Header.Set("CF-IPCountry", "JP")
	rr := httptest.NewRecorder()

	handler.BanUser(rr, req)

	resp := decodeResponse[struct{}](t, rr)
	if resp.Code != 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	for _, entry := range store.auditLogs {
		if entry.Action == "user.ban" {
			if entry.IPLocation != "JP" {
				t.Fatalf("unexpected audit location: %#v", entry)
			}
			return
		}
	}
	t.Fatalf("expected user.ban audit log, got %#v", store.auditLogs)
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

func TestBanUserRequiresIdempotencyKey(t *testing.T) {
	handler := newTestHandler()
	req := adminContext(httptest.NewRequest(http.MethodPost, "/users/ban", strings.NewReader(`{"user_id":"user_bob"}`)), "admin", true)
	rr := httptest.NewRecorder()

	handler.BanUser(rr, req)

	resp := decodeResponse[struct{}](t, rr)
	if resp.Code != 22000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestBanUserAndUnbanUser(t *testing.T) {
	handler, store := newTestHandlerWithStore()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	hub := realtime.DefaultHub()
	bobSink := &sinkCapture{}
	hub.RegisterUser("user_bob", "sess_user_bob_1", bobSink)

	banReq := adminContext(httptest.NewRequest(http.MethodPost, "/users/ban", strings.NewReader(`{"user_id":"user_bob","reason":"spam","duration_days":7}`)), "admin", true)
	banReq.Header.Set("Idempotency-Key", "banuser9abc1234def05678ghij9012klmn")
	banRR := httptest.NewRecorder()
	handler.BanUser(banRR, banReq)
	banResp := decodeResponse[struct{}](t, banRR)
	if banResp.Code != 0 {
		t.Fatalf("unexpected ban response: %#v", banResp)
	}
	state, ok, err := store.GetUserBanState(banReq.Context(), "user_bob")
	if err != nil || !ok || state.Status != UserAccountStatusBanned || state.DurationDays != 7 {
		t.Fatalf("unexpected ban state: %#v ok=%v err=%v", state, ok, err)
	}
	if sessions, _ := store.ListUserSessions(banReq.Context(), "user_bob"); len(sessions) != 0 {
		t.Fatalf("expected sessions revoked, got %#v", sessions)
	}
	foundBanAudit := false
	for _, entry := range store.auditLogs {
		if entry.Action == "user.ban" {
			foundBanAudit = true
			break
		}
	}
	if !foundBanAudit {
		t.Fatalf("unexpected audit logs after ban: %#v", store.auditLogs)
	}
	if len(bobSink.events) != 1 || bobSink.events[0].Type != "AccountBanned" || !bobSink.closed {
		t.Fatalf("unexpected ban event delivery: %#v closed=%v", bobSink.events, bobSink.closed)
	}

	unbanReq := adminContext(httptest.NewRequest(http.MethodPost, "/users/unban", strings.NewReader(`{"user_id":"user_bob","reason":"appeal approved"}`)), "admin", true)
	unbanRR := httptest.NewRecorder()
	handler.UnbanUser(unbanRR, unbanReq)
	unbanResp := decodeResponse[struct{}](t, unbanRR)
	if unbanResp.Code != 0 {
		t.Fatalf("unexpected unban response: %#v", unbanResp)
	}
	state, ok, err = store.GetUserBanState(unbanReq.Context(), "user_bob")
	if err != nil || !ok || state.Status != UserAccountStatusNormal {
		t.Fatalf("unexpected unban state: %#v ok=%v err=%v", state, ok, err)
	}
	foundUnbanAudit := false
	for _, entry := range store.auditLogs {
		if entry.Action == "user.unban" {
			foundUnbanAudit = true
			break
		}
	}
	if !foundUnbanAudit {
		t.Fatalf("unexpected audit logs after unban: %#v", store.auditLogs)
	}
}

func TestSearchUsersRequiresPermission(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())

	createRoleReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/roles/create", strings.NewReader(`{"name":"GroupModerator"}`)), "admin", true)
	createRoleReq.Header.Set("Idempotency-Key", "usread9abc1234def05678ghij9012klmn")
	createRoleRR := httptest.NewRecorder()
	handler.CreateRole(createRoleRR, createRoleReq)
	createdRole := decodeResponse[AdminRoleItem](t, createRoleRR)
	if createdRole.Code != 0 || createdRole.Data == nil {
		t.Fatalf("unexpected create role response: %#v", createdRole)
	}

	grantReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/admins/grant", strings.NewReader(`{"user_id":"user_bob","role_ids":["`+createdRole.Data.RoleID+`"]}`)), "admin", true)
	grantReq.Header.Set("Idempotency-Key", "usread9abc1234def05678ghij9012klmo")
	grantRR := httptest.NewRecorder()
	handler.GrantAdmin(grantRR, grantReq)
	grantResp := decodeResponse[struct{}](t, grantRR)
	if grantResp.Code != 0 {
		t.Fatalf("unexpected grant response: %#v", grantResp)
	}

	req := adminContext(httptest.NewRequest(http.MethodPost, "/users/search", strings.NewReader(`{}`)), "user_bob", true)
	rr := httptest.NewRecorder()
	handler.SearchUsers(rr, req)
	resp := decodeResponse[PaginatedAdminUsers](t, rr)
	if resp.Code != 10200 || resp.Message != "系统管理员权限不足: admin.users.read" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSearchUsersReturnsResults(t *testing.T) {
	handler := newTestHandler()
	req := adminContext(httptest.NewRequest(http.MethodPost, "/users/search", strings.NewReader(`{"nickname":"bob","pagination":{"limit":10}}`)), "admin", true)
	rr := httptest.NewRecorder()

	handler.SearchUsers(rr, req)

	resp := decodeResponse[PaginatedAdminUsers](t, rr)
	if resp.Code != 0 || resp.Data == nil || len(resp.Data.Items) != 1 || resp.Data.Items[0].User.UserID != "user_bob" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestOpenAdminUserReturnsNotFound(t *testing.T) {
	handler := newTestHandler()
	req := adminContext(httptest.NewRequest(http.MethodPost, "/users/open_the_door", strings.NewReader(`{"user_id":"missing_user"}`)), "admin", true)
	rr := httptest.NewRecorder()

	handler.OpenAdminUser(rr, req)

	resp := decodeResponse[AdminUserItem](t, rr)
	if resp.Code != 10300 || resp.Message != "用户不存在" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

type sinkCapture struct {
	events []realtime.Event
	closed bool
}

func (s *sinkCapture) Send(e realtime.Event) error {
	s.events = append(s.events, e)
	return nil
}

func (s *sinkCapture) Close() error {
	s.closed = true
	return nil
}

func TestSearchGroupsAndDismissGroup(t *testing.T) {
	handler, store := newTestHandlerWithStore()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	hub := realtime.DefaultHub()
	aliceSink := &sinkCapture{}
	bobSink := &sinkCapture{}
	groupSink := &sinkCapture{}
	hub.RegisterUser("user_alice", "sess_user_alice_1", aliceSink)
	hub.RegisterUser("user_bob", "sess_user_bob_1", bobSink)
	hub.SetGroupSubscriptions([]string{"group_hotspot"}, groupSink)

	searchReq := adminContext(httptest.NewRequest(http.MethodPost, "/groups/search", strings.NewReader(`{"name":"hot","pagination":{"limit":10}}`)), "admin", true)
	searchRR := httptest.NewRecorder()
	handler.SearchGroups(searchRR, searchReq)
	searchResp := decodeResponse[PaginatedAdminGroups](t, searchRR)
	if searchResp.Code != 0 || searchResp.Data == nil || len(searchResp.Data.Items) != 1 || searchResp.Data.Items[0].Group.GroupID != "group_hotspot" {
		t.Fatalf("unexpected search response: %#v", searchResp)
	}

	dismissReq := adminContext(httptest.NewRequest(http.MethodPost, "/groups/dismiss", strings.NewReader(`{"group_id":"group_hotspot","reason":"policy"}`)), "admin", true)
	dismissReq.Header.Set("Idempotency-Key", "dismiss001abcd1234abcd1234abcd12")
	dismissRR := httptest.NewRecorder()
	handler.DismissGroup(dismissRR, dismissReq)
	dismissResp := decodeResponse[struct{}](t, dismissRR)
	if dismissResp.Code != 0 {
		t.Fatalf("unexpected dismiss response: %#v", dismissResp)
	}
	if _, found, _ := store.GetGroup(dismissReq.Context(), "group_hotspot"); found {
		t.Fatal("expected group to be removed")
	}
	if members, _ := store.ListGroupMemberIDs(dismissReq.Context(), "group_hotspot"); len(members) != 0 {
		t.Fatalf("expected group members removed, got %#v", members)
	}
	foundDismissAudit := false
	for _, entry := range store.auditLogs {
		if entry.Action == "group.dismiss" {
			foundDismissAudit = true
			break
		}
	}
	if !foundDismissAudit {
		t.Fatalf("unexpected audit logs: %#v", store.auditLogs)
	}
	if len(aliceSink.events) != 1 || aliceSink.events[0].Type != "GroupDismissed" {
		t.Fatalf("unexpected alice events: %#v", aliceSink.events)
	}
	if len(bobSink.events) != 1 || bobSink.events[0].Type != "GroupDismissed" {
		t.Fatalf("unexpected bob events: %#v", bobSink.events)
	}
	if len(groupSink.events) != 1 || groupSink.events[0].Type != "GroupDismissed" {
		t.Fatalf("unexpected group events: %#v", groupSink.events)
	}
}

func TestOpenAdminGroupReturnsNotFound(t *testing.T) {
	handler := newTestHandler()
	req := adminContext(httptest.NewRequest(http.MethodPost, "/groups/open_the_door", strings.NewReader(`{"group_id":"missing_group"}`)), "admin", true)
	rr := httptest.NewRecorder()

	handler.OpenAdminGroup(rr, req)

	resp := decodeResponse[AdminGroupItem](t, rr)
	if resp.Code != 10300 || resp.Message != "群组不存在" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSearchReportsAndHandleReport(t *testing.T) {
	handler, store := newTestHandlerWithStore()
	idempotency.SetStore(idempotency.NewInMemoryStore())

	searchReq := adminContext(httptest.NewRequest(http.MethodPost, "/reports/search", strings.NewReader(`{"status":"pending","pagination":{"limit":10}}`)), "admin", true)
	searchRR := httptest.NewRecorder()
	handler.SearchReports(searchRR, searchReq)
	searchResp := decodeResponse[PaginatedAdminReports](t, searchRR)
	if searchResp.Code != 0 || searchResp.Data == nil || len(searchResp.Data.Items) != 1 || searchResp.Data.Items[0].ReportID != "report_1" {
		t.Fatalf("unexpected search response: %#v", searchResp)
	}

	handleReq := adminContext(httptest.NewRequest(http.MethodPost, "/reports/handle", strings.NewReader(`{"report_id":"report_1","action":"resolve","note":"confirmed","ban_duration_days":3}`)), "admin", true)
	handleReq.Header.Set("Idempotency-Key", "report001abcd1234abcd1234abcd123")
	handleRR := httptest.NewRecorder()
	handler.HandleReport(handleRR, handleReq)
	handleResp := decodeResponse[struct{}](t, handleRR)
	if handleResp.Code != 0 {
		t.Fatalf("unexpected handle response: %#v", handleResp)
	}
	record, found, err := store.GetReportHandleRecord(handleReq.Context(), "report_1")
	if err != nil || !found || record.Item.Status != ReportStatusResolved || record.Item.HandleNote != "confirmed" {
		t.Fatalf("unexpected report record: %#v found=%v err=%v", record, found, err)
	}
	state, ok, err := store.GetUserBanState(handleReq.Context(), "user_bob")
	if err != nil || !ok || state.Status != UserAccountStatusBanned || state.DurationDays != 3 {
		t.Fatalf("unexpected user ban state after report handle: %#v ok=%v err=%v", state, ok, err)
	}
	foundHandleAudit := false
	for _, entry := range store.auditLogs {
		if entry.Action == "report.handle" {
			foundHandleAudit = true
			break
		}
	}
	if !foundHandleAudit {
		t.Fatalf("unexpected audit logs: %#v", store.auditLogs)
	}
}

func TestSearchTagsAndDeleteTag(t *testing.T) {
	handler, store := newTestHandlerWithStore()

	searchReq := adminContext(httptest.NewRequest(http.MethodPost, "/tags/search", strings.NewReader(`{"keyword":"hot","pagination":{"limit":10}}`)), "admin", true)
	searchRR := httptest.NewRecorder()
	handler.SearchTags(searchRR, searchReq)
	searchResp := decodeResponse[PaginatedAdminTags](t, searchRR)
	if searchResp.Code != 0 || searchResp.Data == nil || len(searchResp.Data.Items) != 1 || searchResp.Data.Items[0].Name != "hotspot" {
		t.Fatalf("unexpected search response: %#v", searchResp)
	}

	deleteReq := adminContext(httptest.NewRequest(http.MethodPost, "/tags/delete", strings.NewReader(`{"name":"hotspot"}`)), "admin", true)
	deleteRR := httptest.NewRecorder()
	handler.DeleteTag(deleteRR, deleteReq)
	deleteResp := decodeResponse[struct{}](t, deleteRR)
	if deleteResp.Code != 0 {
		t.Fatalf("unexpected delete response: %#v", deleteResp)
	}

	searchAfterReq := adminContext(httptest.NewRequest(http.MethodPost, "/tags/search", strings.NewReader(`{"keyword":"hot","pagination":{"limit":10}}`)), "admin", true)
	searchAfterRR := httptest.NewRecorder()
	handler.SearchTags(searchAfterRR, searchAfterReq)
	searchAfterResp := decodeResponse[PaginatedAdminTags](t, searchAfterRR)
	if searchAfterResp.Code != 0 || searchAfterResp.Data == nil || len(searchAfterResp.Data.Items) != 0 {
		t.Fatalf("unexpected search-after-delete response: %#v", searchAfterResp)
	}

	deleteAgainReq := adminContext(httptest.NewRequest(http.MethodPost, "/tags/delete", strings.NewReader(`{"name":"hotspot"}`)), "admin", true)
	deleteAgainRR := httptest.NewRecorder()
	handler.DeleteTag(deleteAgainRR, deleteAgainReq)
	deleteAgainResp := decodeResponse[struct{}](t, deleteAgainRR)
	if deleteAgainResp.Code != 0 {
		t.Fatalf("unexpected second delete response: %#v", deleteAgainResp)
	}
	missingReq := adminContext(httptest.NewRequest(http.MethodPost, "/tags/delete", strings.NewReader(`{"name":"missing"}`)), "admin", true)
	missingRR := httptest.NewRecorder()
	handler.DeleteTag(missingRR, missingReq)
	missingResp := decodeResponse[struct{}](t, missingRR)
	if missingResp.Code != 10300 || missingResp.Message != "标签不存在" {
		t.Fatalf("unexpected missing delete response: %#v", missingResp)
	}
	foundTagDeleteAudit := false
	tagDeleteAuditCount := 0
	for _, entry := range store.auditLogs {
		if entry.Action == AuditActionTagDelete {
			tagDeleteAuditCount++
			foundTagDeleteAudit = true
			if entry.TargetType != AuditTargetTypeTag || entry.TargetID != "hotspot" {
				t.Fatalf("unexpected tag delete audit target: %#v", entry)
			}
			if entry.Details["type"] != string(AuditActionTagDelete) || entry.Details["name"] != "hotspot" {
				t.Fatalf("unexpected tag delete audit details: %#v", entry.Details)
			}
		}
	}
	if !foundTagDeleteAudit {
		t.Fatalf("expected tag delete audit log, got %#v", store.auditLogs)
	}
	if tagDeleteAuditCount != 1 {
		t.Fatalf("expected exactly one tag delete audit log, got %d %#v", tagDeleteAuditCount, store.auditLogs)
	}
}

func TestCreateAnnouncementRequiresIdempotencyKey(t *testing.T) {
	handler := newTestHandler()
	req := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(`{"title":"维护通知","content":"今晚维护","severity":"warning"}`)), "admin", true)
	rr := httptest.NewRecorder()

	handler.CreateAnnouncement(rr, req)

	resp := decodeResponse[AnnouncementCreated](t, rr)
	if resp.Code != 22000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSearchAnnouncementsCreateUpdateAndDelete(t *testing.T) {
	handler, store := newTestHandlerWithPushStore()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	hub := realtime.DefaultHub()
	adminSink := &sinkCapture{}
	aliceSink := &sinkCapture{}
	hub.RegisterUser("admin", "sess_admin_1", adminSink)
	hub.RegisterUser("user_alice", "sess_user_alice_1", aliceSink)

	createReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(`{"title":"维护通知","content":"今晚维护","severity":"warning","action_url":"https://example.com/maint","send_push":true}`)), "admin", true)
	createReq.Header.Set("Idempotency-Key", "announce001abcd1234abcd1234abcd12")
	createRR := httptest.NewRecorder()
	handler.CreateAnnouncement(createRR, createReq)
	createResp := decodeResponse[AnnouncementCreated](t, createRR)
	if createResp.Code != 0 || createResp.Data == nil || createResp.Data.AnnouncementID == "" {
		t.Fatalf("unexpected create response: %#v", createResp)
	}
	if len(adminSink.events) != 1 || len(aliceSink.events) != 1 {
		t.Fatalf("unexpected announcement push events: %#v %#v", adminSink.events, aliceSink.events)
	}
	if adminSink.events[0].Type != "SystemAnnouncement" || aliceSink.events[0].Type != "SystemAnnouncement" {
		t.Fatalf("unexpected announcement event types: %#v %#v", adminSink.events, aliceSink.events)
	}
	adminPayload, ok := adminSink.events[0].Payload.(systemmodule.SystemAnnouncementItem)
	if !ok {
		t.Fatalf("unexpected admin payload type: %#v", adminSink.events[0].Payload)
	}
	alicePayload, ok := aliceSink.events[0].Payload.(systemmodule.SystemAnnouncementItem)
	if !ok {
		t.Fatalf("unexpected alice payload type: %#v", aliceSink.events[0].Payload)
	}
	if adminPayload.AnnouncementID != createResp.Data.AnnouncementID || adminPayload.Title != "维护通知" || adminPayload.Content != "今晚维护" || adminPayload.Severity != AnnouncementSeverityWarning || adminPayload.ActionURL != "https://example.com/maint" {
		t.Fatalf("unexpected admin payload: %#v", adminPayload)
	}
	if alicePayload != adminPayload {
		t.Fatalf("expected broadcast payloads to match: %#v %#v", adminPayload, alicePayload)
	}

	searchReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/search", strings.NewReader(`{"severity":"warning","pagination":{"limit":10}}`)), "admin", true)
	searchRR := httptest.NewRecorder()
	handler.SearchAnnouncements(searchRR, searchReq)
	searchResp := decodeResponse[PaginatedAdminAnnouncements](t, searchRR)
	if searchResp.Code != 0 || searchResp.Data == nil || len(searchResp.Data.Items) != 1 || searchResp.Data.Items[0].AnnouncementID != createResp.Data.AnnouncementID {
		t.Fatalf("unexpected search response: %#v", searchResp)
	}

	updateReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/update", strings.NewReader(`{"announcement_id":"`+createResp.Data.AnnouncementID+`","title":"维护结束","content":"服务恢复","severity":"info","action_url":""}`)), "admin", true)
	updateRR := httptest.NewRecorder()
	handler.UpdateAnnouncement(updateRR, updateReq)
	updateResp := decodeResponse[struct{}](t, updateRR)
	if updateResp.Code != 0 {
		t.Fatalf("unexpected update response: %#v", updateResp)
	}

	searchUpdatedReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/search", strings.NewReader(`{"severity":"info","pagination":{"limit":10}}`)), "admin", true)
	searchUpdatedRR := httptest.NewRecorder()
	handler.SearchAnnouncements(searchUpdatedRR, searchUpdatedReq)
	searchUpdatedResp := decodeResponse[PaginatedAdminAnnouncements](t, searchUpdatedRR)
	if searchUpdatedResp.Code != 0 || searchUpdatedResp.Data == nil || len(searchUpdatedResp.Data.Items) != 1 {
		t.Fatalf("unexpected updated search response: %#v", searchUpdatedResp)
	}
	updated := searchUpdatedResp.Data.Items[0]
	if updated.Title != "维护结束" || updated.Content != "服务恢复" || updated.ActionURL != "" {
		t.Fatalf("unexpected updated item: %#v", updated)
	}

	deleteReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/delete", strings.NewReader(`{"announcement_id":"`+createResp.Data.AnnouncementID+`"}`)), "admin", true)
	deleteRR := httptest.NewRecorder()
	handler.DeleteAnnouncement(deleteRR, deleteReq)
	deleteResp := decodeResponse[struct{}](t, deleteRR)
	if deleteResp.Code != 0 {
		t.Fatalf("unexpected delete response: %#v", deleteResp)
	}
	deleteAgainReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/delete", strings.NewReader(`{"announcement_id":"`+createResp.Data.AnnouncementID+`"}`)), "admin", true)
	deleteAgainRR := httptest.NewRecorder()
	handler.DeleteAnnouncement(deleteAgainRR, deleteAgainReq)
	deleteAgainResp := decodeResponse[struct{}](t, deleteAgainRR)
	if deleteAgainResp.Code != 0 {
		t.Fatalf("unexpected second delete response: %#v", deleteAgainResp)
	}

	searchAfterDeleteReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/search", strings.NewReader(`{"pagination":{"limit":10}}`)), "admin", true)
	searchAfterDeleteRR := httptest.NewRecorder()
	handler.SearchAnnouncements(searchAfterDeleteRR, searchAfterDeleteReq)
	searchAfterDeleteResp := decodeResponse[PaginatedAdminAnnouncements](t, searchAfterDeleteRR)
	if searchAfterDeleteResp.Code != 0 || searchAfterDeleteResp.Data == nil || len(searchAfterDeleteResp.Data.Items) < 1 {
		t.Fatalf("unexpected search-after-delete response: %#v", searchAfterDeleteResp)
	}
	var deleted AdminAnnouncementItem
	foundDeleted := false
	for _, item := range searchAfterDeleteResp.Data.Items {
		if item.AnnouncementID == createResp.Data.AnnouncementID {
			deleted = item
			foundDeleted = true
			break
		}
	}
	if !foundDeleted {
		t.Fatalf("expected deleted announcement in search results: %#v", searchAfterDeleteResp.Data.Items)
	}
	if deleted.AnnouncementID != createResp.Data.AnnouncementID || deleted.Title != "维护结束" {
		t.Fatalf("unexpected deleted announcement search result: %#v", deleted)
	}
	foundCreateAudit := false
	foundUpdateAudit := false
	foundDeleteAudit := false
	for _, entry := range store.auditLogs {
		switch entry.Action {
		case AuditActionAnnouncementCreate:
			foundCreateAudit = true
			if entry.TargetType != AuditTargetTypeAnnouncement || entry.TargetID != createResp.Data.AnnouncementID || entry.Details["title"] != "维护通知" {
				t.Fatalf("unexpected announcement create audit: %#v", entry)
			}
		case AuditActionAnnouncementUpdate:
			foundUpdateAudit = true
			changedFields, ok := entry.Details["changed_fields"].([]string)
			if !ok {
				t.Fatalf("unexpected changed_fields type: %#v", entry.Details)
			}
			if entry.TargetType != AuditTargetTypeAnnouncement || entry.TargetID != createResp.Data.AnnouncementID || entry.Details["title"] != "维护结束" || !containsString(changedFields, "title") || !containsString(changedFields, "content") || !containsString(changedFields, "severity") || !containsString(changedFields, "action_url") {
				t.Fatalf("unexpected announcement update audit: %#v", entry)
			}
		case AuditActionAnnouncementDelete:
			foundDeleteAudit = true
			if entry.TargetType != AuditTargetTypeAnnouncement || entry.TargetID != createResp.Data.AnnouncementID || entry.Details["title"] != "维护结束" {
				t.Fatalf("unexpected announcement delete audit: %#v", entry)
			}
		}
	}
	if !foundCreateAudit || !foundUpdateAudit || !foundDeleteAudit {
		t.Fatalf("missing announcement audit logs: %#v", store.auditLogs)
	}
	deleteAuditCount := 0
	for _, entry := range store.auditLogs {
		if entry.Action == AuditActionAnnouncementDelete {
			deleteAuditCount++
		}
	}
	if deleteAuditCount != 1 {
		t.Fatalf("expected exactly one delete audit log, got %d %#v", deleteAuditCount, store.auditLogs)
	}
}

func TestAnnouncementMutationsReturnNotFound(t *testing.T) {
	handler := newTestHandler()

	updateReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/update", strings.NewReader(`{"announcement_id":"missing","title":"维护结束"}`)), "admin", true)
	updateRR := httptest.NewRecorder()
	handler.UpdateAnnouncement(updateRR, updateReq)
	updateResp := decodeResponse[struct{}](t, updateRR)
	if updateResp.Code != 10300 || updateResp.Message != "公告不存在" {
		t.Fatalf("unexpected update response: %#v", updateResp)
	}

	deleteReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/delete", strings.NewReader(`{"announcement_id":"missing"}`)), "admin", true)
	deleteRR := httptest.NewRecorder()
	handler.DeleteAnnouncement(deleteRR, deleteReq)
	deleteResp := decodeResponse[struct{}](t, deleteRR)
	if deleteResp.Code != 10300 || deleteResp.Message != "公告不存在" {
		t.Fatalf("unexpected delete response: %#v", deleteResp)
	}
}
