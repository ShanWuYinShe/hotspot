package feed

import (
	"context"
	"sort"

	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) ListFollowingMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedMoments{}, err
	}
	candidates, err := r.loadFollowingCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedMoments{}, err
	}
	return paginateFollowingMoments(candidates, query.Limit, query.Cursor)
}

func (r *PostgresRepository) ListNearbyMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedMoments{}, err
	}
	candidates, err := r.loadNearbyCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedMoments{}, err
	}
	return paginateNearbyMoments(candidates, query.Limit, query.Cursor)
}

func (r *PostgresRepository) ListNearbyGroups(ctx context.Context, query groupsQuery) (PaginatedGroups, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedGroups{}, err
	}
	candidates, err := r.loadGroupCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedGroups{}, err
	}
	return paginateGroups(candidates, query.Limit, query.Cursor)
}

func (r *PostgresRepository) ListNearbyUsers(ctx context.Context, query usersQuery) (PaginatedUsers, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedUsers{}, err
	}
	candidates, err := r.loadUserCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedUsers{}, err
	}
	return paginateUsers(candidates, query.Limit, query.Cursor)
}

func (r *PostgresRepository) ListTrending(ctx context.Context, viewerUserID string) (TrendingResult, error) {
	queryable, err := r.queryable()
	if err != nil {
		return TrendingResult{}, err
	}
	hotMoments, err := loadTrendingMoments(ctx, queryable, viewerUserID)
	if err != nil {
		return TrendingResult{}, err
	}
	return TrendingResult{HotMoments: hotMoments}, nil
}

func (r *PostgresRepository) queryable() (persistence.QueryableDB, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	return queryable, nil
}

func (r *PostgresRepository) FetchMomentImageRows(ctx context.Context, momentIDs []string) (map[string][]momentImageRow, error) {
	queryable, err := r.queryable()
	if err != nil {
		return map[string][]momentImageRow{}, nil
	}
	return FetchMomentImageRows(ctx, queryable, momentIDs)
}

func loadTrendingMoments(ctx context.Context, queryable persistence.QueryableDB, viewerUserID string) ([]MomentItem, error) {
	rows, err := queryable.Query(ctx, queryFeedTrendingHotMoments, viewerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]momentCandidate, 0)
	momentIDs := make([]string, 0)
	for rows.Next() {
		var momentID, userID, nickname, avatarURL, description, locationName string
		var latitude, longitude float64
		var createdAt time.Time
		var likesCount, commentsCount, sharesCount int
		var isLiked, isFavorited bool
		if err := rows.Scan(&momentID, &userID, &nickname, &avatarURL, &description, &createdAt, &likesCount, &commentsCount, &sharesCount, &locationName, &latitude, &longitude, &isLiked, &isFavorited); err != nil {
			return nil, err
		}
		item := MomentItem{
			MomentID:      momentID,
			Description:   description,
			CreatedAt:     createdAt.UTC().Format(time.RFC3339),
			LikesCount:    likesCount,
			CommentsCount: commentsCount,
			SharesCount:   sharesCount,
			User:          UserInfo{UserID: userID, Nickname: nickname, AvatarURL: types.StrPtr(avatarURL)},
			Location:      &RelativeLocation{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName},
			IsFavorited:   isFavorited,
		}
		reason := RecommendationContext("trending")
		item.Recommendation = &reason
		items = append(items, momentCandidate{Item: item, Score: likesCount + commentsCount + sharesCount, CreatedAt: createdAt.UTC()})
		momentIDs = append(momentIDs, momentID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].Item.Tags = tagsByMomentID[items[idx].Item.MomentID]
	}
	sort.SliceStable(items, func(i, j int) bool { return compareNearbyCandidates(items[i], items[j]) < 0 })
	out := make([]MomentItem, 0, len(items))
	for _, item := range items {
		out = append(out, item.Item)
	}
	return out, nil
}

func loadMomentTags(ctx context.Context, queryable persistence.QueryableDB, momentIDs []string) (map[string][]string, error) {
	if len(momentIDs) == 0 {
		return map[string][]string{}, nil
	}
	result := make(map[string][]string, len(momentIDs))
	rows, err := queryable.Query(ctx, queryFeedMomentTagsByMomentIDs, momentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var momentID, tag string
		if err := rows.Scan(&momentID, &tag); err != nil {
			return nil, err
		}
		result[momentID] = append(result[momentID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
