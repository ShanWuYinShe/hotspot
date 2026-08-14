package users

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/geo"
)

type memoryRelation struct {
	Following   bool
	Blocked     bool
	Remark      string
	FollowedAt  string
	BlockedAt   string
	BlockReason string
}

type reactedMomentRecord struct {
	ReactedAt string
	Moment    momenttypes.MomentItem
}

type InMemoryRepository struct {
	mu             sync.RWMutex
	profiles       map[string]MyProfile
	settings       map[string]UserSettings
	relations      map[string]map[string]memoryRelation
	reactedMoments map[string][]reactedMomentRecord
	groupsByUser   map[string][]grouptypes.GroupItem
	momentsByUser  map[string][]momenttypes.MomentItem
	blockedTags    map[string]struct{}
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		profiles:       make(map[string]MyProfile),
		settings:       make(map[string]UserSettings),
		relations:      make(map[string]map[string]memoryRelation),
		reactedMoments: make(map[string][]reactedMomentRecord),
		groupsByUser:   make(map[string][]grouptypes.GroupItem),
		momentsByUser:  make(map[string][]momenttypes.MomentItem),
		blockedTags:    make(map[string]struct{}),
	}
}

func (r *InMemoryRepository) GetProfile(_ context.Context, userID string) (MyProfile, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	profile, ok := r.profiles[userID]
	return profile, ok, nil
}
func (r *InMemoryRepository) PutProfile(_ context.Context, profile MyProfile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[profile.UserID] = profile
	return nil
}

