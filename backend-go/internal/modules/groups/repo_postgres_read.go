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
	item.ViewerState = myState
	item.Group.MembersCount, err = countGroupMembers(ctx, queryable, groupID)
	if err != nil {
		return GroupItem{}, false, err
	}
	return item, true, nil
}

// Search resolves candidates with a static SQL query (S2 cell prefiltering
// plus coarse keyword/tag/radius-bounds filters), then performs the exact
// radius check, distance computation, sorting and pagination in Go. The SQL
// layer no longer computes haversine distances, relevance scores or keyset
// cursors — that logic lives in the Go search pipeline shared with the
// in-memory repository.
func (r *PostgresRepository) Search(ctx context.Context, req SearchGroupsRequest) (PaginatedGroups, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedGroups{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedGroups{}, persistence.ErrNotImplemented
	}
	req = normalizedSearchRequest(req)
	query, args := buildGroupSearchQuery(req)
	rows, err := queryable.Query(ctx, query, args...)
	if err != nil {
		return PaginatedGroups{}, err
	}
	defer rows.Close()
	principal, _ := platformauth.PrincipalFromContext(ctx)
	reference := searchGeoPoint(req.ReferencePoint)
	var (
		plan    geo.SearchPlan
		hasPlan bool
	)
	if req.Radius != nil {
		plan = geo.NewSearchPlan(reference, *req.Radius, req.SortBy)
		hasPlan = true
	}
	items := make([]GroupItem, 0)
	metas := make(map[string]groupSearchMeta)
	for rows.Next() {
		item, err := scanSearchGroupRow(rows)
		if err != nil {
			return PaginatedGroups{}, err
		}
		meta := groupSearchMeta{CreatedAt: groupCreatedAt(item), NameMatch: searchNameMatchScore(item, req.Keyword)}
		if hasPlan {
			distance, bearing, within := plan.Match(groupLocationPoint(item.Location))
			if !within {
				continue
			}
			meta.Distance = distance
			meta.Bearing = bearing
			value := float64(distance)
			item.Location.Distance = &value
			value = float64(bearing)
			item.Location.BearingDegrees = &value
		} else if req.SortBy == searchSortDistance {
			meta.Distance = geo.DistanceMeters(reference, groupLocationPoint(item.Location))
			value := float64(meta.Distance)
			item.Location.Distance = &value
		}
		items = append(items, item)
		metas[item.Group.GroupID] = meta
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
		items[i].ViewerState = memberships[items[i].Group.GroupID]
	}
	sortGroupsForSearch(items, metas, req.SortBy)
	return paginateSearchGroups(items, metas, req.SortBy, req.Pagination)
}

