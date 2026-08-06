package search

import (
	"context"
	"sort"
	"strings"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Suggest(ctx context.Context, req SearchSuggestRequest) (SearchSuggestResponse, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return SearchSuggestResponse{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return SearchSuggestResponse{}, persistence.ErrNotImplemented
	}

	prefix := strings.TrimSpace(req.Prefix)
	if prefix == "" {
		return SearchSuggestResponse{Items: []SearchSuggestItem{}}, nil
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
		userItems, err := queryUserSuggestions(ctx, queryable, prefix, req.ReferencePoint, req.Radius)
		if err != nil {
			return SearchSuggestResponse{}, err
		}
		items = append(items, userItems...)
	}
	if allowed["group"] {
		groupItems, err := queryGroupSuggestions(ctx, queryable, prefix, req.ReferencePoint, req.Radius)
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

func queryTagSuggestions(ctx context.Context, queryable persistence.QueryableDB, prefix string) ([]scoredSuggestItem, error) {
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

func queryUserSuggestions(ctx context.Context, queryable persistence.QueryableDB, prefix string, reference GeoPoint, radius int) ([]scoredSuggestItem, error) {
	rows, err := queryable.Query(ctx, querySearchSuggestUsers, prefix, maxSuggestRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]scoredSuggestItem, 0)
	for rows.Next() {
		var userID, nickname, avatarURL, bio, locationName, presenceStatus string
		var latitude, longitude float64
		var lastOnline time.Time
		var tags []string
		var momentsCount, friendsCount, followersCount, followingCount, groupsCount int
		if err := rows.Scan(&userID, &nickname, &avatarURL, &bio, &locationName, &latitude, &longitude, &presenceStatus, &lastOnline, &tags, &momentsCount, &friendsCount, &followersCount, &followingCount, &groupsCount); err != nil {
			return nil, err
		}
		distance := distanceFrom(reference, latitude, longitude)
		if radius > 0 && distance > radius {
			continue
		}
		// payload 复用通用 UserItem 结构（spec api_spec_types.md）：
		// 不含 viewer 相关的 @auth-only 字段（relation/social_context），
		// 与 suggest 无登录上下文的查询语义一致。
		user := map[string]any{"user_id": userID, "nickname": nickname}
		if avatarURL != "" {
			user["avatar_url"] = avatarURL
		}
		payload := map[string]any{"user": user, "location": relativeLocationPayload(reference, locationName, latitude, longitude)}
		if bio != "" {
			payload["bio"] = bio
		}
		if len(tags) > 0 {
			payload["tags"] = tags
		}
		if ps := presenceStatusPtr(presenceStatus); ps != "" {
			payload["presence_status"] = ps
		}
		if !lastOnline.IsZero() {
			payload["last_online_at"] = lastOnline.UTC().Format(time.RFC3339)
		}
		payload["stats"] = map[string]any{
			"moments_count":   momentsCount,
			"friends_count":   friendsCount,
			"followers_count": followersCount,
			"following_count": followingCount,
			"groups_count":    groupsCount,
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

func queryGroupSuggestions(ctx context.Context, queryable persistence.QueryableDB, prefix string, reference GeoPoint, radius int) ([]scoredSuggestItem, error) {
	rows, err := queryable.Query(ctx, querySearchSuggestGroups, prefix, maxSuggestRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]scoredSuggestItem, 0)
	for rows.Next() {
		var groupID, name, avatarURL, description, locationName, joinMethod, nextActivity, ownerID string
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
		// payload 复用通用 GroupItem 结构（spec api_spec_types.md）
		group := map[string]any{"group_id": groupID, "name": name, "members_count": membersCount}
		if avatarURL != "" {
			group["avatar_url"] = avatarURL
		}
		payload := map[string]any{
			"group":                group,
			"join_method":          joinMethod,
			"requires_approval":    requiresApproval,
			"location":             relativeLocationPayload(reference, locationName, latitude, longitude),
			"online_members_count": onlineMembersCount,
		}
		if description != "" {
			payload["description"] = description
		}
		if len(tags) > 0 {
			payload["tags"] = tags
		}
		if nextActivity != "" {
			payload["next_activity"] = nextActivity
		}
		if isSearchable {
			payload["is_searchable"] = isSearchable
		}
		if maxMembers > 0 {
			payload["max_members"] = maxMembers
		}
		if !createdAt.IsZero() {
			payload["created_at"] = createdAt.UTC().Format(time.RFC3339)
		}
		if ownerID != "" {
			payload["owner_id"] = ownerID
		}
		items = append(items, scoredSuggestItem{
			Item:       SearchSuggestItem{Type: "group", Payload: payload},
			TypeOrder:  2,
			MatchRank:  minInt(classifyPrefixMatch(prefix, name), classifyPrefixMatch(prefix, description)),
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

func queryMomentSuggestions(ctx context.Context, queryable persistence.QueryableDB, prefix string, reference GeoPoint, radius int) ([]scoredSuggestItem, error) {
	rows, err := queryable.Query(ctx, querySearchSuggestMoments, prefix, maxSuggestRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]scoredSuggestItem, 0)
	for rows.Next() {
		var momentID, userID, nickname, description, locationName string
		var latitude, longitude float64
		var popularity int
		if err := rows.Scan(&momentID, &userID, &nickname, &description, &locationName, &latitude, &longitude, &popularity); err != nil {
			return nil, err
		}
		distance := distanceFrom(reference, latitude, longitude)
		if radius > 0 && distance > radius {
			continue
		}
		payload := map[string]any{
			"moment_id":           momentID,
			"description_preview": truncatePreview(description),
			"author": map[string]any{
				"user_id":  userID,
				"nickname": nickname,
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

func truncatePreview(description string) string {
	description = strings.TrimSpace(description)
	if len(description) <= 60 {
		return description
	}
	return description[:60]
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

var _ Repository = (*PostgresRepository)(nil)
