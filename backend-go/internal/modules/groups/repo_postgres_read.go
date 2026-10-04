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
	"hotspot/backend-go/internal/platform/textutil"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) Get(ctx context.Context, groupID string) (GroupItem, bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
			distance, bearing, within := plan.Match(groupLocationPoint(*item.Location))
			if !within {
				continue
			}
			meta.Distance = distance
			distanceVal := float64(distance)
			bearingVal := float64(bearing)
			item.Location.Distance = &distanceVal
			item.Location.BearingDegrees = &bearingVal
		} else if req.SortBy == searchSortDistance {
			meta.Distance = geo.DistanceMeters(reference, groupLocationPoint(*item.Location))
			distanceVal := float64(meta.Distance)
			item.Location.Distance = &distanceVal
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
	var locationName, joinMethod, description string
	var latitude, longitude float64
	var createdAt, lastActivityAt *time.Time
	var tags []string
	if err := scanner.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.RequiresApproval, &item.NextActivityAt, &item.IsSearchable, &item.MembersSearchable, &item.MaxMembers, &item.OwnerID, &createdAt, &lastActivityAt, &tags, &item.OnlineMembersCount); err != nil {
		return GroupItem{}, err
	}
	location := RelativeLocation{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.Description = types.StrPtr(description)
	item.Location = &location
	item.JoinMethod = GroupJoinMethod(joinMethod)
	item.CreatedAt = formatOptionalTime(createdAt)
	item.LastActivityAt = formatOptionalTime(lastActivityAt)
	item.Tags = tags
	return item, nil
}

func scanSearchGroupRow(scanner interface{ Scan(...any) error }) (GroupItem, error) {
	var item GroupItem
	var locationName, joinMethod, description string
	var latitude, longitude float64
	var createdAt, lastActivityAt *time.Time
	var tags []string
	if err := scanner.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.RequiresApproval, &item.NextActivityAt, &item.IsSearchable, &item.MembersSearchable, &item.MaxMembers, &item.OwnerID, &createdAt, &lastActivityAt, &tags, &item.Group.MembersCount, &item.OnlineMembersCount); err != nil {
		return GroupItem{}, err
	}
	location := RelativeLocation{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.Description = types.StrPtr(description)
	item.Location = &location
	item.JoinMethod = GroupJoinMethod(joinMethod)
	item.CreatedAt = formatOptionalTime(createdAt)
	item.LastActivityAt = formatOptionalTime(lastActivityAt)
	item.Tags = tags
	return item, nil
}

func getGroupBase(ctx context.Context, queryable persistence.Queryable, groupID string) (GroupItem, error) {
	return scanGroupBase(queryable.QueryRow(ctx, queryGetGroup, groupID))
}

// loadGroupMembership 读取单个群的成员状态。注意 UnreadCount 恒为 0、
// IsPinned 恒为 false：spec 未定义未读数/置顶的存储与来源，属于已知限制，
// 勿将这两个字段当作真实数据使用。
func loadGroupMembership(ctx context.Context, queryable persistence.Queryable, groupID, userID string) (*GroupViewerState, error) {
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
func loadGroupMemberships(ctx context.Context, queryable persistence.Queryable, groupIDs []string, userID string) (map[string]*GroupViewerState, error) {
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

func countGroupMembers(ctx context.Context, queryable persistence.Queryable, groupID string) (int, error) {
	var count int
	if err := queryable.QueryRow(ctx, queryCountGroupMembers, groupID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func newRelativeLocationPtr(location Location) *RelativeLocation {
	value := newRelativeLocation(location)
	return &value
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

// groupSearchHardCap 是群搜索候选集的防御性硬上限：搜索的精确半径过滤、
// 排序与分页都在 Go 侧完成，SQL 层不排序不分页，若无上限，候选集会随群
// 规模无界增长（最坏退化一次 Query 全表载入内存）。此上限不是业务分页——
// 正常排序分页逻辑不变，仅防止极端规模下把可搜索全表拉进进程；根治需把
// 排序/裁剪下推 SQL（既定决策：等产品拍板排序口径后再实施，不在本处做）。
const groupSearchHardCap = 1000

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
	builder.WriteString("SELECT g.group_id, g.name, g.avatar_url, g.description, g.has_password, g.location_name, g.latitude, g.longitude, g.join_method, g.requires_approval, g.next_activity_at, g.is_searchable, g.members_searchable, g.max_members, g.owner_id, g.created_at, g.last_activity_at, COALESCE(g.tags, '{}')")
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
		// 转义 LIKE 通配符：与内存实现 strings.Contains 字面语义一致（同 moments 先例）。
		keywordArg := nextArg(textutil.EscapeLikePattern(req.Keyword))
		builder.WriteString(fmt.Sprintf(" AND (g.name ILIKE '%%' || %s || '%%' ESCAPE '\\' OR g.description ILIKE '%%' || %s || '%%' ESCAPE '\\')", keywordArg, keywordArg))
	}
	if req.Tag != "" {
		builder.WriteString(fmt.Sprintf(" AND %s = ANY(g.tags)", nextArg(req.Tag)))
	}
	// 防御性候选集硬上限（见 groupSearchHardCap），非业务分页。
	builder.WriteString(fmt.Sprintf(" LIMIT %d", groupSearchHardCap))
	return builder.String(), args
}
