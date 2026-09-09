package moments

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/types"
)

type InMemoryRepository struct {
	mu      sync.RWMutex
	moments map[string]MomentItem
	byUser  map[string][]string
	// reactedByUser 按 user_id 聚合的反应瞬间 id（ListReactedByUser 消费，
	// 测试经 SeedReactedMoment 填充）。
	reactedByUser map[string][]string
	// favorites 记录收藏时刻（userID -> momentID -> favorited_at），驱动
	// fetch_favorites 的 favorited_at 倒序排序（spec api_spec_moment.md 排序）。
	favorites map[string]map[string]time.Time
	// followGraph 记录关注边（viewer -> target -> true），驱动内存版
	// fetch_following（生产实现走 user_relations 表）。
	followGraph map[string]map[string]bool
	// blockedTags 记录被屏蔽的标签（spec moment create：含被屏蔽标签 → 10400）。
	blockedTags map[string]struct{}
	// seq 保证同纳秒内连续创建的 ID 不碰撞（UnixNano 时间戳在快速循环下会重复）。
	seq uint64
}

// SetBlockedTags 注入需视为被屏蔽的标签（仅测试/内存实现使用；生产走 Postgres admin_blocked_tags 表）。
func (r *InMemoryRepository) SetBlockedTags(tags ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, tag := range tags {
		if r.blockedTags == nil {
			r.blockedTags = make(map[string]struct{})
		}
		r.blockedTags[tag] = struct{}{}
	}
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		moments:       make(map[string]MomentItem),
		byUser:        make(map[string][]string),
		reactedByUser: make(map[string][]string),
		favorites:     make(map[string]map[string]time.Time),
		followGraph:   make(map[string]map[string]bool),
	}
}

// SetFollow 注入内存版关注边（viewer 关注 target），驱动 fetch_following 测试。
func (r *InMemoryRepository) SetFollow(viewerID, targetID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.followGraph[viewerID] == nil {
		r.followGraph[viewerID] = make(map[string]bool)
	}
	r.followGraph[viewerID][targetID] = true
}

func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateMomentRequest) (MomentItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	r.seq++
	momentID := fmt.Sprintf("moment_%d_%d", now.UnixNano(), r.seq)
	privacy := req.PrivacyLevel
	if privacy == "" {
		privacy = "public"
	}
	for _, tag := range normalizeMomentTags(req.Tags) {
		if _, blocked := r.blockedTags[tag]; blocked {
			return MomentItem{}, ErrBlockedTag
		}
	}
	item := MomentItem{MomentID: momentID, Description: req.Description, Tags: normalizeMomentTags(req.Tags), Mentions: normalizeMomentMentions(req.Mentions), CreatedAt: now.Format(time.RFC3339), LikesCount: 0, CommentsCount: 0, SharesCount: 0, User: UserInfo{UserID: userID, Nickname: userID}, Location: req.Location, PrivacyLevel: privacy, IsFavorited: false}
	r.moments[momentID] = item
	r.byUser[userID] = append(r.byUser[userID], momentID)
	return item, nil
}

func (r *InMemoryRepository) Get(_ context.Context, momentID string) (MomentItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.moments[momentID]
	return item, ok, nil
}

