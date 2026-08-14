package moments

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/types"
)

type InMemoryRepository struct {
	mu        sync.RWMutex
	moments   map[string]MomentItem
	byUser    map[string][]string
	favorites map[string]map[string]bool // userID -> momentID -> favorited
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
		moments:   make(map[string]MomentItem),
		byUser:    make(map[string][]string),
		favorites: make(map[string]map[string]bool),
	}
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
	item := MomentItem{MomentID: momentID, Description: req.Description, Tags: normalizeMomentTags(req.Tags), Mentions: normalizeMomentMentions(req.Mentions), CreatedAt: now.Format(time.RFC3339), LikesCount: 0, CommentsCount: 0, SharesCount: 0, User: UserInfo{UserID: userID, Nickname: userID}, Location: &req.Location, PrivacyLevel: privacy, IsFavorited: false}
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
	if limit <= 0 {
		limit = 20
	}
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
		result.NextCursor = items[len(items)-1].MomentID
	}
	return result, nil
}

func (r *InMemoryRepository) Exists(_ context.Context, momentID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.moments[momentID]
	return ok, nil
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

func (r *InMemoryRepository) Search(_ context.Context, req SearchMomentsRequest) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var items []MomentItem
	for _, m := range r.moments {
		if m.PrivacyLevel == "private" {
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
		items = append(items, m)
	}
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "relevance"
	}
	keyword := ""
	if req.Keyword != nil {
		keyword = *req.Keyword
	}
	tag := ""
	if req.Tag != nil {
		tag = *req.Tag
	}
	sortMoments(items, sortBy, keyword, tag)
	if len(items) == 0 {
		items = []MomentItem{}
	}
	return PaginatedMoments{Items: items}, nil
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

func (r *InMemoryRepository) Delete(_ context.Context, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.moments[momentID]
	if !ok {
		return ErrMomentNotFound
	}
	delete(r.moments, momentID)
	userIDs := r.byUser[item.User.UserID]
	for i, id := range userIDs {
		if id == momentID {
			r.byUser[item.User.UserID] = append(userIDs[:i], userIDs[i+1:]...)
			break
		}
	}
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
		r.favorites[userID] = make(map[string]bool)
	}
	r.favorites[userID][momentID] = true
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
		isFavorited = userFavs[momentID]
	}
	return &MyMomentState{IsFavorited: isFavorited}
}

func (r *InMemoryRepository) ListFavorites(_ context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	if limit > types.MaxPageSize {
		limit = types.MaxPageSize
	}
	favMap := r.favorites[userID]
	if favMap == nil {
		return PaginatedMoments{Items: []MomentItem{}}, nil
	}
	var items []MomentItem
	for mid := range favMap {
		if m, ok := r.moments[mid]; ok {
			items = append(items, m)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	start := 0
	if cursor != "" {
		for i, item := range items {
			if item.MomentID == cursor {
				start = i + 1
				break
			}
		}
	}
	items = items[start:]
	if len(items) > limit {
		return PaginatedMoments{Items: items[:limit], NextCursor: items[limit].MomentID}, nil
	}
	if len(items) == 0 {
		items = []MomentItem{}
	}
	return PaginatedMoments{Items: items}, nil
}

func (r *InMemoryRepository) ListFavoriteUserIDs(_ context.Context, momentID string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var userIDs []string
	for userID, favs := range r.favorites {
		if favs[momentID] {
			userIDs = append(userIDs, userID)
		}
	}
	return userIDs, nil
}

// sortMoments sorts a slice of MomentItem in place according to the given mode.
// refPt is used for distance calculation; keyword and tag are used for relevance scoring.
func sortMoments(items []MomentItem, sortBy string, keyword, tag string) {
	switch sortBy {
	case "newest":
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].CreatedAt != items[j].CreatedAt {
				return items[i].CreatedAt > items[j].CreatedAt
			}
			return items[i].MomentID < items[j].MomentID
		})
	case "popular":
		sort.SliceStable(items, func(i, j int) bool {
			si := items[i].LikesCount + items[i].CommentsCount
			sj := items[j].LikesCount + items[j].CommentsCount
			if si != sj {
				return si > sj
			}
			if items[i].CreatedAt != items[j].CreatedAt {
				return items[i].CreatedAt > items[j].CreatedAt
			}
			return items[i].MomentID < items[j].MomentID
		})
	default: // "relevance"
		sort.SliceStable(items, func(i, j int) bool {
			ri := relevanceScore(items[i], keyword, tag)
			rj := relevanceScore(items[j], keyword, tag)
			if ri != rj {
				return ri > rj
			}
			if items[i].CreatedAt != items[j].CreatedAt {
				return items[i].CreatedAt > items[j].CreatedAt
			}
			return items[i].MomentID < items[j].MomentID
		})
	}
}

// relevanceScore returns a simple integer score: items matching both keyword
// and tag score 2, matching either one score 1, otherwise 0.
func relevanceScore(m MomentItem, keyword, tag string) int {
	score := 0
	if keyword != "" && strings.Contains(m.Description, keyword) {
		score++
	}
	if tag != "" && containsTag(m.Tags, tag) {
		score++
	}
	return score
}

// FetchMomentImageRows returns no rows for the in-memory repo, which does not
// persist image associations. Image hydration is a no-op in memory-backed tests.
func (r *InMemoryRepository) FetchMomentImageRows(_ context.Context, _ []string) (map[string][]momentImageRow, error) {
	return map[string][]momentImageRow{}, nil
}
