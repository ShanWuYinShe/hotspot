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

func TestListMomentsRejectsMissingReferencePoint(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(&stubRepository{}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/list_moments", strings.NewReader(`{"source":"following"}`))
	req = req.WithContext(platformauth.WithPrincipal(req.Context(), platformauth.Principal{UserID: "u1"}))

	handler.ListMoments(rr, req)
	assertFeedCode(t, rr, 10000)
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
