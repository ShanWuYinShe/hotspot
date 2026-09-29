package users

import (
	"context"
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
	limit = types.NormalizeLimit(limit)
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
	limit = types.NormalizeLimit(limit)
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
	limit = types.NormalizeLimit(limit)
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
	var remark string
	if err := scanner.Scan(&item.User.User.UserID, &item.User.User.Nickname, &avatarURL, &followedAt, &presenceStatus, &lastOnline, &sortableLastOnline, &remark); err != nil {
		return RelatedUserItem{}, time.Time{}, err
	}
	item.User.User.AvatarURL = types.StrPtr(avatarURL)
	item.Remark = &remark
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

type userMomentCursor struct {
	At string `json:"at"`
	ID string `json:"id"`
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