// DeleteProfile 账户删除匿名化（与 postgres 实现一致）：users 行保留并置占位
// （已发布内容保留 + 作者显示"已注销用户"），个人数据清理。
func (r *InMemoryRepository) DeleteProfile(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if profile, ok := r.profiles[userID]; ok {
		profile.Nickname = "已注销用户"
		profile.AvatarURL = ""
		profile.Bio = ""
		profile.Tags = nil
		profile.Gender = ""
		profile.Email = ""
		profile.Phone = ""
		profile.Location = Location{}
		profile.PresenceStatus = UserPresenceOffline
		profile.IsDeleted = true
		r.profiles[userID] = profile
	}
	delete(r.settings, userID)
	delete(r.relations, userID)
	delete(r.reactedMoments, userID)
	delete(r.groupsByUser, userID)
	delete(r.momentsByUser, userID)
	for sourceUserID := range r.relations {
		delete(r.relations[sourceUserID], userID)
	}
	return nil
}
func (r *InMemoryRepository) GetSettings(_ context.Context, userID string) (UserSettings, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	settings, ok := r.settings[userID]
	return settings, ok, nil
}
func (r *InMemoryRepository) GetSettingsBatch(ctx context.Context, userIDs []string) (map[string]UserSettings, error) {
	result := make(map[string]UserSettings, len(userIDs))
	for _, userID := range userIDs {
		settings, ok, err := r.GetSettings(ctx, userID)
		if err != nil {
			return nil, err
		}
		if ok {
			result[userID] = settings
		}
	}
	return result, nil
}
func (r *InMemoryRepository) PutSettings(_ context.Context, userID string, settings UserSettings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.settings[userID] = settings
	return nil
}
func (r *InMemoryRepository) SearchProfiles(_ context.Context, req SearchUsersRequest) (PaginatedProfiles, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sortBy := normalizeSearchSort(req.SortBy)
	// radius 语义（spec user.md → users/search）：显式提供才按半径过滤；缺省不过滤（全球匹配）
	radiusFilter := req.Radius != nil && *req.Radius > 0
	radius := normalizeSearchRadius(req.Radius)
	plan := geo.NewSearchPlan(geo.Point{Latitude: req.ReferencePoint.Latitude, Longitude: req.ReferencePoint.Longitude}, radius, sortBy)
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	tag := strings.ToLower(strings.TrimSpace(req.Tag))
	items := make([]searchProfileCandidate, 0, len(r.profiles))
	for _, profile := range r.profiles {
		settings, ok := r.settings[profile.UserID]
		if !ok {
			settings = defaultSettings()
		}
		if !settings.Searchable {
			continue
		}
		// location_searchable：仅位置/半径维度过滤；keyword/tag 维度不受影响
		if radiusFilter && !settings.LocationSearchable {
			continue
		}
		if keyword != "" {
			nickname := strings.ToLower(profile.Nickname)
			bio := strings.ToLower(profile.Bio)
			if !(settings.NicknameSearchable && strings.Contains(nickname, keyword)) && !strings.Contains(bio, keyword) {
				continue
			}
		}
		if tag != "" && !containsFold(profile.Tags, tag) {
			continue
		}
		distance, bearing, within := plan.Match(geo.Point{Latitude: profile.Location.Point.Latitude, Longitude: profile.Location.Point.Longitude})
		if radiusFilter && !within {
			continue
		}
		items = append(items, searchProfileCandidate{Profile: profile, Distance: distance, BearingDegrees: bearing, SortBy: sortBy, RelevanceScore: searchRelevanceScore(profile, keyword, tag)})
	}
	sortSearchCandidates(items, sortBy)
	result := paginateSearchCandidates(items, req.Pagination)
	for i := range result.Items {
		profile := result.Items[i]
		profile.Location = Location{Point: GeoPoint{Latitude: profile.Location.Point.Latitude, Longitude: profile.Location.Point.Longitude}, LocationName: profile.Location.LocationName}
		result.Items[i] = profile
	}
	return result, nil
}
func (r *InMemoryRepository) Follow(_ context.Context, userID, targetUserID string, _ FollowSource, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	relation := r.ensureRelation(userID, targetUserID)
	relation.Following = true
	relation.FollowedAt = nowMemoryRFC3339()
	r.relations[userID][targetUserID] = relation
	return nil
}
func (r *InMemoryRepository) Unfollow(_ context.Context, userID, targetUserID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	relation := r.ensureRelation(userID, targetUserID)
	relation.Following = false
	relation.FollowedAt = ""
	r.relations[userID][targetUserID] = relation
	return nil
}
func (r *InMemoryRepository) Block(_ context.Context, userID, targetUserID, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	relation := r.ensureRelation(userID, targetUserID)
	relation.Blocked = true
	relation.BlockedAt = nowMemoryRFC3339()
	relation.BlockReason = reason
	r.relations[userID][targetUserID] = relation
	return nil
}
func (r *InMemoryRepository) Unblock(_ context.Context, userID, targetUserID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	relation := r.ensureRelation(userID, targetUserID)
	relation.Blocked = false
	relation.BlockedAt = ""
	relation.BlockReason = ""
	r.relations[userID][targetUserID] = relation
	return nil
}
func (r *InMemoryRepository) SetRemark(_ context.Context, userID, targetUserID, remark string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	relation := r.ensureRelation(userID, targetUserID)
	relation.Remark = remark
	r.relations[userID][targetUserID] = relation
	return nil
}

// CheckBlockedTags 返回 tags 中命中平台屏蔽标签的标签名（内存实现读取 blockedTags 集合）。
func (r *InMemoryRepository) CheckBlockedTags(_ context.Context, tags []string) ([]string, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	blocked := make([]string, 0)
	seen := make(map[string]struct{})
	for _, tag := range tags {
		if _, ok := r.blockedTags[tag]; ok {
			if _, dup := seen[tag]; !dup {
				blocked = append(blocked, tag)
				seen[tag] = struct{}{}
			}
		}
	}
	return blocked, nil
}

func (r *InMemoryRepository) ListBlocked(_ context.Context, userID string, limit int, cursor string) (PaginatedBlockedUsers, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]BlockedUserItem, 0)
	for targetUserID, relation := range r.relations[userID] {
		if relation.Blocked {
			items = append(items, BlockedUserItem{User: UserInfo{UserID: targetUserID, Nickname: r.nicknameFor(targetUserID)}, BlockedAt: relation.BlockedAt, Reason: relation.BlockReason})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].BlockedAt == items[j].BlockedAt {
			return items[i].User.UserID < items[j].User.UserID
		}
		return items[i].BlockedAt > items[j].BlockedAt
	})
	return paginateBlocked(items, limit, cursor), nil
}
func (r *InMemoryRepository) ListReactedMoments(_ context.Context, userID string, limit int, cursor, viewerUserID string) (momenttypes.PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	records := append([]reactedMomentRecord(nil), r.reactedMoments[userID]...)
	visible := make([]reactedMomentRecord, 0, len(records))
	for _, record := range records {
		if viewerUserID == record.Moment.User.UserID || record.Moment.PrivacyLevel != "private" {
			visible = append(visible, record)
		}
	}
	return paginateReactedMoments(visible, limit, cursor), nil
}

