package moments

import (
	"context"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	platformcursor "hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/types"
)

// LocationProvider 提供已登录用户最近一次 update_location 的位置（reference_point
// 回退链第 2 级，spec api_spec_types.md「reference_point 解析规则」）。由 users 模块
// 实现，模块间通过接口解耦，不直接依赖 users repo。
type LocationProvider interface {
	RecentLocation(ctx context.Context, userID string) (geo.Point, bool)
}

// momentsQuery 是 fetch_following / fetch_nearby 的查询载体（自 feed 模块迁移）。
// HasReferencePoint 标记参考点是否成功解析（spec api_spec_types.md「reference_point
// 解析规则」）：仅在该标记为真时填充响应侧 distance/bearing_degrees。
type momentsQuery struct {
	ViewerUserID      string
	ReferencePoint    GeoPoint
	HasReferencePoint bool
	Radius            int
	Limit             int
	Cursor            string
}

// momentCandidate 是发现流的排序候选（距离/热度/发布时间）。
type momentCandidate struct {
	Item      MomentItem
	Score     int
	Distance  int
	CreatedAt time.Time
}

// itemCursor 是发现流分页游标的载荷。
type itemCursor struct {
	Score     int    `json:"score,omitempty"`
	Distance  int    `json:"distance,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	ID        string `json:"id"`
}

// normalizeFetchLimit 收敛发现流分页大小（与 feed 时期口径一致：默认 20，上限 100）。
func normalizeFetchLimit(pagination *PaginationParams) int {
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

func isValidGeoPoint(point GeoPoint) bool {
	return point.Latitude >= -90 && point.Latitude <= 90 && point.Longitude >= -180 && point.Longitude <= 180
}

// resolveReferencePoint 实现 reference_point 回退链（spec api_spec_types.md）：
// 1. 请求显式传入（合法）→ 直接使用
// 2. 已登录用户最近一次 update_location 的位置（匿名或未更新时跳过）
// 3. ClientConfig.defaults.default_location.point（服务端 appConfig 提供）
// 仅当显式传入的坐标非法时返回 nil。
func (s *Service) resolveReferencePoint(ctx context.Context, ref *GeoPoint) *GeoPoint {
	if ref != nil && !isValidGeoPoint(*ref) {
		return nil
	}
	if ref != nil {
		return ref
	}
	// 回退 2：已登录用户的最近位置
	if principal, ok := platformauth.PrincipalFromContext(ctx); ok && principal.UserID != "" && s.locations != nil {
		if pt, found := s.locations.RecentLocation(ctx, principal.UserID); found {
			return &GeoPoint{Latitude: pt.Latitude, Longitude: pt.Longitude}
		}
	}
	// 回退 3：default_location
	lat, lng := geo.DefaultFallbackLatitude, geo.DefaultFallbackLongitude
	if s.appConfig != nil {
		def := s.appConfig.Get(ctx).Defaults.DefaultLocation
		lat, lng = def.Point.Latitude, def.Point.Longitude
	}
	return &GeoPoint{Latitude: lat, Longitude: lng}
}

// normalizeRadius 收敛附近发现的搜索半径：缺省 5000 米并受 search_radius_max 裁剪。
func (s *Service) normalizeRadius(ctx context.Context, radius *int) int {
	value := 5000
	if radius != nil && *radius > 0 {
		value = *radius
	}
	if s.appConfig != nil {
		if maxRadius := s.appConfig.Get(ctx).Limits.SearchRadiusMax; maxRadius > 0 && value > maxRadius {
			value = maxRadius
		}
	}
	return value
}

// compareFollowingCandidates 排序规则（spec moment.md → fetch_following）：
// 发布时间降序（最新在前）→ momentID 降序（keyset 强一致）。
func compareFollowingCandidates(left, right momentCandidate) int {
	if !left.CreatedAt.Equal(right.CreatedAt) {
		if left.CreatedAt.After(right.CreatedAt) {
			return -1
		}
		return 1
	}
	if left.Item.MomentID > right.Item.MomentID {
		return -1
	}
	if left.Item.MomentID < right.Item.MomentID {
		return 1
	}
	return 0
}

// compareNearbyCandidates 排序规则（spec moment.md → fetch_nearby 排序主序）：
// 距离升序为主序（服务端计算 distance）；热度评分仅作为次级信号——实现为
// 距离升序 → 热度降序 → 发布时间降序 → momentID 降序。
func compareNearbyCandidates(left, right momentCandidate) int {
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if left.Score != right.Score {
		return right.Score - left.Score
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		if left.CreatedAt.After(right.CreatedAt) {
			return -1
		}
		return 1
	}
	if left.Item.MomentID > right.Item.MomentID {
		return -1
	}
	if left.Item.MomentID < right.Item.MomentID {
		return 1
	}
	return 0
}

var fetchCursorCodec = platformcursor.NewCodec("moments.fetch")

func encodeFetchCursor(cursor itemCursor) string {
	token, err := fetchCursorCodec.Encode(cursor)
	if err != nil {
		return ""
	}
	return token
}

func decodeFetchCursor(cursor string) (*itemCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	var decoded itemCursor
	if err := fetchCursorCodec.Decode(cursor, &decoded); err != nil {
		return nil, ErrInvalidCursor
	}
	if decoded.ID == "" {
		return nil, ErrInvalidCursor
	}
	if decoded.CreatedAt != "" {
		if _, err := time.Parse(time.RFC3339Nano, decoded.CreatedAt); err != nil {
			return nil, ErrInvalidCursor
		}
	}
	return &decoded, nil
}

// paginateFollowingMoments：keyset 谓词与 LIMIT 已下推 SQL，items ≤ limit+1
// （+1 用于探测下一页），此处仅裁剪并生成下一页游标。
func paginateFollowingMoments(items []momentCandidate, limit int, cursor string) (PaginatedMoments, error) {
	if _, err := decodeFetchCursor(cursor); err != nil {
		return PaginatedMoments{}, err
	}
	if limit <= 0 {
		limit = 20
	}
	end := min(len(items), limit)
	resp := PaginatedMoments{Items: make([]MomentItem, 0, end)}
	for _, item := range items[:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if len(items) > limit && end > 0 {
		last := items[end-1]
		resp.NextCursor = types.StrPtr(encodeFetchCursor(itemCursor{CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339), ID: last.Item.MomentID}))
	}
	return resp, nil
}

// paginateNearbyMoments：候选集已在 SQL/内存层按最终序排好，此处按游标定位后裁剪。
func paginateNearbyMoments(items []momentCandidate, limit int, cursor string) (PaginatedMoments, error) {
	decoded, err := decodeFetchCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	if limit <= 0 {
		limit = 20
	}
	start := findNearbyCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedMoments{Items: make([]MomentItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		last := items[end-1]
		resp.NextCursor = types.StrPtr(encodeFetchCursor(itemCursor{Score: last.Score, Distance: last.Distance, CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339), ID: last.Item.MomentID}))
	}
	return resp, nil
}

func findNearbyCursorStart(items []momentCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	parsedAt, _ := time.Parse(time.RFC3339Nano, cursor.CreatedAt)
	for idx, item := range items {
		cmp := compareNearbyCandidates(item, momentCandidate{Item: MomentItem{MomentID: cursor.ID}, Score: cursor.Score, Distance: cursor.Distance, CreatedAt: parsedAt.UTC()})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}