func (r *InMemoryRepository) ListMine(_ context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit = types.NormalizeLimit(limit)
	if limit > types.MaxPageSize {
		limit = types.MaxPageSize
	}
	ids := append([]string(nil), r.byUser[userID]...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	startIdx := 0
	if cursor != "" {
		for i, id := range ids {
			if id == cursor {
				startIdx = i + 1
				break
			}
		}
	}
	endIdx := startIdx + limit
	if endIdx > len(ids) {
		endIdx = len(ids)
	}
	items := make([]MomentItem, 0, limit)
	for _, id := range ids[startIdx:endIdx] {
		items = append(items, r.moments[id])
	}
	result := PaginatedMoments{Items: items}
	if endIdx < len(ids) && len(items) > 0 {
		c := items[len(items)-1].MomentID
		result.NextCursor = &c
	}
	return result, nil
}

func (r *InMemoryRepository) Exists(_ context.Context, momentID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.moments[momentID]
	return ok && !item.IsDeleted, nil
}

func (r *InMemoryRepository) IncrementCommentCount(_ context.Context, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.moments[momentID]
	if !ok {
		return nil
	}
	item.CommentsCount++
	r.moments[momentID] = item
	return nil
}

func (r *InMemoryRepository) DecrementCommentCount(_ context.Context, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.moments[momentID]
	if !ok {
		return nil
	}
	item.CommentsCount--
	if item.CommentsCount < 0 {
		item.CommentsCount = 0
	}
	r.moments[momentID] = item
	return nil
}

func (r *InMemoryRepository) IncrementSharesCount(_ context.Context, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.moments[momentID]
	if !ok {
		return ErrMomentNotFound
	}
	item.SharesCount++
	r.moments[momentID] = item
	return nil
}

func (r *InMemoryRepository) Search(_ context.Context, req SearchMomentsRequest, viewerID string) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var items []MomentItem
	for _, m := range r.moments {
		if m.PrivacyLevel == "private" || m.IsDeleted {
			continue
		}
		if req.Keyword != nil && *req.Keyword != "" {
			if !strings.Contains(m.Description, *req.Keyword) {
				continue
			}
		}
		if req.Tag != nil && *req.Tag != "" {
			if !containsTag(m.Tags, *req.Tag) {
				continue
			}
		}
		// 查看者收藏态（spec api_spec_types.md MomentItem is_favorited 恒定布尔）：
		// 仅登录查看者按收藏记录填充，匿名恒为 false（与 Postgres EXISTS 子查询一致）。
		if viewerID != "" {
			m.IsFavorited = false
			if userFavs, ok := r.favorites[viewerID]; ok {
				if _, fav := userFavs[m.MomentID]; fav {
					m.IsFavorited = true
				}
			}
		}
		items = append(items, m)
	}
	limit := 20
	if req.Pagination != nil && req.Pagination.Limit > 0 {
		limit = req.Pagination.Limit
	}
	if limit > types.MaxPageSize {
		limit = types.MaxPageSize
	}
	sortBy := normalizeMomentSearchSort(req.SortBy)
	keyword := ""
	if req.Keyword != nil {
		keyword = *req.Keyword
	}
	tag := ""
	if req.Tag != nil {
		tag = *req.Tag
	}
	// 单次查询内取同一 now：popular 时间衰减分值页内自洽（与 SQL now() 语句级稳定对应）。
	now := time.Now().UTC()
	// 最终排序与 Postgres keyset 语义一致：(score, created_at, moment_id) DESC。
	sortMoments(items, sortBy, keyword, tag, now)
	start := 0
	if req.Pagination != nil && req.Pagination.Cursor != "" {
		searchCursor, err := decodeMomentSearchCursor(req.Pagination.Cursor)
		if err != nil {
			return PaginatedMoments{}, err
		}
		if searchCursor.Sort != sortBy {
			return PaginatedMoments{}, ErrInvalidCursor
		}
		// keyset 谓词定位（与 SQL `(score, created_at, moment_id) < 游标元组` 同语义）：
		// 不要求边界项仍存在；弱一致快照下的衰减漂移只影响游标相邻项。
		start = len(items)
		for i := range items {
			score := momentSearchScore(items[i], sortBy, keyword, tag, now)
			if compareSearchTuple(score, items[i].CreatedAt, items[i].MomentID, *searchCursor) < 0 {
				start = i
				break
			}
		}
	}
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	var nextCursor *string
	if end < len(items) {
		c := encodeMomentSearchCursor(sortBy, items[end], keyword, tag, now)
		nextCursor = &c
	}
	if end > len(items) {
		end = len(items)
	}
	page := items[start:end]
	if len(page) == 0 {
		page = []MomentItem{}
	}
	return PaginatedMoments{Items: page, NextCursor: nextCursor}, nil
}

func (r *InMemoryRepository) Update(_ context.Context, userID string, req UpdateMomentRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.moments[req.MomentID]
	if !ok {
		return ErrMomentNotFound
	}
	if item.User.UserID != userID {
		return ErrPermissionDenied
	}
	if req.Description != nil {
		item.Description = *req.Description
	}
	if req.Location != nil {
		loc := *req.Location
		item.Location = &loc
	}
	if req.PrivacyLevel != nil {
		item.PrivacyLevel = *req.PrivacyLevel
	}
	if req.Tags != nil {
		item.Tags = normalizeMomentTags(req.Tags)
	}
	if req.Mentions != nil {
		item.Mentions = normalizeMomentMentions(req.Mentions)
	}
	r.moments[req.MomentID] = item
	return nil
}

