package feed

import (
	"context"
	"hotspot/backend-go/internal/platform/types"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/slicex"
)

type Service struct {
	repo      Repository
	appConfig appconfig.Source
	images    ImageResolver
	reactions ReactionAggregator
	locations LocationProvider
}

// LocationProvider 提供已登录用户最近一次 update_location 的位置(回退链第 2 级,
// api_spec_types.md「reference_point 解析规则」)。由 users 模块实现,模块间通过
// Service 接口解耦,不直接依赖 users repo。
type LocationProvider interface {
	RecentLocation(ctx context.Context, userID string) (geo.Point, bool)
}

type ReactionAggregator interface {
	AggregateMoments(ctx context.Context, momentIDs []string, viewerID string) (map[string][]Reaction, error)
}

func NewService(source appconfig.Source) *Service {
	return NewServiceWithSourceAndImages(NewPostgresRepository(nil), source, nil)
}

func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithSourceAndImages(repo, nil, nil)
}

func NewServiceWithSource(repo Repository, source appconfig.Source) *Service {
	return NewServiceWithSourceAndImages(repo, source, nil)
}

func NewServiceWithSourceAndImages(repo Repository, source appconfig.Source, images ImageResolver) *Service {
	if repo == nil {
		repo = NewPostgresRepository(nil)
	}
	if source == nil {
		source = appconfig.NewStaticSource(config.Config{})
	}
	return &Service{repo: repo, appConfig: source, images: images}
}

// SetLocationProvider wires the users-module location resolver (fallback level 2
// of the reference_point chain). Optional: nil skips the user-location level.
func (s *Service) SetLocationProvider(lp LocationProvider) { s.locations = lp }

func (s *Service) Following(ctx context.Context, req GetFeedFollowingRequest) (PaginatedMoments, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return PaginatedMoments{}, ErrInvalidFeedRequest
	}
	query := momentsQuery{
		ViewerUserID: principal.UserID,
		Limit:        normalizeFeedLimit(req.Pagination),
		Cursor:       types.PaginationCursor(req.Pagination),
	}
	if refPt := s.resolveReferencePoint(ctx, req.ReferencePoint); refPt != nil {
		query.ReferencePoint = *refPt
	}
	items, err := s.repo.ListFollowingMoments(ctx, query)
	if err != nil {
		observability.L().Error("feed.following failed", "error", err)
		return PaginatedMoments{}, err
	}
	s.hydrateImages(ctx, items.Items)
	s.fillReactions(ctx, items.Items, principal.UserID)
	return items, nil
}

func (s *Service) Nearby(ctx context.Context, req GetFeedNearbyRequest) (PaginatedMoments, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return PaginatedMoments{}, ErrInvalidFeedRequest
	}
	refPt := s.resolveReferencePoint(ctx, req.ReferencePoint)
	if refPt == nil {
		return PaginatedMoments{}, ErrInvalidFeedRequest
	}
	query := momentsQuery{
		ViewerUserID:   principal.UserID,
		ReferencePoint: *refPt,
		Radius:         s.normalizeRadius(ctx, req.Radius),
		Limit:          normalizeFeedLimit(req.Pagination),
		Cursor:         types.PaginationCursor(req.Pagination),
	}
	items, err := s.repo.ListNearbyMoments(ctx, query)
	if err != nil {
		observability.L().Error("feed.nearby failed", "error", err)
		return PaginatedMoments{}, err
	}
	s.hydrateImages(ctx, items.Items)
	s.fillReactions(ctx, items.Items, principal.UserID)
	return items, nil
}

func (s *Service) SetReactionAggregator(ra ReactionAggregator) { s.reactions = ra }

func (s *Service) fillReactions(ctx context.Context, items []MomentItem, viewerID string) {
	if s.reactions == nil || len(items) == 0 {
		return
	}
	ids := make([]string, 0, len(items))
	for i := range items {
		ids = append(ids, items[i].MomentID)
	}
	byID, err := s.reactions.AggregateMoments(ctx, ids, viewerID)
	if err != nil {
		observability.L().Warn("feed.fillReactions failed", "error", err)
		return
	}
	for i := range items {
		if recs := byID[items[i].MomentID]; len(recs) > 0 {
			items[i].Reactions = recs
		}
	}
}

func (s *Service) hydrateImages(ctx context.Context, items []MomentItem) {
	if s.images == nil || len(items) == 0 {
		return
	}
	momentIDs := make([]string, 0, len(items))
	for _, it := range items {
		if it.MomentID != "" {
			momentIDs = append(momentIDs, it.MomentID)
		}
	}
	rowsByMoment, err := s.repo.FetchMomentImageRows(ctx, momentIDs)
	if err != nil {
		observability.L().Error("feed.hydrate_images fetch failed", "error", err)
		return
	}
	hydrateMomentImages(ctx, s.images, rowsByMoment, items)
}

