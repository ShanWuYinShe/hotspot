package moments

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/idgen"
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
	momentID, err := idgen.WithPrefix("moment_")
	if err != nil {
		return MomentItem{}, err
	}
	privacy := req.PrivacyLevel
	if privacy == "" {
		privacy = "public"
	}
	if err := queryable.Exec(ctx, queryInsertMoment, momentID, userID, req.Description, createdAt, 0, 0, 0, req.Location.LocationName, req.Location.Point.Latitude, req.Location.Point.Longitude, privacy, geo.LeafCellID(geo.Point{Latitude: req.Location.Point.Latitude, Longitude: req.Location.Point.Longitude})); err != nil {
		return MomentItem{}, fmt.Errorf("moments.Create: insert moment: %w", err)
	}
	tags := normalizeMomentTags(req.Tags)
	allowedTags, err := filterAllowedMomentTags(ctx, queryable, tags)
	if err != nil {
		return MomentItem{}, fmt.Errorf("moments.Create: filter tags: %w", err)
	}
	if err := insertMomentTags(ctx, queryable, momentID, tags, now); err != nil {
		return MomentItem{}, fmt.Errorf("moments.Create: insert tags: %w", err)
	}
	normalizedMentions := normalizeMomentMentions(req.Mentions)
	if err := replaceMomentMentions(ctx, queryable, momentID, normalizedMentions, now); err != nil {
		return MomentItem{}, fmt.Errorf("moments.Create: insert mentions: %w", err)
	}
	if len(req.ImageFileIDs) > 0 {
		if err := insertMomentImages(ctx, queryable, momentID, req.ImageFileIDs, now); err != nil {
			return MomentItem{}, fmt.Errorf("moments.Create: insert images: %w", err)
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
		MyState:       &MyMomentState{IsFavorited: false},
	}, nil
}

