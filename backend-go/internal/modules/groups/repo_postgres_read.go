package groups

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) Get(ctx context.Context, groupID string) (GroupItem, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return GroupItem{}, false, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return GroupItem{}, false, persistence.ErrNotImplemented
	}
	item, err := getGroupBase(ctx, queryable, groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GroupItem{}, false, nil
		}
		return GroupItem{}, false, err
	}
	principal, _ := platformauth.PrincipalFromContext(ctx)
	myState, err := loadGroupMembership(ctx, queryable, groupID, principal.UserID)
	if err != nil {
		return GroupItem{}, false, err
	}
	item.MyState = myState
	item.Group.MembersCount, err = countGroupMembers(ctx, queryable, groupID)
	if err != nil {
		return GroupItem{}, false, err
	}
	return item, true, nil
}

func (r *PostgresRepository) Search(ctx context.Context, req SearchGroupsRequest) (PaginatedGroups, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedGroups{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedGroups{}, persistence.ErrNotImplemented
	}
	req = normalizedSearchRequest(req)
	query, args, limit, cursor, err := buildGroupSearchQuery(req)
	if err != nil {
		return PaginatedGroups{}, err
	}
	rows, err := queryable.Query(ctx, query, args...)
	if err != nil {
		return PaginatedGroups{}, err
	}
	defer rows.Close()
	principal, _ := platformauth.PrincipalFromContext(ctx)
	items := make([]GroupItem, 0)
	for rows.Next() {
		item, meta, err := scanSearchGroupRow(rows)
		if err != nil {
			return PaginatedGroups{}, err
		}
		if cursor.GroupID != "" && !matchesSearchCursor(meta, item, cursor) {
			continue
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return PaginatedGroups{}, err
	}
	groupIDs := make([]string, len(items))
	for i, item := range items {
		groupIDs[i] = item.Group.GroupID
	}
	memberships, err := loadGroupMemberships(ctx, queryable, groupIDs, principal.UserID)
	if err != nil {
		return PaginatedGroups{}, err
	}
	for i := range items {
		items[i].MyState = memberships[items[i].Group.GroupID]
	}
	response := PaginatedGroups{}
	if len(items) <= limit {
		response.Items = items
		return response, nil
	}
	response.Items = items[:limit]
	last := response.Items[len(response.Items)-1]
	lastDistance := 0
	if last.Location.Distance != nil {
		lastDistance = int(*last.Location.Distance)
	}
	lastCreatedAt := ""
	if last.CreatedAt != nil {
		lastCreatedAt = *last.CreatedAt
	}
	next, err := encodeSearchCursor(groupSearchCursor{SortBy: req.SortBy, GroupID: last.Group.GroupID, Distance: lastDistance, MembersCount: last.Group.MembersCount, NameMatch: searchNameMatchScore(last, req.Keyword), CreatedAt: lastCreatedAt})
	if err != nil {
		return PaginatedGroups{}, err
	}
	response.NextCursor = next
	return response, nil
}

func scanGroupBase(scanner interface{ Scan(...any) error }) (GroupItem, error) {
	var item GroupItem
	var locationName, joinMethod string
	var latitude, longitude float64
	var createdAt, lastActivityAt *time.Time
	var tags []string
	if err := scanner.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers, &item.OwnerID, &createdAt, &lastActivityAt, &tags); err != nil {
		return GroupItem{}, err
	}
	item.Location = RelativeLocation{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.JoinMethod = GroupJoinMethod(joinMethod)
	item.CreatedAt = formatOptionalTime(createdAt)
	item.LastActivityAt = formatOptionalTime(lastActivityAt)
	item.Tags = tags
	return item, nil
}

func scanSearchGroupRow(scanner interface{ Scan(...any) error }) (GroupItem, groupSearchMeta, error) {
	var item GroupItem
	var meta groupSearchMeta
	var locationName, joinMethod string
	var latitude, longitude float64
	var createdAt, lastActivityAt *time.Time
	var tags []string
	var distanceMeters, nameMatchScore int
	if err := scanner.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers, &item.OwnerID, &createdAt, &lastActivityAt, &tags, &item.Group.MembersCount, &distanceMeters, &nameMatchScore); err != nil {
		return GroupItem{}, groupSearchMeta{}, err
	}
	item.Location = RelativeLocation{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	distance := float64(distanceMeters)
	item.Location.Distance = &distance
	item.JoinMethod = GroupJoinMethod(joinMethod)
	item.CreatedAt = formatOptionalTime(createdAt)
	item.LastActivityAt = formatOptionalTime(lastActivityAt)
	item.Tags = tags
	meta.Distance = distanceMeters
	meta.NameMatch = nameMatchScore
	meta.CreatedAt = groupCreatedAt(item)
	return item, meta, nil
}

func getGroupBase(ctx context.Context, queryable persistence.QueryableDB, groupID string) (GroupItem, error) {
	return scanGroupBase(queryable.QueryRow(ctx, queryGetGroup, groupID))
}

func loadGroupMembership(ctx context.Context, queryable persistence.QueryableDB, groupID, userID string) (*MyGroupState, error) {
	if userID == "" {
		return nil, nil
	}
	var role string
	var joinedAt time.Time
	var isMuted bool
	var mutedUntil *time.Time
	if err := queryable.QueryRow(ctx, queryGetGroupMembership, groupID, userID).Scan(&role, &joinedAt, &isMuted, &mutedUntil); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &MyGroupState{Role: GroupRole(role), IsJoined: true, UnreadCount: 0, IsMuted: isMuted, MutedUntil: formatMutedUntil(mutedUntil), IsPinned: false, JoinedAt: joinedAt.UTC().Format(time.RFC3339)}, nil
}

func loadGroupMemberships(ctx context.Context, queryable persistence.QueryableDB, groupIDs []string, userID string) (map[string]*MyGroupState, error) {
	result := make(map[string]*MyGroupState, len(groupIDs))
	if userID == "" || len(groupIDs) == 0 {
		return result, nil
	}
	rows, err := queryable.Query(ctx, queryGetGroupMemberships, userID, groupIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var groupID, role string
		var joinedAt time.Time
		var isMuted bool
		var mutedUntil *time.Time
		if err := rows.Scan(&groupID, &role, &joinedAt, &isMuted, &mutedUntil); err != nil {
			return nil, err
		}
		result[groupID] = &MyGroupState{Role: GroupRole(role), IsJoined: true, UnreadCount: 0, IsMuted: isMuted, MutedUntil: formatMutedUntil(mutedUntil), IsPinned: false, JoinedAt: joinedAt.UTC().Format(time.RFC3339)}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func formatMutedUntil(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func countGroupMembers(ctx context.Context, queryable persistence.QueryableDB, groupID string) (int, error) {
	var count int
	if err := queryable.QueryRow(ctx, queryCountGroupMembers, groupID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func newRelativeLocation(location Location) RelativeLocation {
	return RelativeLocation{
		Point: struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}{
			Latitude:  location.Point.Latitude,
			Longitude: location.Point.Longitude,
		},
		LocationName: location.LocationName,
	}
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func buildGroupSearchQuery(req SearchGroupsRequest) (string, []any, int, groupSearchCursor, error) {
	limit := normalizeSearchLimit(req.Limit)
	cursor, err := decodeSearchCursor(req.Cursor)
	if err != nil {
		return "", nil, 0, groupSearchCursor{}, err
	}
	if cursor.SortBy == "" {
		cursor.SortBy = req.SortBy
	}
	args := make([]any, 0, 32)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	reference := searchGeoPoint(req.ReferencePoint)
	refLat := addArg(reference.Latitude)
	refLng := addArg(reference.Longitude)
	distanceExpr := geo.HaversineDistanceSQL(refLat, refLng)
	nameMatchExpr := "0"
	if req.Keyword != "" {
		keywordArg := addArg(req.Keyword)
		nameMatchExpr = fmt.Sprintf(`CASE WHEN name ILIKE '%%' || %s || '%%' THEN 1 ELSE 0 END`, keywordArg)
	}
	var builder strings.Builder
	builder.WriteString("SELECT * FROM (")
	builder.WriteString("SELECT g.group_id, g.name, g.avatar_url, g.description, g.has_password, g.location_name, g.latitude, g.longitude, g.join_method, g.is_searchable, g.max_members, g.owner_id, g.created_at, g.last_activity_at, COALESCE(g.tags, '{}')")
	builder.WriteString(", (SELECT COUNT(*)::int FROM group_members gm WHERE gm.group_id = g.group_id) AS members_count")
	builder.WriteString(", ")
	builder.WriteString(distanceExpr)
	builder.WriteString(" AS distance_meters, ")
	builder.WriteString(nameMatchExpr)
	builder.WriteString(" AS name_match_score")
	builder.WriteString(" FROM groups g WHERE g.deleted_at IS NULL AND g.is_searchable = true")
	if req.Keyword != "" {
		keywordArg := addArg(req.Keyword)
		builder.WriteString(fmt.Sprintf(" AND (g.name ILIKE '%%' || %s || '%%' OR g.description ILIKE '%%' || %s || '%%')", keywordArg, keywordArg))
	}
	if req.Tag != "" {
		tagArg := addArg(req.Tag)
		builder.WriteString(fmt.Sprintf(" AND %s = ANY(g.tags)", tagArg))
	}
	if req.Radius != nil {
		bounds := geo.BoundsAround(reference, *req.Radius)
		mins, maxs := geo.CoveringArrays(bounds)
		if len(mins) > 0 {
			minArg := addArg(mins)
			maxArg := addArg(maxs)
			builder.WriteString(fmt.Sprintf(" AND EXISTS (SELECT 1 FROM unnest(%s::bigint[], %s::bigint[]) AS t(lo, hi) WHERE g.s2_cell_id BETWEEN t.lo AND t.hi)", minArg, maxArg))
		}
	}
	builder.WriteString(") search WHERE 1=1")
	if req.Radius != nil {
		radiusArg := addArg(*req.Radius)
		builder.WriteString(fmt.Sprintf(" AND distance_meters <= %s", radiusArg))
	}
	if cursor.GroupID != "" {
		switch req.SortBy {
		case searchSortDistance:
			distanceArg := addArg(cursor.Distance)
			groupIDArg := addArg(cursor.GroupID)
			builder.WriteString(fmt.Sprintf(" AND (distance_meters > %s OR (distance_meters = %s AND group_id > %s))", distanceArg, distanceArg, groupIDArg))
		case searchSortNewest:
			createdAtArg := addArg(cursor.CreatedAt)
			groupIDArg := addArg(cursor.GroupID)
			builder.WriteString(fmt.Sprintf(" AND (created_at < %s::timestamptz OR (created_at = %s::timestamptz AND group_id > %s))", createdAtArg, createdAtArg, groupIDArg))
		case searchSortPopular:
			membersArg := addArg(cursor.MembersCount)
			groupIDArg := addArg(cursor.GroupID)
			builder.WriteString(fmt.Sprintf(" AND (members_count < %s OR (members_count = %s AND group_id > %s))", membersArg, membersArg, groupIDArg))
		default:
			nameMatchArg := addArg(cursor.NameMatch)
			membersArg := addArg(cursor.MembersCount)
			groupIDArg := addArg(cursor.GroupID)
			builder.WriteString(fmt.Sprintf(" AND (name_match_score < %s OR (name_match_score = %s AND (members_count < %s OR (members_count = %s AND group_id > %s))))", nameMatchArg, nameMatchArg, membersArg, membersArg, groupIDArg))
		}
	}
	switch req.SortBy {
	case searchSortDistance:
		builder.WriteString(" ORDER BY distance_meters ASC, group_id ASC")
	case searchSortNewest:
		builder.WriteString(" ORDER BY created_at DESC, group_id ASC")
	case searchSortPopular:
		builder.WriteString(" ORDER BY members_count DESC, group_id ASC")
	default:
		builder.WriteString(" ORDER BY name_match_score DESC, members_count DESC, group_id ASC")
	}
	limitArg := addArg(limit + 1)
	builder.WriteString(fmt.Sprintf(" LIMIT %s", limitArg))
	return builder.String(), args, limit, cursor, nil
}
