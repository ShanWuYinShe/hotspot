package users

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/types"
)

type memoryRelation struct {
	Following   bool
	Blocked     bool
	Remark      string
	FollowedAt  string
	BlockedAt   string
	BlockReason string
}

type InMemoryRepository struct {
	mu          sync.RWMutex
	profiles    map[string]MyProfile
	settings    map[string]UserSettings
	relations   map[string]map[string]memoryRelation
	blockedTags map[string]struct{}
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		profiles:    make(map[string]MyProfile),
		settings:    make(map[string]UserSettings),
		relations:   make(map[string]map[string]memoryRelation),
		blockedTags: make(map[string]struct{}),
	}
}

func (r *InMemoryRepository) GetProfile(_ context.Context, userID string) (MyProfile, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	profile, ok := r.profiles[userID]
	return profile, ok, nil
}

func (r *InMemoryRepository) GetProfiles(_ context.Context, userIDs []string) (map[string]MyProfile, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]MyProfile, len(userIDs))
	for _, userID := range userIDs {
		if profile, ok := r.profiles[userID]; ok {
			result[userID] = profile
		}
	}
	return result, nil
}

// MutualFriends 内存版：全量求交（总数精确；PG 版同语义单条 SQL）。
// 预览按最近互关优先（对齐 PG 的 ORDER BY），数量上限 previewLimit。
func (r *InMemoryRepository) MutualFriends(_ context.Context, viewerID, targetID string, previewLimit int) (int, []UserInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if previewLimit < 0 {
		previewLimit = 0
	}
	type candidate struct {
		userID   string
		recentAt string
	}
	candidates := make([]candidate, 0)
	for otherID, rel := range r.relations[viewerID] {
		if !rel.Following || !r.relations[otherID][viewerID].Following {
			continue
		}
		if trel, ok := r.relations[targetID][otherID]; !ok || !trel.Following || !r.relations[otherID][targetID].Following {
			continue
		}
		candidates = append(candidates, candidate{
			userID:   otherID,
			recentAt: maxStringTime(rel.FollowedAt, r.relations[otherID][viewerID].FollowedAt),
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].recentAt != candidates[j].recentAt {
			return candidates[i].recentAt > candidates[j].recentAt
		}
		return candidates[i].userID < candidates[j].userID
	})
	preview := make([]UserInfo, 0, previewLimit)
	for i, c := range candidates {
		if i >= previewLimit {
			break
		}
		preview = append(preview, UserInfo{UserID: c.userID, Nickname: r.nicknameFor(c.userID), AvatarURL: types.StrPtr(r.profiles[c.userID].AvatarURL)})
	}
	return len(candidates), preview, nil
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
			// keyword 为手机号格式时同时按登录手机号精确匹配（受
			// phone_searchable 控制，spec user.md → users/search；与
			// Postgres 实现的 auth_accounts 子查询语义一致，内存版以
			// profile.Phone 承载 login_id 回填值）。
			phoneHit := isPhoneFormatKeyword(keyword) && settings.PhoneSearchable && profile.Phone == keyword
			if !phoneHit && !(settings.NicknameSearchable && strings.Contains(nickname, keyword)) && !strings.Contains(bio, keyword) {
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
func (r *InMemoryRepository) Follow(_ context.Context, userID, targetUserID string, _ FollowSource, _ string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	relation := r.ensureRelation(userID, targetUserID)
	// 与 PG 实现同口径：已关注时是 no-op（changed=false），调用方跳过重复通知。
	if relation.Following {
		return false, nil
	}
	relation.Following = true
	relation.FollowedAt = nowMemoryRFC3339()
	r.relations[userID][targetUserID] = relation
	return true, nil
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
func (r *InMemoryRepository) BlockCascade(_ context.Context, userID, targetUserID, reason string) error {
	// 与 Postgres 版语义一致：单锁内完成解除双向关注 + 拉黑，保证原子可见。
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, pair := range [][2]string{{userID, targetUserID}, {targetUserID, userID}} {
		relation := r.ensureRelation(pair[0], pair[1])
		relation.Following = false
		relation.FollowedAt = ""
		r.relations[pair[0]][pair[1]] = relation
	}
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
			items = append(items, BlockedUserItem{User: UserInfo{UserID: targetUserID, Nickname: r.nicknameFor(targetUserID)}, BlockedAt: relation.BlockedAt, Reason: types.StrPtr(relation.BlockReason)})
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

func (r *InMemoryRepository) Relationship(_ context.Context, userID, targetUserID string) (Relationship, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	forward := r.relations[userID][targetUserID]
	reverse := r.relations[targetUserID][userID]
	return Relationship{IsFollowing: forward.Following, IsFollowedBy: reverse.Following, IsBlocked: forward.Blocked, TargetBlocked: reverse.Blocked}, nil
}

func (r *InMemoryRepository) BlockedEitherWay(_ context.Context, aUserID, bUserID string) (callerBlocked, targetBlocked bool, err error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.relations[aUserID][bUserID].Blocked, r.relations[bUserID][aUserID].Blocked, nil
}

func (r *InMemoryRepository) RelationshipBatch(_ context.Context, userID string, targetUserIDs []string) (map[string]Relationship, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make(map[string]Relationship, len(targetUserIDs))
	for _, target := range targetUserIDs {
		forward := r.relations[userID][target]
		reverse := r.relations[target][userID]
		result[target] = Relationship{IsFollowing: forward.Following, IsFollowedBy: reverse.Following, IsBlocked: forward.Blocked, TargetBlocked: reverse.Blocked}
	}
	return result, nil
}

func (r *InMemoryRepository) ListFriends(_ context.Context, userID string, limit int, cursor string) (PaginatedFriends, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]RelatedUserItem, 0)
	for targetUserID, relation := range r.relations[userID] {
		reverse := r.relations[targetUserID][userID]
		if relation.Following && reverse.Following {
			lastOnline := r.profiles[targetUserID].LastOnline
			presence := r.profiles[targetUserID].PresenceStatus
			remark := relation.Remark
			followedBackAt := maxStringTime(relation.FollowedAt, reverse.FollowedAt)
			item := RelatedUserItem{
				User:       UserItem{User: UserInfo{UserID: targetUserID, Nickname: r.nicknameFor(targetUserID), AvatarURL: types.StrPtr(r.profiles[targetUserID].AvatarURL)}},
				FollowedAt: relation.FollowedAt, FollowedBackAt: &followedBackAt, Remark: &remark,
			}
			if lastOnline != "" {
				item.User.LastOnline = &lastOnline
			}
			item.User.PresenceStatus = &presence
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if *items[i].FollowedBackAt == *items[j].FollowedBackAt {
			return items[i].User.User.UserID < items[j].User.User.UserID
		}
		return *items[i].FollowedBackAt > *items[j].FollowedBackAt
	})
	return paginateFriends(items, limit, cursor), nil
}
func (r *InMemoryRepository) ListFollowing(_ context.Context, userID string, limit int, cursor string) (PaginatedFollowing, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]RelatedUserItem, 0)
	for targetUserID, relation := range r.relations[userID] {
		if relation.Following {
			lastOnline := r.profiles[targetUserID].LastOnline
			presence := r.profiles[targetUserID].PresenceStatus
			item := RelatedUserItem{
				User:       UserItem{User: UserInfo{UserID: targetUserID, Nickname: r.nicknameFor(targetUserID), AvatarURL: types.StrPtr(r.profiles[targetUserID].AvatarURL)}},
				FollowedAt: relation.FollowedAt,
			}
			if lastOnline != "" {
				item.User.LastOnline = &lastOnline
			}
			item.User.PresenceStatus = &presence
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].FollowedAt == items[j].FollowedAt {
			return items[i].User.User.UserID < items[j].User.User.UserID
		}
		return items[i].FollowedAt > items[j].FollowedAt
	})
	return paginateFollowing(items, limit, cursor), nil
}
func (r *InMemoryRepository) ListFollowers(_ context.Context, userID string, limit int, cursor string) (PaginatedFollowers, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]RelatedUserItem, 0)
	for sourceUserID, relationMap := range r.relations {
		relation := relationMap[userID]
		if relation.Following {
			lastOnline := r.profiles[sourceUserID].LastOnline
			presence := r.profiles[sourceUserID].PresenceStatus
			item := RelatedUserItem{
				User:       UserItem{User: UserInfo{UserID: sourceUserID, Nickname: r.nicknameFor(sourceUserID), AvatarURL: types.StrPtr(r.profiles[sourceUserID].AvatarURL)}},
				FollowedAt: relation.FollowedAt,
			}
			if lastOnline != "" {
				item.User.LastOnline = &lastOnline
			}
			item.User.PresenceStatus = &presence
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].FollowedAt == items[j].FollowedAt {
			return items[i].User.User.UserID < items[j].User.User.UserID
		}
		return items[i].FollowedAt > items[j].FollowedAt
	})
	return paginateFollowers(items, limit, cursor), nil
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
	limit = types.NormalizeLimit(limit)
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
		c := encodeUserMomentCursor(items[end-1].BlockedAt, items[end-1].User.UserID)
		resp.NextCursor = &c
	}
	return resp
}
func paginateFriends(items []RelatedUserItem, limit int, cursor string) PaginatedFriends {
	limit = types.NormalizeLimit(limit)
	start, ok := findUserCursorStart(cursor, func(i int) (string, string) {
		at := ""
		if items[i].FollowedBackAt != nil {
			at = *items[i].FollowedBackAt
		}
		return at, items[i].User.User.UserID
	}, len(items))
	if !ok {
		return PaginatedFriends{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := PaginatedFriends{Items: items[start:end]}
	if end < len(items) && end > start {
		c := encodeUserMomentCursor(*items[end-1].FollowedBackAt, items[end-1].User.User.UserID)
		resp.NextCursor = &c
	}
	return resp
}
func paginateFollowing(items []RelatedUserItem, limit int, cursor string) PaginatedFollowing {
	limit = types.NormalizeLimit(limit)
	start, ok := findUserCursorStart(cursor, func(i int) (string, string) { return items[i].FollowedAt, items[i].User.User.UserID }, len(items))
	if !ok {
		return PaginatedFollowing{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := PaginatedFollowing{Items: items[start:end]}
	if end < len(items) && end > start {
		c := encodeUserMomentCursor(items[end-1].FollowedAt, items[end-1].User.User.UserID)
		resp.NextCursor = &c
	}
	return resp
}
func paginateFollowers(items []RelatedUserItem, limit int, cursor string) PaginatedFollowers {
	limit = types.NormalizeLimit(limit)
	start, ok := findUserCursorStart(cursor, func(i int) (string, string) { return items[i].FollowedAt, items[i].User.User.UserID }, len(items))
	if !ok {
		return PaginatedFollowers{}
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}
	resp := PaginatedFollowers{Items: items[start:end]}
	if end < len(items) && end > start {
		c := encodeUserMomentCursor(items[end-1].FollowedAt, items[end-1].User.User.UserID)
		resp.NextCursor = &c
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
