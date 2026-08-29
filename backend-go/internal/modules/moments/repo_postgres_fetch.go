package moments

import (
	"context"
	"sort"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

// queryableDB 解析当前请求可用的 QueryableDB；persistence 未配置（内存模式）
// 时返回 false。
func (r *PostgresRepository) fetchQueryable(ctx context.Context) (persistence.Queryable, bool) {
	return persistence.RepoQueryable(ctx, r.factory)
}

// ListFollowingMoments 关注动态（spec api_spec_moment.md → moments/fetch_following）：
// keyset 谓词与 LIMIT 下推 SQL（此前 feed 实现拉取关注者全部历史进内存再排序分页）。
// 解析出参考点时同步填充 location.distance / bearing_degrees
// （spec api_spec_types.md RelativeLocation：两字段由服务端基于 reference_point 计算）。
func (r *PostgresRepository) ListFollowingMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	queryable, ok := r.fetchQueryable(ctx)
	if !ok {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	decoded, err := decodeFetchCursor(query.Cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	fetchLimit := query.Limit
	if fetchLimit <= 0 {
		fetchLimit = 20
	}
	fetchLimit++ // limit+1 探测下一页
	var cursorAt any
	var cursorID string
	if decoded != nil {
		cursorID = decoded.ID
		if decoded.CreatedAt != "" {
			if parsed, parseErr := time.Parse(time.RFC3339Nano, decoded.CreatedAt); parseErr == nil {
				cursorAt = parsed.UTC()
			}
		}
	}
	rows, err := queryable.Query(ctx, queryFollowingMoments, query.ViewerUserID, cursorAt, cursorID, fetchLimit)
	if err != nil {
		return PaginatedMoments{}, err
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
			return PaginatedMoments{}, err
		}
		_ = isLiked // 兼容保留列：likes_count 已从契约删除，reactions 聚合另行填充
		item := MomentItem{
			MomentID:      momentID,
			Description:   description,
			CreatedAt:     createdAt.UTC().Format(time.RFC3339),
			LikesCount:    likesCount,
			CommentsCount: commentsCount,
			SharesCount:   sharesCount,
			User:          UserInfo{UserID: userID, Nickname: nickname, AvatarURL: types.StrPtr(avatarURL)},
			Location:      &Location{Point: pointStruct(latitude, longitude), LocationName: locationName},
			IsFavorited:   isFavorited,
		}
		if query.HasReferencePoint {
			ref := geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}
			fillRelativeLocation(item.Location, geo.DistanceMeters(ref, geo.Point{Latitude: latitude, Longitude: longitude}), geo.BearingDegrees(ref, geo.Point{Latitude: latitude, Longitude: longitude}))
		}
		items = append(items, momentCandidate{Item: item, CreatedAt: createdAt.UTC()})
		momentIDs = append(momentIDs, momentID)
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, err
	}
	for idx := range items {
		items[idx].Item.Tags = tagsByMomentID[items[idx].Item.MomentID]
	}
	sort.SliceStable(items, func(i, j int) bool { return compareFollowingCandidates(items[i], items[j]) < 0 })
	return paginateFollowingMoments(items, query.Limit, query.Cursor)
}

// ListNearbyMoments 附近瞬间（spec moment.md → moments/fetch_nearby）：
// S2 覆盖区间粗筛 + geo.SearchPlan 精筛（radius 内）+ 距离主序排序。
func (r *PostgresRepository) ListNearbyMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	queryable, ok := r.fetchQueryable(ctx)
	if !ok {
		return PaginatedMoments{}, persistence.ErrNotImplemented
	}
	bounds := geo.BoundsAround(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius)
	mins, maxs := geo.CoveringArrays(bounds)
	if len(mins) == 0 {
		return PaginatedMoments{}, nil
	}
	rows, err := queryable.Query(ctx, queryNearbyMoments, mins, maxs, query.ViewerUserID)
	if err != nil {
		return PaginatedMoments{}, err
	}
	defer rows.Close()
	plan := geo.NewSearchPlan(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius, "relevance")
	items := make([]momentCandidate, 0)
	momentIDs := make([]string, 0)
	for rows.Next() {
		var momentID, userID, nickname, avatarURL, description, locationName string
		var latitude, longitude float64
		var createdAt time.Time
		var likesCount, commentsCount, sharesCount int
		var isLiked, isFavorited bool
		var isFollowing bool
		if err := rows.Scan(&momentID, &userID, &nickname, &avatarURL, &description, &createdAt, &likesCount, &commentsCount, &sharesCount, &locationName, &latitude, &longitude, &isLiked, &isFavorited, &isFollowing); err != nil {
			return PaginatedMoments{}, err
		}
		_ = isLiked
		distance, bearing, ok := plan.Match(geo.Point{Latitude: latitude, Longitude: longitude})
		if !ok {
			continue
		}
		score := likesCount + commentsCount + sharesCount
		if isFollowing {
			score += 1000
		}
		score += max(0, query.Radius-distance)
		item := MomentItem{
			MomentID:      momentID,
			Description:   description,
			CreatedAt:     createdAt.UTC().Format(time.RFC3339),
			LikesCount:    likesCount,
			CommentsCount: commentsCount,
			SharesCount:   sharesCount,
			User:          UserInfo{UserID: userID, Nickname: nickname, AvatarURL: types.StrPtr(avatarURL)},
			Location:      &Location{Point: pointStruct(latitude, longitude), LocationName: locationName},
			IsFavorited:   isFavorited,
		}
		// 附近流参考点恒已解析（service 层兜底），distance/bearing_degrees 恒填充
		//（spec api_spec_types.md RelativeLocation；匿名版同样基于 reference_point 计算）。
		fillRelativeLocation(item.Location, distance, bearing)
		items = append(items, momentCandidate{Item: item, Score: score, Distance: distance, CreatedAt: createdAt.UTC()})
		momentIDs = append(momentIDs, momentID)
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return PaginatedMoments{}, err
	}
	for idx := range items {
		items[idx].Item.Tags = tagsByMomentID[items[idx].Item.MomentID]
	}
	sort.SliceStable(items, func(i, j int) bool { return compareNearbyCandidates(items[i], items[j]) < 0 })
	return paginateNearbyMoments(items, query.Limit, query.Cursor)
}

// pointStruct 构造 Location 的匿名 Point 子结构（与 scanMomentRow 保持同一形状）。
func pointStruct(latitude, longitude float64) struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
} {
	return struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}
}

var _ Repository = (*PostgresRepository)(nil)