func (r *InMemoryRepository) NotificationContext(_ context.Context, momentID string) (MomentItem, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.moments[momentID]
	return item, ok, nil
}

// Delete 软删除（spec api_spec_moment.md「删除语义」：瞬间不物理删除，标记
// is_deleted=true 并清空 description）：作者 fetch_created、收藏者 fetch_favorites
// 继续返回占位项；search/fetch_following/fetch_nearby/Exists 按不存在过滤。幂等：
// 重复删除返回成功（与 Postgres UPDATE 语义一致）。
func (r *InMemoryRepository) Delete(_ context.Context, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.moments[momentID]
	if !ok {
		return ErrMomentNotFound
	}
	item.IsDeleted = true
	item.Description = ""
	r.moments[momentID] = item
	return nil
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func (r *InMemoryRepository) Favorite(_ context.Context, userID string, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.moments[momentID]; !ok {
		return ErrMomentNotFound
	}
	if r.favorites[userID] == nil {
		r.favorites[userID] = make(map[string]time.Time)
	}
	// 与 Postgres ON CONFLICT DO NOTHING 一致：重复收藏保留首次 favorited_at。
	if _, exists := r.favorites[userID][momentID]; !exists {
		r.favorites[userID][momentID] = time.Now().UTC()
	}
	return nil
}

func (r *InMemoryRepository) Unfavorite(_ context.Context, userID string, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.favorites[userID] != nil {
		delete(r.favorites[userID], momentID)
	}
	return nil
}

func (r *InMemoryRepository) LoadMyState(_ context.Context, momentID, userID string) *MyMomentState {
	if userID == "" {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	isFavorited := false
	if userFavs, ok := r.favorites[userID]; ok {
		_, isFavorited = userFavs[momentID]
	}
	return &MyMomentState{IsFavorited: isFavorited}
}

// ListFavorites 我的收藏（spec api_spec_moment.md → moments/fetch_favorites）：
// favorited_at 时间倒序 + moment_id DESC 稳定 tie-break，keyset 游标与 Postgres
// 实现同编码（加密不透明）；含被收藏后软删的瞬间占位项（spec「删除语义」）。
func (r *InMemoryRepository) ListFavorites(_ context.Context, userID string, limit int, cursorValue string) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	limit = normalizeMomentInteractionLimit(limit)
	cursorAt, cursorMomentID, err := decodeMomentFavoriteCursor(cursorValue)
	if err != nil {
		return PaginatedMoments{}, err
	}
	favMap := r.favorites[userID]
	if len(favMap) == 0 {
		return PaginatedMoments{Items: []MomentItem{}}, nil
	}
	type favoriteEntry struct {
		item        MomentItem
		favoritedAt time.Time
	}
	entries := make([]favoriteEntry, 0, len(favMap))
	for mid, favoritedAt := range favMap {
		m, ok := r.moments[mid]
		if !ok {
			continue
		}
		entries = append(entries, favoriteEntry{item: m, favoritedAt: favoritedAt})
	}
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].favoritedAt.Equal(entries[j].favoritedAt) {
			return entries[i].favoritedAt.After(entries[j].favoritedAt)
		}
		return entries[i].item.MomentID > entries[j].item.MomentID
	})
	start := 0
	if parsedAt, ok := cursorAt.(time.Time); ok {
		start = len(entries)
		for i, e := range entries {
			if e.favoritedAt.Before(parsedAt) || (e.favoritedAt.Equal(parsedAt) && e.item.MomentID < cursorMomentID) {
				start = i
				break
			}
		}
	}
	page := entries[start:]
	var nextCursor *string
	if len(page) > limit {
		boundary := page[limit-1]
		c := encodeMomentFavoriteCursor(boundary.favoritedAt.UTC().Format(time.RFC3339), boundary.item.MomentID)
		nextCursor = &c
		page = page[:limit]
	}
	items := make([]MomentItem, 0, len(page))
	for _, e := range page {
		e.item.IsFavorited = true
		e.item.FavoritedAt = e.favoritedAt.UTC().Format(time.RFC3339)
		items = append(items, e.item)
	}
	if len(items) == 0 {
		items = []MomentItem{}
	}
	return PaginatedMoments{Items: items, NextCursor: nextCursor}, nil
}

