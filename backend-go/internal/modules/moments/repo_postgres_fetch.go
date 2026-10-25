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
	for rows.Next() {
		var momentID, userID, nickname, avatarURL, description, locationName string
		var latitude, longitude float64
		var createdAt time.Time
		var likesCount, commentsCount, sharesCount int
		var isFavorited bool
		var tags []string
		if err := rows.Scan(&momentID, &userID, &nickname, &avatarURL, &description, &createdAt, &likesCount, &commentsCount, &sharesCount, &locationName, &latitude, &longitude, &isFavorited, &tags); err != nil {
			return PaginatedMoments{}, err
		}
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
			Tags:          tags,
		}
		if query.HasReferencePoint {
			ref := geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}
			fillRelativeLocation(item.Location, geo.DistanceMeters(ref, geo.Point{Latitude: latitude, Longitude: longitude}), geo.BearingDegrees(ref, geo.Point{Latitude: latitude, Longitude: longitude}))
		}
		items = append(items, momentCandidate{Item: item, CreatedAt: createdAt.UTC()})
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, err
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
	// 候选集只扫描排序所需列（id/时间/坐标/热度计数/关注标记）：
	// 昵称、头像、描述、位置名等仅分页结果需要的字符串列改为分页后按页面 ID 一次补全，
	// 降低密集城区（候选上千行）时的行解码与字符串分配（pprof 显示 Scan/Decode 为主要剩余开销）。
	candidates := make([]nearbyCandidate, 0, 256)
	for rows.Next() {
		var candidate nearbyCandidate
		if err := rows.Scan(&candidate.momentID, &candidate.createdAt, &candidate.likesCount, &candidate.commentsCount, &candidate.sharesCount, &candidate.latitude, &candidate.longitude, &candidate.isFollowing); err != nil {
			return PaginatedMoments{}, err
		}
		distance, bearing, ok := plan.Match(geo.Point{Latitude: candidate.latitude, Longitude: candidate.longitude})
		if !ok {
			continue
		}
		candidate.distance = distance
		candidate.bearing = bearing
		candidate.score = candidate.likesCount + candidate.commentsCount + candidate.sharesCount
		if candidate.isFollowing {
			candidate.score += 1000
		}
		candidate.score += max(0, query.Radius-distance)
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return PaginatedMoments{}, err
	}
	// 排序规则与 compareNearbyCandidates 保持一致：距离升序 → 热度降序 → 时间降序 → ID 降序。
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if left.distance != right.distance {
			return left.distance < right.distance
		}
		if left.score != right.score {
			return left.score > right.score
		}
		if !left.createdAt.Equal(right.createdAt) {
			return left.createdAt.After(right.createdAt)
		}
		return left.momentID > right.momentID
	})
	limit := types.NormalizeLimit(query.Limit)
	start := findNearbyCandidateStart(candidates, query.Cursor)
	end := min(len(candidates), start+limit)
	page := PaginatedMoments{Items: make([]MomentItem, 0, max(0, end-start))}
	if end > start {
		pageCandidates := candidates[start:end]
		pageIDs := make([]string, 0, len(pageCandidates))
		for _, candidate := range pageCandidates {
			pageIDs = append(pageIDs, candidate.momentID)
		}
		details, err := loadNearbyMomentDetails(ctx, queryable, pageIDs)
		if err != nil {
			return PaginatedMoments{}, err
		}
		tagsByMomentID, err := loadMomentTags(ctx, queryable, pageIDs)
		if err != nil {
			return PaginatedMoments{}, err
		}
		favoritedMomentIDs, err := loadMomentFavorites(ctx, queryable, query.ViewerUserID, pageIDs)
		if err != nil {
			return PaginatedMoments{}, err
		}
		for _, candidate := range pageCandidates {
			detail := details[candidate.momentID]
			item := MomentItem{
				MomentID:      candidate.momentID,
				Description:   detail.description,
				CreatedAt:     candidate.createdAt.UTC().Format(time.RFC3339),
				LikesCount:    candidate.likesCount,
				CommentsCount: candidate.commentsCount,
				SharesCount:   candidate.sharesCount,
				User:          UserInfo{UserID: detail.userID, Nickname: detail.nickname, AvatarURL: types.StrPtr(detail.avatarURL)},
				Location:      &Location{Point: pointStruct(candidate.latitude, candidate.longitude), LocationName: detail.locationName},
				Tags:          tagsByMomentID[candidate.momentID],
				IsFavorited:   favoritedMomentIDs[candidate.momentID],
			}
			// 附近流参考点恒已解析（service 层兜底），distance/bearing_degrees 恒填充
			//（spec api_spec_types.md RelativeLocation；匿名版同样基于 reference_point 计算）。
			fillRelativeLocation(item.Location, candidate.distance, candidate.bearing)
			page.Items = append(page.Items, item)
		}
	}
	if end < len(candidates) && end > start {
		last := candidates[end-1]
		page.NextCursor = types.StrPtr(encodeFetchCursor(itemCursor{Score: last.score, Distance: last.distance, CreatedAt: last.createdAt.UTC().Format(time.RFC3339), ID: last.momentID}))
	}
	return page, nil
}

