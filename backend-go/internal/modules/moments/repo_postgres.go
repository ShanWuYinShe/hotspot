package moments

import (
	"context"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory  *persistence.Factory
	fallback Repository
}

func NewPostgresRepository(factory *persistence.Factory, fallback Repository) *PostgresRepository {
	if fallback == nil {
		fallback = NewInMemoryRepository()
	}
	return &PostgresRepository{factory: factory, fallback: fallback}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateMomentRequest) (MomentItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Create(ctx, userID, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MomentItem{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	createdAt := now.Format(time.RFC3339)
	momentID := fmt.Sprintf("moment_%d", now.UnixNano())
	privacy := req.PrivacyLevel
	if privacy == "" {
		privacy = "public"
	}
	if err := queryable.Exec(ctx, queryInsertMoment, momentID, userID, req.Description, createdAt, 0, 0, 0, req.Location.LocationName, req.Location.Point.Latitude, req.Location.Point.Longitude, privacy); err != nil {
		return MomentItem{}, err
	}
	tags := normalizeMomentTags(req.Tags)
	allowedTags, err := filterAllowedMomentTags(ctx, queryable, tags)
	if err != nil {
		return MomentItem{}, err
	}
	if err := insertMomentTags(ctx, queryable, momentID, tags, now); err != nil {
		return MomentItem{}, err
	}
	return MomentItem{
		MomentID:      momentID,
		Description:   req.Description,
		Tags:          allowedTags,
		CreatedAt:     createdAt,
		LikesCount:    0,
		CommentsCount: 0,
		SharesCount:   0,
		User:          UserInfo{UserID: userID, Nickname: userID},
		Location:      req.Location,
		PrivacyLevel:  privacy,
		MyState:       &MyMomentState{IsLiked: false, IsFavorited: false},
	}, nil
}

func (r *PostgresRepository) IncrementCommentCount(ctx context.Context, momentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.IncrementCommentCount(ctx, momentID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryIncrementMomentCommentCount, momentID)
}

func (r *PostgresRepository) DecrementCommentCount(ctx context.Context, momentID string) error {
	return r.fallback.DecrementCommentCount(ctx, momentID)
}

func (r *PostgresRepository) Search(ctx context.Context, req SearchMomentsRequest) (PaginatedMoments, error) {
	return r.fallback.Search(ctx, req)
}

func (r *PostgresRepository) Update(ctx context.Context, userID string, req EditMomentRequest) error {
	return r.fallback.Update(ctx, userID, req)
}

func (r *PostgresRepository) Delete(ctx context.Context, momentID string) error {
	return r.fallback.Delete(ctx, momentID)
}

func (r *PostgresRepository) Like(ctx context.Context, userID string, momentID string) error {
	return r.fallback.Like(ctx, userID, momentID)
}

func (r *PostgresRepository) Unlike(ctx context.Context, userID string, momentID string) error {
	return r.fallback.Unlike(ctx, userID, momentID)
}

func (r *PostgresRepository) Favorite(ctx context.Context, userID string, momentID string) error {
	return r.fallback.Favorite(ctx, userID, momentID)
}

func (r *PostgresRepository) Unfavorite(ctx context.Context, userID string, momentID string) error {
	return r.fallback.Unfavorite(ctx, userID, momentID)
}

func (r *PostgresRepository) ListLikes(ctx context.Context, momentID string, limit int, cursor string) ([]LikeItem, string, error) {
	return r.fallback.ListLikes(ctx, momentID, limit, cursor)
}

func (r *PostgresRepository) ListFavorites(ctx context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	return r.fallback.ListFavorites(ctx, userID, limit, cursor)
}
