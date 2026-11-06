package users

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/httpx"
)

// spec api_spec_public.md：匿名 UserItem 不含 relation / social_context；
// tags 恒为数组；open_the_door 的 recent_moments 使用 PublicMomentItem 形状。
func TestToPublicUserItemStripsViewerFields(t *testing.T) {
	item := UserItem{
		User:          UserInfo{UserID: "u1", Nickname: "Alice", AvatarURL: typesStrPtr("http://avatar/u1")},
		Bio:           typesStrPtr("hi"),
		Relation:      &Relationship{IsFollowing: true},
		SocialContext: &SocialContext{MutualFriendsCount: 2},
	}
	pub := toPublicUserItem(item)
	if pub.Tags == nil {
		t.Fatal("tags must normalize to empty slice")
	}
	blob, _ := json.Marshal(pub)
	for _, forbidden := range []string{"relation", "social_context"} {
		if strings.Contains(string(blob), forbidden) {
			t.Fatalf("public user must not expose %q: %s", forbidden, blob)
		}
	}
	if !strings.Contains(string(blob), "avatar_url") {
		t.Fatalf("public user must keep avatar_url: %s", blob)
	}
}

func TestToPublicUserProfileResultMapsRecentMoments(t *testing.T) {
	bundle := UserProfileBundle{
		User: UserItem{User: UserInfo{UserID: "u1", Nickname: "Alice"}},
		RecentMoments: &momenttypes.PaginatedMoments{Items: []momenttypes.MomentItem{{
			MomentID:       "m1",
			Description:    "hi",
			User:           UserInfo{UserID: "u1", Nickname: "Alice"},
			IsFavorited:    true,
			Reactions:      []momenttypes.Reaction{{Emoji: "like", Count: 1, ReactedByMe: true}},
			AuthorRelation: &momenttypes.UserRelationView{IsFollowing: true},
		}}},
	}
	result := toPublicUserProfileResult(bundle)
	if result.RecentMoments == nil || len(result.RecentMoments.Items) != 1 {
		t.Fatalf("recent moments must map: %#v", result.RecentMoments)
	}
	blob, _ := json.Marshal(result)
	for _, forbidden := range []string{"is_favorited", "reacted_by_me", "author_relation", "relation", "social_context"} {
		if strings.Contains(string(blob), forbidden) {
			t.Fatalf("public profile must not expose %q: %s", forbidden, blob)
		}
	}
}

func TestPublicUserHandlers(t *testing.T) {
	repo := &stubUserRepo{
		profile:      MyProfile{UserID: "u1", Nickname: "Alice", AvatarURL: "http://avatar/u1", Tags: []string{"x"}},
		profileFound: true,
		profiles: map[string]MyProfile{
			"u1": {UserID: "u1", Nickname: "Alice", AvatarURL: "http://avatar/u1"},
		},
	}
	handler := NewHandler(NewServiceWithRepo(repo))

	// open_the_door：公开形状（无 relation/social_context）。
	rr := httptest.NewRecorder()
	handler.PublicOpenTheDoor(rr, httptest.NewRequest(http.MethodPost, "/public/v1/users/open_the_door", strings.NewReader(`{"user_id":"u1"}`)))
	var openResp struct {
		Code httpx.Code     `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&openResp); err != nil {
		t.Fatalf("decode open: %v body=%s", err, rr.Body.String())
	}
	if openResp.Code != 0 {
		t.Fatalf("open code=%d body=%s", openResp.Code, rr.Body.String())
	}
	userPayload, _ := openResp.Data["user"].(map[string]any)
	if userPayload == nil {
		t.Fatalf("missing user payload: %s", rr.Body.String())
	}
	for _, forbidden := range []string{"relation", "social_context"} {
		if _, ok := userPayload[forbidden]; ok {
			t.Fatalf("public profile must not contain %q: %s", forbidden, rr.Body.String())
		}
	}
	if tags, ok := userPayload["tags"].([]any); !ok || len(tags) != 1 {
		t.Fatalf("tags must be array: %#v", userPayload["tags"])
	}

	// search：缺少条件 10000；有 keyword 但空结果 → 0 + items 数组。
	rr = httptest.NewRecorder()
	handler.PublicSearch(rr, httptest.NewRequest(http.MethodPost, "/public/v1/users/search", strings.NewReader(`{}`)))
	if got := decodeUsersCode(t, rr); got != httpx.CodeInvalidRequest {
		t.Fatalf("empty search expected 10000, got %d", got)
	}
	rr = httptest.NewRecorder()
	handler.PublicSearch(rr, httptest.NewRequest(http.MethodPost, "/public/v1/users/search", strings.NewReader(`{"keyword":"alice"}`)))
	var searchResp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&searchResp); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	if searchResp.Code != 0 || searchResp.Data.Items == nil {
		t.Fatalf("unexpected search response: %s", rr.Body.String())
	}

	// batch_fetch：共享 UsersBatchResult（user_id -> UserInfo），含头像。
	rr = httptest.NewRecorder()
	handler.PublicBatchGet(rr, httptest.NewRequest(http.MethodPost, "/public/v1/users/batch_fetch", strings.NewReader(`{"user_ids":["u1"]}`)))
	var batchResp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Users map[string]map[string]any `json:"users"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&batchResp); err != nil {
		t.Fatalf("decode batch: %v", err)
	}
	if batchResp.Code != 0 || batchResp.Data.Users["u1"]["avatar_url"] != "http://avatar/u1" {
		t.Fatalf("unexpected batch response: %s", rr.Body.String())
	}
	// 空 ids 与超 100 → 10000。
	rr = httptest.NewRecorder()
	handler.PublicBatchGet(rr, httptest.NewRequest(http.MethodPost, "/public/v1/users/batch_fetch", strings.NewReader(`{"user_ids":[]}`)))
	if got := decodeUsersCode(t, rr); got != httpx.CodeInvalidRequest {
		t.Fatalf("empty batch expected 10000, got %d", got)
	}
}

func decodeUsersCode(t *testing.T, rr *httptest.ResponseRecorder) httpx.Code {
	t.Helper()
	var envelope struct {
		Code httpx.Code `json:"code"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode envelope: %v body=%s", err, rr.Body.String())
	}
	return envelope.Code
}

func typesStrPtr(s string) *string { return &s }