func (s *Service) NearbyGroups(ctx context.Context, req GetFeedNearbyGroupsRequest) (PaginatedGroups, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	viewerUserID := ""
	if ok {
		viewerUserID = principal.UserID
	}
	query, err := s.newNearbyGroupsQuery(ctx, viewerUserID, req)
	if err != nil {
		return PaginatedGroups{}, err
	}
	items, err := s.repo.ListNearbyGroups(ctx, query)
	if err != nil {
		observability.L().Error("feed.nearby_groups failed", "error", err)
		return PaginatedGroups{}, err
	}
	return items, nil
}

func (s *Service) NearbyUsers(ctx context.Context, req GetFeedNearbyUsersRequest) (PaginatedUsers, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	viewerUserID := ""
	if ok {
		viewerUserID = principal.UserID
	}
	query, err := s.newNearbyUsersQuery(ctx, viewerUserID, req)
	if err != nil {
		return PaginatedUsers{}, err
	}
	items, err := s.repo.ListNearbyUsers(ctx, query)
	if err != nil {
		observability.L().Error("feed.nearby_users failed", "error", err)
		return PaginatedUsers{}, err
	}
	return items, nil
}

func (s *Service) Trending(ctx context.Context) (TrendingBundle, error) {
	viewerUserID := ""
	if principal, ok := platformauth.PrincipalFromContext(ctx); ok {
		viewerUserID = principal.UserID
	}
	bundle, err := s.repo.ListTrending(ctx, viewerUserID)
	if err != nil {
		observability.L().Error("feed.trending failed", "error", err)
		return TrendingBundle{}, err
	}
	cfg := s.appConfig.Get(ctx)
	bundle.HotMoments = slicex.LimitCopy(bundle.HotMoments, cfg.ResponseDefaults.HotMomentsCount)
	s.hydrateImages(ctx, bundle.HotMoments)
	s.fillReactions(ctx, bundle.HotMoments, viewerUserID)
	return bundle, nil
}

func (s *Service) newNearbyGroupsQuery(ctx context.Context, viewerUserID string, req GetFeedNearbyGroupsRequest) (groupsQuery, error) {
	refPt := s.resolveReferencePoint(ctx, req.ReferencePoint)
	if refPt == nil {
		return groupsQuery{}, ErrInvalidFeedRequest
	}
	return groupsQuery{
		ViewerUserID:   viewerUserID,
		ReferencePoint: *refPt,
		Radius:         s.normalizeRadius(ctx, req.Radius),
		Limit:          normalizeFeedLimit(req.Pagination),
		Cursor:         types.PaginationCursor(req.Pagination),
	}, nil
}

func (s *Service) newNearbyUsersQuery(ctx context.Context, viewerUserID string, req GetFeedNearbyUsersRequest) (usersQuery, error) {
	refPt := s.resolveReferencePoint(ctx, req.ReferencePoint)
	if refPt == nil {
		return usersQuery{}, ErrInvalidFeedRequest
	}
	return usersQuery{
		ViewerUserID:   viewerUserID,
		ReferencePoint: *refPt,
		Radius:         s.normalizeRadius(ctx, req.Radius),
		Limit:          normalizeFeedLimit(req.Pagination),
		Cursor:         types.PaginationCursor(req.Pagination),
	}, nil
}

func (s *Service) normalizeRadius(ctx context.Context, radius *int) int {
	value := 5000
	if radius != nil && *radius > 0 {
		value = *radius
	}
	maxRadius := s.appConfig.Get(ctx).Limits.SearchRadiusMax
	if maxRadius > 0 && value > maxRadius {
		value = maxRadius
	}
	return value
}

func normalizeFeedLimit(pagination *PaginationParams) int {
	if pagination == nil {
		return 20
	}
	switch {
	case pagination.Limit <= 0:
		return 20
	case pagination.Limit > 100:
		return 100
	default:
		return pagination.Limit
	}
}

func isValidGeoPoint(point GeoPoint) bool {
	return point.Latitude >= -90 && point.Latitude <= 90 && point.Longitude >= -180 && point.Longitude <= 180
}

// resolveReferencePoint 实现 reference_point 回退链(api_spec_types.md):
// 1. 请求显式传入(合法) → 直接使用
// 2. 已登录用户最近一次 update_location 的位置(匿名或未更新时跳过)
// 3. ClientConfig.defaults.default_location.point(服务端 appConfig 提供)
// 仅当显式传入的坐标非法时返回 nil。
func (s *Service) resolveReferencePoint(ctx context.Context, ref *GeoPoint) *GeoPoint {
	if ref != nil && !isValidGeoPoint(*ref) {
		return nil
	}
	if ref != nil {
		return ref
	}
	// 回退 2:已登录用户的最近位置
	if principal, ok := platformauth.PrincipalFromContext(ctx); ok && principal.UserID != "" && s.locations != nil {
		if pt, found := s.locations.RecentLocation(ctx, principal.UserID); found {
			return &GeoPoint{Latitude: pt.Latitude, Longitude: pt.Longitude}
		}
	}
	// 回退 3:default_location
	lat, lng := geo.DefaultFallbackLatitude, geo.DefaultFallbackLongitude
	if s.appConfig != nil {
		def := s.appConfig.Get(ctx).Defaults.DefaultLocation
		lat, lng = def.Point.Latitude, def.Point.Longitude
	}
	return &GeoPoint{Latitude: lat, Longitude: lng}
}