func (r *InMemoryRepository) ListFavoriteUserIDs(_ context.Context, momentID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var userIDs []string
	for userID, favs := range r.favorites {
		if _, fav := favs[momentID]; fav {
			userIDs = append(userIDs, userID)
		}
	}
	return userIDs, nil
}

// sortMoments sorts a slice of MomentItem in place by the shared keyset tuple
// (score, created_at, moment_id) in descending order. SQL 与内存实现共用
// momentSearchScore，避免翻页语义漂移；now 为本次查询的统一时钟
// （popular 时间衰减分值的计算基准，与 SQL now() 语句级稳定对应）。
func sortMoments(items []MomentItem, sortBy string, keyword, tag string, now time.Time) {
	sort.SliceStable(items, func(i, j int) bool {
		si := momentSearchScore(items[i], sortBy, keyword, tag, now)
		sj := momentSearchScore(items[j], sortBy, keyword, tag, now)
		if si != sj {
			return si > sj
		}
		if items[i].CreatedAt != items[j].CreatedAt {
			return items[i].CreatedAt > items[j].CreatedAt
		}
		return items[i].MomentID > items[j].MomentID
	})
}

// FetchMomentImageRows returns no rows for the in-memory repo, which does not
// persist image associations. Image hydration is a no-op in memory-backed tests.
func (r *InMemoryRepository) FetchMomentImageRows(_ context.Context, _ []string) (map[string][]momentImageRow, error) {
	return map[string][]momentImageRow{}, nil
}

// ListFollowingMoments 内存版关注动态：viewer 关注的人发布的公开瞬间，
// created_at DESC, moment_id DESC keyset 分页（与 Postgres 实现语义一致）；
// 参考点已解析时填充 location 的 distance/bearing_degrees
// （spec api_spec_types.md RelativeLocation）。
func (r *InMemoryRepository) ListFollowingMoments(_ context.Context, query momentsQuery) (PaginatedMoments, error) {
	decoded, err := decodeFetchCursor(query.Cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	r.mu.RLock()
	followed := r.followGraph[query.ViewerUserID]
	candidates := make([]momentCandidate, 0)
	ref := geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}
	for _, m := range r.moments {
		if !followed[m.User.UserID] || m.PrivacyLevel != "public" || m.IsDeleted {
			continue
		}
		createdAt, parseErr := time.Parse(time.RFC3339Nano, m.CreatedAt)
		if parseErr != nil {
			continue
		}
		if userFavs, ok := r.favorites[query.ViewerUserID]; ok {
			if _, fav := userFavs[m.MomentID]; fav {
				m.IsFavorited = true
			}
		}
		if query.HasReferencePoint && m.Location != nil {
			target := geo.Point{Latitude: m.Location.Point.Latitude, Longitude: m.Location.Point.Longitude}
			fillRelativeLocation(m.Location, geo.DistanceMeters(ref, target), geo.BearingDegrees(ref, target))
		}
		candidates = append(candidates, momentCandidate{Item: m, CreatedAt: createdAt.UTC()})
	}
	r.mu.RUnlock()
	sort.SliceStable(candidates, func(i, j int) bool { return compareFollowingCandidates(candidates[i], candidates[j]) < 0 })
	// keyset 定位：跳过游标项及之前的所有项。
	if decoded != nil {
		parsedAt, _ := time.Parse(time.RFC3339Nano, decoded.CreatedAt)
		start := 0
		for idx, item := range candidates {
			cmp := compareFollowingCandidates(item, momentCandidate{Item: MomentItem{MomentID: decoded.ID}, CreatedAt: parsedAt.UTC()})
			if cmp > 0 {
				start = idx
				break
			}
			start = idx + 1
		}
		candidates = candidates[start:]
	}
	return paginateFollowingMoments(candidates, query.Limit, "")
}

