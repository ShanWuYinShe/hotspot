package search

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
)

// spec api_spec_public.md → PublicSearchSuggestResponse：匿名 payload 为
// TagItem / PublicUserItem / PublicGroupItem / PublicMomentSuggestItem 判别
// 联合，不得含 relation / social_context / viewer_state / friends_in_group。
func TestPublicSuggestStripsViewerFields(t *testing.T) {
	avatar := "http://avatar/u1"
	repo := &stubRepository{response: SearchSuggestResponse{Items: []SearchSuggestItem{
		{Type: "tag", Payload: TagItem{Name: "hot", UsageCount: 3}},
		{Type: "user", Payload: SearchSuggestUserPayload{
			User:          SearchSuggestUser{UserID: "u1", Nickname: "Alice", AvatarURL: &avatar},
			Tags:          []string{"x"},
			Relation:      &SearchSuggestUserRelation{IsFollowing: true},
			SocialContext: nil,
		}},
		{Type: "group", Payload: SearchSuggestGroupPayload{
			Group:       SearchSuggestGroup{GroupID: "g1", Name: "Group"},
			Tags:        []string{"x"},
			ViewerState: map[string]any{"role": "owner"},
		}},
		{Type: "moment", Payload: SearchSuggestMomentPayload{
			MomentID:           "m1",
			DescriptionPreview: "preview",
			Author:             SearchSuggestUser{UserID: "u1", Nickname: "Alice", AvatarURL: &avatar},
		}},
	}}}
	service := NewServiceWithSource(repo, appconfig.NewStaticSource(config.Load()))
	handler := NewHandler(service)

	rr := httptest.NewRecorder()
	handler.PublicSuggest(rr, httptest.NewRequest(http.MethodPost, "/public/v1/search/suggest", strings.NewReader(`{"prefix":"hot"}`)))
	var resp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Items []struct {
				Type    string         `json:"type"`
				Payload map[string]any `json:"payload"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v body=%s", err, rr.Body.String())
	}
	if resp.Code != 0 || len(resp.Data.Items) != 4 {
		t.Fatalf("unexpected response: %s", rr.Body.String())
	}
	for _, item := range resp.Data.Items {
		for _, forbidden := range []string{"relation", "social_context", "viewer_state", "friends_in_group"} {
			if _, ok := item.Payload[forbidden]; ok {
				t.Fatalf("public %s payload must not expose %q: %#v", item.Type, forbidden, item.Payload)
			}
		}
	}
	byType := map[string]map[string]any{}
	for _, item := range resp.Data.Items {
		byType[item.Type] = item.Payload
	}
	if byType["tag"]["name"] != "hot" {
		t.Fatalf("tag payload: %#v", byType["tag"])
	}
	user, _ := byType["user"]["user"].(map[string]any)
	if user == nil || user["user_id"] != "u1" || user["avatar_url"] != avatar {
		t.Fatalf("user payload: %#v", byType["user"])
	}
	group, _ := byType["group"]["group"].(map[string]any)
	if group == nil || group["group_id"] != "g1" {
		t.Fatalf("group payload: %#v", byType["group"])
	}
	if byType["moment"]["description_preview"] != "preview" {
		t.Fatalf("moment payload: %#v", byType["moment"])
	}

	// 缺 prefix → 10000。
	rr = httptest.NewRecorder()
	handler.PublicSuggest(rr, httptest.NewRequest(http.MethodPost, "/public/v1/search/suggest", strings.NewReader(`{}`)))
	var bad struct {
		Code httpx.Code `json:"code"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&bad)
	if bad.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected 10000, got %d", bad.Code)
	}
}
