package moments

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateMomentRequest) (MomentItem, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MomentItem{}, persistence.ErrNotImplemented
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
	if err := queryable.Exec(ctx, queryInsertMoment, momentID, userID, req.Description, createdAt, 0, 0, 0, req.Location.LocationName, req.Location.Point.Latitude, req.Location.Point.Longitude, privacy, geo.LeafCellID(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude})); err != nil {
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
	if len(req.ImageFileIDs) > 0 {
		if err := insertMomentImages(ctx, queryable, momentID, req.ImageFileIDs, now); err != nil {
			return MomentItem{}, err
		}
	}
	nickname, err := lookupUserNickname(ctx, queryable, userID)
	if err != nil {
		return MomentItem{}, err
	}
	if nickname == "" {
		nickname = userID
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
		User:          UserInfo{UserID: userID, Nickname: nickname},
		Location:      req.Location,
		PrivacyLevel:  privacy,
		MyState:       &MyMomentState{IsLiked: false, IsFavorited: false},
	}, nil
}

func lookupUserNickname(ctx context.Context, queryable persistence.QueryableDB, userID string) (string, error) {
	var nickname string
	err := queryable.QueryRow(ctx, queryGetUserNickname, userID).Scan(&nickname)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return nickname, nil
}

func (r *PostgresRepository) IncrementCommentCount(ctx context.Context, momentID string) error {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryIncrementMomentCommentCount, momentID)
}

func (r *PostgresRepository) DecrementCommentCount(ctx context.Context, momentID string) error {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDecrementMomentCommentCount, momentID)
}

func (r *PostgresRepository) IncrementSharesCount(ctx context.Context, momentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryIncrementMomentShareCount, momentID)
}

