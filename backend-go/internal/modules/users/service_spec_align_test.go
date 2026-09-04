package users

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	groups "hotspot/backend-go/internal/modules/groups"
	moments "hotspot/backend-go/internal/modules/moments"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

// users/spec_align 契约对齐测试（v1 spec）。
// 覆盖：user_id 必传(10000)、10200 必带 forbidden_reason=privacy_setting、list_moments 纯公开、
// UserItem 恒定 nullable 序列化、update_account 屏蔽标签 10400、delete_account confirmation 10000、batch_get UsersBatchResult。

func specAlignCtx(userID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	if userID != "" {
		req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: userID, SessionID: "sess_1"}))
	}
	return req
}

func decodeUserResp(t *testing.T, rr *httptest.ResponseRecorder) struct {
	Code httpx.Code                `json:"code"`
	Data *httpx.ForbiddenErrorData `json:"data"`
} {
	t.Helper()
	var resp struct {
		Code httpx.Code                `json:"code"`
		Data *httpx.ForbiddenErrorData `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return resp
}

// TestListEndpointsRequireUserID：六个列表端点的 user_id 必传，缺失返回 10000。
func TestListEndpointsRequireUserID(t *testing.T) {
	handler := NewHandler(NewService())
	cases := []struct {
		name string
		hit  func(w http.ResponseWriter, r *http.Request)
	}{
		{"list_friends", handler.ListFriends},
		{"list_following", handler.ListFollowing},
		{"list_followers", handler.ListFollowers},
		{"list_groups", handler.ListGroups},
		{"list_moments", handler.ListMoments},
		{"list_reacted_moments", handler.ListReactedMoments},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			tc.hit(rr, specAlignCtx(""))
			if got := decodeUserResp(t, rr).Code; got != httpx.CodeInvalidRequest {
				t.Fatalf("%s missing user_id: expected 10000, got %d", tc.name, got)
			}
		})
	}
}

// TestListPrivacyForbiddenReason：查他人列表受隐私开关(整表拒绝) → 10200 + forbidden_reason=privacy_setting。
func TestListPrivacyForbiddenReason(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	reqCtx := specAlignCtx("viewer").Context()
	_ = repo.PutProfile(context.Background(), defaultTestProfile("target"))
	_ = repo.PutSettings(context.Background(), "target", UserSettings{SocialListsVisible: false, GroupsVisible: false, ReactedMomentsVisible: false})

	cases := []struct {
		name string
		body string
		hit  func(w http.ResponseWriter, r *http.Request)
	}{
		{"list_friends", `{"user_id":"target"}`, handler.ListFriends},
		{"list_following", `{"user_id":"target"}`, handler.ListFollowing},
		{"list_followers", `{"user_id":"target"}`, handler.ListFollowers},
		{"list_groups", `{"user_id":"target"}`, handler.ListGroups},
		{"list_reacted_moments", `{"user_id":"target"}`, handler.ListReactedMoments},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body)).WithContext(reqCtx)
			rr := httptest.NewRecorder()
			tc.hit(rr, req)
			resp := decodeUserResp(t, rr)
			if resp.Code != httpx.CodeForbidden {
				t.Fatalf("%s: expected 10200, got %d", tc.name, resp.Code)
			}
			if resp.Data == nil || resp.Data.ForbiddenReason != httpx.ReasonPrivacySetting {
				t.Fatalf("%s: 10200 must carry forbidden_reason=privacy_setting, got %#v", tc.name, resp.Data)
			}
		})
	}
}

// TestListMomentsPublicOnly：list_moments 只返回 public 瞬间，忽略 private。
func TestListMomentsPublicOnly(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	_ = repo.PutProfile(ctx, defaultTestProfile("alice"))
	// 数据源走 moments Service（D1 迁移后 users 不再直查 moments 表）
	momentsRepo := moments.NewInMemoryRepository()
	momentsRepo.SeedMomentsByUser("alice", []momenttypes.MomentItem{
		{MomentID: "m_pub", User: momenttypes.UserInfo{UserID: "alice"}, Description: "public", PrivacyLevel: "public"},
		{MomentID: "m_priv", User: momenttypes.UserInfo{UserID: "alice"}, Description: "private", PrivacyLevel: "private"},
	})
	momentsSvc := moments.NewServiceWithRepo(momentsRepo)
	groupsSvc := groups.NewServiceWithRepo(groups.NewInMemoryRepository())
	svc := NewServiceWithRepo(repo)
	svc.SetContentProviders(groupsSvc, momentsSvc)
	res, err := svc.ListMoments(platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: "alice"}), UserMomentsRequest{UserID: "alice"})
	if err != nil {
		t.Fatalf("ListMoments: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].MomentID != "m_pub" {
		t.Fatalf("list_moments must be public-only, got %#v", res.Items)
	}
	if res.Items[0].PrivacyLevel != "" {
		t.Fatalf("privacy_level is owner data, must be empty on public view, got %q", res.Items[0].PrivacyLevel)
	}
}

// TestUserItemConstantNullable：UserItem 的 presence_status/last_online_at/location/stats/relation/social_context
// 序列化恒在，nil→null（恒定 nullable 通则，spec api_spec_base.md「类型稳定性」）。
func TestUserItemConstantNullable(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	// presence 隐藏：presence_visible=false → presence_status/last_online_at 为 null。
	profile := defaultTestProfile("alice")
	profile.PresenceStatus = UserPresenceOnline
	_ = repo.PutProfile(ctx, profile)
	_ = repo.PutSettings(ctx, "alice", UserSettings{PresenceVisible: false, LocationVisible: false, StatsVisible: false})
	svc := NewServiceWithRepo(repo)
	item, ok, err := svc.OpenTheDoor(platformauth.WithPrincipal(ctx, platformauth.Principal{UserID: "viewer"}), "alice")
	if err != nil || !ok {
		t.Fatalf("OpenTheDoor: ok=%v err=%v", ok, err)
	}
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"presence_status", "last_online_at", "location", "stats", "relation", "social_context"} {
		val, present := obj[key]
		if !present {
			t.Fatalf("UserItem.%s must be always present (constant nullable)", key)
		}
		if string(val) != "null" {
			t.Fatalf("UserItem.%s must serialize null when hidden, got %s", key, string(val))
		}
	}
}

// TestUpdateAccountBlockedTagRejects：tags 含被屏蔽标签 → 10400（spec update_account 错误码）。
func TestUpdateAccountBlockedTagRejects(t *testing.T) {
	repo := NewInMemoryRepository()
	repo.blockedTags["slur"] = struct{}{}
	svc := NewServiceWithDeps(repo, nil, nil)
	ctx := specAlignCtx("alice").Context()
	tags := []string{"coffee", "slur"}
	err := svc.UpdateMyProfile(ctx, UpdateAccountRequest{Tags: &tags})
	if !errors.Is(err, ErrBlockedTag) {
		t.Fatalf("expected ErrBlockedTag, got %v", err)
	}
}

// TestDeleteAccountWrongConfirmationInvalidRequest：confirmation 不匹配 → 10000（ErrInvalidRequest）。
func TestDeleteAccountWrongConfirmationInvalidRequest(t *testing.T) {
	svc := NewServiceWithRepo(NewInMemoryRepository())
	err := svc.DeleteAccount(specAlignCtx("alice").Context(), DeleteAccountRequest{Confirmation: "NOPE"})
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("expected ErrInvalidRequest, got %v", err)
	}
}

// TestBatchGetUsersResultShape：batch_get 响应为 UsersBatchResult{users map}。
func TestBatchGetUsersResultShape(t *testing.T) {
	repo := NewInMemoryRepository()
	handler := NewHandler(NewServiceWithRepo(repo))
	_ = repo.PutProfile(context.Background(), defaultTestProfile("alice"))
	req := httptest.NewRequest(http.MethodPost, "/batch_get", strings.NewReader(`{"user_ids":["alice","ghost"]}`)).WithContext(specAlignCtx("viewer").Context())
	rr := httptest.NewRecorder()
	handler.BatchGet(rr, req)
	var resp struct {
		Code httpx.Code       `json:"code"`
		Data UsersBatchResult `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected 0, got %d", resp.Code)
	}
	if resp.Data.Users == nil || len(resp.Data.Users) != 1 {
		t.Fatalf("expected exactly 1 user (alice), got %#v", resp.Data.Users)
	}
	if _, ok := resp.Data.Users["alice"]; !ok {
		t.Fatalf("alice missing, got %#v", resp.Data.Users)
	}
	if _, ok := resp.Data.Users["ghost"]; ok {
		t.Fatalf("ghost must be absent, got %#v", resp.Data.Users)
	}
}
