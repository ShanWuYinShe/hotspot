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
	"hotspot/backend-go/internal/platform/realtime"
)

func newTestHandler() *Handler {
	handler, _ := newTestHandlerWithStore()
	return handler
}

func newTestHandlerWithStore() (*Handler, *InMemoryRepository) {
	hub := realtime.NewHub()
	realtime.SetDefaultHub(hub)
	store := NewInMemoryRepository()
	provider := NewAppConfigProvider(appconfig.NewStaticSource(config.Load()))
	return NewHandler(NewService(store, provider, nil, nil)), store
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
