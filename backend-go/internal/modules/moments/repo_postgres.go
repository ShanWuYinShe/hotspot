package moments

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
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
	normalizedMentions := normalizeMomentMentions(req.Mentions)
	if err := replaceMomentMentions(ctx, queryable, momentID, normalizedMentions, now); err != nil {
		return MomentItem{}, err
	}
	return MomentItem{
		MomentID:      momentID,
		Description:   req.Description,
		Tags:          allowedTags,
		Mentions:      normalizedMentions,
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
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.DecrementCommentCount(ctx, momentID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDecrementMomentCommentCount, momentID)
}

func (r *PostgresRepository) Search(ctx context.Context, req SearchMomentsRequest) (PaginatedMoments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Search(ctx, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if limit > 100 {
		limit = 100
	}
	var keyword *string
	if req.Keyword != nil && *req.Keyword != "" {
		keyword = req.Keyword
	}
	rows, err := queryable.Query(ctx, querySearchMoments, keyword, req.Tag, limit+1)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	for rows.Next() {
		item, err := scanMomentRow(rows)
		if err != nil {
			return PaginatedMoments{}, err
		}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, err
	}
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
		items[i].MyState = &MyMomentState{}
	}
	if len(items) > limit {
		return PaginatedMoments{Items: items[:limit], NextCursor: items[limit].MomentID}, nil
	}
	return PaginatedMoments{Items: items}, nil
}

