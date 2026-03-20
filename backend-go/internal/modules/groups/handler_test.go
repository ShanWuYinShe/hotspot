package groups

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

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
	service := NewService()
	handler := NewHandler(service)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	service.Create(ctx, CreateGroupRequest{Name: "Searchable", Location: Location{LocationName: "Here"}})

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
