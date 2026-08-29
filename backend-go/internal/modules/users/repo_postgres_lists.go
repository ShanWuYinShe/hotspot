package users

import (
	"context"
	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
	"time"
)

func listFriendsPostgres(ctx context.Context, factory *persistence.Factory, userID string, limit int, cursor string) (PaginatedFriends, error) {
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedFriends{}, persistence.ErrNotImplemented
	}
	limit = normalizeUserMomentLimit(limit)
	cursorAt, cursorID, err := decodeUserCursor(cursor)
	if err != nil {
		return PaginatedFriends{}, ErrInvalidCursor
	}
	rows, err := queryable.Query(ctx, queryListFriends, userID, cursorAt, cursorID, limit+1)
	if err != nil {
		return PaginatedFriends{}, err
	}
	defer rows.Close()
	items := make([]RelatedUserItem, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, sortAt, err := scanFriendRow(rows)
		if err != nil {
			return PaginatedFriends{}, err
		}
		items = append(items, item)
		cursors = append(cursors, encodeUserCursor(sortAt, item.User.User.UserID))
		if len(items) == limit+1 {
			next := cursors[limit-1]
			return PaginatedFriends{Items: items[:limit], NextCursor: &next}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return PaginatedFriends{}, err
	}
	return PaginatedFriends{Items: items}, nil
}

func listFollowingPostgres(ctx context.Context, factory *persistence.Factory, userID string, limit int, cursor string) (PaginatedFollowing, error) {
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedFollowing{}, persistence.ErrNotImplemented
	}
	limit = normalizeUserMomentLimit(limit)
	cursorAt, cursorID, err := decodeUserCursor(cursor)
	if err != nil {
		return PaginatedFollowing{}, ErrInvalidCursor
	}
	rows, err := queryable.Query(ctx, queryListFollowing, userID, cursorAt, cursorID, limit+1)
	if err != nil {
		return PaginatedFollowing{}, err
	}
	defer rows.Close()
	items := make([]RelatedUserItem, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, sortAt, err := scanFollowingRow(rows)
		if err != nil {
			return PaginatedFollowing{}, err
		}
		items = append(items, item)
		cursors = append(cursors, encodeUserCursor(sortAt, item.User.User.UserID))
		if len(items) == limit+1 {
			next := cursors[limit-1]
			return PaginatedFollowing{Items: items[:limit], NextCursor: &next}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return PaginatedFollowing{}, err
	}
	return PaginatedFollowing{Items: items}, nil
}

func listFollowersPostgres(ctx context.Context, factory *persistence.Factory, userID string, limit int, cursor string) (PaginatedFollowers, error) {
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedFollowers{}, persistence.ErrNotImplemented
	}
	limit = normalizeUserMomentLimit(limit)
	cursorAt, cursorID, err := decodeUserCursor(cursor)
	if err != nil {
		return PaginatedFollowers{}, ErrInvalidCursor
	}
	rows, err := queryable.Query(ctx, queryListFollowers, userID, cursorAt, cursorID, limit+1)
	if err != nil {
		return PaginatedFollowers{}, err
	}
	defer rows.Close()
	items := make([]RelatedUserItem, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, sortAt, err := scanFollowerRow(rows)
		if err != nil {
			return PaginatedFollowers{}, err
		}
		items = append(items, item)
		cursors = append(cursors, encodeUserCursor(sortAt, item.User.User.UserID))
		if len(items) == limit+1 {
			next := cursors[limit-1]
			return PaginatedFollowers{Items: items[:limit], NextCursor: &next}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return PaginatedFollowers{}, err
	}
	return PaginatedFollowers{Items: items}, nil
}