// ListNearbyMoments 内存版附近瞬间：radius 内距离主序（距离 → 热度 → 时间 → ID）。
func (r *InMemoryRepository) ListNearbyMoments(_ context.Context, query momentsQuery) (PaginatedMoments, error) {
	decoded, err := decodeFetchCursor(query.Cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	ref := geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}
	r.mu.RLock()
	candidates := make([]momentCandidate, 0)
	for _, m := range r.moments {
		if m.PrivacyLevel != "public" || m.IsDeleted || m.Location == nil {
			continue
		}
		target := geo.Point{Latitude: m.Location.Point.Latitude, Longitude: m.Location.Point.Longitude}
		distance := geo.DistanceMeters(ref, target)
		if distance > query.Radius {
			continue
		}
		createdAt, parseErr := time.Parse(time.RFC3339Nano, m.CreatedAt)
		if parseErr != nil {
			continue
		}
		score := m.LikesCount + m.CommentsCount + m.SharesCount
		if query.ViewerUserID != "" && r.followGraph[query.ViewerUserID][m.User.UserID] {
			score += 1000
		}
		m.IsFavorited = false
		if userFavs, ok := r.favorites[query.ViewerUserID]; ok {
			if _, fav := userFavs[m.MomentID]; fav {
				m.IsFavorited = true
			}
		}
		// 相对位置字段恒填充（附近流参考点恒已解析，spec api_spec_types.md RelativeLocation）。
		fillRelativeLocation(m.Location, distance, geo.BearingDegrees(ref, target))
		candidates = append(candidates, momentCandidate{Item: m, Score: score, Distance: distance, CreatedAt: createdAt.UTC()})
	}
	r.mu.RUnlock()
	sort.SliceStable(candidates, func(i, j int) bool { return compareNearbyCandidates(candidates[i], candidates[j]) < 0 })
	start := findNearbyCursorStart(candidates, decoded)
	candidates = candidates[start:]
	return paginateNearbyMoments(candidates, query.Limit, "")
}

// ListMomentsByAuthor 内存实现：byUser 索引已有（open_the_door 测试路径），
// 隐私过滤与 users 原版一致。
func (r *InMemoryRepository) ListMomentsByAuthor(_ context.Context, userID string, limit int, cursor string, includePrivate bool) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := r.byUser[userID]
	items := make([]MomentItem, 0, len(ids))
	for _, id := range ids {
		item, ok := r.moments[id]
		if !ok || item.IsDeleted {
			continue
		}
		if !includePrivate && item.PrivacyLevel == "private" {
			continue
		}
		items = append(items, item)
	}
	limit = types.NormalizeLimit(limit)
	if len(items) > limit {
		return PaginatedMoments{Items: items[:limit]}, nil
	}
	return PaginatedMoments{Items: items}, nil
}

// ListReactedByUser 内存实现：测试数据经 SeedReactedMoment 填充。
func (r *InMemoryRepository) ListReactedByUser(_ context.Context, userID, viewerUserID string, limit int, cursor string) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]MomentItem, 0)
	for _, id := range r.reactedByUser[userID] {
		item, ok := r.moments[id]
		if !ok || item.IsDeleted {
			continue
		}
		if item.PrivacyLevel == "private" && viewerUserID != item.User.UserID && viewerUserID != userID {
			continue
		}
		items = append(items, item)
	}
	limit = types.NormalizeLimit(limit)
	if len(items) > limit {
		return PaginatedMoments{Items: items[:limit]}, nil
	}
	return PaginatedMoments{Items: items}, nil
}

// SeedReactedMoment 为 ListReactedByUser 的内存测试填充一条反应记录
// （reactedAt 倒序由调用方保证插入顺序或自行排序）。
func (r *InMemoryRepository) SeedReactedMoment(userID string, item MomentItem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.moments == nil {
		r.moments = make(map[string]MomentItem)
	}
	if r.byUser == nil {
		r.byUser = make(map[string][]string)
	}
	if r.reactedByUser == nil {
		r.reactedByUser = make(map[string][]string)
	}
	r.moments[item.MomentID] = item
	r.byUser[item.User.UserID] = append(r.byUser[item.User.UserID], item.MomentID)
	r.reactedByUser[userID] = append(r.reactedByUser[userID], item.MomentID)
}

// SeedMomentsByUser 填充「按作者聚合的瞬间列表」测试数据（供跨包 handler
// 测试构造 users 聚合场景）。
func (r *InMemoryRepository) SeedMomentsByUser(userID string, items []MomentItem) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, item := range items {
		r.moments[item.MomentID] = item
		r.byUser[userID] = append(r.byUser[userID], item.MomentID)
	}
}