func scanGroupBase(scanner interface{ Scan(...any) error }) (GroupItem, error) {
	var item GroupItem
	var locationName, joinMethod string
	var latitude, longitude float64
	var createdAt, lastActivityAt *time.Time
	var tags []string
	if err := scanner.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.RequiresApproval, &item.NextActivityAt, &item.IsSearchable, &item.MembersSearchable, &item.MaxMembers, &item.OwnerID, &createdAt, &lastActivityAt, &tags, &item.OnlineMembersCount); err != nil {
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

func scanSearchGroupRow(scanner interface{ Scan(...any) error }) (GroupItem, error) {
	var item GroupItem
	var locationName, joinMethod string
	var latitude, longitude float64
	var createdAt, lastActivityAt *time.Time
	var tags []string
	if err := scanner.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.RequiresApproval, &item.NextActivityAt, &item.IsSearchable, &item.MembersSearchable, &item.MaxMembers, &item.OwnerID, &createdAt, &lastActivityAt, &tags, &item.Group.MembersCount, &item.OnlineMembersCount); err != nil {
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

func getGroupBase(ctx context.Context, queryable persistence.QueryableDB, groupID string) (GroupItem, error) {
	return scanGroupBase(queryable.QueryRow(ctx, queryGetGroup, groupID))
}

// loadGroupMembership 读取单个群的成员状态。注意 UnreadCount 恒为 0、
// IsPinned 恒为 false：spec 未定义未读数/置顶的存储与来源，属于已知限制，
// 勿将这两个字段当作真实数据使用。
func loadGroupMembership(ctx context.Context, queryable persistence.QueryableDB, groupID, userID string) (*GroupViewerState, error) {
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
	return &GroupViewerState{Role: GroupRole(role), UnreadCount: 0, IsMuted: isMuted, MutedUntil: formatMutedUntil(mutedUntil), IsPinned: false, JoinedAt: joinedAt.UTC().Format(time.RFC3339)}, nil
}

// loadGroupMemberships 批量读取成员状态（同 loadGroupMembership，未读数/置顶
// 为已知限制，恒为 0/false）。
func loadGroupMemberships(ctx context.Context, queryable persistence.QueryableDB, groupIDs []string, userID string) (map[string]*GroupViewerState, error) {
	result := make(map[string]*GroupViewerState, len(groupIDs))
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
		result[groupID] = &GroupViewerState{Role: GroupRole(role), UnreadCount: 0, IsMuted: isMuted, MutedUntil: formatMutedUntil(mutedUntil), IsPinned: false, JoinedAt: joinedAt.UTC().Format(time.RFC3339)}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func formatMutedUntil(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

// groupMutedUntil converts an in-memory muted_until string storage to a
// truthy nullable pointer ("" → nil, meaning not muted).
func groupMutedUntil(value string) *string {
	if value == "" {
		return nil
	}
	v := value
	return &v
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

// stringValue 安全解引用可空字符串(*nil → "")。
func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// buildGroupSearchQuery assembles the static search SQL. S2 cell prefiltering
// for the radius bounds (EXISTS unnest) and the coarse keyword/tag filters stay
// in SQL; exact radius filtering, sorting and pagination are applied in Go on
// the returned rows. The query carries no haversine distance column, no
// relevance CASE expression, no keyset cursor predicates and no ORDER BY.
func buildGroupSearchQuery(req SearchGroupsRequest) (string, []any) {
	var builder strings.Builder
	args := make([]any, 0, 4)
	nextArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}
	builder.WriteString("SELECT g.group_id, g.name, g.avatar_url, g.description, g.has_password, g.location_name, g.latitude, g.longitude, g.join_method, g.requires_approval, g.next_activity_at, g.is_searchable, g.max_members, g.owner_id, g.created_at, g.last_activity_at, COALESCE(g.tags, '{}')")
	builder.WriteString(", (SELECT COUNT(*)::int FROM group_members gm WHERE gm.group_id = g.group_id) AS members_count")
	builder.WriteString(", (SELECT COUNT(*)::int FROM group_members gm JOIN users u ON u.user_id = gm.user_id WHERE gm.group_id = g.group_id AND u.presence_status = 'online') AS online_members_count")
	builder.WriteString(" FROM groups g WHERE g.deleted_at IS NULL AND g.is_searchable = true")
	if req.Radius != nil {
		bounds := geo.BoundsAround(searchGeoPoint(req.ReferencePoint), *req.Radius)
		mins, maxs := geo.CoveringArrays(bounds)
		if len(mins) > 0 {
			builder.WriteString(fmt.Sprintf(" AND EXISTS (SELECT 1 FROM unnest(%s::bigint[], %s::bigint[]) AS t(lo, hi) WHERE g.s2_cell_id BETWEEN t.lo AND t.hi)", nextArg(mins), nextArg(maxs)))
		}
	}
	if req.Keyword != "" {
		keywordArg := nextArg(req.Keyword)
		builder.WriteString(fmt.Sprintf(" AND (g.name ILIKE '%%' || %s || '%%' OR g.description ILIKE '%%' || %s || '%%')", keywordArg, keywordArg))
	}
	if req.Tag != "" {
		builder.WriteString(fmt.Sprintf(" AND %s = ANY(g.tags)", nextArg(req.Tag)))
	}
	return builder.String(), args
}
