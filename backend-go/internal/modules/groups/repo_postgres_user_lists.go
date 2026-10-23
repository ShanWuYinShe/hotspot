package groups

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
)

// groupsByUserCursorCodec 承载「按用户聚合的群组列表」keyset 游标（joined_at DESC,
// group_id ASC），编解码仅存在于 Repository 层（AGENTS.md 分页规范）。
var groupsByUserCursorCodec = cursor.NewCodec("groups-by-user")

type groupsByUserCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

func encodeGroupsByUserCursor(joinedAt time.Time, groupID string) string {
	token, err := groupsByUserCursorCodec.Encode(groupsByUserCursor{At: joinedAt.UTC().Format(time.RFC3339Nano), ID: groupID})
	if err != nil {
		return ""
	}
	return token
}

func decodeGroupsByUserCursor(raw string) (time.Time, string, error) {
	if raw == "" {
		return time.Time{}, "", nil
	}
	var decoded groupsByUserCursor
	if err := groupsByUserCursorCodec.Decode(raw, &decoded); err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	if decoded.At == "" || decoded.ID == "" {
		return time.Time{}, "", ErrInvalidCursor
	}
	parsed, err := time.Parse(time.RFC3339Nano, decoded.At)
	if err != nil {
		return time.Time{}, "", ErrInvalidCursor
	}
	return parsed.UTC(), decoded.ID, nil
}

// ListGroupsByUser returns a user's visible groups (searchable, not deleted)
// in join-order desc, keyset-paginated. Consumed by the users module via the
// Service interface for the profile aggregation endpoints.
func (r *PostgresRepository) ListGroupsByUser(ctx context.Context, userID string, limit int, cursor string) (PaginatedGroups, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return PaginatedGroups{}, persistence.ErrNotImplemented
	}
	limit = types.NormalizeLimit(limit)
	cursorJoinedAt, cursorGroupID, err := decodeGroupsByUserCursor(cursor)
	if err != nil {
		return PaginatedGroups{}, ErrInvalidCursor
	}
	var cursorJoinedAtValue any
	if !cursorJoinedAt.IsZero() {
		cursorJoinedAtValue = cursorJoinedAt
	}
	rows, err := queryable.Query(ctx, queryListGroupsByUser, userID, cursorJoinedAtValue, cursorGroupID, limit+1)
	if err != nil {
		return PaginatedGroups{}, err
	}
	defer rows.Close()
	items := make([]GroupItem, 0, limit+1)
	joinedAts := make([]time.Time, 0, limit+1)
	for rows.Next() {
		item, joinedAt, err := scanGroupByUserRow(rows)
		if err != nil {
			return PaginatedGroups{}, err
		}
		items = append(items, item)
		joinedAts = append(joinedAts, joinedAt)
	}
	if err := rows.Err(); err != nil {
		return PaginatedGroups{}, err
	}
	if len(items) <= limit {
		return PaginatedGroups{Items: items}, nil
	}
	next := encodeGroupsByUserCursor(joinedAts[limit-1], items[limit-1].Group.GroupID)
	return PaginatedGroups{Items: items[:limit], NextCursor: &next}, nil
}

func scanGroupByUserRow(scanner interface{ Scan(...any) error }) (GroupItem, time.Time, error) {
	var item GroupItem
	var locationName, joinMethod, description string
	var latitude, longitude float64
	var membersCount int
	var joinedAt time.Time
	if err := scanner.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers, &membersCount, &joinedAt); err != nil {
		return GroupItem{}, time.Time{}, err
	}
	location := Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.Description = types.StrPtr(description)
	item.Location = &location
	item.JoinMethod = GroupJoinMethod(joinMethod)
	// SQL 已带回真实成员计数（审查 P1-5）：此前硬编码 1，users/fetch_groups
	// 对外 members_count 恒为占位错误值。
	item.Group.MembersCount = membersCount
	return item, joinedAt.UTC(), nil
}

// CountCommonGroups counts the groups shared by two users (both not deleted).
func (r *PostgresRepository) CountCommonGroups(ctx context.Context, viewerUserID, targetUserID string) (int, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var count int
	if err := queryable.QueryRow(ctx, queryCountCommonGroups, viewerUserID, targetUserID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// CountOwnedGroups counts non-dismissed groups where the user holds the owner role.
func (r *PostgresRepository) CountOwnedGroups(ctx context.Context, userID string) (int, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var count int
	if err := queryable.QueryRow(ctx, queryCountOwnedGroups, userID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
