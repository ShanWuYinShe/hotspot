package feed

import (
	"context"

	"hotspot/backend-go/internal/platform/appconfig"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/slicex"
)

type Service struct {
	repo      Repository
	appConfig appconfig.Source
	images    ImageResolver
	reactions ReactionAggregator
}

type ReactionAggregator interface {
	AggregateMoments(ctx context.Context, momentIDs []string, viewerID string) (map[string][]Reaction, error)
}

func NewService(source appconfig.Source) *Service {
	return NewServiceWithSourceAndImages(NewPersistenceRepository(nil), source, nil)
}

func NewServiceWithRepo(repo Repository) *Service {
	return NewServiceWithSourceAndImages(repo, nil, nil)
}

func NewServiceWithSource(repo Repository, source appconfig.Source) *Service {
	return NewServiceWithSourceAndImages(repo, source, nil)
}

func NewServiceWithSourceAndImages(repo Repository, source appconfig.Source, images ImageResolver) *Service {
	if repo == nil {
		repo = NewPersistenceRepository(nil)
	}
	if source == nil {
		source = appconfig.NewStaticSource(config.Config{})
	}
	return &Service{repo: repo, appConfig: source, images: images}
}

func (s *Service) Following(ctx context.Context, req GetFeedFollowingRequest) (PaginatedMoments, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return PaginatedMoments{}, ErrInvalidFeedRequest
	}
	query := momentsQuery{
		ViewerUserID: principal.UserID,
		Limit:        normalizeFeedLimit(req.Pagination),
		Cursor:       paginationCursor(req.Pagination),
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
	if req.ReferencePoint == nil || !isValidGeoPoint(*req.ReferencePoint) {
		return PaginatedMoments{}, ErrInvalidFeedRequest
	}
	query := momentsQuery{
		ViewerUserID:   principal.UserID,
		ReferencePoint: *req.ReferencePoint,
		Radius:         s.normalizeRadius(ctx, req.Radius),
		Limit:          normalizeFeedLimit(req.Pagination),
		Cursor:         paginationCursor(req.Pagination),
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
		Cursor:         paginationCursor(req.Pagination),
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
		Cursor:         paginationCursor(req.Pagination),
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

func paginationCursor(pagination *PaginationParams) string {
	if pagination == nil {
		return ""
	}
	return pagination.Cursor
}

func isValidGeoPoint(point GeoPoint) bool {
	return point.Latitude >= -90 && point.Latitude <= 90 && point.Longitude >= -180 && point.Longitude <= 180
}

func (s *Service) resolveReferencePoint(ctx context.Context, ref *GeoPoint) *GeoPoint {
	if ref != nil && !isValidGeoPoint(*ref) {
		return nil
	}
	if ref != nil {
		return ref
	}
	if s.appConfig == nil {
		return nil
	}
	def := s.appConfig.Get(ctx).Defaults.DefaultLocation
	if def.Point.Latitude != 0 || def.Point.Longitude != 0 {
		return &GeoPoint{Latitude: def.Point.Latitude, Longitude: def.Point.Longitude}
	}
	return nil
}