func listUserGroupsPostgres(ctx context.Context, factory *persistence.Factory, userID string, limit int, cursor string) (grouptypes.PaginatedGroups, error) {
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		return grouptypes.PaginatedGroups{}, persistence.ErrNotImplemented
	}
	limit = normalizeUserMomentLimit(limit)
	cursorAt, cursorID, err := decodeUserCursor(cursor)
	if err != nil {
		return grouptypes.PaginatedGroups{}, ErrInvalidCursor
	}
	rows, err := queryable.Query(ctx, queryListUserGroups, userID, cursorAt, cursorID, limit+1)
	if err != nil {
		return grouptypes.PaginatedGroups{}, err
	}
	defer rows.Close()
	items := make([]grouptypes.GroupItem, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, joinedAt, err := scanUserGroupRow(rows)
		if err != nil {
			return grouptypes.PaginatedGroups{}, err
		}
		items = append(items, item)
		cursors = append(cursors, encodeUserCursor(joinedAt, item.Group.GroupID))
		if len(items) == limit+1 {
			next := cursors[limit-1]
			return grouptypes.PaginatedGroups{Items: items[:limit], NextCursor: &next}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return grouptypes.PaginatedGroups{}, err
	}
	return grouptypes.PaginatedGroups{Items: items}, nil
}

func listUserMomentsPostgres(ctx context.Context, factory *persistence.Factory, userID string, limit int, cursor string, includePrivate bool) (momenttypes.PaginatedMoments, error) {
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		return momenttypes.PaginatedMoments{}, persistence.ErrNotImplemented
	}
	limit = normalizeUserMomentLimit(limit)
	cursorAt, cursorID, err := decodeUserMomentCursor(cursor)
	if err != nil {
		return momenttypes.PaginatedMoments{}, ErrInvalidCursor
	}
	rows, err := queryable.Query(ctx, queryListUserMoments, userID, includePrivate, cursorAt, cursorID, limit+1)
	if err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]momenttypes.MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, err := scanMomentItem(rows)
		if err != nil {
			return momenttypes.PaginatedMoments{}, err
		}
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		parsed, _ := time.Parse(time.RFC3339Nano, item.CreatedAt)
		cursors = append(cursors, encodeUserMomentCursor(parsed.UTC().Format(time.RFC3339), item.MomentID))
		if len(items) == limit+1 {
			tagsByMomentID, err := loadUserMomentTags(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return momenttypes.PaginatedMoments{}, err
			}
			for i := 0; i < limit; i++ {
				items[i].Tags = tagsByMomentID[items[i].MomentID]
			}
			c := cursors[limit-1]
			return momenttypes.PaginatedMoments{Items: items[:limit], NextCursor: &c}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	tagsByMomentID, err := loadUserMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
	}
	return momenttypes.PaginatedMoments{Items: items}, nil
}

