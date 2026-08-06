package groups

import (
	"encoding/json"
	"errors"
	"hotspot/backend-go/internal/platform/httpx"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func decodeGroupCode(t *testing.T, rr *httptest.ResponseRecorder) httpx.Code {
	t.Helper()
	var resp struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp.Code
}

func TestListApplicationsHandlerErrorPaths(t *testing.T) {
	// 参数错误 → httpx.CodeInvalidRequest
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/applications/list", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.ListApplications(rr, req)
	if code := decodeGroupCode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("empty group id: expected httpx.CodeInvalidRequest, got %d", code)
	}

	// 群不存在 → httpx.CodeNotFound
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: false}))
	req = httptest.NewRequest(http.MethodPost, "/applications/list", strings.NewReader(`{"group_id":"g1"}`))
	rr = httptest.NewRecorder()
	handler.ListApplications(rr, req)
	if code := decodeGroupCode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("missing group: expected httpx.CodeNotFound, got %d", code)
	}

	// 非管理员 → httpx.CodeForbidden
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: true, getResult: GroupItem{}}))
	req = httptest.NewRequest(http.MethodPost, "/applications/list", strings.NewReader(`{"group_id":"g1"}`))
	rr = httptest.NewRecorder()
	handler.ListApplications(rr, req)
	if code := decodeGroupCode(t, rr); code != httpx.CodeForbidden {
		t.Fatalf("non-admin: expected httpx.CodeForbidden, got %d", code)
	}

	// 服务错误 → httpx.CodeInternalUnavailable
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: true, getResult: ownerGroup(), listApplicationsErr: errors.New("boom")}))
	req = httptest.NewRequest(http.MethodPost, "/applications/list", strings.NewReader(`{"group_id":"g1"}`))
	rr = httptest.NewRecorder()
	handler.ListApplications(rr, req)
	if code := decodeGroupCode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("service error: expected httpx.CodeInternalUnavailable, got %d", code)
	}
}

func TestInviteMemberHandlerErrorPaths(t *testing.T) {
	// 参数错误 → httpx.CodeInvalidRequest
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/members/invite", strings.NewReader(`{"group_id":"g1"}`))
	rr := httptest.NewRecorder()
	handler.InviteMember(rr, req)
	if code := decodeGroupCode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("missing user id: expected httpx.CodeInvalidRequest, got %d", code)
	}

	// 群不存在 → httpx.CodeNotFound
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: false}))
	req = httptest.NewRequest(http.MethodPost, "/members/invite", strings.NewReader(`{"group_id":"g1","user_id":"u2"}`))
	rr = httptest.NewRecorder()
	handler.InviteMember(rr, req)
	if code := decodeGroupCode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("missing group: expected httpx.CodeNotFound, got %d", code)
	}

	// 服务错误 → httpx.CodeInternalUnavailable
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: true, getResult: ownerGroup(), createErr: errors.New("boom")}))
	req = httptest.NewRequest(http.MethodPost, "/members/invite", strings.NewReader(`{"group_id":"g1","user_id":"u2"}`))
	rr = httptest.NewRecorder()
	handler.InviteMember(rr, req)
	if code := decodeGroupCode(t, rr); code != httpx.CodeInternalUnavailable {
		t.Fatalf("service error: expected httpx.CodeInternalUnavailable, got %d", code)
	}
}

func TestSetMemberRoleHandlerErrorPaths(t *testing.T) {
	// 非法角色 → httpx.CodeInvalidRequest
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/members/set_role", strings.NewReader(`{"group_id":"g1","user_id":"u2","role":"boss"}`))
	rr := httptest.NewRecorder()
	handler.SetMemberRole(rr, req)
	if code := decodeGroupCode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("bad role: expected httpx.CodeInvalidRequest, got %d", code)
	}

	// 群不存在 → httpx.CodeNotFound
	handler = NewHandler(NewServiceWithRepo(&groupErrorRepo{getFound: false}))
	req = httptest.NewRequest(http.MethodPost, "/members/set_role", strings.NewReader(`{"group_id":"g1","user_id":"u2","role":"admin"}`))
	rr = httptest.NewRecorder()
	handler.SetMemberRole(rr, req)
	if code := decodeGroupCode(t, rr); code != httpx.CodeNotFound {
		t.Fatalf("missing group: expected httpx.CodeNotFound, got %d", code)
	}
}
