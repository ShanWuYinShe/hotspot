package feed

import (
	"context"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/appconfig"
	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/observability"
	"hotspot/backend-go/internal/platform/slicex"
)

type Service struct {
	repo      Repository
	appConfig appconfig.Source
}

func NewService(source appconfig.Source) *Service { return NewServiceWithSource(NewPersistenceRepository(nil), source) }

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

func (s *Service) ListMoments(ctx context.Context, req ListFeedMomentsRequest) (PaginatedMoments, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return PaginatedMoments{}, ErrInvalidFeedRequest
	}
	query, err := s.newMomentsQuery(ctx, principal.UserID, req)
	if err != nil {
		return PaginatedMoments{}, err
	}
	items, err := s.repo.ListMoments(ctx, query)
	if err != nil {
		observability.L().Error("feed.list_moments failed", "error", err)
		return PaginatedMoments{}, err
	}
	return items, nil
}

func (s *Service) ListGroups(ctx context.Context, req ListFeedGroupsRequest) (PaginatedGroups, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return PaginatedGroups{}, ErrInvalidFeedRequest
	}
	query, err := s.newGroupsQuery(ctx, principal.UserID, req)
	if err != nil {
		return PaginatedGroups{}, err
	}
	items, err := s.repo.ListGroups(ctx, query)
	if err != nil {
		observability.L().Error("feed.list_groups failed", "error", err)
		return PaginatedGroups{}, err
	}
	return items, nil
}

func (s *Service) ListUsers(ctx context.Context, req ListFeedUsersRequest) (PaginatedUsers, error) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return PaginatedUsers{}, ErrInvalidFeedRequest
	}
	query, err := s.newUsersQuery(ctx, principal.UserID, req)
	if err != nil {
		return PaginatedUsers{}, err
	}
	items, err := s.repo.ListUsers(ctx, query)
	if err != nil {
		observability.L().Error("feed.list_users failed", "error", err)
		return PaginatedUsers{}, err
	}
	return items, nil
}

func (s *Service) ListTrending(ctx context.Context) (TrendingBundle, error) {
	viewerUserID := ""
	if principal, ok := platformauth.PrincipalFromContext(ctx); ok {
		viewerUserID = principal.UserID
	}
	bundle, err := s.repo.ListTrending(ctx, viewerUserID)
	if err != nil {
		observability.L().Error("feed.list_trending failed", "error", err)
		return TrendingBundle{}, err
	}
	cfg := s.appConfig.Get(ctx)
	bundle.TrendingTags = slicex.LimitCopy(bundle.TrendingTags, cfg.ResponseDefaults.TrendingTagsCount)
	bundle.HotMoments = slicex.LimitCopy(bundle.HotMoments, cfg.ResponseDefaults.HotMomentsCount)
	return bundle, nil
}

func (s *Service) Dismiss(ctx context.Context, req DismissFeedItemRequest) error {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok || principal.UserID == "" {
		return ErrInvalidFeedRequest
	}
	cmd, err := s.newDismissCommand(principal.UserID, req)
	if err != nil {
		return err
	}
	if err := s.repo.Dismiss(ctx, cmd); err != nil {
		observability.L().Error("feed.dismiss failed", "error", err)
		return err
	}
	return nil
}

func (s *Service) newMomentsQuery(ctx context.Context, viewerUserID string, req ListFeedMomentsRequest) (momentsQuery, error) {
	if req.ReferencePoint == nil || !isValidGeoPoint(*req.ReferencePoint) {
		return momentsQuery{}, ErrInvalidFeedRequest
	}
	source := normalizeFeedSource(req.Source)
	if source == "" {
		return momentsQuery{}, ErrInvalidFeedRequest
	}
	return momentsQuery{
		ViewerUserID:   viewerUserID,
		Source:         source,
		ReferencePoint: *req.ReferencePoint,
		Radius:         s.normalizeRadius(ctx, req.Radius),
		Limit:          normalizeFeedLimit(req.Pagination),
		Cursor:         paginationCursor(req.Pagination),
	}, nil
}

func (s *Service) newGroupsQuery(ctx context.Context, viewerUserID string, req ListFeedGroupsRequest) (groupsQuery, error) {
	if req.ReferencePoint == nil || !isValidGeoPoint(*req.ReferencePoint) {
		return groupsQuery{}, ErrInvalidFeedRequest
	}
	return groupsQuery{
		ViewerUserID:   viewerUserID,
		ReferencePoint: *req.ReferencePoint,
		Radius:         s.normalizeRadius(ctx, req.Radius),
		Limit:          normalizeFeedLimit(req.Pagination),
		Cursor:         paginationCursor(req.Pagination),
	}, nil
}

func (s *Service) newUsersQuery(ctx context.Context, viewerUserID string, req ListFeedUsersRequest) (usersQuery, error) {
	if req.ReferencePoint == nil || !isValidGeoPoint(*req.ReferencePoint) {
		return usersQuery{}, ErrInvalidFeedRequest
	}
	return usersQuery{
		ViewerUserID:   viewerUserID,
		ReferencePoint: *req.ReferencePoint,
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

func normalizeFeedSource(source string) string {
	switch source {
	case "following", "nearby", "all":
		return source
	default:
		return ""
	}
}

func (s *Service) newDismissCommand(viewerUserID string, req DismissFeedItemRequest) (dismissCommand, error) {
	targetType := normalizeDismissTargetType(req.TargetType)
	targetID := req.TargetID
	if targetType == "" || targetID == "" {
		return dismissCommand{}, ErrInvalidFeedRequest
	}
	return dismissCommand{ViewerUserID: viewerUserID, TargetType: targetType, TargetID: targetID}, nil
}

func isValidGeoPoint(point GeoPoint) bool {
	return point.Latitude >= -90 && point.Latitude <= 90 && point.Longitude >= -180 && point.Longitude <= 180
}