func (r *PostgresRepository) Update(ctx context.Context, userID string, req EditMomentRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Update(ctx, userID, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var locationName, latStr, lngStr *string
	if req.Location != nil {
		locationName = &req.Location.LocationName
		s := fmt.Sprintf("%f", req.Location.Point.Latitude)
		latStr = &s
		s2 := fmt.Sprintf("%f", req.Location.Point.Longitude)
		lngStr = &s2
	}
	if err := queryable.Exec(ctx, queryUpdateMoment, req.MomentID, req.Description, req.PrivacyLevel, locationName, latStr, lngStr); err != nil {
		return err
	}
	if req.Tags != nil {
		if err := queryable.Exec(ctx, queryDeleteMomentTagsByMomentID, req.MomentID); err != nil {
			return err
		}
		if err := insertMomentTags(ctx, queryable, req.MomentID, req.Tags, time.Now().UTC()); err != nil {
			return err
		}
	}
	if req.Mentions != nil {
		if err := replaceMomentMentions(ctx, queryable, req.MomentID, req.Mentions, time.Now().UTC()); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, momentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Delete(ctx, momentID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteMoment, momentID)
}

func (r *PostgresRepository) Like(ctx context.Context, userID string, momentID string) (bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Like(ctx, userID, momentID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	exists, err := momentLikeExists(ctx, queryable, momentID, userID)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryInsertMomentLike, momentID, userID, now); err != nil {
		return false, err
	}
	if err := queryable.Exec(ctx, queryIncrementMomentLikeCount, momentID); err != nil {
		return false, err
	}
	return true, nil
}

func (r *PostgresRepository) Unlike(ctx context.Context, userID string, momentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Unlike(ctx, userID, momentID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	exists, err := momentLikeExists(ctx, queryable, momentID, userID)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := queryable.Exec(ctx, queryDeleteMomentLike, momentID, userID); err != nil {
		return err
	}
	return queryable.Exec(ctx, queryDecrementMomentLikeCount, momentID)
}

func (r *PostgresRepository) Favorite(ctx context.Context, userID string, momentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Favorite(ctx, userID, momentID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryInsertMomentFavorite, userID, momentID, time.Now().UTC())
}

func (r *PostgresRepository) Unfavorite(ctx context.Context, userID string, momentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Unfavorite(ctx, userID, momentID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteMomentFavorite, userID, momentID)
}

func (r *PostgresRepository) ListLikes(ctx context.Context, momentID string, limit int, cursor string) ([]LikeItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListLikes(ctx, momentID, limit, cursor)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, "", persistence.ErrNotImplemented
	}
	limit = normalizeMomentInteractionLimit(limit)
	cursorLikedAt, cursorUserID, err := decodeMomentLikeCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	rows, err := queryable.Query(ctx, queryListMomentLikes, momentID, cursorLikedAt, cursorUserID, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := make([]LikeItem, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		var item LikeItem
		if err := rows.Scan(&item.UserID, &item.LikedAt); err != nil {
			return nil, "", err
		}
		item.Nickname = item.UserID
		items = append(items, item)
		cursors = append(cursors, encodeMomentLikeCursor(item.LikedAt, item.UserID))
		if len(items) == limit+1 {
			return items[:limit], cursors[limit-1], nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, "", nil
}

func (r *PostgresRepository) NotificationContext(ctx context.Context, momentID string) (MomentItem, bool, error) {
	return r.Get(ctx, momentID)
}

func (r *PostgresRepository) ListFavorites(ctx context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.ListFavorites(ctx, userID, limit, cursor)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	limit = normalizeMomentInteractionLimit(limit)
	cursorAt, cursorMomentID, err := decodeMomentFavoriteCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	rows, err := queryable.Query(ctx, queryListFavoriteMoments, userID, cursorAt, cursorMomentID, limit+1)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, favoritedAt, err := scanFavoriteMomentRow(rows)
		if err != nil {
			return PaginatedMoments{}, err
		}
		item.MyState = &MyMomentState{IsFavorited: true}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		cursors = append(cursors, encodeMomentFavoriteCursor(favoritedAt, item.MomentID))
		if len(items) == limit+1 {
			tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return PaginatedMoments{}, err
			}
			for i := 0; i < limit; i++ {
				items[i].Tags = tagsByMomentID[items[i].MomentID]
			}
			return PaginatedMoments{Items: items[:limit], NextCursor: cursors[limit-1]}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, err
	}
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
	}
	return PaginatedMoments{Items: items}, nil
}

type momentInteractionCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

func normalizeMomentInteractionLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > 100:
		return 100
	default:
		return limit
	}
}

func encodeMomentLikeCursor(likedAt, userID string) string {
	blob, err := json.Marshal(momentInteractionCursor{At: likedAt, ID: userID})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(blob)
}

func decodeMomentLikeCursor(cursor string) (any, string, error) {
	if cursor == "" {
		return nil, "", nil
	}
	blob, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, "", persistence.ErrNotImplemented
	}
	var decoded momentInteractionCursor
	if err := json.Unmarshal(blob, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	likedAt, err := time.Parse(time.RFC3339, decoded.At)
	if err != nil {
		return nil, "", persistence.ErrNotImplemented
	}
	return likedAt.UTC(), decoded.ID, nil
}

func encodeMomentFavoriteCursor(favoritedAt, momentID string) string {
	blob, err := json.Marshal(momentInteractionCursor{At: favoritedAt, ID: momentID})
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(blob)
}

func decodeMomentFavoriteCursor(cursor string) (any, string, error) {
	if cursor == "" {
		return nil, "", nil
	}
	blob, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, "", persistence.ErrNotImplemented
	}
	var decoded momentInteractionCursor
	if err := json.Unmarshal(blob, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", persistence.ErrNotImplemented
	}
	favoritedAt, err := time.Parse(time.RFC3339, decoded.At)
	if err != nil {
		return nil, "", persistence.ErrNotImplemented
	}
	return favoritedAt.UTC(), decoded.ID, nil
}

func momentLikeExists(ctx context.Context, queryable persistence.QueryableDB, momentID, userID string) (bool, error) {
	var marker int
	err := queryable.QueryRow(ctx, queryGetMomentLike, momentID, userID).Scan(&marker)
	if err == nil {
		return true, nil
	}
	if err == pgx.ErrNoRows {
		return false, nil
	}
	return false, err
}
