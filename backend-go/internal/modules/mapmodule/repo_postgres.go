package mapmodule

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/textutil"
	"hotspot/backend-go/internal/platform/types"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

type mapCandidate struct {
	ObjectType   string
	ObjectID     string
	Title        string
	Subtitle     string
	CoverURL     string
	Location     GeoPoint
	LocationName string
	CreatedAt    string
	Distance     int
	Bearing      int
}

type mapCursor struct {
	Distance  int    `json:"distance"`
	Type      string `json:"type"`
	ID        string `json:"id"`
	CreatedAt string `json:"created_at,omitempty"`
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Heatmap(ctx context.Context, query canvasQuery) (MapHeatCanvas, error) {
	items, err := r.loadCandidates(ctx, query)
	if err != nil {
		return MapHeatCanvas{}, err
	}
	buckets := aggregateHeatBuckets(items, query.Level)
	return MapHeatCanvas{Items: buckets}, nil
}

func (r *PostgresRepository) Hotspots(ctx context.Context, query canvasQuery) (MapHotspotsCanvas, error) {
	items, err := r.loadCandidates(ctx, query)
	if err != nil {
		return MapHotspotsCanvas{}, err
	}
	return paginateHotspots(items, query.Limit, query.Cursor)
}

func (r *PostgresRepository) loadCandidates(ctx context.Context, query canvasQuery) ([]mapCandidate, error) {
	queryable, err := r.queryable(ctx)
	if err != nil {
		return nil, err
	}
	// Compute the S2 covering arrays once for the request bounds; the three
	// queryMap* helpers used to each recompute geo.CoveringArrays themselves.
	mins, maxs := geo.CoveringArrays(query.Bounds)
	if len(mins) == 0 {
		return nil, nil
	}
	items := make([]mapCandidate, 0)
	// The three candidate queries are intentionally kept sequential. Running
	// them concurrently with errgroup would hold up to three pgxpool
	// connections per request, which risks pool exhaustion under load and the
	// extra latency from connection waits can eclipse the saved query time.
	// The single-covering optimisation above already removes the redundant
	// geo work that was the dominant per-request overhead.
	if query.DataType == "all" || query.DataType == "user" {
		users, err := queryMapUsers(ctx, queryable, query, mins, maxs)
		if err != nil {
			return nil, err
		}
		items = append(items, users...)
	}
	if query.DataType == "all" || query.DataType == "group" {
		groups, err := queryMapGroups(ctx, queryable, query, mins, maxs)
		if err != nil {
			return nil, err
		}
		items = append(items, groups...)
	}
	if query.DataType == "all" || query.DataType == "moment" {
		moments, err := queryMapMoments(ctx, queryable, query, mins, maxs)
		if err != nil {
			return nil, err
		}
		items = append(items, moments...)
	}
	sort.SliceStable(items, func(i, j int) bool { return compareMapCandidates(items[i], items[j]) < 0 })
	return items, nil
}

func (r *PostgresRepository) queryable(ctx context.Context) (persistence.Queryable, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	return queryable, nil
}

func queryMapUsers(ctx context.Context, queryable persistence.Queryable, query canvasQuery, mins, maxs []int64) ([]mapCandidate, error) {
	rows, err := queryable.Query(ctx, queryMapCanvasUsers, mins, maxs, maxMapScanRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]mapCandidate, 0)
	for rows.Next() {
		var id, title, coverURL, subtitle string
		var lat, lng float64
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &coverURL, &subtitle, &lat, &lng, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, newMapCandidate("user", id, title, subtitle, coverURL, lat, lng, createdAt.UTC().Format(time.RFC3339), query.Center))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func queryMapGroups(ctx context.Context, queryable persistence.Queryable, query canvasQuery, mins, maxs []int64) ([]mapCandidate, error) {
	rows, err := queryable.Query(ctx, queryMapCanvasGroups, mins, maxs, maxMapScanRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]mapCandidate, 0)
	for rows.Next() {
		var id, title, coverURL, subtitle string
		var lat, lng float64
		if err := rows.Scan(&id, &title, &coverURL, &subtitle, &lat, &lng); err != nil {
			return nil, err
		}
		items = append(items, newMapCandidate("group", id, title, subtitle, coverURL, lat, lng, "", query.Center))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func queryMapMoments(ctx context.Context, queryable persistence.Queryable, query canvasQuery, mins, maxs []int64) ([]mapCandidate, error) {
	var startTime, endTime *time.Time
	if !query.StartTime.IsZero() {
		t := query.StartTime.UTC()
		startTime = &t
	}
	if !query.EndTime.IsZero() {
		t := query.EndTime.UTC()
		endTime = &t
	}
	rows, err := queryable.Query(ctx, queryMapCanvasMoments, mins, maxs, maxMapScanRows, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]mapCandidate, 0)
	for rows.Next() {
		var id, title, subtitle string
		var lat, lng float64
		var createdAt time.Time
		if err := rows.Scan(&id, &title, &subtitle, &lat, &lng, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, newMapCandidate("moment", id, truncateMapTitle(title), subtitle, "", lat, lng, createdAt.UTC().Format(time.RFC3339), query.Center))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func newMapCandidate(objectType, id, title, subtitle, coverURL string, lat, lng float64, createdAt string, center geo.Point) mapCandidate {
	location := GeoPoint{Latitude: lat, Longitude: lng}
	return mapCandidate{
		ObjectType: objectType,
		ObjectID:   id,
		Title:      title,
		Subtitle:   subtitle,
		CoverURL:   coverURL,
		Location:   location,
		CreatedAt:  createdAt,
		Distance:   geo.DistanceMeters(center, geo.Point{Latitude: lat, Longitude: lng}),
	}
}

func aggregateHeatBuckets(items []mapCandidate, level int) []MapHeatBucket {
	byToken := map[string]*MapHeatBucket{}
	for _, item := range items {
		token := geo.CellTokenAtLevel(geo.Point{Latitude: item.Location.Latitude, Longitude: item.Location.Longitude}, level)
		bucket, ok := byToken[token]
		if !ok {
			bucket = &MapHeatBucket{CellToken: token, Level: level, Location: item.Location, Breakdown: &MapHeatBreakdown{}}
			byToken[token] = bucket
		}
		bucket.TotalCount++
		switch item.ObjectType {
		case "user":
			if bucket.Breakdown.User == nil {
				zero := 0
				bucket.Breakdown.User = &zero
			}
			*bucket.Breakdown.User++
		case "group":
			if bucket.Breakdown.Group == nil {
				zero := 0
				bucket.Breakdown.Group = &zero
			}
			*bucket.Breakdown.Group++
		case "moment":
			if bucket.Breakdown.Moment == nil {
				zero := 0
				bucket.Breakdown.Moment = &zero
			}
			*bucket.Breakdown.Moment++
		}
	}
	out := make([]MapHeatBucket, 0, len(byToken))
	for _, bucket := range byToken {
		out = append(out, *bucket)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].TotalCount != out[j].TotalCount {
			return out[i].TotalCount > out[j].TotalCount
		}
		return out[i].CellToken < out[j].CellToken
	})
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}

func paginateHotspots(items []mapCandidate, limit int, cursor string) (MapHotspotsCanvas, error) {
	decoded, err := decodeMapCursor(cursor)
	if err != nil {
		return MapHotspotsCanvas{}, err
	}
	start := 0
	if decoded != nil {
		start = findMapCursorStart(items, *decoded)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := MapHotspotsCanvas{Items: make([]MapHotspotItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		distance := float64(item.Distance)
		bearing := float64(item.Bearing)
		resp.Items = append(resp.Items, MapHotspotItem{
			ObjectType: item.ObjectType,
			ObjectID:   item.ObjectID,
			Title:      item.Title,
			Subtitle:   types.StrPtr(item.Subtitle),
			CoverURL:   types.StrPtr(item.CoverURL),
			Location: types.RelativeLocation{
				Point:          types.GeoPoint{Latitude: item.Location.Latitude, Longitude: item.Location.Longitude},
				LocationName:   item.LocationName,
				Distance:       &distance,
				BearingDegrees: &bearing,
			},
			CreatedAt: types.StrPtr(item.CreatedAt),
		})
	}
	if end < len(items) && end > start {
		c := encodeMapCursor(items[end-1])
		resp.NextCursor = &c
	}
	return resp, nil
}

func compareMapCandidates(left, right mapCandidate) int {
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if left.ObjectType != right.ObjectType {
		if left.ObjectType < right.ObjectType {
			return -1
		}
		return 1
	}
	if left.CreatedAt != right.CreatedAt {
		if left.CreatedAt > right.CreatedAt {
			return -1
		}
		return 1
	}
	if left.ObjectID < right.ObjectID {
		return -1
	}
	if left.ObjectID > right.ObjectID {
		return 1
	}
	return 0
}

func normalizeMapDataType(dataType string) string {
	normalized := strings.ToLower(strings.TrimSpace(dataType))
	switch normalized {
	case "", "all":
		return "all"
	case "user", "group", "moment":
		return normalized
	default:
		return "invalid"
	}
}

func normalizeMapLimit(limit int) int {
	switch {
	case limit <= 0:
		return 50
	case limit > 200:
		return 200
	default:
		return limit
	}
}

func truncateMapTitle(title string) string {
	return textutil.TruncateRunes(title, 60)
}

// mapCursorCodec is a process-wide cached AES-GCM codec for map cursors,
// keyed with a namespace salt to isolate cursors across modules.
var mapCursorCodec = cursor.NewCodec("map")

func encodeMapCursor(item mapCandidate) string {
	payload := mapCursor{Distance: item.Distance, Type: item.ObjectType, ID: item.ObjectID, CreatedAt: item.CreatedAt}
	token, err := mapCursorCodec.Encode(payload)
	if err != nil {
		return ""
	}
	return token
}

func decodeMapCursor(raw string) (*mapCursor, error) {
	if raw == "" {
		return nil, nil
	}
	var decoded mapCursor
	if err := mapCursorCodec.Decode(raw, &decoded); err != nil {
		return nil, ErrInvalidCursor
	}
	if decoded.Type == "" || decoded.ID == "" {
		return nil, ErrInvalidCursor
	}
	return &decoded, nil
}

func findMapCursorStart(items []mapCandidate, decoded mapCursor) int {
	for idx, item := range items {
		cmp := compareMapCandidates(item, mapCandidate{ObjectType: decoded.Type, ObjectID: decoded.ID, CreatedAt: decoded.CreatedAt, Distance: decoded.Distance})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

var _ Repository = (*PostgresRepository)(nil)

var ErrInvalidCursor = errors.New("map: invalid cursor")
