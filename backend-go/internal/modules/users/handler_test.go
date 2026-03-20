package users

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

func TestGetMyProfileReturnsCurrentUser(t *testing.T) {
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/me/profile/get", strings.NewReader(`{}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetMyProfile(rr, req)

	var resp struct {
		Code int       `json:"code"`
		Data MyProfile `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || resp.Data.UserID != "user_alice" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestUpdateMyProfilePersistsChanges(t *testing.T) {
	service := NewService()
	handler := NewHandler(service)
	req := httptest.NewRequest(http.MethodPost, "/me/profile/update", strings.NewReader(`{"nickname":"Alice","bio":"hello"}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.UpdateMyProfile(rr, req)

	getReq := httptest.NewRequest(http.MethodPost, "/me/profile/get", strings.NewReader(`{}`))
	getReq = getReq.WithContext(ctx)
	getRR := httptest.NewRecorder()
	handler.GetMyProfile(getRR, getReq)

	var resp struct {
		Code int       `json:"code"`
		Data MyProfile `json:"data"`
	}
	if err := json.NewDecoder(getRR.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Data.Nickname != "Alice" || resp.Data.Bio != "hello" {
		t.Fatalf("unexpected profile: %#v", resp.Data)
	}
}

func TestGetMySettingsReturnsDefaults(t *testing.T) {
	handler := NewHandler(NewService())
	req := httptest.NewRequest(http.MethodPost, "/me/settings/get", strings.NewReader(`{}`))
	ctx := platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.GetMySettings(rr, req)

	var resp struct {
		Code int          `json:"code"`
		Data UserSettings `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 || !resp.Data.AllowFollow || resp.Data.AllowPrivateChat == "" {
		t.Fatalf("unexpected settings: %#v", resp)
	}
}

func TestFollowBlockRemarkAndOpenTheDoor(t *testing.T) {
	service := NewService()
	handler := NewHandler(service)
	ctx := platformauth.WithPrincipal(httptest.NewRequest(http.MethodPost, "/", nil).Context(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})

	followReq := httptest.NewRequest(http.MethodPost, "/me/follow", strings.NewReader(`{"user_id":"user_bob"}`))
	followReq = followReq.WithContext(ctx)
	followRR := httptest.NewRecorder()
	handler.Follow(followRR, followReq)

	remarkReq := httptest.NewRequest(http.MethodPost, "/me/set_remark", strings.NewReader(`{"user_id":"user_bob","remark":"buddy"}`))
	remarkReq = remarkReq.WithContext(ctx)
	remarkRR := httptest.NewRecorder()
	handler.SetRemark(remarkRR, remarkReq)

	blockReq := httptest.NewRequest(http.MethodPost, "/me/block", strings.NewReader(`{"user_id":"user_bob"}`))
	blockReq = blockReq.WithContext(ctx)
	blockRR := httptest.NewRecorder()
	handler.Block(blockRR, blockReq)

	openReq := httptest.NewRequest(http.MethodPost, "/open_the_door", strings.NewReader(`{"user_id":"user_bob"}`))
	openReq = openReq.WithContext(ctx)
	openRR := httptest.NewRecorder()
	handler.OpenTheDoor(openRR, openReq)
	var openResp struct {
		Code int      `json:"code"`
		Data UserItem `json:"data"`
	}
	if err := json.NewDecoder(openRR.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open response: %v", err)
	}
	if openResp.Code != 0 || openResp.Data.User.UserID != "user_bob" || openResp.Data.Relation != RelationshipBlocked {
		t.Fatalf("unexpected open response: %#v", openResp)
	}

	blockedReq := httptest.NewRequest(http.MethodPost, "/me/list_blocked", strings.NewReader(`{}`))
	blockedReq = blockedReq.WithContext(ctx)
	blockedRR := httptest.NewRecorder()
	handler.ListBlocked(blockedRR, blockedReq)
	var blockedResp struct {
		Code int                   `json:"code"`
		Data PaginatedBlockedUsers `json:"data"`
	}
	if err := json.NewDecoder(blockedRR.Body).Decode(&blockedResp); err != nil {
		t.Fatalf("decode blocked response: %v", err)
	}
	if blockedResp.Code != 0 || len(blockedResp.Data.Items) != 1 || blockedResp.Data.Items[0].Remark != "buddy" {
		t.Fatalf("unexpected blocked response: %#v", blockedResp)
	}
}