func (r *InMemoryRepository) Relationship(_ context.Context, userID, targetUserID string) (Relationship, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	forward := r.relations[userID][targetUserID]
	reverse := r.relations[targetUserID][userID]
	return Relationship{IsFollowing: forward.Following, IsFollowedBy: reverse.Following, IsBlocked: forward.Blocked}, nil
}

func (r *InMemoryRepository) RelationshipBatch(_ context.Context, userID string, targetUserIDs []string) (map[string]Relationship, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]Relationship, len(targetUserIDs))
	for _, target := range targetUserIDs {
		forward := r.relations[userID][target]
		reverse := r.relations[target][userID]
		result[target] = Relationship{IsFollowing: forward.Following, IsFollowedBy: reverse.Following, IsBlocked: forward.Blocked}
	}
	return result, nil
}

func (r *InMemoryRepository) ListFriends(_ context.Context, userID string, limit int, cursor string) (PaginatedFriends, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]FriendItem, 0)
	for targetUserID, relation := range r.relations[userID] {
		reverse := r.relations[targetUserID][userID]
		if relation.Following && reverse.Following {
			lastOnline := r.profiles[targetUserID].LastOnline
			presence := r.profiles[targetUserID].PresenceStatus
			remark := relation.Remark
			item := FriendItem{User: UserInfo{UserID: targetUserID, Nickname: r.nicknameFor(targetUserID), AvatarURL: r.profiles[targetUserID].AvatarURL}, Remark: &remark, FollowedAt: relation.FollowedAt, FollowedBackAt: maxStringTime(relation.FollowedAt, reverse.FollowedAt)}
			if lastOnline != "" {
				item.LastOnline = &lastOnline
			}
			item.PresenceStatus = &presence
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].FollowedBackAt == items[j].FollowedBackAt {
			return items[i].User.UserID < items[j].User.UserID
		}
		return items[i].FollowedBackAt > items[j].FollowedBackAt
	})
	return paginateFriends(items, limit, cursor), nil
}
func (r *InMemoryRepository) ListFollowing(_ context.Context, userID string, limit int, cursor string) (PaginatedFollowing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]FollowingItem, 0)
	for targetUserID, relation := range r.relations[userID] {
		if relation.Following {
			lastOnline := r.profiles[targetUserID].LastOnline
			presence := r.profiles[targetUserID].PresenceStatus
			item := FollowingItem{User: UserInfo{UserID: targetUserID, Nickname: r.nicknameFor(targetUserID), AvatarURL: r.profiles[targetUserID].AvatarURL}, FollowedAt: relation.FollowedAt, IsMutual: r.relations[targetUserID][userID].Following}
			if lastOnline != "" {
				item.LastOnline = &lastOnline
			}
			item.PresenceStatus = &presence
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].FollowedAt == items[j].FollowedAt {
			return items[i].User.UserID < items[j].User.UserID
		}
		return items[i].FollowedAt > items[j].FollowedAt
	})
	return paginateFollowing(items, limit, cursor), nil
}
func (r *InMemoryRepository) ListFollowers(_ context.Context, userID string, limit int, cursor string) (PaginatedFollowers, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]FollowerItem, 0)
	for sourceUserID, relationMap := range r.relations {
		relation := relationMap[userID]
		if relation.Following {
			lastOnline := r.profiles[sourceUserID].LastOnline
			presence := r.profiles[sourceUserID].PresenceStatus
			item := FollowerItem{User: UserInfo{UserID: sourceUserID, Nickname: r.nicknameFor(sourceUserID), AvatarURL: r.profiles[sourceUserID].AvatarURL}, FollowedAt: relation.FollowedAt, IsMutual: r.relations[userID][sourceUserID].Following}
			if lastOnline != "" {
				item.LastOnline = &lastOnline
			}
			item.PresenceStatus = &presence
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].FollowedAt == items[j].FollowedAt {
			return items[i].User.UserID < items[j].User.UserID
		}
		return items[i].FollowedAt > items[j].FollowedAt
	})
	return paginateFollowers(items, limit, cursor), nil
}
func (r *InMemoryRepository) ListGroups(_ context.Context, userID string, limit int, cursor string) (grouptypes.PaginatedGroups, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := append([]grouptypes.GroupItem(nil), r.groupsByUser[userID]...)
	return paginateGroups(items, limit, cursor), nil
}
func (r *InMemoryRepository) ListMoments(_ context.Context, userID string, limit int, cursor string, includePrivate bool) (momenttypes.PaginatedMoments, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := append([]momenttypes.MomentItem(nil), r.momentsByUser[userID]...)
	filtered := make([]momenttypes.MomentItem, 0, len(items))
	for _, item := range items {
		if includePrivate || item.PrivacyLevel != "private" {
			filtered = append(filtered, item)
		}
	}
	return paginateMoments(filtered, limit, cursor), nil
}
func (r *InMemoryRepository) CommonGroupCount(_ context.Context, viewerUserID, targetUserID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	targetSet := make(map[string]struct{}, len(r.groupsByUser[targetUserID]))
	for _, item := range r.groupsByUser[targetUserID] {
		targetSet[item.Group.GroupID] = struct{}{}
	}
	count := 0
	seen := make(map[string]struct{})
	for _, item := range r.groupsByUser[viewerUserID] {
		groupID := item.Group.GroupID
		if _, ok := seen[groupID]; ok {
			continue
		}
		seen[groupID] = struct{}{}
		if _, ok := targetSet[groupID]; ok {
			count++
		}
	}
	return count, nil
}
func (r *InMemoryRepository) UserExists(_ context.Context, userID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.profiles[userID]
	return ok, nil
}

