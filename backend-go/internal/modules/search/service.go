package search

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
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

// resolveReferencePoint:reference_point 缺失(零值)时回退到 default_location。
// search 模块无用户位置访问,仅回退到 default_location(回退链第 3 级)。
func (s *Service) resolveReferencePoint(ctx context.Context, ref GeoPoint) GeoPoint {
	if ref.Latitude != 0 || ref.Longitude != 0 {
		return ref
	}
	return s.defaultReferencePoint(ctx)
}

func (s *Service) defaultReferencePoint(ctx context.Context) GeoPoint {
	lat, lng := 35.154922, 112.682524
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
