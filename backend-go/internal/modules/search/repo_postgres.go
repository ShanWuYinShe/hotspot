package search

import (
	"context"
	"hotspot/backend-go/internal/platform/textutil"
	"sort"
	"strings"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Suggest(ctx context.Context, req SearchSuggestRequest) (SearchSuggestResponse, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return SearchSuggestResponse{}, persistence.ErrNotImplemented
	}

	prefix := strings.TrimSpace(req.Prefix)
	if prefix == "" {
		return SearchSuggestResponse{Items: []SearchSuggestItem{}}, nil
	}

	// 登录视角（/api/v1）与匿名视角（/public/v1）共用实现：以 ctx 中的
	// principal 区分。登录视角按 spec 填充 relation/social_context 与
	// viewer_state 等查看者字段；匿名视角不含查看者字段（Public 对偶类型）。
	viewerID := ""
	if principal, ok := platformauth.PrincipalFromContext(ctx); ok {
		viewerID = principal.UserID
	}

	allowed := normalizeSuggestTypes(req.Types)
	items := make([]scoredSuggestItem, 0)
	if allowed["tag"] {
		tagItems, err := queryTagSuggestions(ctx, queryable, prefix)
		if err != nil {
			return SearchSuggestResponse{}, err
		}
		items = append(items, tagItems...)
	}
	if allowed["user"] {
		userItems, err := queryUserSuggestions(ctx, queryable, viewerID, prefix, req.ReferencePoint, req.Radius)
		if err != nil {
			return SearchSuggestResponse{}, err
		}
		items = append(items, userItems...)
	}
	if allowed["group"] {
		groupItems, err := queryGroupSuggestions(ctx, queryable, viewerID, prefix, req.ReferencePoint, req.Radius)
		if err != nil {
			return SearchSuggestResponse{}, err
		}
		items = append(items, groupItems...)
	}
	if allowed["moment"] {
		momentItems, err := queryMomentSuggestions(ctx, queryable, prefix, req.ReferencePoint, req.Radius)
		if err != nil {
			return SearchSuggestResponse{}, err
		}
		items = append(items, momentItems...)
	}
	items = dedupeSuggestItems(items)
	sort.SliceStable(items, func(i, j int) bool { return compareSuggestItems(items[i], items[j]) < 0 })
	out := make([]SearchSuggestItem, 0, len(items))
	for _, item := range items {
		out = append(out, item.Item)
	}
	return SearchSuggestResponse{Items: out}, nil
}

type scoredSuggestItem struct {
	Item       SearchSuggestItem
	TypeOrder  int
	MatchRank  int
	Distance   int
	Popularity int
	TieBreaker string
}

func normalizeSuggestTypes(types []string) map[string]bool {
	allowed := map[string]bool{"tag": false, "user": false, "group": false, "moment": false}
	if len(types) == 0 {
		for key := range allowed {
			allowed[key] = true
		}
		return allowed
	}
	for _, typ := range types {
		normalized := strings.ToLower(strings.TrimSpace(typ))
		switch normalized {
		case "tag", "user", "group", "moment":
			allowed[normalized] = true
		}
	}
	if !allowed["tag"] && !allowed["user"] && !allowed["group"] && !allowed["moment"] {
		for key := range allowed {
			allowed[key] = true
		}
	}
	return allowed
}