// nearbyCandidate 附近流排序候选（值类型，仅排序所需字段；展示字段分页后补全）。
type nearbyCandidate struct {
	momentID                               string
	createdAt                              time.Time
	latitude, longitude                    float64
	distance, bearing                      int
	likesCount, commentsCount, sharesCount int
	isFollowing                            bool
	score                                  int
}

// nearbyMomentDetail 分页结果的展示字段（一次批量查询补全）。
type nearbyMomentDetail struct {
	userID, nickname, avatarURL, description, locationName string
}

// loadNearbyMomentDetails 批量补全附近流页面内的展示字段。
func loadNearbyMomentDetails(ctx context.Context, queryable persistence.Queryable, momentIDs []string) (map[string]nearbyMomentDetail, error) {
	details := make(map[string]nearbyMomentDetail, len(momentIDs))
	if len(momentIDs) == 0 {
		return details, nil
	}
	rows, err := queryable.Query(ctx, queryNearbyMomentDetails, momentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var momentID string
		var detail nearbyMomentDetail
		if err := rows.Scan(&momentID, &detail.userID, &detail.nickname, &detail.avatarURL, &detail.description, &detail.locationName); err != nil {
			return nil, err
		}
		details[momentID] = detail
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return details, nil
}

// findNearbyCandidateStart 按游标定位候选起始下标（与 compareNearbyCandidates 同序）。
func findNearbyCandidateStart(candidates []nearbyCandidate, cursor string) int {
	decoded, err := decodeFetchCursor(cursor)
	if err != nil || decoded == nil {
		return 0
	}
	parsedAt, _ := time.Parse(time.RFC3339Nano, decoded.CreatedAt)
	for idx, candidate := range candidates {
		cmp := compareNearbyCandidate(candidate, decoded, parsedAt.UTC())
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(candidates)
}

// compareNearbyCandidate 与 compareNearbyCandidates 保持同一全序。
func compareNearbyCandidate(candidate nearbyCandidate, cursor *itemCursor, cursorCreatedAt time.Time) int {
	if candidate.distance != cursor.Distance {
		if candidate.distance < cursor.Distance {
			return -1
		}
		return 1
	}
	if candidate.score != cursor.Score {
		if candidate.score > cursor.Score {
			return -1
		}
		return 1
	}
	candidateCreated := candidate.createdAt.UTC()
	if !candidateCreated.Equal(cursorCreatedAt) {
		if candidateCreated.After(cursorCreatedAt) {
			return -1
		}
		return 1
	}
	if candidate.momentID > cursor.ID {
		return -1
	}
	if candidate.momentID < cursor.ID {
		return 1
	}
	return 0
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
