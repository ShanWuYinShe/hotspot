package moments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/idgen"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

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
	now := time.Now().UTC()
	createdAt := now.Format(time.RFC3339)
	momentID, err := idgen.WithPrefix("moment_")
	if err != nil {
		return MomentItem{}, err
	}
	privacy := req.PrivacyLevel
	if privacy == "" {
		privacy = "public"
	}
	// moment + tags + mentions + images 五步写包进同一事务：任一步失败
	// 整体回滚，避免「moment 已落库但接口报错」导致客户端重试产生重复。
	var allowedTags []string
	normalizedMentions := normalizeMomentMentions(req.Mentions)
	nickname := userID
	if err := persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		txQueryable, ok := queryable(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := txQueryable.Exec(txCtx, queryInsertMoment, momentID, userID, req.Description, createdAt, 0, 0, 0, req.Location.LocationName, req.Location.Point.Latitude, req.Location.Point.Longitude, privacy, geo.LeafCellID(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude})); err != nil {
			return fmt.Errorf("moments.Create: insert moment: %w", err)
		}
		tags := normalizeMomentTags(req.Tags)
		allowed, err := filterAllowedMomentTags(txCtx, txQueryable, tags)
		if err != nil {
			return fmt.Errorf("moments.Create: filter tags: %w", err)
		}
		allowedTags = allowed
		if err := insertMomentTags(txCtx, txQueryable, momentID, tags, now); err != nil {
			return fmt.Errorf("moments.Create: insert tags: %w", err)
		}
		if err := replaceMomentMentions(txCtx, txQueryable, momentID, normalizedMentions, now); err != nil {
			return fmt.Errorf("moments.Create: insert mentions: %w", err)
		}
		if len(req.ImageFileIDs) > 0 {
			if err := insertMomentImages(txCtx, txQueryable, momentID, req.ImageFileIDs, now); err != nil {
				return fmt.Errorf("moments.Create: insert images: %w", err)
			}
		}
		n, err := lookupUserNickname(txCtx, txQueryable, userID)
		if err != nil {
			return err
		}
		if n != "" {
			nickname = n
		}
		return nil
	}); err != nil {
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
		User:          UserInfo{UserID: userID, Nickname: nickname},
		Location:      req.Location,
		PrivacyLevel:  privacy,
		MyState:       &MyMomentState{IsFavorited: false},
	}, nil
}

func lookupUserNickname(ctx context.Context, queryable persistence.Queryable, userID string) (string, error) {
	var nickname string
	err := queryable.QueryRow(ctx, queryGetUserNickname, userID).Scan(&nickname)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
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
	if limit > types.MaxPageSize {
		limit = types.MaxPageSize
	}
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "relevance"
	}
	sql, args := buildSearchMomentsQuery(req.Keyword, req.Tag, sortBy, limit+1)
	if sql == "" {
		return PaginatedMoments{}, nil
	}
	rows, err := queryable.Query(ctx, sql, args...)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]MomentItem, 0)
	momentIDs := make([]string, 0)
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
	sortKeyword := ""
	if req.Keyword != nil {
		sortKeyword = *req.Keyword
	}
	sortTag := ""
	if req.Tag != nil {
		sortTag = *req.Tag
	}
	sortMoments(items, sortBy, sortKeyword, sortTag)
	if len(items) > limit {
		return PaginatedMoments{Items: items[:limit], NextCursor: items[limit].MomentID}, nil
	}
	return PaginatedMoments{Items: items}, nil
}

// buildSearchMomentsQuery：全球 keyword/tag 搜索（spec moment.md → moments/search 收口，
// 无半径过滤；附近发现走 feed/nearby）
func buildSearchMomentsQuery(keyword *string, tag *string, sortBy string, limit int) (string, []any) {
	args := make([]any, 0, 6)
	var builder strings.Builder
	builder.WriteString(`SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level, m.is_deleted FROM moments m LEFT JOIN users u ON u.user_id = m.user_id WHERE m.privacy_level = 'public' AND m.is_deleted = FALSE`)
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

func (r *PostgresRepository) Update(ctx context.Context, userID string, req UpdateMomentRequest) error {
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
	// update + tags/mentions/images 替换包进同一事务：中途失败不会留下
	// 「标签已删但未插回」的半更新状态。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		q, ok := queryable(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := q.Exec(txCtx, queryUpdateMoment, req.MomentID, req.Description, req.PrivacyLevel, locationName, latStr, lngStr, s2CellID); err != nil {
			return fmt.Errorf("moments.Update: update moment: %w", err)
		}
		if req.Tags != nil {
			if err := q.Exec(txCtx, queryDeleteMomentTagsByMomentID, req.MomentID); err != nil {
				return fmt.Errorf("moments.Update: delete tags: %w", err)
			}
			if err := insertMomentTags(txCtx, q, req.MomentID, req.Tags, time.Now().UTC()); err != nil {
				return fmt.Errorf("moments.Update: insert tags: %w", err)
			}
		}
		if req.Mentions != nil {
			if err := replaceMomentMentions(txCtx, q, req.MomentID, req.Mentions, time.Now().UTC()); err != nil {
				return fmt.Errorf("moments.Update: replace mentions: %w", err)
			}
		}
		if req.ImageFileIDs != nil {
			if err := replaceMomentImages(txCtx, q, req.MomentID, req.ImageFileIDs, time.Now().UTC()); err != nil {
				return fmt.Errorf("moments.Update: replace images: %w", err)
			}
		}
		return nil
	})
}

func (r *PostgresRepository) Delete(ctx context.Context, momentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, queryDeleteMoment, momentID); err != nil {
		return fmt.Errorf("moments.Delete: %w", err)
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
		item.FavoritedAt = favoritedAt // spec moment.md → FavoritedMomentItem.favorited_at
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

func (r *PostgresRepository) ListFavoriteUserIDs(ctx context.Context, momentID string) ([]string, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListMomentFavoriteUserIDs, momentID)
	if err != nil {
		return nil, fmt.Errorf("moments.ListFavoriteUserIDs query: %w", err)
	}
	defer rows.Close()
	var userIDs []string
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("moments.ListFavoriteUserIDs scan: %w", err)
		}
		userIDs = append(userIDs, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("moments.ListFavoriteUserIDs rows: %w", err)
	}
	return userIDs, nil
}

type momentInteractionCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

func normalizeMomentInteractionLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > types.MaxPageSize:
		return types.MaxPageSize
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

// momentFavoriteCursorNamespace 是收藏列表分页游标的加密命名空间。
const momentFavoriteCursorNamespace = "moments.favorites"

func encodeMomentFavoriteCursor(favoritedAt, momentID string) string {
	value, err := cursor.Encode(momentFavoriteCursorNamespace, momentInteractionCursor{At: favoritedAt, ID: momentID})
	if err != nil {
		return ""
	}
	return value
}

func decodeMomentFavoriteCursor(cursorValue string) (any, string, error) {
	if cursorValue == "" {
		return nil, "", nil
	}
	var decoded momentInteractionCursor
	if err := cursor.Decode(momentFavoriteCursorNamespace, cursorValue, &decoded); err != nil || decoded.At == "" || decoded.ID == "" {
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
