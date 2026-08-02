package feed

import (
	"fmt"
	"strings"

	"hotspot/backend-go/internal/platform/geo"
)

// The three feed queries below previously used positional $1/$2/$3 constants
// (mins, maxs, viewerUserID). After pushing distance/radius/sort into SQL per
// AGENTS.md they are now built dynamically: the reference lat/lng come first
// (HaversineDistanceSQL needs them), then the S2 covering arrays, the viewer
// user id and the radius. Distance, radius filtering and ORDER BY now happen in
// PostgreSQL, mirroring buildSearchUsersQuery / buildSearchMomentsQuery.

// buildFeedUsersQuery builds the nearby-users SQL. Distance is computed by the
// Haversine expression in the SELECT; radius filtering and ORDER BY happen in
// the DB. The outer wrapper lets the radius predicate reference the computed
// distance alias (PostgreSQL forbids referencing SELECT aliases in the same
// WHERE level), mirroring buildSearchUsersQuery. The returned args slice is in
// the order [refLat, refLng, mins, maxs, viewerUserID, radius].
func buildFeedUsersQuery(refLat, refLng float64, mins, maxs []int64, viewerUserID string, radius int) (string, []any) {
	args := make([]any, 0, 6)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	refLatArg := addArg(refLat)
	refLngArg := addArg(refLng)
	distanceExpr := geo.HaversineDistanceSQL(refLatArg, refLngArg)
	minsArg := addArg(mins)
	maxsArg := addArg(maxs)
	viewerArg := addArg(viewerUserID)
	radiusArg := addArg(radius)
	var builder strings.Builder
	builder.WriteString("SELECT * FROM (")
	builder.WriteString("SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), COALESCE(u.bio, ''), u.location_name, u.latitude, u.longitude,")
	builder.WriteString(" EXISTS(SELECT 1 FROM user_relations mine WHERE mine.user_id = ")
	builder.WriteString(viewerArg)
	builder.WriteString(" AND mine.target_user_id = u.user_id AND mine.is_following = TRUE),")
	builder.WriteString(" EXISTS(SELECT 1 FROM user_relations rev WHERE rev.user_id = u.user_id AND rev.target_user_id = ")
	builder.WriteString(viewerArg)
	builder.WriteString(" AND rev.is_following = TRUE),")
	builder.WriteString(" COALESCE((SELECT COUNT(*) FROM user_relations mine JOIN user_relations theirs ON theirs.user_id = u.user_id AND theirs.target_user_id = mine.target_user_id AND theirs.is_following = TRUE WHERE mine.user_id = ")
	builder.WriteString(viewerArg)
	builder.WriteString(" AND mine.is_following = TRUE), 0),")
	builder.WriteString(" COALESCE((SELECT COUNT(*) FROM group_members gm1 JOIN group_members gm2 ON gm2.group_id = gm1.group_id AND gm2.user_id = u.user_id JOIN groups g ON g.group_id = gm1.group_id WHERE gm1.user_id = ")
	builder.WriteString(viewerArg)
	builder.WriteString(" AND g.deleted_at IS NULL), 0),")
	builder.WriteString(distanceExpr)
	builder.WriteString(" AS distance_meters")
	builder.WriteString(" FROM users u JOIN user_settings us ON us.user_id = u.user_id")
	builder.WriteString(" WHERE us.searchable = TRUE AND us.location_visible = TRUE AND us.privacy_level = 'public'")
	builder.WriteString(" AND EXISTS (SELECT 1 FROM unnest(")
	builder.WriteString(minsArg)
	builder.WriteString("::bigint[], ")
	builder.WriteString(maxsArg)
	builder.WriteString("::bigint[]) AS t(lo, hi) WHERE u.s2_cell_id BETWEEN t.lo AND t.hi)")
	builder.WriteString(") feed WHERE distance_meters <= ")
	builder.WriteString(radiusArg)
	builder.WriteString(" ORDER BY distance_meters ASC, u.user_id ASC")
	return builder.String(), args
}

// buildFeedGroupsQuery builds the nearby-groups SQL. The inner GROUP BY is kept
// intact and wrapped so the outer radius/order predicates can reference the
// computed distance_meters alias without forcing the distance expression into
// the GROUP BY (it is added there defensively anyway for strict-mode safety).
func buildFeedGroupsQuery(refLat, refLng float64, mins, maxs []int64, viewerUserID string, radius int) (string, []any) {
	args := make([]any, 0, 6)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	refLatArg := addArg(refLat)
	refLngArg := addArg(refLng)
	distanceExpr := geo.HaversineDistanceSQL(refLatArg, refLngArg)
	minsArg := addArg(mins)
	maxsArg := addArg(maxs)
	viewerArg := addArg(viewerUserID)
	radiusArg := addArg(radius)
	var builder strings.Builder
	builder.WriteString("SELECT * FROM (")
	builder.WriteString("SELECT g.group_id, g.name, COALESCE(g.avatar_url, ''), COALESCE(g.description, ''), g.has_password, g.location_name, g.latitude, g.longitude,")
	builder.WriteString(" COUNT(gm.user_id) AS members_count,")
	builder.WriteString(" EXISTS(SELECT 1 FROM group_members viewer_gm WHERE viewer_gm.group_id = g.group_id AND viewer_gm.user_id = ")
	builder.WriteString(viewerArg)
	builder.WriteString("),")
	builder.WriteString(" COALESCE((SELECT COUNT(*) FROM group_members gm_friend JOIN user_relations ur ON ur.target_user_id = gm_friend.user_id AND ur.user_id = ")
	builder.WriteString(viewerArg)
	builder.WriteString(" AND ur.is_following = TRUE WHERE gm_friend.group_id = g.group_id), 0),")
	builder.WriteString(distanceExpr)
	builder.WriteString(" AS distance_meters")
	builder.WriteString(" FROM groups g LEFT JOIN group_members gm ON gm.group_id = g.group_id")
	builder.WriteString(" WHERE g.deleted_at IS NULL AND g.is_searchable = TRUE")
	builder.WriteString(" AND EXISTS (SELECT 1 FROM unnest(")
	builder.WriteString(minsArg)
	builder.WriteString("::bigint[], ")
	builder.WriteString(maxsArg)
	builder.WriteString("::bigint[]) AS t(lo, hi) WHERE g.s2_cell_id BETWEEN t.lo AND t.hi)")
	builder.WriteString(" GROUP BY g.group_id, g.name, g.avatar_url, g.description, g.has_password, g.location_name, g.latitude, g.longitude")
	builder.WriteString(") feed WHERE distance_meters <= ")
	builder.WriteString(radiusArg)
	builder.WriteString(" ORDER BY distance_meters ASC, g.group_id ASC")
	return builder.String(), args
}

