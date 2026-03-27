package groups

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

type groupErrorRepo struct {
	createErr error
	getErr    error
	searchErr error
	updateErr error
	dismissErr error
	listMembersErr error
	searchMembersErr error
	listMyGroupsErr error
	joinGroupErr error
}

func (r *groupErrorRepo) Create(context.Context, string, CreateGroupRequest) (GroupItem, error) {
	return GroupItem{}, r.createErr
}
func (r *groupErrorRepo) Get(context.Context, string) (GroupItem, bool, error) {
	return GroupItem{}, false, r.getErr
}
func (r *groupErrorRepo) Search(context.Context, string) (PaginatedGroups, error) {
	return PaginatedGroups{}, r.searchErr
}
func (r *groupErrorRepo) Update(context.Context, string, UpdateGroupRequest) error {
	return r.updateErr
}
func (r *groupErrorRepo) Dismiss(context.Context, string) error {
	return r.dismissErr
}
func (r *groupErrorRepo) ListMembers(context.Context, string, *GroupRole, int, string) ([]GroupMemberItem, string, error) {
	return nil, "", r.listMembersErr
}
func (r *groupErrorRepo) SearchMembers(context.Context, string, string, *GroupRole, int, string) ([]GroupMemberItem, string, error) {
	return nil, "", r.searchMembersErr
}
func (r *groupErrorRepo) ListMyGroups(context.Context, string, int, string) ([]GroupItem, string, error) {
	return nil, "", r.listMyGroupsErr
}
func (r *groupErrorRepo) JoinGroup(context.Context, string, string, *string, *string) (JoinGroupResult, error) {
	return JoinGroupResult{}, r.joinGroupErr
}

func TestCreateAndOpenGroup(t *testing.T) {
	handler := NewHandler(NewService())
	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"name":"Hot Group","location":{"point":{"latitude":1,"longitude":2},"location_name":"Somewhere"}}`))
	createReq.Header.Set("Idempotency-Key", "abcd1234abcd1234abcd1234abcd1234")
	ctx := platformauth.WithPrincipal(createReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	createReq = createReq.WithContext(ctx)
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)

	var createResp struct {
		Code int       `json:"code"`
		Data GroupItem `json:"data"`
	}
	if err := json.NewDecoder(createRR.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if createResp.Code != 0 || createResp.Data.Group.GroupID == "" {
		t.Fatalf("unexpected create response: %#v", createResp)
	}

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"group_id":"`+createResp.Data.Group.GroupID+`"}`))
	openReq = openReq.WithContext(ctx)
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)

	var openResp struct {
		Code int       `json:"code"`
		Data GroupItem `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResp.Code != 0 || openResp.Data.Group.Name != "Hot Group" {
		t.Fatalf("unexpected open response: %#v", openResp)
	}
}

func TestSearchGroupsReturnsMatches(t *testing.T) {
	repo := NewInMemoryRepository()
	service := NewServiceWithRepo(repo)
	handler := NewHandler(service)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	if _, err := service.Create(ctx, CreateGroupRequest{Name: "Searchable", Location: Location{LocationName: "Here"}}); err != nil {
		t.Fatalf("create group: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"keyword":"Searchable"}`))
	rr := httptest.NewRecorder()
	handler.Search(rr, req)

	var resp struct {
		Code int             `json:"code"`
		Data PaginatedGroups `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Items) == 0 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestGroupsHandlerReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&groupErrorRepo{createErr: errors.New("boom"), getErr: errors.New("boom"), searchErr: errors.New("boom")}))

	createReq := httptest.NewRequest(http.MethodPost, "/create", strings.NewReader(`{"name":"Hot Group","location":{"point":{"latitude":1,"longitude":2},"location_name":"Somewhere"}}`))
	createReq.Header.Set("Idempotency-Key", "efgh1234efgh1234efgh1234efgh1234")
	createReq = createReq.WithContext(platformauth.WithPrincipal(createReq.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"}))
	createRR := httptest.NewRecorder()
	handler.Create(createRR, createReq)
	assertGroupCode(t, createRR, 10600)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"group_id":"group_1"}`))
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)
	assertGroupCode(t, openRR, 10600)

	searchReq := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(`{"keyword":"Hot Group"}`))
	searchRR := httptest.NewRecorder()
	handler.Search(searchRR, searchReq)
	assertGroupCode(t, searchRR, 10600)
}

func assertGroupCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
