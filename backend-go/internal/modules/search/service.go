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
	return NewServiceWithSource(NewPersistenceRepository(nil), source)
}

func NewServiceWithRepo(repo Repository) *Service { return NewServiceWithSource(repo, nil) }

func NewServiceWithSource(repo Repository, source appconfig.Source) *Service {
	if repo == nil {
		repo = NewPersistenceRepository(nil)
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
	// reference_point 缺失时回退到 default_location(见 api_spec_types.md)
	req.ReferencePoint = s.resolveReferencePoint(ctx, req.ReferencePoint)
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

// resolveReferencePoint:reference_point 缺失(零值)时按回退链解析
// (api_spec_types.md):显式传参 → 已登录用户最近位置 → default_location。
func (s *Service) resolveReferencePoint(ctx context.Context, ref GeoPoint) GeoPoint {
	if ref.Latitude != 0 || ref.Longitude != 0 {
		return ref
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