func lookupUserNickname(ctx context.Context, queryable persistence.QueryableDB, userID string) (string, error) {
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
	pageToken := ""
	if req.Pagination != nil {
		if req.Pagination.Limit > 0 {
			limit = req.Pagination.Limit
		}
		if limit > 100 {
			limit = 100
		}
		pageToken = req.Pagination.Cursor
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
	pageCursor, err := decodeMomentsCursor(pageToken)
	if err != nil {
		return PaginatedMoments{}, ErrInvalidCursor
	}
	if pageCursor.SortBy == "" {
		pageCursor.SortBy = sortBy
	}
	sql, args := buildSearchMomentsQuery(req, refPt, radius, pageCursor, limit+1)
	if sql == "" {
		return PaginatedMoments{}, nil
	}
	rows, err := queryable.Query(ctx, sql, args...)
	if err != nil {
		return PaginatedMoments{}, fmt.Errorf("moments.Search query: %w", err)
	}
	defer rows.Close()
	items := make([]MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	var lastMeta momentSearchMeta
	for rows.Next() {
		item, meta, err := scanMomentRowWithMeta(rows)
		if err != nil {
			return PaginatedMoments{}, err
		}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		lastMeta = meta
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, fmt.Errorf("moments.Search rows: %w", err)
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, err
	}
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
		items[i].MyState = &MyMomentState{}
	}
	// Distance/radius filtering and ordering now happen in SQL; detect the
	// next page by fetching limit+1 and encoding a keyset cursor from the last
	// returned row.
	nextCursor := ""
	if len(items) > limit {
		last := items[limit-1]
		nextCursor = encodeMomentsCursor(momentSearchCursor{
			SortBy:    pageCursor.SortBy,
			MomentID:  last.MomentID,
			Distance:  lastMeta.distance,
			CreatedAt: last.CreatedAt,
			Popular:   last.LikesCount + last.CommentsCount,
		})
		items = items[:limit]
	}
	return PaginatedMoments{Items: items, NextCursor: nextCursor}, nil
}

// buildSearchMomentsQuery constructs the keyset-paginated moments search with
// distance (Haversine), radius filtering, popularity and relevance computed in
// SQL — mirroring buildSearchUsersQuery. The previous version pulled candidates
// into Go and ran plan.Match / sortMoments there, violating the
// "distance/range/sort must happen in PostgreSQL" rule (AGENTS.md).
func buildSearchMomentsQuery(req SearchMomentsRequest, ref geo.Point, radius int, cursor momentSearchCursor, limit int) (string, []any) {
	bounds := geo.BoundsAround(ref, radius)
	mins, maxs := geo.CoveringArrays(bounds)
	if len(mins) == 0 {
		return "", nil
	}
	args := make([]any, 0, 12)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	refLat := addArg(ref.Latitude)
	refLng := addArg(ref.Longitude)
	distanceExpr := geo.HaversineDistanceSQL(refLat, refLng)
	// Relevance mirrors relevanceScore: description containing the keyword (+1)
	// and a matching tag (+1), computed entirely in SQL.
	relevanceExpr := "0"
	keyword := ""
	if req.Keyword != nil {
		keyword = strings.TrimSpace(*req.Keyword)
	}
	tag := ""
	if req.Tag != nil {
		tag = strings.TrimSpace(*req.Tag)
	}
	if keyword != "" || tag != "" {
		var parts []string
		if keyword != "" {
			parts = append(parts, fmt.Sprintf("(CASE WHEN m.description ILIKE '%%' || %s || '%%' THEN 1 ELSE 0 END)", addArg(keyword)))
		}
		if tag != "" {
			parts = append(parts, fmt.Sprintf("(CASE WHEN EXISTS (SELECT 1 FROM moment_tags mt WHERE mt.moment_id = m.moment_id AND mt.tag_name = %s) THEN 1 ELSE 0 END)", addArg(tag)))
		}
		relevanceExpr = strings.Join(parts, " + ")
	}
	var builder strings.Builder
	builder.WriteString("SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude, m.privacy_level")
	builder.WriteString(", ")
	builder.WriteString(distanceExpr)
	builder.WriteString(" AS distance_meters, (m.likes_count + m.comments_count + m.shares_count) AS popularity, ")
	builder.WriteString(relevanceExpr)
	builder.WriteString(" AS relevance_score")
	builder.WriteString(" FROM moments m LEFT JOIN users u ON u.user_id = m.user_id")
	builder.WriteString(" WHERE m.privacy_level = 'public'")
	minArg := addArg(mins)
	maxArg := addArg(maxs)
	builder.WriteString(fmt.Sprintf(" AND EXISTS (SELECT 1 FROM unnest(%s::bigint[], %s::bigint[]) AS t(lo, hi) WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)", minArg, maxArg))
	builder.WriteString(fmt.Sprintf(" AND %s <= %s", distanceExpr, addArg(radius)))
	if keyword != "" {
		builder.WriteString(fmt.Sprintf(" AND m.description ILIKE '%%' || %s || '%%'", addArg(keyword)))
	}
	if tag != "" {
		builder.WriteString(fmt.Sprintf(" AND EXISTS (SELECT 1 FROM moment_tags mt WHERE mt.moment_id = m.moment_id AND mt.tag_name = %s)", addArg(tag)))
	}
	// Keyset predicate (secondary key: moment_id ASC for a stable tiebreak).
	if cursor.MomentID != "" {
		switch cursor.SortBy {
		case "distance":
			d := addArg(cursor.Distance)
			m := addArg(cursor.MomentID)
			builder.WriteString(fmt.Sprintf(" AND (distance_meters > %s OR (distance_meters = %s AND m.moment_id > %s))", d, d, m))
		case "newest":
			c := addArg(cursor.CreatedAt)
			m := addArg(cursor.MomentID)
			builder.WriteString(fmt.Sprintf(" AND (m.created_at < %s::timestamptz OR (m.created_at = %s::timestamptz AND m.moment_id > %s))", c, c, m))
		case "popular":
			p := addArg(cursor.Popular)
			c := addArg(cursor.CreatedAt)
			m := addArg(cursor.MomentID)
			builder.WriteString(fmt.Sprintf(" AND (popularity < %s OR (popularity = %s AND (m.created_at < %s::timestamptz OR (m.created_at = %s::timestamptz AND m.moment_id > %s))))", p, p, c, c, m))
		default: // relevance
			r := addArg(cursor.Relevance)
			c := addArg(cursor.CreatedAt)
			m := addArg(cursor.MomentID)
			builder.WriteString(fmt.Sprintf(" AND (relevance_score < %s OR (relevance_score = %s AND (m.created_at < %s::timestamptz OR (m.created_at = %s::timestamptz AND m.moment_id > %s))))", r, r, c, c, m))
		}
	}
	switch cursor.SortBy {
	case "distance":
		builder.WriteString(" ORDER BY distance_meters ASC, m.moment_id ASC")
	case "newest":
		builder.WriteString(" ORDER BY m.created_at DESC, m.moment_id ASC")
	case "popular":
		builder.WriteString(" ORDER BY popularity DESC, m.created_at DESC, m.moment_id ASC")
	default:
		builder.WriteString(" ORDER BY relevance_score DESC, m.created_at DESC, m.moment_id ASC")
	}
	builder.WriteString(fmt.Sprintf(" LIMIT %s", addArg(limit)))
	return builder.String(), args
}

func (r *PostgresRepository) Update(ctx context.Context, userID string, req EditMomentRequest) error {
	q, ok := queryable(ctx, r.factory)
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
	if err := q.Exec(ctx, queryUpdateMoment, req.MomentID, req.Description, req.PrivacyLevel, locationName, latStr, lngStr, s2CellID); err != nil {
		return fmt.Errorf("moments.Update: update moment: %w", err)
	}
	if req.Tags != nil {
		if err := q.Exec(ctx, queryDeleteMomentTagsByMomentID, req.MomentID); err != nil {
			return fmt.Errorf("moments.Update: delete tags: %w", err)
		}
		if err := insertMomentTags(ctx, q, req.MomentID, req.Tags, time.Now().UTC()); err != nil {
			return fmt.Errorf("moments.Update: insert tags: %w", err)
		}
	}
	if req.Mentions != nil {
		if err := replaceMomentMentions(ctx, q, req.MomentID, req.Mentions, time.Now().UTC()); err != nil {
			return fmt.Errorf("moments.Update: replace mentions: %w", err)
		}
	}
	if req.ImageFileIDs != nil {
		if err := replaceMomentImages(ctx, q, req.MomentID, req.ImageFileIDs, time.Now().UTC()); err != nil {
			return fmt.Errorf("moments.Update: replace images: %w", err)
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

// momentSearchCursor is the keyset cursor for moment Search. Each sort mode
// carries the columns its ORDER BY uses so the next-page predicate resumes
// deterministically (secondary key: moment_id ASC).
type momentSearchCursor struct {
	SortBy    string `json:"sort_by"`
	MomentID  string `json:"moment_id"`
	Distance  int    `json:"distance,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Popular   int    `json:"popular,omitempty"`
	Relevance int    `json:"relevance,omitempty"`
}

// momentSearchMeta holds the computed columns read back from a search row, used
// only to encode the next-page cursor.
type momentSearchMeta struct {
	distance  int
	popular   int
	relevance int
}

const momentsSearchCursorNamespace = "moments.search"

func decodeMomentsCursor(raw string) (momentSearchCursor, error) {
	if raw == "" {
		return momentSearchCursor{}, nil
	}
	var decoded momentSearchCursor
	if err := cursor.Decode(momentsSearchCursorNamespace, raw, &decoded); err != nil {
		return momentSearchCursor{}, ErrInvalidCursor
	}
	if decoded.MomentID == "" {
		return momentSearchCursor{}, ErrInvalidCursor
	}
	return decoded, nil
}

func encodeMomentsCursor(c momentSearchCursor) string {
	value, err := cursor.Encode(momentsSearchCursorNamespace, c)
	if err != nil {
		return ""
	}
	return value
}

// scanMomentRowWithMeta scans a search row that carries the computed
// distance_meters / popularity / relevance_score columns appended by
// buildSearchMomentsQuery. Returns the item plus the meta needed to encode a
// follow-on cursor.
func scanMomentRowWithMeta(scanner interface{ Scan(...any) error }) (MomentItem, momentSearchMeta, error) {
	var item MomentItem
	var meta momentSearchMeta
	var userID, nickname, locationName, privacyLevel string
	var createdAt time.Time
	var latitude, longitude float64
	if err := scanner.Scan(&item.MomentID, &userID, &nickname, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel, &meta.distance, &meta.popular, &meta.relevance); err != nil {
		return MomentItem{}, momentSearchMeta{}, fmt.Errorf("moments.scanMomentRowWithMeta scan: %w", err)
	}
	if nickname == "" {
		nickname = userID
	}
	item.User = UserInfo{UserID: userID, Nickname: nickname}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.Location = Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.PrivacyLevel = privacyLevel
	return item, meta, nil
}
