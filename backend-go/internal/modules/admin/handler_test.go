package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	systemmodule "hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/idempotency"
)

func newTestHandler() *Handler {
	store := NewInMemoryRepository()
	provider := NewAppConfigProvider(appconfig.NewStaticSource(config.Load()))
	moderationStore := NewInMemoryModerationStore()
	return NewHandler(NewService(store, provider, moderationStore, nil))
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

func TestSearchTagsRequiresPermission(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())

	createRoleReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/roles/create", strings.NewReader(`{"name":"TagViewer"}`)), "admin", true)
	createRoleReq.Header.Set("Idempotency-Key", "tagv1234abcd1234abcd1234abcd1234")
	createRoleRR := httptest.NewRecorder()
	handler.CreateRole(createRoleRR, createRoleReq)
	createdRole := decodeResponse[AdminRoleItem](t, createRoleRR)
	if createdRole.Code != 0 || createdRole.Data == nil {
		t.Fatalf("unexpected create role response: %#v", createdRole)
	}

	grantReq := adminContext(httptest.NewRequest(http.MethodPost, "/rbac/admins/grant", strings.NewReader(`{"user_id":"user_alice","role_ids":["`+createdRole.Data.RoleID+`"]}`)), "admin", true)
	grantReq.Header.Set("Idempotency-Key", "tagg1234abcd1234abcd1234abcd1234")
	grantRR := httptest.NewRecorder()
	handler.GrantAdmin(grantRR, grantReq)
	grantResp := decodeResponse[struct{}](t, grantRR)
	if grantResp.Code != 0 {
		t.Fatalf("unexpected grant response: %#v", grantResp)
	}

	req := adminContext(httptest.NewRequest(http.MethodPost, "/tags/search", strings.NewReader(`{"keyword":"hot"}`)), "user_alice", true)
	rr := httptest.NewRecorder()
	handler.SearchTags(rr, req)

	resp := decodeResponse[PaginatedAdminTags](t, rr)
	if resp.Code != 10200 || resp.Message != "系统管理员权限不足: admin.tags.read" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestSearchTagsReturnsItems(t *testing.T) {
	handler := newTestHandler()
	req := adminContext(httptest.NewRequest(http.MethodPost, "/tags/search", strings.NewReader(`{"keyword":"hot","pagination":{"limit":10}}`)), "admin", true)
	rr := httptest.NewRecorder()

	handler.SearchTags(rr, req)

	resp := decodeResponse[PaginatedAdminTags](t, rr)
	if resp.Code != 0 || resp.Data == nil || len(resp.Data.Items) != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if resp.Data.Items[0].Name != "hotspot" || resp.Data.Items[0].UsageCount != 3 {
		t.Fatalf("unexpected tag item: %#v", resp.Data.Items[0])
	}
}

func TestDeleteTagIsIdempotent(t *testing.T) {
	handler := newTestHandler()
	makeReq := func() *http.Request {
		return adminContext(httptest.NewRequest(http.MethodPost, "/tags/delete", strings.NewReader(`{"name":"hotspot"}`)), "admin", true)
	}
	rr := httptest.NewRecorder()
	handler.DeleteTag(rr, makeReq())
	resp := decodeResponse[struct{}](t, rr)
	if resp.Code != 0 {
		t.Fatalf("unexpected first response: %#v", resp)
	}
	second := httptest.NewRecorder()
	handler.DeleteTag(second, makeReq())
	secondResp := decodeResponse[struct{}](t, second)
	if secondResp.Code != 0 {
		t.Fatalf("unexpected second response: %#v", secondResp)
	}
}

func TestCreateAnnouncementRequiresIdempotencyKey(t *testing.T) {
	handler := newTestHandler()
	req := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(`{"title":"维护","content":"今晚维护","severity":"warning"}`)), "admin", true)
	rr := httptest.NewRecorder()

	handler.CreateAnnouncement(rr, req)

	resp := decodeResponse[AnnouncementCreated](t, rr)
	if resp.Code != 22000 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestCreateAnnouncementReplaysSameIdempotencyKey(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	makeReq := func() *http.Request {
		req := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(`{"title":"维护","content":"今晚维护","severity":"warning","send_push":true}`)), "admin", true)
		req.Header.Set("Idempotency-Key", "annn1234abcd1234abcd1234abcd1234")
		return req
	}

	first := httptest.NewRecorder()
	handler.CreateAnnouncement(first, makeReq())
	firstResp := decodeResponse[AnnouncementCreated](t, first)
	if firstResp.Code != 0 || firstResp.Data == nil || firstResp.Data.AnnouncementID == "" {
		t.Fatalf("unexpected first response: %#v", firstResp)
	}

	second := httptest.NewRecorder()
	handler.CreateAnnouncement(second, makeReq())
	secondResp := decodeResponse[AnnouncementCreated](t, second)
	if secondResp.Code != 0 || secondResp.Data == nil || secondResp.Data.AnnouncementID != firstResp.Data.AnnouncementID {
		t.Fatalf("unexpected second response: %#v", secondResp)
	}
}