// buildFeedMomentsNearbyQuery builds the nearby-moments SQL. Distance and radius
// filtering are pushed to SQL; the Go-side relevance score (popularity +
// following bonus + radius-distance bonus) still uses the SQL-computed distance
// value — it no longer calls geo.NewSearchPlan / plan.Match.
func buildFeedMomentsNearbyQuery(refLat, refLng float64, mins, maxs []int64, viewerUserID string, radius int) (string, []any) {
	args := make([]any, 0, 6)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	refLatArg := addArg(refLat)
	refLngArg := addArg(refLng)
	distanceExpr := geo.HaversineDistanceSQL(refLatArg, refLngArg)
	minsArg := addArg(mins)
	maxsArg := addArg(maxs)
	viewerArg := addArg(viewerUserID)
	radiusArg := addArg(radius)
	var builder strings.Builder
	builder.WriteString("SELECT m.moment_id, m.user_id, COALESCE(u.nickname, ''), COALESCE(u.avatar_url, ''), m.description, m.created_at, m.likes_count, m.comments_count, m.shares_count, m.location_name, m.latitude, m.longitude,")
	builder.WriteString(" EXISTS(SELECT 1 FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = m.moment_id AND r.user_id = ")
	builder.WriteString(viewerArg)
	builder.WriteString("),")
	builder.WriteString(" EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = ")
	builder.WriteString(viewerArg)
	builder.WriteString("),")
	builder.WriteString(" EXISTS(SELECT 1 FROM user_relations ur WHERE ur.target_user_id = m.user_id AND ur.user_id = ")
	builder.WriteString(viewerArg)
	builder.WriteString(" AND ur.is_following = TRUE),")
	builder.WriteString(distanceExpr)
	builder.WriteString(" AS distance_meters")
	builder.WriteString(" FROM moments m JOIN users u ON u.user_id = m.user_id")
	builder.WriteString(" WHERE m.privacy_level = 'public'")
	builder.WriteString(" AND EXISTS (SELECT 1 FROM unnest(")
	builder.WriteString(minsArg)
	builder.WriteString("::bigint[], ")
	builder.WriteString(maxsArg)
	builder.WriteString("::bigint[]) AS t(lo, hi) WHERE m.s2_cell_id BETWEEN t.lo AND t.hi)")
	builder.WriteString(" AND ")
	builder.WriteString(distanceExpr)
	builder.WriteString(" <= ")
	builder.WriteString(radiusArg)
	return builder.String(), args
}

const (
	queryFeedMomentTags = `
SELECT tag_name
FROM moment_tags
WHERE moment_id = $1
ORDER BY tag_name ASC`

	queryFeedMomentTagsByMomentIDs = `
SELECT moment_id, tag_name
FROM moment_tags
WHERE moment_id = ANY($1)
ORDER BY moment_id, tag_name ASC`

	queryFeedMomentsFollowing = `
SELECT m.moment_id,
       m.user_id,
       COALESCE(u.nickname, ''),
       COALESCE(u.avatar_url, ''),
       m.description,
       m.created_at,
       m.likes_count,
       m.comments_count,
       m.shares_count,
       m.location_name,
       m.latitude,
       m.longitude,
       EXISTS(SELECT 1 FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = m.moment_id AND r.user_id = $1),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $1)
FROM moments m
JOIN users u ON u.user_id = m.user_id
JOIN user_relations ur ON ur.target_user_id = m.user_id AND ur.user_id = $1 AND ur.is_following = TRUE
WHERE m.privacy_level = 'public'`

	queryFeedTrendingHotMoments = `
SELECT m.moment_id,
       m.user_id,
       COALESCE(u.nickname, ''),
       COALESCE(u.avatar_url, ''),
       m.description,
       m.created_at,
       m.likes_count,
       m.comments_count,
       m.shares_count,
       m.location_name,
       m.latitude,
       m.longitude,
       EXISTS(SELECT 1 FROM reactions r WHERE r.target_type = 'moment' AND r.target_id = m.moment_id AND r.user_id = $1),
       EXISTS(SELECT 1 FROM moment_favorites mf WHERE mf.moment_id = m.moment_id AND mf.user_id = $1)
FROM moments m
JOIN users u ON u.user_id = m.user_id
WHERE m.privacy_level = 'public'`
)
