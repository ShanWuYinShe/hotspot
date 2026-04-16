package moments

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/geo"
)

type likeEntry struct {
	userID   string
	nickname string
	likedAt  string
}

type InMemoryRepository struct {
	mu        sync.RWMutex
	moments   map[string]MomentItem
	byUser    map[string][]string
	likes     map[string]map[string]likeEntry // momentID -> userID -> entry
	favorites map[string]map[string]bool      // userID -> momentID -> favorited
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		moments:   make(map[string]MomentItem),
		byUser:    make(map[string][]string),
		likes:     make(map[string]map[string]likeEntry),
		favorites: make(map[string]map[string]bool),
	}
}

func (r *InMemoryRepository) Create(_ context.Context, userID string, req CreateMomentRequest) (MomentItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	momentID := fmt.Sprintf("moment_%d", now.UnixNano())
	privacy := req.PrivacyLevel
	if privacy == "" {
		privacy = "public"
	}
	item := MomentItem{MomentID: momentID, Description: req.Description, Tags: normalizeMomentTags(req.Tags), Mentions: normalizeMomentMentions(req.Mentions), CreatedAt: now.Format(time.RFC3339), LikesCount: 0, CommentsCount: 0, SharesCount: 0, User: UserInfo{UserID: userID, Nickname: userID}, Location: req.Location, PrivacyLevel: privacy, MyState: &MyMomentState{IsLiked: false, IsFavorited: false}}
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

func (r *InMemoryRepository) ListMine(_ context.Context, userID string) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids := append([]string(nil), r.byUser[userID]...)
	sort.Slice(ids, func(i, j int) bool { return ids[i] > ids[j] })
	items := make([]MomentItem, 0, len(ids))
	for _, id := range ids {
		items = append(items, r.moments[id])
	}
	return PaginatedMoments{Items: items}, nil
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
	refPt := geo.Point{Latitude: req.ReferencePoint.Latitude, Longitude: req.ReferencePoint.Longitude}
	keyword := ""
	if req.Keyword != nil {
		keyword = *req.Keyword
	}
	tag := ""
	if req.Tag != nil {
		tag = *req.Tag
	}
	sortMoments(items, sortBy, refPt, keyword, tag)
	if len(items) == 0 {
		items = []MomentItem{}
	}
	return PaginatedMoments{Items: items}, nil
}

func (r *InMemoryRepository) Update(_ context.Context, userID string, req EditMomentRequest) error {
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
		item.Location = *req.Location
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

func (r *InMemoryRepository) Like(_ context.Context, userID string, momentID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.moments[momentID]; !ok {
		return false, ErrMomentNotFound
	}
	if r.likes[momentID] == nil {
		r.likes[momentID] = make(map[string]likeEntry)
	}
	if _, exists := r.likes[momentID][userID]; exists {
		return false, nil
	}
	r.likes[momentID][userID] = likeEntry{
		userID:   userID,
		nickname: userID,
		likedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	item := r.moments[momentID]
	item.LikesCount++
	r.moments[momentID] = item
	return true, nil
}

func (r *InMemoryRepository) Unlike(_ context.Context, userID string, momentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.likes[momentID] != nil {
		if _, exists := r.likes[momentID][userID]; exists {
			delete(r.likes[momentID], userID)
			item := r.moments[momentID]
			if item.LikesCount > 0 {
				item.LikesCount--
			}
			r.moments[momentID] = item
		}
	}
	return nil
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

func (r *InMemoryRepository) ListLikes(_ context.Context, momentID string, limit int, cursor string) ([]LikeItem, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.moments[momentID]; !ok {
		return nil, "", ErrMomentNotFound
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	userMap := r.likes[momentID]
	if userMap == nil {
		return []LikeItem{}, "", nil
	}
	items := make([]LikeItem, 0, len(userMap))
	for uid, entry := range userMap {
		items = append(items, LikeItem{UserID: uid, Nickname: entry.nickname, LikedAt: entry.likedAt})
	}
	_ = cursor
	if len(items) == 0 {
		return []LikeItem{}, "", nil
	}
	if len(items) > limit {
		return items[:limit], items[limit].UserID, nil
	}
	return items, "", nil
}

func (r *InMemoryRepository) ListFavorites(_ context.Context, userID string, limit int, cursor string) (PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
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
	_ = cursor
	if len(items) > limit {
		return PaginatedMoments{Items: items[:limit], NextCursor: items[limit].MomentID}, nil
	}
	if len(items) == 0 {
		items = []MomentItem{}
	}
	return PaginatedMoments{Items: items}, nil
}

// sortMoments sorts a slice of MomentItem in place according to the given mode.
// refPt is used for distance calculation; keyword and tag are used for relevance scoring.
func sortMoments(items []MomentItem, sortBy string, refPt geo.Point, keyword, tag string) {
	switch sortBy {
	case "distance":
		sort.SliceStable(items, func(i, j int) bool {
			di := geo.DistanceMeters(refPt, geo.Point{Latitude: items[i].Location.Point.Latitude, Longitude: items[i].Location.Point.Longitude})
			dj := geo.DistanceMeters(refPt, geo.Point{Latitude: items[j].Location.Point.Latitude, Longitude: items[j].Location.Point.Longitude})
			if di != dj {
				return di < dj
			}
			return items[i].MomentID < items[j].MomentID
		})
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
