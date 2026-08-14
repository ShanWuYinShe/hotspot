package feed

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/httpx"
)

func TestFollowingRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/following", strings.NewReader(`{"reference_point":`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Following(rr, req)
	assertFeedCode(t, rr, httpx.CodeInvalidRequest)
}

func TestFollowingUsesDefaultLocationWhenMissingReferencePoint(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/following", strings.NewReader(`{}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Following(rr, req)
	assertFeedCode(t, rr, 0)
}

func TestNearbyFallsBackWhenMissingReferencePoint(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nearby", strings.NewReader(`{}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Nearby(rr, req)
	assertFeedCode(t, rr, 0)
}

func TestNearbyRejectsInvalidReferencePoint(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nearby", strings.NewReader(`{"reference_point":{"latitude":999,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Nearby(rr, req)
	assertFeedCode(t, rr, httpx.CodeInvalidRequest)
}

func TestNearbyUsersRejectsInvalidRadiusViaService(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nearby_users", strings.NewReader(`{"radius":1,"reference_point":{"latitude":999,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.NearbyUsers(rr, req)
	assertFeedCode(t, rr, httpx.CodeInvalidRequest)
}

func TestNearbyGroupsRejectsInvalidCursor(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{err: ErrInvalidCursor}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nearby_groups", strings.NewReader(`{"reference_point":{"latitude":1,"longitude":2},"pagination":{"cursor":"bad"}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.NearbyGroups(rr, req)
	assertFeedCode(t, rr, httpx.CodeInvalidRequest)
}

func TestNearbyUsersReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{err: errors.New("boom")}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nearby_users", strings.NewReader(`{"reference_point":{"latitude":1,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.NearbyUsers(rr, req)
	assertFeedCode(t, rr, httpx.CodeInternalUnavailable)
}

func TestTrendingRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trending", strings.NewReader(`[`))

	handler.Trending(rr, req)
	assertFeedCode(t, rr, httpx.CodeInvalidRequest)
}

func TestTrendingReturnsBundle(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{trendingResult: TrendingResult{HotMoments: []MomentItem{{MomentID: "m1"}}}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trending", strings.NewReader(`{}`))

	handler.Trending(rr, req)
	resp := decodeFeedResponse[TrendingResult](t, rr)
	if resp.Code != 0 || len(resp.Data.HotMoments) != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestFollowingReturnsPaginatedMoments(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{momentsResult: PaginatedMoments{Items: []MomentItem{{MomentID: "m1"}}}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/following", strings.NewReader(`{"reference_point":{"latitude":1,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Following(rr, req)
	resp := decodeFeedResponse[PaginatedMoments](t, rr)
	if resp.Code != 0 || len(resp.Data.Items) != 1 || resp.Data.Items[0].MomentID != "m1" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestFollowingReturnsPaginatedShape(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{momentsResult: PaginatedMoments{Items: []MomentItem{{MomentID: "m1"}}, NextCursor: strPtr("cursor1")}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/following", strings.NewReader(`{}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Following(rr, req)
	resp := decodeFeedResponse[map[string]any](t, rr)
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	data := resp.Data
	items, ok := data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected items array with 1 element, got %#v", data["items"])
	}
	nextCursor, ok := data["next_cursor"].(string)
	if !ok || nextCursor != "cursor1" {
		t.Fatalf("expected next_cursor 'cursor1', got %#v", data["next_cursor"])
	}
}

func TestNearbyUsersReturnsPaginatedShape(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{usersResult: PaginatedUsers{Items: []UserItem{{User: UserInfo{UserID: "u2"}}}, NextCursor: strPtr("cursor2")}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nearby_users", strings.NewReader(`{"reference_point":{"latitude":1,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.NearbyUsers(rr, req)
	resp := decodeFeedResponse[map[string]any](t, rr)
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	data := resp.Data
	items, ok := data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected items array with 1 element, got %#v", data["items"])
	}
	nextCursor, ok := data["next_cursor"].(string)
	if !ok || nextCursor != "cursor2" {
		t.Fatalf("expected next_cursor 'cursor2', got %#v", data["next_cursor"])
	}
}

func TestNearbyGroupsReturnsPaginatedShape(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{groupsResult: PaginatedGroups{Items: []GroupItem{{Group: GroupInfo{GroupID: "g1"}}}, NextCursor: strPtr("cursor3")}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/nearby_groups", strings.NewReader(`{"reference_point":{"latitude":1,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.NearbyGroups(rr, req)
	resp := decodeFeedResponse[map[string]any](t, rr)
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	data := resp.Data
	items, ok := data["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected items array with 1 element, got %#v", data["items"])
	}
	nextCursor, ok := data["next_cursor"].(string)
	if !ok || nextCursor != "cursor3" {
		t.Fatalf("expected next_cursor 'cursor3', got %#v", data["next_cursor"])
	}
}

func TestTrendingReturnsBundleShape(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{trendingResult: TrendingResult{
		HotMoments: []MomentItem{{MomentID: "m1"}},
	}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trending", strings.NewReader(`{}`))

	handler.Trending(rr, req)
	resp := decodeFeedResponse[map[string]any](t, rr)
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	data := resp.Data
	_, ok := data["hot_moments"].([]any)
	if !ok {
		t.Fatalf("expected hot_moments array, got %#v", data["hot_moments"])
	}
}

func TestTrendingReturns10600OnRepoError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{err: errors.New("repo error")}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/trending", strings.NewReader(`{}`))

	handler.Trending(rr, req)
	assertFeedCode(t, rr, httpx.CodeInternalUnavailable)
}

type feedAPIResponse[T any] struct {
	Code httpx.Code `json:"code"`
	Data T          `json:"data"`
}

func decodeFeedResponse[T any](t *testing.T, rr *httptest.ResponseRecorder) feedAPIResponse[T] {
	t.Helper()
	var resp feedAPIResponse[T]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func assertFeedCode(t *testing.T, rr *httptest.ResponseRecorder, expected httpx.Code) {
	t.Helper()
	resp := decodeFeedResponse[map[string]any](t, rr)
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
