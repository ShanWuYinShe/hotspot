package search

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/slicex"
)

const (
	defaultSearchSuggestRadius = 5000
	maxSearchSuggestRadius     = 50000
)

type Service struct {
	repo      Repository
	appConfig appconfig.Source
	locations LocationProvider // optional; nil => skip user-location fallback
}

// LocationProvider 提供已登录用户最近一次 update_location 的位置(回退链第 2 级,
// api_spec_types.md「reference_point 解析规则」)。由 users 模块实现。
type LocationProvider interface {
	RecentLocation(ctx context.Context, userID string) (geo.Point, bool)
}

func NewService(source appconfig.Source) *Service {
	return NewServiceWithSource(NewPostgresRepository(nil), source)
}

func NewServiceWithRepo(repo Repository) *Service { return NewServiceWithSource(repo, nil) }

func NewServiceWithSource(repo Repository, source appconfig.Source) *Service {
	if repo == nil {
		repo = NewPostgresRepository(nil)
	}
	if source == nil {
		source = appconfig.NewStaticSource(config.Config{})
	}
	return &Service{repo: repo, appConfig: source}
}

// SetLocationProvider wires the users-module location resolver (fallback level 2
// of the reference_point chain). Optional: nil skips the user-location level.
func (s *Service) SetLocationProvider(lp LocationProvider) { s.locations = lp }

func (s *Service) Suggest(ctx context.Context, req SearchSuggestRequest) (SearchSuggestResponse, error) {
	// reference_point 可选（指针语义）：nil=未传走回退链；回退链解析结果
	// 恒有值（default_location 兜底），下游无需再判空。
	resolved := s.resolveReferencePoint(ctx, req.ReferencePoint)
	req.ReferencePoint = &resolved
	req.Radius = normalizeSuggestRadius(s.appConfig.Get(ctx).Limits.SearchRadiusMax, req.Radius)
	resp, err := s.repo.Suggest(ctx, req)
	if err != nil {
		observability.L().Error("search.suggest failed", "error", err)
		return SearchSuggestResponse{}, err
	}
	limit := s.appConfig.Get(ctx).ResponseDefaults.SearchSuggestCount
	resp.Items = slicex.LimitCopy(resp.Items, limit)
	return resp, nil
}

func normalizeSuggestRadius(configMax, radius int) int {
	if radius <= 0 {
		radius = defaultSearchSuggestRadius
	}
	if radius > maxSearchSuggestRadius {
		radius = maxSearchSuggestRadius
	}
	if configMax > 0 && radius > configMax {
		radius = configMax
	}
	return radius
}

// resolveReferencePoint 按「是否传入」（指针 nil 判定）走回退链
// (api_spec_types.md):显式传参 → 已登录用户最近位置 → default_location。
// 此前以 (0,0) 零值哨兵判「未传」，赤道/本初子午线交点这一合法显式
// 坐标会被误吞进回退链（moments Create 已修同款问题，P2-11 对齐）。
func (s *Service) resolveReferencePoint(ctx context.Context, ref *GeoPoint) GeoPoint {
	if ref != nil {
		return *ref
	}
	// 回退 2:已登录用户的最近位置
	if principal, ok := platformauth.PrincipalFromContext(ctx); ok && principal.UserID != "" && s.locations != nil {
		if pt, found := s.locations.RecentLocation(ctx, principal.UserID); found {
			return GeoPoint{Latitude: pt.Latitude, Longitude: pt.Longitude}
		}
	}
	// 回退 3:default_location
	return s.defaultReferencePoint(ctx)
}

func (s *Service) defaultReferencePoint(ctx context.Context) GeoPoint {
	lat, lng := geo.DefaultFallbackLatitude, geo.DefaultFallbackLongitude
	if s.appConfig != nil {
		defaults := s.appConfig.Get(ctx).Defaults
		lat = defaults.DefaultLocation.Point.Latitude
		lng = defaults.DefaultLocation.Point.Longitude
	}
	return GeoPoint{Latitude: lat, Longitude: lng}
}

func isValidGeoPoint(point GeoPoint) bool {
	return point.Latitude >= -90 && point.Latitude <= 90 && point.Longitude >= -180 && point.Longitude <= 180
}