func (r *PostgresRepository) Search(ctx context.Context, req SearchMomentsRequest) (PaginatedMoments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedMoments{}, persistence.ErrNotImplemented
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
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "relevance"
	}
	refPt := geo.Point{Latitude: req.ReferencePoint.Latitude, Longitude: req.ReferencePoint.Longitude}
	radius := 50000
	if req.Radius != nil && *req.Radius > 0 {
		radius = *req.Radius
	}
	if radius > 50000 {
		radius = 50000
	}
	bounds := geo.BoundsAround(refPt, radius)
	sql, args := buildSearchMomentsQuery(bounds, req.Keyword, req.Tag, sortBy, limit+1)
	if sql == "" {
		return PaginatedMoments{}, nil
	}
	rows, err := queryable.Query(ctx, sql, args...)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	plan := geo.NewSearchPlan(refPt, radius, sortBy)
	items := make([]MomentItem, 0)
	momentIDs := make([]string, 0)
	for rows.Next() {
		item, err := scanMomentRow(rows)
		if err != nil {
			return PaginatedMoments{}, err
		}
		_, _, within := plan.Match(geo.Point{Latitude: item.Location.Point.Latitude, Longitude: item.Location.Point.Longitude})
		if !within {
			continue
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
	sortKeyword := ""
	if req.Keyword != nil {
		sortKeyword = *req.Keyword
	}
	sortTag := ""
	if req.Tag != nil {
		sortTag = *req.Tag
	}
	sortMoments(items, sortBy, refPt, sortKeyword, sortTag)
	if len(items) > limit {
		return PaginatedMoments{Items: items[:limit], NextCursor: items[limit].MomentID}, nil
	}
	return PaginatedMoments{Items: items}, nil
}

func buildSearchMomentsQuery(bounds geo.Bounds, keyword *string, tag *string, sortBy string, limit int) (string, []any) {
	mins, maxs := geo.CoveringArrays(bounds)
	if len(mins) == 0 {
		return "", nil
	}
	args := make([]any, 0, 6)
	var builder strings.Builder
	builder.WriteString(`SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level FROM moments m LEFT JOIN users u ON u.user_id = m.user_id WHERE m.privacy_level = 'public' AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi) WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)`)
	args = append(args, mins, maxs)
	if keyword != nil && *keyword != "" {
		args = append(args, *keyword)
		builder.WriteString(fmt.Sprintf(" AND description ILIKE '%%' || $%d || '%%'", len(args)))
	}
	if tag != nil && *tag != "" {
		args = append(args, *tag)
		builder.WriteString(fmt.Sprintf(" AND moment_id IN (SELECT moment_id FROM moment_tags WHERE tag_name = $%d)", len(args)))
	}
	if sortBy == "newest" {
		builder.WriteString(" ORDER BY created_at DESC, moment_id DESC")
		args = append(args, limit)
		builder.WriteString(fmt.Sprintf(" LIMIT $%d", len(args)))
	} else {
		// Non-newest sorts rank in Go after the geo radius prefilter, so we
		// cannot push the page LIMIT to the DB. Cap the scan with a hard
		// ceiling so a dense viewport can never pull the whole table into
		// memory. The ceiling comfortably covers a page plus headroom for the
		// radius filter to discard candidates.
		args = append(args, maxSearchScanRows)
		builder.WriteString(fmt.Sprintf(" LIMIT $%d", len(args)))
	}
	return builder.String(), args
}

func (r *PostgresRepository) Update(ctx context.Context, userID string, req EditMomentRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var locationName, latStr, lngStr *string
	var s2CellID *int64
	if req.Location != nil {
		locationName = &req.Location.LocationName
		s := fmt.Sprintf("%f", req.Location.Point.Latitude)
		latStr = &s
		s2 := fmt.Sprintf("%f", req.Location.Point.Longitude)
		lngStr = &s2
		id := geo.LeafCellID(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude})
		s2CellID = &id
	}
	if err := queryable.Exec(ctx, queryUpdateMoment, req.MomentID, req.Description, req.PrivacyLevel, locationName, latStr, lngStr, s2CellID); err != nil {
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
	if req.ImageFileIDs != nil {
		if err := replaceMomentImages(ctx, queryable, req.MomentID, req.ImageFileIDs, time.Now().UTC()); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, momentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteMoment, momentID)
}

func (r *PostgresRepository) Like(ctx context.Context, userID string, momentID string) (bool, error) {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	// Single atomic statement: insert the like row and bump the counter only if
	// the insert was real (not a conflict). The existence check used to live in
	// a separate query, leaving a window where two concurrent likes could both
	// pass it and double-increment the counter.
	now := time.Now().UTC()
	var inserted bool
	if err := queryable.QueryRow(ctx, queryInsertMomentLikeAndCount, momentID, userID, now).Scan(&inserted); err != nil {
		// On conflict the UPDATE matches no rows and returns no row; treat that
		// (pgx.ErrNoRows) as "already liked" rather than an error.
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return inserted, nil
}

func (r *PostgresRepository) Unlike(ctx context.Context, userID string, momentID string) error {
	queryable, ok := queryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	// Single atomic statement: delete the like row and decrement only if a row
	// was actually removed, so repeated unlikes never undercount.
	var deleted bool
	if err := queryable.QueryRow(ctx, queryDeleteMomentLikeAndCount, momentID, userID).Scan(&deleted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	return nil
}

func (r *PostgresRepository) Favorite(ctx context.Context, userID string, momentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryInsertMomentFavorite, userID, momentID, time.Now().UTC())
}

func (r *PostgresRepository) Unfavorite(ctx context.Context, userID string, momentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryDeleteMomentFavorite, userID, momentID)
}

func (r *PostgresRepository) ListLikes(ctx context.Context, momentID string, limit int, cursor string) ([]LikeItem, string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, "", persistence.ErrNotImplemented
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
		var likedAt time.Time
		if err := rows.Scan(&item.User.UserID, &item.User.Nickname, &likedAt); err != nil {
			return nil, "", err
		}
		item.LikedAt = likedAt.UTC().Format(time.RFC3339)
		items = append(items, item)
		cursors = append(cursors, encodeMomentLikeCursor(item.LikedAt, item.User.UserID))
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
		return PaginatedMoments{}, persistence.ErrNotImplemented
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

// maxSearchScanRows bounds the worst-case number of rows a geo/ranking search
// loads into memory before the Go-side radius prefilter and sort. It is a
// safety ceiling against unbounded scans in dense viewports, not the page size
// (which is applied after sorting). 500 comfortably exceeds any single result
// page while keeping memory predictable.
const maxSearchScanRows = 500

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
		return nil, "", ErrInvalidCursor
	}
	var decoded momentInteractionCursor
	if err := json.Unmarshal(blob, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", ErrInvalidCursor
	}
	likedAt, err := time.Parse(time.RFC3339, decoded.At)
	if err != nil {
		return nil, "", ErrInvalidCursor
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
		return nil, "", ErrInvalidCursor
	}
	var decoded momentInteractionCursor
	if err := json.Unmarshal(blob, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
		return nil, "", ErrInvalidCursor
	}
	favoritedAt, err := time.Parse(time.RFC3339, decoded.At)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return favoritedAt.UTC(), decoded.ID, nil
}

// queryable resolves the Queryable to use for this request: an in-flight
// transaction bound to ctx when present, otherwise the shared pool. Returns
// ok=false when persistence is not configured (in-memory mode).
func queryable(ctx context.Context, factory *persistence.Factory) (persistence.Queryable, bool) {
	if q, ok := persistence.QueryableFromContext(ctx); ok {
		return q, true
	}
	if factory == nil || factory.DB == nil || factory.Config.PostgresDSN == "" {
		return nil, false
	}
	q, ok := factory.DB.(persistence.QueryableDB)
	return q, ok
}
