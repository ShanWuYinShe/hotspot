package groups

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeGroupCode(t *testing.T, rr *httptest.ResponseRecorder) int {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.Code
}

func TestListApplicationsHandlerErrorPaths(t *testing.T) {
	// 参数错误 → 10000
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/applications/list", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.ListApplications(rr, req)
	if code := decodeGroupCode(t, rr); code != 10000 {
		t.Fatalf("empty group id: expected 10000, got %d", code)
	}

	// 群不存在 → 10300
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: false}))
	req = httptest.NewRequest(http.MethodPost, "/applications/list", strings.NewReader(`{"group_id":"g1"}`))
	rr = httptest.NewRecorder()
	handler.ListApplications(rr, req)
	if code := decodeGroupCode(t, rr); code != 10300 {
		t.Fatalf("missing group: expected 10300, got %d", code)
	}

	// 非管理员 → 10200
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: true, getResult: GroupItem{}}))
	req = httptest.NewRequest(http.MethodPost, "/applications/list", strings.NewReader(`{"group_id":"g1"}`))
	rr = httptest.NewRecorder()
	handler.ListApplications(rr, req)
	if code := decodeGroupCode(t, rr); code != 10200 {
		t.Fatalf("non-admin: expected 10200, got %d", code)
	}

	// 服务错误 → 10600
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: true, getResult: ownerGroup(), listApplicationsErr: errors.New("boom")}))
	req = httptest.NewRequest(http.MethodPost, "/applications/list", strings.NewReader(`{"group_id":"g1"}`))
	rr = httptest.NewRecorder()
	handler.ListApplications(rr, req)
	if code := decodeGroupCode(t, rr); code != 10600 {
		t.Fatalf("service error: expected 10600, got %d", code)
	}
}

func TestInviteMemberHandlerErrorPaths(t *testing.T) {
	// 参数错误 → 10000
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/members/invite", strings.NewReader(`{"group_id":"g1"}`))
	rr := httptest.NewRecorder()
	handler.InviteMember(rr, req)
	if code := decodeGroupCode(t, rr); code != 10000 {
		t.Fatalf("missing user id: expected 10000, got %d", code)
	}

	// 群不存在 → 10300
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: false}))
	req = httptest.NewRequest(http.MethodPost, "/members/invite", strings.NewReader(`{"group_id":"g1","user_id":"u2"}`))
	rr = httptest.NewRecorder()
	handler.InviteMember(rr, req)
	if code := decodeGroupCode(t, rr); code != 10300 {
		t.Fatalf("missing group: expected 10300, got %d", code)
	}

	// 服务错误 → 10600
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: true, getResult: ownerGroup(), createErr: errors.New("boom")}))
	req = httptest.NewRequest(http.MethodPost, "/members/invite", strings.NewReader(`{"group_id":"g1","user_id":"u2"}`))
	rr = httptest.NewRecorder()
	handler.InviteMember(rr, req)
	if code := decodeGroupCode(t, rr); code != 10600 {
		t.Fatalf("service error: expected 10600, got %d", code)
	}
}

func TestSetMemberRoleHandlerErrorPaths(t *testing.T) {
	// 非法角色 → 10000
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/members/set_role", strings.NewReader(`{"group_id":"g1","user_id":"u2","role":"boss"}`))
	rr := httptest.NewRecorder()
	handler.SetMemberRole(rr, req)
	if code := decodeGroupCode(t, rr); code != 10000 {
		t.Fatalf("bad role: expected 10000, got %d", code)
	}

	// 群不存在 → 10300
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: false}))
	req = httptest.NewRequest(http.MethodPost, "/members/set_role", strings.NewReader(`{"group_id":"g1","user_id":"u2","role":"admin"}`))
	rr = httptest.NewRecorder()
	handler.SetMemberRole(rr, req)
	if code := decodeGroupCode(t, rr); code != 10300 {
		t.Fatalf("missing group: expected 10300, got %d", code)
	}
}
