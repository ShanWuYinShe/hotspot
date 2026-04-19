package mapmodule

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/config"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

type mapCandidate struct {
	ObjectType string
	ObjectID   string
	Title      string
	Subtitle   string
	CoverURL   string
	Location   GeoPoint
	CreatedAt  string
	Distance   int
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
	queryable, err := r.queryable()
	if err != nil {
		return nil, err
	}
	items := make([]mapCandidate, 0)
	if query.DataType == "all" || query.DataType == "user" {
		users, err := queryMapUsers(ctx, queryable, query)
		if err != nil {
			return nil, err
		}
		items = append(items, users...)
	}
	if query.DataType == "all" || query.DataType == "group" {
		groups, err := queryMapGroups(ctx, queryable, query)
		if err != nil {
			return nil, err
		}
		items = append(items, groups...)
	}
	if query.DataType == "all" || query.DataType == "moment" {
		moments, err := queryMapMoments(ctx, queryable, query)
		if err != nil {
			return nil, err
		}
		items = append(items, moments...)
	}
	sort.SliceStable(items, func(i, j int) bool { return compareMapCandidates(items[i], items[j]) < 0 })
	return items, nil
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

func queryMapUsers(ctx context.Context, queryable persistence.QueryableDB, query canvasQuery) ([]mapCandidate, error) {
	rows, err := queryable.Query(ctx, queryMapCanvasUsers, query.Bounds.South, query.Bounds.North, query.Bounds.West, query.Bounds.East)
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

func queryMapGroups(ctx context.Context, queryable persistence.QueryableDB, query canvasQuery) ([]mapCandidate, error) {
	rows, err := queryable.Query(ctx, queryMapCanvasGroups, query.Bounds.South, query.Bounds.North, query.Bounds.West, query.Bounds.East)
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

func queryMapMoments(ctx context.Context, queryable persistence.QueryableDB, query canvasQuery) ([]mapCandidate, error) {
	rows, err := queryable.Query(ctx, queryMapCanvasMoments, query.Bounds.South, query.Bounds.North, query.Bounds.West, query.Bounds.East)
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
		if !query.StartTime.IsZero() && createdAt.UTC().Before(query.StartTime) {
			continue
		}
		if !query.EndTime.IsZero() && createdAt.UTC().After(query.EndTime) {
			continue
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
			bucket.Breakdown.User++
		case "group":
			bucket.Breakdown.Group++
		case "moment":
			bucket.Breakdown.Moment++
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
	resp := MapHotspotsCanvas{Items: make([]MapHotspotItem, 0, maxInt(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, MapHotspotItem{
			ObjectType:     item.ObjectType,
			ObjectID:       item.ObjectID,
			Title:          item.Title,
			Subtitle:       item.Subtitle,
			CoverURL:       item.CoverURL,
			Location:       item.Location,
			CreatedAt:      item.CreatedAt,
			DistanceMeters: item.Distance,
		})
	}
	if end < len(items) && end > start {
		resp.NextCursor = encodeMapCursor(items[end-1])
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
	title = strings.TrimSpace(title)
	runes := []rune(title)
	if len(runes) <= 60 {
		return title
	}
	return string(runes[:60])
}

func encodeMapCursor(item mapCandidate) string {
	block, err := newMapCursorBlock()
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	payload, err := json.Marshal(mapCursor{Distance: item.Distance, Type: item.ObjectType, ID: item.ObjectID, CreatedAt: item.CreatedAt})
	if err != nil {
		return ""
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return ""
	}
	sealed := gcm.Seal(nonce, nonce, payload, nil)
	return base64.RawURLEncoding.EncodeToString(sealed)
}

func decodeMapCursor(cursor string) (*mapCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	block, err := newMapCursorBlock()
	if err != nil {
		return nil, ErrInvalidCursor
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	blob, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil || len(blob) < gcm.NonceSize() {
		return nil, ErrInvalidCursor
	}
	nonce, ciphertext := blob[:gcm.NonceSize()], blob[gcm.NonceSize():]
	payload, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidCursor
	}
	var decoded mapCursor
	if err := json.Unmarshal(payload, &decoded); err != nil {
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

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func newMapCursorBlock() (cipher.Block, error) {
	secret := config.Load().AuthTokenSecret
	if secret == "" {
		secret = "dev-secret-change-me"
	}
	key := sha256.Sum256([]byte(secret))
	return aes.NewCipher(key[:])
}

var _ Repository = (*PostgresRepository)(nil)

var ErrInvalidCursor = errors.New("map: invalid cursor")