func TestSearchAnnouncementsReturnsAdminFields(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	createReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(`{"title":"维护","content":"今晚维护","severity":"warning","action_url":"https://example.com/maintenance","send_push":true}`)), "admin", true)
	createReq.Header.Set("Idempotency-Key", "annn2234abcd1234abcd1234abcd1234")
	createRR := httptest.NewRecorder()
	handler.CreateAnnouncement(createRR, createReq)
	created := decodeResponse[AnnouncementCreated](t, createRR)
	if created.Code != 0 || created.Data == nil {
		t.Fatalf("unexpected create response: %#v", created)
	}

	searchReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/search", strings.NewReader(`{"severity":"warning","pagination":{"limit":10}}`)), "admin", true)
	searchRR := httptest.NewRecorder()
	handler.SearchAnnouncements(searchRR, searchReq)
	resp := decodeResponse[PaginatedAdminAnnouncements](t, searchRR)
	if resp.Code != 0 || resp.Data == nil || len(resp.Data.Items) != 1 {
		t.Fatalf("unexpected search response: %#v", resp)
	}
	item := resp.Data.Items[0]
	if item.AnnouncementID != created.Data.AnnouncementID || item.CreatedBy.UserID != "admin" || !item.SendPush || item.PushSentAt == "" || item.ActionURL != "https://example.com/maintenance" {
		t.Fatalf("unexpected announcement item: %#v", item)
	}
}

func TestUpdateAndDeleteAnnouncementAreIdempotent(t *testing.T) {
	handler := newTestHandler()
	idempotency.SetStore(idempotency.NewInMemoryStore())
	createReq := adminContext(httptest.NewRequest(http.MethodPost, "/announcements/create", strings.NewReader(`{"title":"维护","content":"今晚维护","severity":"warning"}`)), "admin", true)
	createReq.Header.Set("Idempotency-Key", "annn3234abcd1234abcd1234abcd1234")
	createRR := httptest.NewRecorder()
	handler.CreateAnnouncement(createRR, createReq)
	created := decodeResponse[AnnouncementCreated](t, createRR)
	if created.Code != 0 || created.Data == nil {
		t.Fatalf("unexpected create response: %#v", created)
	}

	makeUpdateReq := func() *http.Request {
		return adminContext(httptest.NewRequest(http.MethodPost, "/announcements/update", strings.NewReader(`{"announcement_id":"`+created.Data.AnnouncementID+`","title":"恢复","content":"已恢复","severity":"info","action_url":""}`)), "admin", true)
	}
	updateRR := httptest.NewRecorder()
	handler.UpdateAnnouncement(updateRR, makeUpdateReq())
	updateResp := decodeResponse[struct{}](t, updateRR)
	if updateResp.Code != 0 {
		t.Fatalf("unexpected update response: %#v", updateResp)
	}
	updateAgain := httptest.NewRecorder()
	handler.UpdateAnnouncement(updateAgain, makeUpdateReq())
	updateAgainResp := decodeResponse[struct{}](t, updateAgain)
	if updateAgainResp.Code != 0 {
		t.Fatalf("unexpected second update response: %#v", updateAgainResp)
	}

	makeSearchReq := func() *http.Request {
		return adminContext(httptest.NewRequest(http.MethodPost, "/announcements/search", strings.NewReader(`{"pagination":{"limit":10}}`)), "admin", true)
	}
	searchRR := httptest.NewRecorder()
	handler.SearchAnnouncements(searchRR, makeSearchReq())
	searchResp := decodeResponse[PaginatedAdminAnnouncements](t, searchRR)
	if searchResp.Code != 0 || searchResp.Data == nil || len(searchResp.Data.Items) != 1 {
		t.Fatalf("unexpected search response after update: %#v", searchResp)
	}
	if searchResp.Data.Items[0].Title != "恢复" || searchResp.Data.Items[0].Severity != systemmodule.AnnouncementSeverityInfo || searchResp.Data.Items[0].ActionURL != "" {
		t.Fatalf("unexpected updated item: %#v", searchResp.Data.Items[0])
	}

	makeDeleteReq := func() *http.Request {
		return adminContext(httptest.NewRequest(http.MethodPost, "/announcements/delete", strings.NewReader(`{"announcement_id":"`+created.Data.AnnouncementID+`"}`)), "admin", true)
	}
	deleteRR := httptest.NewRecorder()
	handler.DeleteAnnouncement(deleteRR, makeDeleteReq())
	deleteResp := decodeResponse[struct{}](t, deleteRR)
	if deleteResp.Code != 0 {
		t.Fatalf("unexpected delete response: %#v", deleteResp)
	}
	deleteAgain := httptest.NewRecorder()
	handler.DeleteAnnouncement(deleteAgain, makeDeleteReq())
	deleteAgainResp := decodeResponse[struct{}](t, deleteAgain)
	if deleteAgainResp.Code != 0 {
		t.Fatalf("unexpected second delete response: %#v", deleteAgainResp)
	}

	searchAfterDelete := httptest.NewRecorder()
	handler.SearchAnnouncements(searchAfterDelete, makeSearchReq())
	searchAfterDeleteResp := decodeResponse[PaginatedAdminAnnouncements](t, searchAfterDelete)
	if searchAfterDeleteResp.Code != 0 || searchAfterDeleteResp.Data == nil || len(searchAfterDeleteResp.Data.Items) != 0 {
		t.Fatalf("unexpected search response after delete: %#v", searchAfterDeleteResp)
	}
}