func (r *InMemoryRepository) ensureRelation(userID, targetUserID string) memoryRelation {
	if r.relations[userID] == nil {
		r.relations[userID] = map[string]memoryRelation{}
	}
	return r.relations[userID][targetUserID]
}

func (r *InMemoryRepository) nicknameFor(userID string) string {
	if profile, ok := r.profiles[userID]; ok && profile.Nickname != "" {
		return profile.Nickname
	}
	return userID
}

func nowMemoryRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }
func maxStringTime(a, b string) string {
	if a >= b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a >= b {
		return a
	}
	return b
}
func containsFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func paginateBlocked(items []BlockedUserItem, limit int, cursor string) PaginatedBlockedUsers {
	if limit <= 0 {
		limit = 20
	}
	start, ok := findUserCursorStart(cursor, func(i int) (string, string) { return items[i].BlockedAt, items[i].User.UserID }, len(items))
	if !ok {
		return PaginatedBlockedUsers{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := PaginatedBlockedUsers{Items: items[start:end]}
	if end < len(items) && end > start {
		resp.NextCursor = encodeUserMomentCursor(items[end-1].BlockedAt, items[end-1].User.UserID)
	}
	return resp
}
func paginateFriends(items []FriendItem, limit int, cursor string) PaginatedFriends {
	if limit <= 0 {
		limit = 20
	}
	start, ok := findUserCursorStart(cursor, func(i int) (string, string) { return items[i].FollowedBackAt, items[i].User.UserID }, len(items))
	if !ok {
		return PaginatedFriends{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := PaginatedFriends{Items: items[start:end]}
	if end < len(items) && end > start {
		resp.NextCursor = encodeUserMomentCursor(items[end-1].FollowedBackAt, items[end-1].User.UserID)
	}
	return resp
}
func paginateFollowing(items []FollowingItem, limit int, cursor string) PaginatedFollowing {
	if limit <= 0 {
		limit = 20
	}
	start, ok := findUserCursorStart(cursor, func(i int) (string, string) { return items[i].FollowedAt, items[i].User.UserID }, len(items))
	if !ok {
		return PaginatedFollowing{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := PaginatedFollowing{Items: items[start:end]}
	if end < len(items) && end > start {
		resp.NextCursor = encodeUserMomentCursor(items[end-1].FollowedAt, items[end-1].User.UserID)
	}
	return resp
}
func paginateFollowers(items []FollowerItem, limit int, cursor string) PaginatedFollowers {
	if limit <= 0 {
		limit = 20
	}
	start, ok := findUserCursorStart(cursor, func(i int) (string, string) { return items[i].FollowedAt, items[i].User.UserID }, len(items))
	if !ok {
		return PaginatedFollowers{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := PaginatedFollowers{Items: items[start:end]}
	if end < len(items) && end > start {
		resp.NextCursor = encodeUserMomentCursor(items[end-1].FollowedAt, items[end-1].User.UserID)
	}
	return resp
}
func paginateGroups(items []grouptypes.GroupItem, limit int, cursor string) grouptypes.PaginatedGroups {
	if limit <= 0 {
		limit = 20
	}
	start, ok := findUserCursorStart(cursor, func(i int) (string, string) {
		joinedAt := ""
		if items[i].ViewerState != nil {
			joinedAt = items[i].ViewerState.JoinedAt
		}
		return joinedAt, items[i].Group.GroupID
	}, len(items))
	if !ok {
		return grouptypes.PaginatedGroups{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := grouptypes.PaginatedGroups{Items: items[start:end]}
	if end < len(items) && end > start {
		joinedAt := ""
		if items[end-1].ViewerState != nil {
			joinedAt = items[end-1].ViewerState.JoinedAt
		}
		next := encodeUserMomentCursor(joinedAt, items[end-1].Group.GroupID)
		resp.NextCursor = &next
	}
	return resp
}
func paginateMoments(items []momenttypes.MomentItem, limit int, cursor string) momenttypes.PaginatedMoments {
	if limit <= 0 {
		limit = 20
	}
	start, ok := findUserCursorStart(cursor, func(i int) (string, string) { return items[i].CreatedAt, items[i].MomentID }, len(items))
	if !ok {
		return momenttypes.PaginatedMoments{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := momenttypes.PaginatedMoments{Items: items[start:end]}
	if end < len(items) && end > start {
		resp.NextCursor = encodeUserMomentCursor(items[end-1].CreatedAt, items[end-1].MomentID)
	}
	return resp
}

func paginateReactedMoments(items []reactedMomentRecord, limit int, cursor string) momenttypes.PaginatedMoments {
	if limit <= 0 {
		limit = 20
	}
	start, ok := findUserCursorStart(cursor, func(i int) (string, string) { return items[i].ReactedAt, items[i].Moment.MomentID }, len(items))
	if !ok {
		return momenttypes.PaginatedMoments{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := momenttypes.PaginatedMoments{Items: make([]momenttypes.MomentItem, 0, end-start)}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Moment)
	}
	if end < len(items) && end > start {
		resp.NextCursor = encodeUserMomentCursor(items[end-1].ReactedAt, items[end-1].Moment.MomentID)
	}
	return resp
}

func findUserCursorStart(cursor string, keyAt func(i int) (string, string), length int) (int, bool) {
	if cursor == "" {
		return 0, true
	}
	decodedAt, decodedID, err := decodeUserMomentCursor(cursor)
	if err != nil {
		return 0, false
	}
	parsedAt, ok := decodedAt.(time.Time)
	if !ok {
		return 0, false
	}
	needleAt := parsedAt.UTC().Format(time.RFC3339)
	for i := 0; i < length; i++ {
		itemAt, itemID := keyAt(i)
		if itemAt == needleAt && itemID == decodedID {
			return i + 1, true
		}
	}
	return 0, false
}

type searchProfileCandidate struct {
	Profile        MyProfile
	Distance       int
	BearingDegrees int
	SortBy         string
	RelevanceScore int
}

type searchCursor struct {
	SortBy     string `json:"sort_by"`
	UserID     string `json:"user_id"`
	Distance   int    `json:"distance,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	LastOnline string `json:"last_online,omitempty"`
}

func normalizeSearchRadius(radius *int) int {
	value := 5000
	if radius != nil {
		value = *radius
	}
	if value < 100 {
		value = 100
	}
	if value > 50000 {
		value = 50000
	}
	return value
}

func normalizeSearchSort(sortBy string) string {
	switch sortBy {
	case "distance", "newest":
		return sortBy
	default:
		return "relevance"
	}
}

func sortSearchCandidates(items []searchProfileCandidate, sortBy string) {
	sort.Slice(items, func(i, j int) bool {
		left, right := items[i], items[j]
		switch sortBy {
		case "distance":
			if left.Distance != right.Distance {
				return left.Distance < right.Distance
			}
			if left.RelevanceScore != right.RelevanceScore {
				return left.RelevanceScore > right.RelevanceScore
			}
			if left.Profile.LastOnline != right.Profile.LastOnline {
				return left.Profile.LastOnline > right.Profile.LastOnline
			}
		case "newest":
			if left.Profile.CreatedAt != right.Profile.CreatedAt {
				return left.Profile.CreatedAt > right.Profile.CreatedAt
			}
		default:
			if left.RelevanceScore != right.RelevanceScore {
				return left.RelevanceScore > right.RelevanceScore
			}
			if left.Profile.LastOnline != right.Profile.LastOnline {
				return left.Profile.LastOnline > right.Profile.LastOnline
			}
		}
		return left.Profile.UserID < right.Profile.UserID
	})
}

func searchRelevanceScore(profile MyProfile, keyword, tag string) int {
	score := 0
	lowerNickname := strings.ToLower(profile.Nickname)
	lowerBio := strings.ToLower(profile.Bio)
	if keyword != "" {
		switch {
		case lowerNickname == keyword:
			score += 6
		case strings.HasPrefix(lowerNickname, keyword):
			score += 4
		case strings.Contains(lowerNickname, keyword):
			score += 3
		}
		if strings.Contains(lowerBio, keyword) {
			score += 2
		}
	}
	if tag != "" && containsFold(profile.Tags, tag) {
		score += 3
	}
	return score
}

func paginateSearchCandidates(items []searchProfileCandidate, p *PaginationParams) PaginatedProfiles {
	limit, cursor := normalizePagination(p)
	start := 0
	if cursor != "" {
		decoded, err := decodeSearchCursor(cursor)
		if err != nil {
			return PaginatedProfiles{InvalidCursor: true}
		}
		matched := false
		for i, item := range items {
			if searchCursorForItem(item) == decoded {
				start = i + 1
				matched = true
				break
			}
		}
		if !matched {
			return PaginatedProfiles{InvalidCursor: true}
		}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	result := PaginatedProfiles{Items: make([]MyProfile, 0, maxInt(end-start, 0))}
	for _, item := range items[start:end] {
		result.Items = append(result.Items, item.Profile)
	}
	if end < len(items) && end > start {
		result.NextCursor = encodeSearchCursor(items[end-1])
	}
	return result
}

func searchCursorForItem(item searchProfileCandidate) searchCursor {
	return searchCursor{SortBy: item.SortBy, UserID: item.Profile.UserID, Distance: item.Distance, CreatedAt: item.Profile.CreatedAt, LastOnline: item.Profile.LastOnline}
}

func encodeSearchCursor(item searchProfileCandidate) string {
	token, err := usersCursorCodec.Encode(searchCursorForItem(item))
	if err != nil {
		return ""
	}
	return token
}

func decodeSearchCursor(raw string) (searchCursor, error) {
	var decoded searchCursor
	if err := usersCursorCodec.Decode(raw, &decoded); err != nil {
		return searchCursor{}, ErrInvalidCursor
	}
	if decoded.SortBy == "" || decoded.UserID == "" {
		return searchCursor{}, ErrInvalidCursor
	}
	return decoded, nil
}
