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
	reactions ReactionAggregator // optional; nil => reactions[] not filled
}

// ReactionAggregator computes per-emoji reaction aggregates for feed moments.
// Implemented by the reactions module (dependency inversion). viewerID drives
// the @auth-only reacted_by_me flag.
type ReactionAggregator interface {
	AggregateMoments(ctx context.Context, momentIDs []string, viewerID string) (map[string][]Reaction, error)
}

func NewService(source appconfig.Source) *Service {
	return NewServiceWithSourceAndImages(NewPersistenceRepository(nil), source, nil)
}

func NewServiceWithRepo(repo Repository) *Service { return NewServiceWithSourceAndImages(repo, nil, nil) }

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
	s.hydrateImages(ctx, items.Items)
	s.fillReactions(ctx, items.Items, principal.UserID)
	return items, nil
}

// SetReactionAggregator wires the reactions aggregator (called by bootstrap).
func (s *Service) SetReactionAggregator(ra ReactionAggregator) { s.reactions = ra }

// fillReactions populates MomentItem.reactions[] for a list via the aggregator.
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

// hydrateImages fills the Images field on each moment in place via a single
// batched query, presigning read URLs through the image resolver. No-op when
// no resolver is configured (e.g. in tests).
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
	s.hydrateImages(ctx, bundle.HotMoments)
	s.fillReactions(ctx, bundle.HotMoments, viewerUserID)
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
	refPt := s.resolveReferencePoint(ctx, req.ReferencePoint)
	if refPt == nil {
		return momentsQuery{}, ErrInvalidFeedRequest
	}
	source := normalizeFeedSource(req.Source)
	if source == "" {
		return momentsQuery{}, ErrInvalidFeedRequest
	}
	return momentsQuery{
		ViewerUserID:   viewerUserID,
		Source:         source,
		ReferencePoint: *refPt,
		Radius:         s.normalizeRadius(ctx, req.Radius),
		Limit:          normalizeFeedLimit(req.Pagination),
		Cursor:         paginationCursor(req.Pagination),
	}, nil
}

func (s *Service) newGroupsQuery(ctx context.Context, viewerUserID string, req ListFeedGroupsRequest) (groupsQuery, error) {
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

func (s *Service) newUsersQuery(ctx context.Context, viewerUserID string, req ListFeedUsersRequest) (usersQuery, error) {
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

// resolveReferencePoint implements the reference_point fallback chain from
// api_spec_types.md: when the request omits reference_point (nil), use the
// user's stored location from config defaults. Returns nil when no fallback
// is available (caller treats as invalid feed request).
func (s *Service) resolveReferencePoint(ctx context.Context, ref *GeoPoint) *GeoPoint {
	// Provided but invalid → reject.
	if ref != nil && !isValidGeoPoint(*ref) {
		return nil
	}
	// Provided and valid → use as-is.
	if ref != nil {
		return ref
	}
	// Not provided → fall back to config defaults.
	if s.appConfig == nil {
		return nil
	}
	def := s.appConfig.Get(ctx).Defaults.DefaultLocation
	if def.Point.Latitude != 0 || def.Point.Longitude != 0 {
		return &GeoPoint{Latitude: def.Point.Latitude, Longitude: def.Point.Longitude}
	}
	return nil
}
