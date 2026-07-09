package feed

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformauth "hotspot/backend-go/internal/platform/auth"
)

func TestListMomentsRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_moments", strings.NewReader(`{"source":`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListMoments(rr, req)
	assertFeedCode(t, rr, 10000)
}

func TestListMomentsUsesDefaultLocationWhenMissingReferencePoint(t *testing.T) {
	// Per api_spec_types.md, reference_point is optional; when omitted the
	// server falls back to ClientConfig.defaults.default_location.point.
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_moments", strings.NewReader(`{"source":"following"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListMoments(rr, req)
	assertFeedCode(t, rr, 0)
}

func TestListMomentsRejectsInvalidSource(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_moments", strings.NewReader(`{"source":"bad","reference_point":{"latitude":1,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListMoments(rr, req)
	assertFeedCode(t, rr, 10000)
}

func TestListUsersRejectsInvalidRadiusViaService(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_users", strings.NewReader(`{"radius":1,"reference_point":{"latitude":999,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListUsers(rr, req)
	assertFeedCode(t, rr, 10000)
}

func TestListGroupsRejectsInvalidCursor(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{err: ErrInvalidCursor}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_groups", strings.NewReader(`{"reference_point":{"latitude":1,"longitude":2},"pagination":{"cursor":"bad"}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListGroups(rr, req)
	assertFeedCode(t, rr, 10000)
}

func TestListUsersReturns10600OnServiceError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{err: errors.New("boom")}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_users", strings.NewReader(`{"reference_point":{"latitude":1,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListUsers(rr, req)
	assertFeedCode(t, rr, 10600)
}

func TestListTrendingRejectsInvalidJSON(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_trending", strings.NewReader(`[`))

	handler.ListTrending(rr, req)
	assertFeedCode(t, rr, 10000)
}

func TestListTrendingReturnsBundle(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{trendingResult: TrendingBundle{TrendingTags: []TrendingTagItem{{Name: "hot", UsageCount: 1, Trend: "rising"}}}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_trending", strings.NewReader(`{}`))

	handler.ListTrending(rr, req)
	resp := decodeFeedResponse[TrendingBundle](t, rr)
	if resp.Code != 0 || len(resp.Data.TrendingTags) != 1 {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestListMomentsReturnsPaginatedMoments(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{momentsResult: PaginatedMoments{Items: []MomentItem{{MomentID: "m1"}}}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_moments", strings.NewReader(`{"source":"following","reference_point":{"latitude":1,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListMoments(rr, req)
	resp := decodeFeedResponse[PaginatedMoments](t, rr)
	if resp.Code != 0 || len(resp.Data.Items) != 1 || resp.Data.Items[0].MomentID != "m1" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestListMomentsReturnsPaginatedShape(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{momentsResult: PaginatedMoments{Items: []MomentItem{{MomentID: "m1"}}, NextCursor: "cursor1"}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_moments", strings.NewReader(`{"source":"following","reference_point":{"latitude":1,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListMoments(rr, req)
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

func TestListUsersReturnsPaginatedShape(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{usersResult: PaginatedUsers{Items: []UserItem{{User: UserInfo{UserID: "u2"}}}, NextCursor: "cursor2"}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_users", strings.NewReader(`{"reference_point":{"latitude":1,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListUsers(rr, req)
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

func TestListGroupsReturnsPaginatedShape(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{groupsResult: PaginatedGroups{Items: []GroupItem{{Group: GroupInfo{GroupID: "g1"}}}, NextCursor: "cursor3"}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_groups", strings.NewReader(`{"reference_point":{"latitude":1,"longitude":2}}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListGroups(rr, req)
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

func TestListTrendingReturnsBundleShape(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{trendingResult: TrendingBundle{
		TrendingTags: []TrendingTagItem{{Name: "tag1", UsageCount: 10, Trend: "rising"}},
		HotMoments:   []MomentItem{{MomentID: "m1"}},
	}}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_trending", strings.NewReader(`{}`))

	handler.ListTrending(rr, req)
	resp := decodeFeedResponse[map[string]any](t, rr)
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	data := resp.Data
	_, ok := data["trending_tags"].([]any)
	if !ok {
		t.Fatalf("expected trending_tags array, got %#v", data["trending_tags"])
	}
	_, ok = data["hot_moments"].([]any)
	if !ok {
		t.Fatalf("expected hot_moments array, got %#v", data["hot_moments"])
	}
}

func TestListTrendingReturns10600OnRepoError(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{err: errors.New("repo error")}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_trending", strings.NewReader(`{}`))

	handler.ListTrending(rr, req)
	assertFeedCode(t, rr, 10600)
}

func TestDismissIsIdempotent(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/dismiss", strings.NewReader(`{"target_type":"moment","target_id":"m1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Dismiss(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	assertFeedCode(t, rr, 0)

	// second dismiss should also return success
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/me/dismiss", strings.NewReader(`{"target_type":"moment","target_id":"m1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))
	handler.Dismiss(rr, req)
	assertFeedCode(t, rr, 0)
}

func TestDismissDoesNotRequireIdempotencyKey(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/dismiss", strings.NewReader(`{"target_type":"moment","target_id":"m1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Dismiss(rr, req)
	assertFeedCode(t, rr, 0)
}

func TestDismissRejectsInvalidRequest(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/dismiss", strings.NewReader(`{"target_type":"bad","target_id":"m1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Dismiss(rr, req)
	assertFeedCode(t, rr, 10000)
}

func TestDismissReturnsNotFound(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{err: ErrDismissTargetNotFound}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/dismiss", strings.NewReader(`{"target_type":"moment","target_id":"m1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Dismiss(rr, req)
	assertFeedCode(t, rr, 10300)
}

func TestDismissReturnsSuccess(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/me/dismiss", strings.NewReader(`{"target_type":"moment","target_id":"m1"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.Dismiss(rr, req)
	assertFeedCode(t, rr, 0)
}

type feedAPIResponse[T any] struct {
	Code int `json:"code"`
	Data T   `json:"data"`
}

func decodeFeedResponse[T any](t *testing.T, rr *httptest.ResponseRecorder) feedAPIResponse[T] {
	t.Helper()
	var resp feedAPIResponse[T]
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

func assertFeedCode(t *testing.T, rr *httptest.ResponseRecorder, expected int) {
	t.Helper()
	resp := decodeFeedResponse[map[string]any](t, rr)
	if resp.Code != expected {
		t.Fatalf("expected code %d, got %#v", expected, resp)
	}
}