func listReactedMomentsPostgres(ctx context.Context, factory *persistence.Factory, userID string, limit int, cursor string, viewerUserID string) (momenttypes.PaginatedMoments, error) {
	queryable, ok := persistence.RepoQueryable(ctx, factory)
	if !ok {
		return momenttypes.PaginatedMoments{}, persistence.ErrNotImplemented
	}
	limit = normalizeUserMomentLimit(limit)
	cursorReactedAt, cursorMomentID, err := decodeUserMomentCursor(cursor)
	if err != nil {
		return momenttypes.PaginatedMoments{}, ErrInvalidCursor
	}
	rows, err := queryable.Query(ctx, queryListReactedMoments, userID, viewerUserID, cursorReactedAt, cursorMomentID, limit+1)
	if err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	defer rows.Close()
	items := make([]momenttypes.MomentItem, 0, limit+1)
	momentIDs := make([]string, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, reactedAt, err := scanUserReactedMomentRow(rows)
		if err != nil {
			return momenttypes.PaginatedMoments{}, err
		}
		// 已反应列表：对查看者本人而言必然已收藏（spec api_spec_moment.md → moments/list_reacted）
		item.IsFavorited = true
		item.FavoritedAt = reactedAt
		items = append(items, item)
		momentIDs = append(momentIDs, item.MomentID)
		cursors = append(cursors, encodeUserMomentCursor(reactedAt, item.MomentID))
		if len(items) == limit+1 {
			tagsByMomentID, err := loadUserMomentTags(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return momenttypes.PaginatedMoments{}, err
			}
			for i := 0; i < limit; i++ {
				items[i].Tags = tagsByMomentID[items[i].MomentID]
			}
			c := cursors[limit-1]
			return momenttypes.PaginatedMoments{Items: items[:limit], NextCursor: &c}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	tagsByMomentID, err := loadUserMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return momenttypes.PaginatedMoments{}, err
	}
	for i := range items {
		items[i].Tags = tagsByMomentID[items[i].MomentID]
	}
	return momenttypes.PaginatedMoments{Items: items}, nil
}

func scanFriendRow(scanner interface{ Scan(...any) error }) (RelatedUserItem, time.Time, error) {
	var item RelatedUserItem
	var avatarURL string
	var presenceStatus string
	var lastOnline time.Time
	var sortableLastOnline time.Time
	var followedAt, followedBackAt time.Time
	var remark string
	if err := scanner.Scan(&item.User.User.UserID, &item.User.User.Nickname, &avatarURL, &presenceStatus, &lastOnline, &sortableLastOnline, &remark, &followedAt, &followedBackAt); err != nil {
		return RelatedUserItem{}, time.Time{}, err
	}
	item.User.User.AvatarURL = types.StrPtr(avatarURL)
	ps := UserPresenceStatus(presenceStatus)
	item.User.PresenceStatus = &ps
	item.Remark = &remark
	lastOnlineStr := lastOnline.UTC().Format(time.RFC3339)
	item.User.LastOnline = &lastOnlineStr
	item.FollowedAt = followedAt.UTC().Format(time.RFC3339)
	followedBackAtStr := followedBackAt.UTC().Format(time.RFC3339)
	item.FollowedBackAt = &followedBackAtStr
	return item, followedBackAt.UTC(), nil
}

func scanFollowingRow(scanner interface{ Scan(...any) error }) (RelatedUserItem, time.Time, error) {
	var item RelatedUserItem
	var avatarURL string
	var followedAt time.Time
	var presenceStatus string
	var lastOnline time.Time
	var sortableLastOnline time.Time
	if err := scanner.Scan(&item.User.User.UserID, &item.User.User.Nickname, &avatarURL, &followedAt, &presenceStatus, &lastOnline, &sortableLastOnline); err != nil {
		return RelatedUserItem{}, time.Time{}, err
	}
	item.User.User.AvatarURL = types.StrPtr(avatarURL)
	item.FollowedAt = followedAt.UTC().Format(time.RFC3339)
	ps := UserPresenceStatus(presenceStatus)
	item.User.PresenceStatus = &ps
	lastOnlineStr := lastOnline.UTC().Format(time.RFC3339)
	item.User.LastOnline = &lastOnlineStr
	return item, followedAt.UTC(), nil
}

func scanFollowerRow(scanner interface{ Scan(...any) error }) (RelatedUserItem, time.Time, error) {
	var item RelatedUserItem
	var avatarURL string
	var followedAt time.Time
	var presenceStatus string
	var lastOnline time.Time
	var sortableLastOnline time.Time
	if err := scanner.Scan(&item.User.User.UserID, &item.User.User.Nickname, &avatarURL, &followedAt, &presenceStatus, &lastOnline, &sortableLastOnline); err != nil {
		return RelatedUserItem{}, time.Time{}, err
	}
	item.User.User.AvatarURL = types.StrPtr(avatarURL)
	item.FollowedAt = followedAt.UTC().Format(time.RFC3339)
	ps := UserPresenceStatus(presenceStatus)
	item.User.PresenceStatus = &ps
	lastOnlineStr := lastOnline.UTC().Format(time.RFC3339)
	item.User.LastOnline = &lastOnlineStr
	return item, followedAt.UTC(), nil
}

func scanUserGroupRow(scanner interface{ Scan(...any) error }) (grouptypes.GroupItem, time.Time, error) {
	var item grouptypes.GroupItem
	var locationName, joinMethod, description string
	var latitude, longitude float64
	var joinedAt time.Time
	if err := scanner.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers, &joinedAt); err != nil {
		return grouptypes.GroupItem{}, time.Time{}, err
	}
	location := grouptypes.Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.Description = types.StrPtr(description)
	item.Location = &location
	item.JoinMethod = grouptypes.GroupJoinMethod(joinMethod)
	item.Group.MembersCount = 1
	return item, joinedAt.UTC(), nil
}

