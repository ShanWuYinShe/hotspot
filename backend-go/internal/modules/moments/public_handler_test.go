package moments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

// spec api_spec_public.md「类型体系」：匿名响应不得出现查看者状态字段
// （is_favorited / author_relation / recommendation），且数组字段恒为数组。
func TestPublicOpenTheDoorOmitsViewerFields(t *testing.T) {
	service := NewServiceWithRepo(NewInMemoryRepository())
	handler := NewHandler(service)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "user_alice", SessionID: "sess_1"})
	created, err := service.Create(ctx, CreateMomentRequest{Description: "hello", Location: &Location{LocationName: "Somewhere"}})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/public/v1/moments/open_the_door", strings.NewReader(`{"moment_id":"`+created.MomentID+`"}`))
	rr := httptest.NewRecorder()
	handler.PublicOpenTheDoor(rr, req)

	var resp struct {
		Code httpx.Code     `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("unexpected code: %d body=%s", resp.Code, rr.Body.String())
	}
	for _, forbidden := range []string{"is_favorited", "author_relation", "recommendation"} {
		if _, ok := resp.Data[forbidden]; ok {
			t.Fatalf("public response must not contain viewer field %q: %s", forbidden, rr.Body.String())
		}
	}
	for _, required := range []string{"moment_id", "description", "images", "tags", "mentions", "created_at", "comments_count", "shares_count", "user", "is_deleted", "location", "reactions"} {
		if _, ok := resp.Data[required]; !ok {
			t.Fatalf("public response missing contract field %q: %s", required, rr.Body.String())
		}
	}
	for _, arrayField := range []string{"images", "tags", "mentions", "reactions"} {
		if _, ok := resp.Data[arrayField].([]any); !ok {
			t.Fatalf("public array field %q must be an array, got %#v", arrayField, resp.Data[arrayField])
		}
	}
}

func TestPublicSearchUsesPublicEnvelopeAndValidation(t *testing.T) {
	service := NewServiceWithRepo(NewInMemoryRepository())
	handler := NewHandler(service)

	// 空条件 → 10000（与登录版同口径）。
	badReq := httptest.NewRequest(http.MethodPost, "/public/v1/moments/search", strings.NewReader(`{}`))
	badRR := httptest.NewRecorder()
	handler.PublicSearch(badRR, badReq)
	var badResp struct {
		Code httpx.Code `json:"code"`
	}
	_ = json.NewDecoder(badRR.Body).Decode(&badResp)
	if badResp.Code != httpx.CodeInvalidRequest {
		t.Fatalf("expected 10000 for empty search, got %d", badResp.Code)
	}

	okReq := httptest.NewRequest(http.MethodPost, "/public/v1/moments/search", strings.NewReader(`{"sort_by":"popular","pagination":{"limit":5}}`))
	okRR := httptest.NewRecorder()
	handler.PublicSearch(okRR, okReq)
	var okResp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Items      []map[string]any `json:"items"`
			NextCursor *string          `json:"next_cursor"`
		} `json:"data"`
	}
	if err := json.NewDecoder(okRR.Body).Decode(&okResp); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if okResp.Code != 0 || okResp.Data.Items == nil {
		t.Fatalf("unexpected search response: %s", okRR.Body.String())
	}
}

func TestPublicFetchNearbyReturnsPublicPage(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/public/v1/moments/fetch_nearby", strings.NewReader(`{"reference_point":{"latitude":34.26,"longitude":117.18},"radius":5000,"pagination":{"limit":5}}`))
	rr := httptest.NewRecorder()
	handler.PublicFetchNearby(rr, req)
	var resp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode nearby response: %v", err)
	}
	if resp.Code != 0 || resp.Data.Items == nil {
		t.Fatalf("unexpected nearby response: %s", rr.Body.String())
	}
}

func TestToPublicMomentItemStripsViewerStateAndNormalizesArrays(t *testing.T) {
	item := MomentItem{
		MomentID:       "m1",
		Description:    "hello",
		CreatedAt:      "2026-01-01T00:00:00Z",
		IsFavorited:    true,
		AuthorRelation: &UserRelationView{IsFollowing: true},
		Reactions:      []Reaction{{Emoji: "like", Count: 3, ReactedByMe: true}},
	}
	pub := ToPublicMomentItem(item)
	if pub.Images == nil || pub.Tags == nil || pub.Mentions == nil {
		t.Fatalf("arrays must normalize to empty slices: %#v", pub)
	}
	if len(pub.Reactions) != 1 || pub.Reactions[0].Emoji != "like" || pub.Reactions[0].Count != 3 {
		t.Fatalf("unexpected public reactions: %#v", pub.Reactions)
	}
	page := ToPublicPaginatedMoments(PaginatedMoments{Items: []MomentItem{item, item}})
	if len(page.Items) != 2 || page.Items[0].MomentID != "m1" {
		t.Fatalf("unexpected public page: %#v", page)
	}
}
