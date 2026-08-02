package search

import (
	"context"
	"sort"
	"strings"

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
	rows, err := queryable.Query(ctx, querySearchSuggestUsers, prefix, reference.Latitude, reference.Longitude, radius, maxSuggestRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]scoredSuggestItem, 0)
	for rows.Next() {
		var userID, nickname, avatarURL, bio, locationName string
		var latitude, longitude float64
		var distance int
		if err := rows.Scan(&userID, &nickname, &avatarURL, &bio, &locationName, &latitude, &longitude, &distance); err != nil {
			return nil, err
		}
		// Radius filtering now happens in SQL (the ($4 = 0 OR distance <= $4)
		// predicate); distance is read from the computed distance_meters column
		// rather than recomputed in Go.
		payload := map[string]any{"user_id": userID, "nickname": nickname}
		if avatarURL != "" {
			payload["avatar_url"] = avatarURL
		}
		if bio != "" {
			payload["bio"] = bio
		}
		payload["location"] = relativeLocationPayloadWithDistance(reference, locationName, latitude, longitude, distance)
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

func queryGroupSuggestions(ctx context.Context, queryable persistence.QueryableDB, prefix string, reference GeoPoint, radius int) ([]scoredSuggestItem, error) {
	rows, err := queryable.Query(ctx, querySearchSuggestGroups, prefix, reference.Latitude, reference.Longitude, radius, maxSuggestRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]scoredSuggestItem, 0)
	for rows.Next() {
		var groupID, name, avatarURL, description, locationName string
		var latitude, longitude float64
		var membersCount int
		var distance int
		if err := rows.Scan(&groupID, &name, &avatarURL, &description, &locationName, &latitude, &longitude, &membersCount, &distance); err != nil {
			return nil, err
		}
		payload := map[string]any{"group_id": groupID, "name": name, "members_count": membersCount}
		if avatarURL != "" {
			payload["avatar_url"] = avatarURL
		}
		if description != "" {
			payload["description"] = description
		}
		payload["location"] = relativeLocationPayloadWithDistance(reference, locationName, latitude, longitude, distance)
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
	rows, err := queryable.Query(ctx, querySearchSuggestMoments, prefix, reference.Latitude, reference.Longitude, radius, maxSuggestRows)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]scoredSuggestItem, 0)
	for rows.Next() {
		var momentID, userID, nickname, description, locationName string
		var latitude, longitude float64
		var popularity, distance int
		if err := rows.Scan(&momentID, &userID, &nickname, &description, &locationName, &latitude, &longitude, &popularity, &distance); err != nil {
			return nil, err
		}
		payload := map[string]any{
			"moment_id":           momentID,
			"description_preview": truncatePreview(description),
			"author": map[string]any{
				"user_id":  userID,
				"nickname": nickname,
			},
			"location": relativeLocationPayloadWithDistance(reference, locationName, latitude, longitude, distance),
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

// relativeLocationPayloadWithDistance is like relativeLocationPayload but
// reuses the distance already computed by PostgreSQL (distance_meters column)
// instead of recomputing it in Go. bearing is still Go-side because it is a
// display-only value that never drives ordering.
func relativeLocationPayloadWithDistance(reference GeoPoint, locationName string, latitude, longitude float64, distance int) map[string]any {
	_, bearing := computeRelativeDistance(reference, GeoPoint{Latitude: latitude, Longitude: longitude})
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