func scanMomentItem(scanner interface{ Scan(...any) error }) (momenttypes.MomentItem, error) {
	var item momenttypes.MomentItem
	var userID, nickname, locationName, privacyLevel string
	var latitude, longitude float64
	var createdAt time.Time
	if err := scanner.Scan(&item.MomentID, &userID, &nickname, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel); err != nil {
		return momenttypes.MomentItem{}, err
	}
	if nickname == "" {
		nickname = userID
	}
	item.User = momenttypes.UserInfo{UserID: userID, Nickname: nickname}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.Location = &momenttypes.Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.PrivacyLevel = privacyLevel
	return item, nil
}

func scanUserReactedMomentRow(scanner interface{ Scan(...any) error }) (momenttypes.MomentItem, string, error) {
	var item momenttypes.MomentItem
	var userID, nickname, locationName, privacyLevel string
	var latitude, longitude float64
	var createdAt, reactedAt time.Time
	if err := scanner.Scan(&item.MomentID, &userID, &nickname, &item.Description, &createdAt, &item.LikesCount, &item.CommentsCount, &item.SharesCount, &locationName, &latitude, &longitude, &privacyLevel, &reactedAt); err != nil {
		return momenttypes.MomentItem{}, "", err
	}
	if nickname == "" {
		nickname = userID
	}
	item.User = momenttypes.UserInfo{UserID: userID, Nickname: nickname}
	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.Location = &momenttypes.Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	item.PrivacyLevel = privacyLevel
	return item, reactedAt.UTC().Format(time.RFC3339), nil
}

type userMomentCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
}

func normalizeUserMomentLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > types.MaxPageSize:
		return 100
	default:
		return limit
	}
}

// usersCursorCodec caches the AES-GCM cipher used for user moment and
// search cursors, keyed with a namespace salt to isolate cursors across
// modules.
var usersCursorCodec = cursor.NewCodec("users")

func encodeUserMomentCursor(at, id string) string {
	token, err := usersCursorCodec.Encode(userMomentCursor{At: at, ID: id})
	if err != nil {
		return ""
	}
	return token
}

func decodeUserMomentCursor(raw string) (any, string, error) {
	if raw == "" {
		return nil, "", nil
	}
	var decoded userMomentCursor
	if err := usersCursorCodec.Decode(raw, &decoded); err != nil {
		return nil, "", ErrInvalidCursor
	}
	if decoded.At == "" || decoded.ID == "" {
		return nil, "", ErrInvalidCursor
	}
	parsed, err := time.Parse(time.RFC3339Nano, decoded.At)
	if err != nil {
		return nil, "", ErrInvalidCursor
	}
	return parsed.UTC(), decoded.ID, nil
}

func encodeUserCursor(at time.Time, id string) string {
	return encodeUserMomentCursor(at.UTC().Format(time.RFC3339), id)
}

func decodeUserCursor(cursor string) (any, string, error) {
	return decodeUserMomentCursor(cursor)
}

func loadUserMomentTags(ctx context.Context, queryable persistence.Queryable, momentIDs []string) (map[string][]string, error) {
	if len(momentIDs) == 0 {
		return map[string][]string{}, nil
	}
	tagsByMomentID := make(map[string][]string, len(momentIDs))
	rows, err := queryable.Query(ctx, momenttypesQueryListMomentTagsByMomentIDs, momentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var momentID, tag string
		if err := rows.Scan(&momentID, &tag); err != nil {
			return nil, err
		}
		tagsByMomentID[momentID] = append(tagsByMomentID[momentID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tagsByMomentID, nil
}
