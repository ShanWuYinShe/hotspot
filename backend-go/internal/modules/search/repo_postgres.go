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
	// SQL 里的 ILIKE '%..%' 用 prefix 作模式：先转义 % _ \，与 users/groups
	// 搜索口径一致（R38 同类问题——未转义时关键词里的通配符会改变匹配语义）。
	// 计分（classifyPrefixMatch）仍用原始 prefix，保持字面匹配语义。
	escapedPrefix := textutil.EscapeLikePattern(prefix)

	// 登录视角（/api/v1）与匿名视角（/public/v1）共用实现：以 ctx 中的
	// principal 区分。登录视角按 spec 填充 relation/social_context 与
	// viewer_state 等查看者字段；匿名视角不含查看者字段（Public 对偶类型）。
	viewerID := ""
	if principal, ok := platformauth.PrincipalFromContext(ctx); ok {
		viewerID = principal.UserID
	}
	// ReferencePoint 已是指针语义：service 侧回退链解析后恒非 nil，此处
	// 解引用为值供各子查询传参；直接调用 repo 的场景防御性取零值。
	reference := GeoPoint{}
	if req.ReferencePoint != nil {
		reference = *req.ReferencePoint
	}

	allowed := normalizeSuggestTypes(req.Types)
	items := make([]scoredSuggestItem, 0)
	if allowed["tag"] {
		tagItems, err := queryTagSuggestions(ctx, queryable, escapedPrefix)
		if err != nil {
			return SearchSuggestResponse{}, err
		}
		items = append(items, tagItems...)
	}
	if allowed["user"] {
		userItems, err := queryUserSuggestions(ctx, queryable, viewerID, escapedPrefix, reference, req.Radius)
		if err != nil {
			return SearchSuggestResponse{}, err
		}
		items = append(items, userItems...)
	}
	if allowed["group"] {
		groupItems, err := queryGroupSuggestions(ctx, queryable, viewerID, escapedPrefix, reference, req.Radius)
		if err != nil {
			return SearchSuggestResponse{}, err
		}
		items = append(items, groupItems...)
	}
	if allowed["moment"] {
		momentItems, err := queryMomentSuggestions(ctx, queryable, escapedPrefix, reference, req.Radius)
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
			Item:       SearchSuggestItem{Type: "tag", Payload: TagItem{Name: name, UsageCount: usageCount}},
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

// suggestGeoFilter 返回 suggest 的 S2 预过滤区间（[lo,hi] 数组对）。
// radius<=0 或参考坐标非法时返回 nil：SQL 侧按「数组为空 → 不做地理过滤」
// 处理。S2 覆盖是半径圆的包围盒超集，仅作 DB 粗筛，精确距离过滤仍在 Go 层
// （distanceFrom + radius 比较）→ 不会产生假阴性。
func suggestGeoFilter(reference GeoPoint, radius int) (mins, maxs []int64) {
	if radius <= 0 {
		return nil, nil
	}
	point := geo.Point{Latitude: reference.Latitude, Longitude: reference.Longitude}
	if !geo.IsValidPoint(point) {
		return nil, nil
	}
	return geo.CoveringArrays(geo.BoundsAround(point, radius))
}

func queryUserSuggestions(ctx context.Context, queryable persistence.Queryable, viewerID, prefix string, reference GeoPoint, radius int) ([]scoredSuggestItem, error) {
	coverMins, coverMaxs := suggestGeoFilter(reference, radius)
	rows, err := queryable.Query(ctx, querySearchSuggestUsers, prefix, maxSuggestRows, coverMins, coverMaxs)
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
		// payload 使用类型化 UserItem 形状（spec api_spec_types.md）：可空字段
		// 显式 null（bio/location/presence_status/last_online_at/stats，tags 恒
		// 数组）。L1 可见性门控隐藏即 null（location 含精确坐标与 distance/
		// bearing，隐藏防三角定位）；登录视角补充 relation/social_context。
		payload := SearchSuggestUserPayload{
			User:          SearchSuggestUser{UserID: userID, Nickname: nickname, AvatarURL: types.StrPtr(avatarURL)},
			Bio:           types.StrPtr(bio),
			Tags:          firstNonNilStrings(tags),
			SocialContext: nil,
		}
		if locationVisible {
			payload.Location = relativeLocationView(reference, locationName, latitude, longitude)
		}
		if presenceVisible {
			if ps := presenceStatusPtr(presenceStatus); ps != "" {
				payload.PresenceStatus = &ps
			}
			if !lastOnline.IsZero() {
				formatted := lastOnline.UTC().Format(time.RFC3339)
				payload.LastOnlineAt = &formatted
			}
		}
		if statsVisible {
			payload.Stats = &SearchSuggestUserStats{
				MomentsCount:   momentsCount,
				FriendsCount:   friendsCount,
				FollowersCount: followersCount,
				FollowingCount: followingCount,
				GroupsCount:    groupsCount,
			}
		}
		if viewerID != "" && (following[userID] || followedBy[userID] || blocked[userID]) {
			payload.Relation = &SearchSuggestUserRelation{IsFollowing: following[userID], IsFollowedBy: followedBy[userID], IsBlocked: blocked[userID]}
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
	coverMins, coverMaxs := suggestGeoFilter(reference, radius)
	rows, err := queryable.Query(ctx, querySearchSuggestGroups, prefix, maxSuggestRows, coverMins, coverMaxs)
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
		payload := SearchSuggestGroupPayload{
			Group:              SearchSuggestGroup{GroupID: groupID, Name: name, MembersCount: membersCount, AvatarURL: avatarURL},
			Description:        description,
			Tags:               firstNonNilStrings(tags),
			JoinMethod:         joinMethod,
			RequiresApproval:   requiresApproval,
			Location:           relativeLocationView(reference, locationName, latitude, longitude),
			OnlineMembersCount: onlineMembersCount,
			NextActivityAt:     nextActivity,
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
	coverMins, coverMaxs := suggestGeoFilter(reference, radius)
	rows, err := queryable.Query(ctx, querySearchSuggestMoments, prefix, maxSuggestRows, coverMins, coverMaxs)
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
		payload := SearchSuggestMomentPayload{
			MomentID:           momentID,
			DescriptionPreview: truncatePreview(description),
			CoverURL:           types.StrPtr(coverURL),
			Author:             SearchSuggestUser{UserID: userID, Nickname: nickname, AvatarURL: types.StrPtr(authorAvatar)},
			Location:           relativeLocationView(reference, locationName, latitude, longitude),
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

func relativeLocationView(reference GeoPoint, locationName string, latitude, longitude float64) *RelativeLocation {
	distance, bearing := computeRelativeDistance(reference, GeoPoint{Latitude: latitude, Longitude: longitude})
	distanceValue, bearingValue := float64(distance), float64(bearing)
	return &RelativeLocation{
		Point:          GeoPoint{Latitude: latitude, Longitude: longitude},
		LocationName:   locationName,
		Distance:       &distanceValue,
		BearingDegrees: &bearingValue,
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