func queryTagSuggestions(ctx context.Context, queryable persistence.Queryable, prefix string) ([]scoredSuggestItem, error) {
	rows, err := queryable.Query(ctx, querySearchSuggestTags, prefix, maxSuggestRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]scoredSuggestItem, 0)
	for rows.Next() {
		var name string
		var usageCount int
		if err := rows.Scan(&name, &usageCount); err != nil {
			return nil, err
		}
		items = append(items, scoredSuggestItem{
			Item:       SearchSuggestItem{Type: "tag", Payload: map[string]any{"name": name, "usage_count": usageCount}},
			TypeOrder:  0,
			MatchRank:  classifyPrefixMatch(prefix, name),
			Popularity: usageCount,
			TieBreaker: name,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func queryUserSuggestions(ctx context.Context, queryable persistence.Queryable, viewerID, prefix string, reference GeoPoint, radius int) ([]scoredSuggestItem, error) {
	rows, err := queryable.Query(ctx, querySearchSuggestUsers, prefix, maxSuggestRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type scannedUser struct {
		userID, nickname, avatarURL, bio, locationName, presenceStatus          string
		latitude, longitude                                                     float64
		lastOnline                                                              time.Time
		tags                                                                    []string
		momentsCount, friendsCount, followersCount, followingCount, groupsCount int
		locationVisible, presenceVisible, statsVisible                          bool
		distance                                                                int
	}
	scanned := make([]scannedUser, 0, maxSuggestRows)
	for rows.Next() {
		var u scannedUser
		if err := rows.Scan(&u.userID, &u.nickname, &u.avatarURL, &u.bio, &u.locationName, &u.latitude, &u.longitude, &u.presenceStatus, &u.lastOnline, &u.tags, &u.momentsCount, &u.friendsCount, &u.followersCount, &u.followingCount, &u.groupsCount, &u.locationVisible, &u.presenceVisible, &u.statsVisible); err != nil {
			return nil, err
		}
		u.distance = distanceFrom(reference, u.latitude, u.longitude)
		if radius > 0 && u.distance > radius {
			continue
		}
		scanned = append(scanned, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// 登录视角批量取查看者关系（正向关注/拉黑 + 反向被关注），一次往返。
	following, followedBy, blocked := map[string]bool{}, map[string]bool{}, map[string]bool{}
	if viewerID != "" && len(scanned) > 0 {
		ids := make([]string, 0, len(scanned))
		for _, u := range scanned {
			ids = append(ids, u.userID)
		}
		if err := collectViewerRelations(ctx, queryable, viewerID, ids, following, followedBy, blocked); err != nil {
			return nil, err
		}
	}
	items := make([]scoredSuggestItem, 0, len(scanned))
	for _, u := range scanned {
		userID, nickname, avatarURL, bio := u.userID, u.nickname, u.avatarURL, u.bio
		lastOnline, tags := u.lastOnline, u.tags
		momentsCount, friendsCount, followersCount, followingCount, groupsCount := u.momentsCount, u.friendsCount, u.followersCount, u.followingCount, u.groupsCount
		locationVisible, presenceVisible, statsVisible := u.locationVisible, u.presenceVisible, u.statsVisible
		presenceStatus, locationName, latitude, longitude := u.presenceStatus, u.locationName, u.latitude, u.longitude
		distance := u.distance
		// payload 复用通用 UserItem 结构（spec api_spec_types.md，形状恒定）：
		// 可空字段显式 null（bio/location/presence_status/last_online_at/stats、
		// tags 恒为数组）。L1 可见性门控（spec api_spec_authz.md #28/#108，
		// 对齐 users open_the_door 的 isSelf||settings.XVisible 口径；suggest
		// 无 isSelf 场景）：隐藏即 null（location 含精确坐标与 distance/bearing，
		// 隐藏防三角定位）。登录视角补充查看者字段 relation（无任何关系时为
		// null）与 social_context（spec：搜索/列表场景恒 null）；匿名视角不含
		// 查看者字段（Public 对偶类型）。
		user := map[string]any{"user_id": userID, "nickname": nickname, "avatar_url": types.StrPtr(avatarURL)}
		payload := map[string]any{"user": user, "bio": types.StrPtr(bio)}
		if tags == nil {
			payload["tags"] = []string{}
		} else {
			payload["tags"] = tags
		}
		if locationVisible {
			payload["location"] = relativeLocationPayload(reference, locationName, latitude, longitude)
		} else {
			payload["location"] = nil
		}
		if presenceVisible {
			ps := presenceStatusPtr(presenceStatus)
			if ps == "" {
				payload["presence_status"] = nil
			} else {
				payload["presence_status"] = ps
			}
			if lastOnline.IsZero() {
				payload["last_online_at"] = nil
			} else {
				payload["last_online_at"] = lastOnline.UTC().Format(time.RFC3339)
			}
		} else {
			payload["presence_status"] = nil
			payload["last_online_at"] = nil
		}
		if statsVisible {
			payload["stats"] = map[string]any{
				"moments_count":   momentsCount,
				"friends_count":   friendsCount,
				"followers_count": followersCount,
				"following_count": followingCount,
				"groups_count":    groupsCount,
			}
		} else {
			payload["stats"] = nil
		}
		if viewerID != "" {
			payload["social_context"] = nil
			if following[userID] || followedBy[userID] || blocked[userID] {
				payload["relation"] = map[string]any{
					"is_following":   following[userID],
					"is_followed_by": followedBy[userID],
					"is_blocked":     blocked[userID],
				}
			} else {
				payload["relation"] = nil
			}
		}
		items = append(items, scoredSuggestItem{
			Item:       SearchSuggestItem{Type: "user", Payload: payload},
			TypeOrder:  1,
			MatchRank:  minInt(classifyPrefixMatch(prefix, nickname), classifyPrefixMatch(prefix, bio)),
			Distance:   distance,
			TieBreaker: userID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// presenceStatusPtr normalizes the raw presence column; empty or unknown
// values are omitted entirely (privacy-controlled field).
func presenceStatusPtr(raw string) string {
	switch raw {
	case "online", "offline":
		return raw
	default:
		return ""
	}
}

func queryGroupSuggestions(ctx context.Context, queryable persistence.Queryable, viewerID, prefix string, reference GeoPoint, radius int) ([]scoredSuggestItem, error) {
	rows, err := queryable.Query(ctx, querySearchSuggestGroups, prefix, maxSuggestRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]scoredSuggestItem, 0)
	for rows.Next() {
		var groupID, name, locationName, joinMethod, ownerID string
		// avatar_url/description/next_activity_at 均为可空列（migrations/004），
		// 必须扫进 *string，否则 NULL 行触发 scan 错误（曾致 suggest 10600）。
		var avatarURL, description, nextActivity *string
		var latitude, longitude float64
		var requiresApproval, isSearchable bool
		var membersCount, onlineMembersCount, maxMembers int
		var tags []string
		var createdAt time.Time
		if err := rows.Scan(&groupID, &name, &avatarURL, &description, &locationName, &latitude, &longitude, &membersCount, &joinMethod, &requiresApproval, &tags, &nextActivity, &isSearchable, &maxMembers, &createdAt, &ownerID, &onlineMembersCount); err != nil {
			return nil, err
		}
		distance := distanceFrom(reference, latitude, longitude)
		if radius > 0 && distance > radius {
			continue
		}
		// payload 复用通用 GroupItem 结构（spec api_spec_types.md，形状恒定）：
		// 群管理字段（is_searchable/max_members）不属于 GroupItem（属
		// GroupSettings）；created_at/owner_id 为「详情返回」，搜索场景显式
		// null；last_activity_at/viewer_state/friends_in_group 建议场景 null
		//（viewer_state/friends_in_group 仅登录视角携带键，匿名 Public 类型无）。
		group := map[string]any{"group_id": groupID, "name": name, "members_count": membersCount, "avatar_url": avatarURL}
		payload := map[string]any{
			"group":                group,
			"description":          description,
			"tags":                 firstNonNilStrings(tags),
			"join_method":          joinMethod,
			"requires_approval":    requiresApproval,
			"location":             relativeLocationPayload(reference, locationName, latitude, longitude),
			"last_activity_at":     nil,
			"online_members_count": onlineMembersCount,
			"next_activity_at":     nextActivity,
			"created_at":           nil,
			"owner_id":             nil,
		}
		if viewerID != "" {
			payload["viewer_state"] = nil
			payload["friends_in_group"] = nil
		}
		items = append(items, scoredSuggestItem{
			Item:       SearchSuggestItem{Type: "group", Payload: payload},
			TypeOrder:  2,
			MatchRank:  minInt(classifyPrefixMatch(prefix, name), classifyPrefixMatch(prefix, derefString(description))),
			Distance:   distance,
			Popularity: membersCount,
			TieBreaker: groupID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func queryMomentSuggestions(ctx context.Context, queryable persistence.Queryable, prefix string, reference GeoPoint, radius int) ([]scoredSuggestItem, error) {
	rows, err := queryable.Query(ctx, querySearchSuggestMoments, prefix, maxSuggestRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]scoredSuggestItem, 0)
	for rows.Next() {
		var momentID, userID, nickname, authorAvatar, description, locationName, coverURL string
		var latitude, longitude float64
		var popularity int
		if err := rows.Scan(&momentID, &userID, &nickname, &authorAvatar, &description, &locationName, &latitude, &longitude, &coverURL, &popularity); err != nil {
			return nil, err
		}
		distance := distanceFrom(reference, latitude, longitude)
		if radius > 0 && distance > radius {
			continue
		}
		payload := map[string]any{
			"moment_id":           momentID,
			"description_preview": truncatePreview(description),
			"cover_url":           types.StrPtr(coverURL),
			"author": map[string]any{
				"user_id":    userID,
				"nickname":   nickname,
				"avatar_url": types.StrPtr(authorAvatar),
			},
			"location": relativeLocationPayload(reference, locationName, latitude, longitude),
		}
		items = append(items, scoredSuggestItem{
			Item:       SearchSuggestItem{Type: "moment", Payload: payload},
			TypeOrder:  3,
			MatchRank:  classifyPrefixMatch(prefix, description),
			Distance:   distance,
			Popularity: popularity,
			TieBreaker: momentID,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func compareSuggestItems(left, right scoredSuggestItem) int {
	if left.MatchRank != right.MatchRank {
		return left.MatchRank - right.MatchRank
	}
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if left.Popularity != right.Popularity {
		if left.Popularity > right.Popularity {
			return -1
		}
		return 1
	}
	if left.TypeOrder != right.TypeOrder {
		return left.TypeOrder - right.TypeOrder
	}
	if left.TieBreaker < right.TieBreaker {
		return -1
	}
	if left.TieBreaker > right.TieBreaker {
		return 1
	}
	return 0
}

func classifyPrefixMatch(prefix, value string) int {
	needle := strings.ToLower(strings.TrimSpace(prefix))
	haystack := strings.ToLower(strings.TrimSpace(value))
	if needle == "" || haystack == "" {
		return 3
	}
	switch {
	case haystack == needle:
		return 0
	case strings.HasPrefix(haystack, needle):
		return 1
	case strings.Contains(haystack, needle):
		return 2
	default:
		return 3
	}
}

func relativeLocationPayload(reference GeoPoint, locationName string, latitude, longitude float64) map[string]any {
	distance, bearing := computeRelativeDistance(reference, GeoPoint{Latitude: latitude, Longitude: longitude})
	return map[string]any{
		"point":           map[string]any{"latitude": latitude, "longitude": longitude},
		"location_name":   locationName,
		"distance":        distance,
		"bearing_degrees": bearing,
	}
}

func computeRelativeDistance(reference GeoPoint, candidate GeoPoint) (int, int) {
	point := geo.Point{Latitude: reference.Latitude, Longitude: reference.Longitude}
	other := geo.Point{Latitude: candidate.Latitude, Longitude: candidate.Longitude}
	return geo.DistanceMeters(point, other), geo.BearingDegrees(point, other)
}

func distanceFrom(reference GeoPoint, latitude, longitude float64) int {
	distance, _ := computeRelativeDistance(reference, GeoPoint{Latitude: latitude, Longitude: longitude})
	return distance
}

// collectViewerRelations 批量取查看者对目标用户的正向/反向关系边（口径与
// users 模块 RelationshipBatch 一致），用于登录视角 suggest 的 relation 填充。
func collectViewerRelations(ctx context.Context, queryable persistence.Queryable, viewerID string, targetIDs []string, following, followedBy, blocked map[string]bool) error {
	if len(targetIDs) == 0 {
		return nil
	}
	rows, err := queryable.Query(ctx, querySuggestViewerRelations, viewerID, targetIDs)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var targetUserID string
		var isFollowing, isBlocked bool
		if err := rows.Scan(&targetUserID, &isFollowing, &isBlocked); err != nil {
			return err
		}
		following[targetUserID] = isFollowing
		blocked[targetUserID] = isBlocked
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	// 反向边（对方是否关注查看者）：口径同 users 模块 queryListUserRelationsReverse。
	reverseRows, err := queryable.Query(ctx, querySuggestViewerRelationsReverse, viewerID, targetIDs)
	if err != nil {
		return err
	}
	defer reverseRows.Close()
	for reverseRows.Next() {
		var sourceUserID string
		var revFollowing bool
		if err := reverseRows.Scan(&sourceUserID, &revFollowing); err != nil {
			return err
		}
		followedBy[sourceUserID] = revFollowing
	}
	return reverseRows.Err()
}

// firstNonNilStrings 返回 tags；nil 切片归一为空数组（spec：数组恒在）。
func firstNonNilStrings(tags []string) []string {
	if tags == nil {
		return []string{}
	}
	return tags
}

func truncatePreview(description string) string {
	// 按 rune 截断：字节切片会把多字节字符（中文）拦腰截断产生非法 UTF-8。
	// 上限 100 对齐 spec MomentSummary.description_preview。
	return textutil.TruncateRunes(description, 100)
}

func dedupeSuggestItems(items []scoredSuggestItem) []scoredSuggestItem {
	seen := make(map[string]struct{}, len(items))
	out := make([]scoredSuggestItem, 0, len(items))
	for _, item := range items {
		key := item.Item.Type + ":" + item.TieBreaker
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

// derefString nil 安全解引用（可空列 → 参与字符串比较时按空串处理）。
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

var _ Repository = (*PostgresRepository)(nil)
