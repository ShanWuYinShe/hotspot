package moments

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/httpx"
	"hotspot/backend-go/internal/platform/testdb"
)

// seedFollowedMoment 预置一条 author 发布的公开瞬间（含位置）。
func seedFollowedMoment(t *testing.T, repo *InMemoryRepository, author, description string, lat, lng float64) MomentItem {
	t.Helper()
	item, err := repo.Create(context.Background(), author, CreateMomentRequest{
		Description: description,
		Location: &Location{Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{Latitude: lat, Longitude: lng}, LocationName: "某地"},
	})
	if err != nil {
		t.Fatalf("create moment: %v", err)
	}
	return item
}

func TestServiceFetchFollowingReturnsFollowedPublicMoments(t *testing.T) {
	repo := NewInMemoryRepository()
	seedFollowedMoment(t, repo, "author_a", "first", 31.0, 121.0)
	private, err := repo.Create(context.Background(), "author_a", CreateMomentRequest{Description: "private", Location: &Location{LocationName: "x"}, PrivacyLevel: "private"})
	if err != nil {
		t.Fatalf("create private moment: %v", err)
	}
	_ = private
	seedFollowedMoment(t, repo, "author_b", "second", 31.1, 121.1)
	seedFollowedMoment(t, repo, "author_c", "unfollowed", 31.2, 121.2)
	repo.SetFollow("viewer", "author_a")
	repo.SetFollow("viewer", "author_b")
	svc := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "viewer", SessionID: "s1"})

	result, err := svc.FetchFollowing(ctx, FetchFollowingMomentsRequest{})
	if err != nil {
		t.Fatalf("fetch following: %v", err)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 followed public moments, got %d", len(result.Items))
	}
	// created_at DESC：后创建的 second 在前
	if result.Items[0].Description != "second" || result.Items[1].Description != "first" {
		t.Fatalf("unexpected order: %q %q", result.Items[0].Description, result.Items[1].Description)
	}
	// 恒定形状：数组字段非 null
	if result.Items[0].Tags == nil || result.Items[0].Images == nil || result.Items[0].Mentions == nil {
		t.Fatalf("expected non-nil constant arrays, got %#v", result.Items[0])
	}
}

func TestServiceFetchFollowingRequiresPrincipal(t *testing.T) {
	svc := NewServiceWithRepo(NewInMemoryRepository())
	if _, err := svc.FetchFollowing(context.Background(), FetchFollowingMomentsRequest{}); err == nil {
		t.Fatal("expected error without principal")
	}
}

func TestServiceFetchFollowingKeysetPagination(t *testing.T) {
	repo := NewInMemoryRepository()
	repo.SetFollow("viewer", "author")
	for i := 0; i < 3; i++ {
		seedFollowedMoment(t, repo, "author", "m", 31.0, 121.0)
	}
	svc := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "viewer", SessionID: "s1"})

	page1, err := svc.FetchFollowing(ctx, FetchFollowingMomentsRequest{Pagination: &PaginationParams{Limit: 2}})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(page1.Items) != 2 || page1.NextCursor == nil {
		t.Fatalf("expected page of 2 with cursor, got %#v", page1)
	}
	page2, err := svc.FetchFollowing(ctx, FetchFollowingMomentsRequest{Pagination: &PaginationParams{Limit: 2, Cursor: *page1.NextCursor}})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(page2.Items) != 1 {
		t.Fatalf("expected 1 item on page2, got %d", len(page2.Items))
	}
	// 不重不漏
	if page2.Items[0].MomentID == page1.Items[0].MomentID || page2.Items[0].MomentID == page1.Items[1].MomentID {
		t.Fatalf("page2 repeated page1 item: %#v", page2.Items)
	}
}

func TestServiceFetchNearbySortsByDistance(t *testing.T) {
	repo := NewInMemoryRepository()
	seedFollowedMoment(t, repo, "far", "far", 31.5, 121.5)
	seedFollowedMoment(t, repo, "near", "near", 31.001, 121.001)
	private, err := repo.Create(context.Background(), "near", CreateMomentRequest{Description: "private", Location: &Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: 31.0, Longitude: 121.0}, LocationName: "x"}, PrivacyLevel: "private"})
	if err != nil {
		t.Fatalf("create private: %v", err)
	}
	_ = private
	svc := NewServiceWithRepo(repo)

	result, err := svc.FetchNearby(context.Background(), FetchNearbyMomentsRequest{
		ReferencePoint: &GeoPoint{Latitude: 31.0, Longitude: 121.0},
		Radius:         intPtr(5000),
	})
	if err != nil {
		t.Fatalf("fetch nearby: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected only the nearby public moment (far out of radius, private hidden), got %#v", result.Items)
	}
	if result.Items[0].Description != "near" {
		t.Fatalf("unexpected item: %#v", result.Items[0])
	}
	// 相对位置字段（spec api_spec_types.md RelativeLocation）：服务端基于
	// reference_point 计算并返回 distance / bearing_degrees。
	loc := result.Items[0].Location
	if loc == nil || loc.Distance == nil || loc.BearingDegrees == nil {
		t.Fatalf("nearby location must carry server-computed distance/bearing_degrees, got %#v", loc)
	}
	if *loc.Distance < 0 || *loc.Distance > 5000 {
		t.Fatalf("distance out of radius: %v", *loc.Distance)
	}
	if *loc.BearingDegrees < 0 || *loc.BearingDegrees >= 360 {
		t.Fatalf("bearing_degrees must be within [0,360): %v", *loc.BearingDegrees)
	}
}

// TestServiceFetchFollowingFillsRelativeLocation：关注动态解析出参考点时同样填充
// distance / bearing_degrees（spec api_spec_moment.md fetch_following：reference_point
// 用于 MomentItem.location 的距离/方位计算）。
func TestServiceFetchFollowingFillsRelativeLocation(t *testing.T) {
	repo := NewInMemoryRepository()
	seedFollowedMoment(t, repo, "author", "first", 31.0, 121.0)
	repo.SetFollow("viewer", "author")
	svc := NewServiceWithRepo(repo)
	ctx := platformauth.WithPrincipal(context.Background(), platformauth.Principal{UserID: "viewer", SessionID: "s1"})

	result, err := svc.FetchFollowing(ctx, FetchFollowingMomentsRequest{
		ReferencePoint: &GeoPoint{Latitude: 31.0, Longitude: 121.0},
	})
	if err != nil {
		t.Fatalf("fetch following: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected 1 item, got %#v", result.Items)
	}
	loc := result.Items[0].Location
	if loc == nil || loc.Distance == nil || loc.BearingDegrees == nil {
		t.Fatalf("following location must carry server-computed distance/bearing_degrees, got %#v", loc)
	}
	if *loc.Distance != 0 {
		t.Fatalf("co-located moment should have ~0 distance, got %v", *loc.Distance)
	}
}

// TestServiceFetchNearbyRadiusCappedByConfig：radius 缺省 5000 并受
// search_radius_max 裁剪。
func TestServiceFetchNearbyRadiusCappedByConfig(t *testing.T) {
	repo := NewInMemoryRepository()
	seedFollowedMoment(t, repo, "author", "edge", 31.05, 121.05) // ~7.4km 外
	svc := NewServiceWithRepo(repo)
	ctx := context.Background()

	// 缺省 5000：7.4km 外不可见
	result, err := svc.FetchNearby(ctx, FetchNearbyMomentsRequest{ReferencePoint: &GeoPoint{Latitude: 31.0, Longitude: 121.0}})
	if err != nil {
		t.Fatalf("default radius: %v", err)
	}
	if len(result.Items) != 0 {
		t.Fatalf("expected empty page within default 5000m radius, got %#v", result.Items)
	}
	// 显式大 radius 仍受 search_radius_max=10000 裁剪
	source := appconfig.NewStaticSource(config.Config{})
	_, _ = source.Update(nil, func(current appconfig.AppConfig) appconfig.AppConfig {
		current.Limits.SearchRadiusMax = 10000
		return current
	})
	svc.appConfig = source
	result, err = svc.FetchNearby(ctx, FetchNearbyMomentsRequest{
		ReferencePoint: &GeoPoint{Latitude: 31.0, Longitude: 121.0},
		Radius:         intPtr(999999),
	})
	if err != nil {
		t.Fatalf("capped radius: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected capped radius to include edge moment, got %#v", result.Items)
	}
}

// TestServiceFetchNearbyReferencePointFallback：不传 reference_point 时按
// 回退链解析（无 appConfig 时用 geo 缺省坐标）。
func TestServiceFetchNearbyReferencePointFallback(t *testing.T) {
	repo := NewInMemoryRepository()
	// geo.DefaultFallbackLatitude/Longitude 附近
	seedFollowedMoment(t, repo, "author", "fallback", 35.154922, 112.682524)
	svc := NewServiceWithRepo(repo)
	result, err := svc.FetchNearby(context.Background(), FetchNearbyMomentsRequest{})
	if err != nil {
		t.Fatalf("fallback reference point: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected fallback-anchored moment, got %#v", result.Items)
	}
}

// TestHandlerFetchFollowingRoutes：handler 参数解析与错误映射（未登录 → 10000）。
func TestHandlerFetchFollowingRoutes(t *testing.T) {
	handler := NewHandler(NewServiceWithRepo(NewInMemoryRepository()))
	req := httptest.NewRequest(http.MethodPost, "/fetch_following", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.FetchFollowing(rr, req)
	if code := testdb.DecodeAPICode(t, rr); code != httpx.CodeInvalidRequest {
		t.Fatalf("anonymous following: expected httpx.CodeInvalidRequest, got %d", code)
	}
}

// TestHandlerFetchNearbyAnonymousAllowed：匿名 fetch_nearby 可用（public 版语义），
// 且相对位置字段仍由服务端计算（distance/bearing_degrees 非 null）。
func TestHandlerFetchNearbyAnonymousAllowed(t *testing.T) {
	repo := NewInMemoryRepository()
	seedFollowedMoment(t, repo, "author", "nearby", 35.154922, 112.682524)
	handler := NewHandler(NewServiceWithRepo(repo))
	req := httptest.NewRequest(http.MethodPost, "/fetch_nearby", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	handler.FetchNearby(rr, req)
	var resp struct {
		Code httpx.Code `json:"code"`
		Data struct {
			Items []struct {
				Location struct {
					Distance       *float64 `json:"distance"`
					BearingDegrees *float64 `json:"bearing_degrees"`
				} `json:"location"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Code != 0 || len(resp.Data.Items) != 1 {
		t.Fatalf("unexpected anonymous fetch_nearby response: %#v", resp)
	}
	if resp.Data.Items[0].Location.Distance == nil || resp.Data.Items[0].Location.BearingDegrees == nil {
		t.Fatalf("anonymous nearby must still carry server-computed relative fields: %#v", resp.Data.Items[0].Location)
	}
}

func intPtr(v int) *int { return &v }
