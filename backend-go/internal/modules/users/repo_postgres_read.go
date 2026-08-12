package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) GetProfile(ctx context.Context, userID string) (MyProfile, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MyProfile{}, false, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MyProfile{}, false, persistence.ErrNotImplemented
	}
	profile, err := scanUserProfileRow(queryable.QueryRow(ctx, queryGetUserProfile, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MyProfile{}, false, nil
		}
		return MyProfile{}, false, err
	}
	stats, err := countUserStats(ctx, queryable, userID)
	if err != nil {
		return MyProfile{}, false, err
	}
	profile.Stats = stats
	return profile, true, nil
}

func (r *PostgresRepository) GetSettings(ctx context.Context, userID string) (UserSettings, bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return UserSettings{}, false, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return UserSettings{}, false, persistence.ErrNotImplemented
	}
	var settings UserSettings
	var allowPrivateChat string
	if err := queryable.QueryRow(ctx, queryGetUserSettings, userID).Scan(&settings.PrivacyLevel, &settings.LocationVisible, &settings.PresenceVisible, &settings.SocialListsVisible, &settings.GroupsVisible, &settings.ReactedMomentsVisible, &settings.StatsVisible, &settings.AllowFollow, &allowPrivateChat, &settings.Searchable, &settings.PhoneSearchable, &settings.NicknameSearchable, &settings.LocationSearchable, &settings.NearbyMomentNotify); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return UserSettings{}, false, nil
		}
		return UserSettings{}, false, err
	}
	settings.AllowPrivateChat = ChatPerm(allowPrivateChat)
	return settings, true, nil
}

func (r *PostgresRepository) GetSettingsBatch(ctx context.Context, userIDs []string) (map[string]UserSettings, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	result := make(map[string]UserSettings, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := queryable.Query(ctx, queryGetUserSettingsBatch, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID string
		var settings UserSettings
		var allowPrivateChat string
		if err := rows.Scan(&userID, &settings.PrivacyLevel, &settings.LocationVisible, &settings.PresenceVisible, &settings.SocialListsVisible, &settings.GroupsVisible, &settings.ReactedMomentsVisible, &settings.StatsVisible, &settings.AllowFollow, &allowPrivateChat, &settings.Searchable, &settings.PhoneSearchable, &settings.NicknameSearchable, &settings.LocationSearchable, &settings.NearbyMomentNotify); err != nil {
			return nil, err
		}
		settings.AllowPrivateChat = ChatPerm(allowPrivateChat)
		result[userID] = settings
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepository) SearchProfiles(ctx context.Context, req SearchUsersRequest) (PaginatedProfiles, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedProfiles{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedProfiles{}, persistence.ErrNotImplemented
	}
	radius := normalizeSearchRadius(req.Radius)
	refPt := geo.Point{Latitude: req.ReferencePoint.Latitude, Longitude: req.ReferencePoint.Longitude}
	bounds := geo.BoundsAround(refPt, radius)
	sql, args := buildSearchUsersQuery(bounds, req.Keyword)
	if sql == "" {
		return PaginatedProfiles{}, nil
	}
	rows, err := queryable.Query(ctx, sql, args...)
	if err != nil {
		return PaginatedProfiles{}, err
	}
	defer rows.Close()
	candidates := make([]searchProfileCandidate, 0)
	sortBy := normalizeSearchSort(req.SortBy)
	plan := geo.NewSearchPlan(refPt, radius, sortBy)
	keyword := strings.ToLower(strings.TrimSpace(req.Keyword))
	tag := strings.ToLower(strings.TrimSpace(req.Tag))
	// Scan every candidate row first, then load profile tags for the whole
	// page in a single query (avoids an N+1 of loadUserProfileTags calls
	// inside the loop).
	scanned := make([]MyProfile, 0)
	for rows.Next() {
		profile, err := scanUserProfileRow(rows)
		if err != nil {
			return PaginatedProfiles{}, err
		}
		scanned = append(scanned, profile)
	}
	if err := rows.Err(); err != nil {
		return PaginatedProfiles{}, err
	}
	if len(scanned) == 0 {
		return PaginatedProfiles{}, nil
	}
	userIDs := make([]string, 0, len(scanned))
	for _, profile := range scanned {
		userIDs = append(userIDs, profile.UserID)
	}
	tagsByUserID, err := loadUserProfileTagsBatch(ctx, queryable, userIDs)
	if err != nil {
		return PaginatedProfiles{}, err
	}
	for _, profile := range scanned {
		profile.Tags = tagsByUserID[profile.UserID]
		if tag != "" && !containsFold(profile.Tags, tag) {
			continue
		}
		distance, bearing, within := plan.Match(geo.Point{Latitude: profile.Location.Point.Latitude, Longitude: profile.Location.Point.Longitude})
		if !within {
			continue
		}
		candidates = append(candidates, searchProfileCandidate{Profile: profile, Distance: distance, BearingDegrees: bearing, SortBy: sortBy, RelevanceScore: searchRelevanceScore(profile, keyword, tag)})
	}
	sortSearchCandidates(candidates, normalizeSearchSort(req.SortBy))
	return paginateSearchCandidates(candidates, req.Pagination), nil
}

func buildSearchUsersQuery(bounds geo.Bounds, keyword string) (string, []any) {
	mins, maxs := geo.CoveringArrays(bounds)
	if len(mins) == 0 {
		return "", nil
	}
	args := make([]any, 0, 4)
	var builder strings.Builder
	builder.WriteString(`SELECT u.user_id, u.nickname, COALESCE(u.avatar_url, ''), u.user_type, COALESCE(u.bio, ''), COALESCE(u.gender, ''), COALESCE(u.email, ''), COALESCE(u.phone, ''), u.location_name, u.latitude, u.longitude, u.presence_status, u.last_online, u.created_at, u.is_deleted FROM users u JOIN user_settings us ON us.user_id = u.user_id WHERE us.searchable = TRUE AND u.is_deleted = FALSE AND us.location_visible = TRUE AND us.privacy_level = 'public' AND EXISTS (SELECT 1 FROM unnest($1::bigint[], $2::bigint[]) AS t(lo, hi) WHERE u.s2_cell_id BETWEEN t.lo AND t.hi)`)
	args = append(args, mins, maxs)
	if keyword != "" {
		args = append(args, keyword)
		builder.WriteString(fmt.Sprintf(" AND ($%d = '' OR (us.nickname_searchable = TRUE AND u.nickname ILIKE '%%' || $%d || '%%') OR u.bio ILIKE '%%' || $%d || '%%')", len(args), len(args), len(args)))
	}
	return builder.String(), args
}

func (r *PostgresRepository) Relationship(ctx context.Context, userID, targetUserID string) (Relationship, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return Relationship{}, persistence.ErrNotImplemented
	}
	following, blocked, _, _, _, _, err := r.getRelationFields(ctx, userID, targetUserID)
	if err != nil {
		return Relationship{}, err
	}
	followedBy, _, _, _, _, _, err := r.getRelationFields(ctx, targetUserID, userID)
	if err != nil {
		return Relationship{}, err
	}
	return Relationship{IsFollowing: following, IsFollowedBy: followedBy, IsBlocked: blocked}, nil
}

// RelationshipBatch resolves the relationship between fromUserID and every
// entry in toUserIDs in two queries (one for the forward edge, one for the
// reverse edge), rather than one query per target. It replaces the N+1 that
// Search previously triggered by calling Relationship once per profile.
func (r *PostgresRepository) RelationshipBatch(ctx context.Context, fromUserID string, toUserIDs []string) (map[string]Relationship, error) {
	result := make(map[string]Relationship, len(toUserIDs))
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return result, persistence.ErrNotImplemented
	}
	if len(toUserIDs) == 0 {
		return result, nil
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return result, persistence.ErrNotImplemented
	}
	// Initialise defaults so missing rows still produce a zero-value Relationship.
	for _, target := range toUserIDs {
		result[target] = Relationship{}
	}
	// Forward edge: am I following / did I block each target?
	forwardRows, err := queryable.Query(ctx, queryListUserRelationsForward, fromUserID, toUserIDs)
	if err != nil {
		return nil, err
	}
	for forwardRows.Next() {
		var targetUserID string
		var following, blocked bool
		if err := forwardRows.Scan(&targetUserID, &following, &blocked); err != nil {
			forwardRows.Close()
			return nil, err
		}
		rel := result[targetUserID]
		rel.IsFollowing = following
		rel.IsBlocked = blocked
		result[targetUserID] = rel
	}
	if err := forwardRows.Err(); err != nil {
		forwardRows.Close()
		return nil, err
	}
	forwardRows.Close()
	// Reverse edge: is each target following me?
	reverseRows, err := queryable.Query(ctx, queryListUserRelationsReverse, fromUserID, toUserIDs)
	if err != nil {
		return nil, err
	}
	for reverseRows.Next() {
		var sourceUserID string
		var following bool
		if err := reverseRows.Scan(&sourceUserID, &following); err != nil {
			reverseRows.Close()
			return nil, err
		}
		rel := result[sourceUserID]
		rel.IsFollowedBy = following
		result[sourceUserID] = rel
	}
	if err := reverseRows.Err(); err != nil {
		reverseRows.Close()
		return nil, err
	}
	reverseRows.Close()
	return result, nil
}

func (r *PostgresRepository) getRelationFields(ctx context.Context, userID, targetUserID string) (bool, bool, string, any, any, string, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, false, "", nil, nil, "", persistence.ErrNotImplemented
	}
	var following, blocked bool
	var remark string
	var followedAt, blockedAt *time.Time
	var blockReason string
	if err := queryable.QueryRow(ctx, queryGetUserRelation, userID, targetUserID).Scan(&following, &blocked, &remark, &followedAt, &blockedAt, &blockReason); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, "", nil, nil, "", nil
		}
		return false, false, "", nil, nil, "", err
	}
	return following, blocked, remark, timePtrToAny(followedAt), timePtrToAny(blockedAt), blockReason, nil
}

func listBlockedUsersPostgres(ctx context.Context, factory *persistence.Factory, userID string, limit int, cursor string) (PaginatedBlockedUsers, error) {
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedBlockedUsers{}, persistence.ErrNotImplemented
	}
	limit = normalizeUserMomentLimit(limit)
	cursorAt, cursorID, err := decodeUserCursor(cursor)
	if err != nil {
		return PaginatedBlockedUsers{}, ErrInvalidCursor
	}
	rows, err := queryable.Query(ctx, queryListBlockedUsers, userID, cursorAt, cursorID, limit+1)
	if err != nil {
		return PaginatedBlockedUsers{}, err
	}
	defer rows.Close()
	items := make([]BlockedUserItem, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		var targetUserID, reason string
		var blockedAt time.Time
		if err := rows.Scan(&targetUserID, &blockedAt, &reason); err != nil {
			return PaginatedBlockedUsers{}, err
		}
		items = append(items, BlockedUserItem{User: UserInfo{UserID: targetUserID, Nickname: targetUserID}, BlockedAt: blockedAt.UTC().Format(time.RFC3339), Reason: reason})
		cursors = append(cursors, encodeUserCursor(blockedAt, targetUserID))
		if len(items) == limit+1 {
			return PaginatedBlockedUsers{Items: items[:limit], NextCursor: cursors[limit-1]}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return PaginatedBlockedUsers{}, err
	}
	return PaginatedBlockedUsers{Items: items}, nil
}

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
	items := make([]FriendItem, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, sortAt, err := scanFriendRow(rows)
		if err != nil {
			return PaginatedFriends{}, err
		}
		items = append(items, item)
		cursors = append(cursors, encodeUserCursor(sortAt, item.User.UserID))
		if len(items) == limit+1 {
			return PaginatedFriends{Items: items[:limit], NextCursor: cursors[limit-1]}, nil
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
	items := make([]FollowingItem, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, sortAt, err := scanFollowingRow(rows)
		if err != nil {
			return PaginatedFollowing{}, err
		}
		items = append(items, item)
		cursors = append(cursors, encodeUserCursor(sortAt, item.User.UserID))
		if len(items) == limit+1 {
			return PaginatedFollowing{Items: items[:limit], NextCursor: cursors[limit-1]}, nil
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
	items := make([]FollowerItem, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		item, sortAt, err := scanFollowerRow(rows)
		if err != nil {
			return PaginatedFollowers{}, err
		}
		items = append(items, item)
		cursors = append(cursors, encodeUserCursor(sortAt, item.User.UserID))
		if len(items) == limit+1 {
			return PaginatedFollowers{Items: items[:limit], NextCursor: cursors[limit-1]}, nil
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
			return grouptypes.PaginatedGroups{Items: items[:limit], NextCursor: cursors[limit-1]}, nil
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
		parsed, _ := time.Parse(time.RFC3339, item.CreatedAt)
		cursors = append(cursors, encodeUserMomentCursor(parsed.UTC().Format(time.RFC3339), item.MomentID))
		if len(items) == limit+1 {
			tagsByMomentID, err := loadUserMomentTags(ctx, queryable, momentIDs[:limit])
			if err != nil {
				return momenttypes.PaginatedMoments{}, err
			}
			for i := 0; i < limit; i++ {
				items[i].Tags = tagsByMomentID[items[i].MomentID]
			}
			return momenttypes.PaginatedMoments{Items: items[:limit], NextCursor: cursors[limit-1]}, nil
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
	if factory == nil || factory.DB == nil || factory.Config.PostgresDSN == "" {
		return momenttypes.PaginatedMoments{}, persistence.ErrNotImplemented
	}
	queryable, ok := factory.DB.(persistence.QueryableDB)
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
		item.MyState = &momenttypes.MyMomentState{}
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
			return momenttypes.PaginatedMoments{Items: items[:limit], NextCursor: cursors[limit-1]}, nil
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

func scanUserProfileRow(scanner interface{ Scan(...any) error }) (MyProfile, error) {
	var profile MyProfile
	var avatarURL, bio, gender, email, phone string
	var locationName string
	var latitude, longitude float64
	var presenceStatus, userType string
	var lastOnline, createdAt time.Time
	if err := scanner.Scan(&profile.UserID, &profile.Nickname, &avatarURL, &userType, &bio, &gender, &email, &phone, &locationName, &latitude, &longitude, &presenceStatus, &lastOnline, &createdAt, &profile.IsDeleted); err != nil {
		return MyProfile{}, err
	}
	profile.AvatarURL = avatarURL
	profile.UserType = UserType(userType)
	profile.Bio = bio
	profile.Gender = Gender(gender)
	profile.Email = email
	profile.Phone = phone
	profile.Location = Location{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
	profile.PresenceStatus = UserPresenceStatus(presenceStatus)
	profile.LastOnline = lastOnline.UTC().Format(time.RFC3339)
	profile.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return profile, nil
}

func countUserStats(ctx context.Context, queryable persistence.QueryableDB, userID string) (UserStats, error) {
	var stats UserStats
	if err := queryable.QueryRow(ctx, queryCountUserStats, userID).Scan(&stats.MomentsCount, &stats.FriendsCount, &stats.FollowersCount, &stats.FollowingCount, &stats.GroupsCount); err != nil {
		return UserStats{}, err
	}
	return stats, nil
}

func scanFriendRow(scanner interface{ Scan(...any) error }) (FriendItem, time.Time, error) {
	var item FriendItem
	var avatarURL string
	var presenceStatus string
	var lastOnline time.Time
	var sortableLastOnline time.Time
	var followedAt, followedBackAt time.Time
	if err := scanner.Scan(&item.User.UserID, &item.User.Nickname, &avatarURL, &presenceStatus, &lastOnline, &sortableLastOnline, &item.Remark, &followedAt, &followedBackAt); err != nil {
		return FriendItem{}, time.Time{}, err
	}
	item.User.AvatarURL = avatarURL
	ps := UserPresenceStatus(presenceStatus)
	item.PresenceStatus = &ps
	lastOnlineStr := lastOnline.UTC().Format(time.RFC3339)
	item.LastOnline = &lastOnlineStr
	item.FollowedAt = followedAt.UTC().Format(time.RFC3339)
	item.FollowedBackAt = followedBackAt.UTC().Format(time.RFC3339)
	return item, followedBackAt.UTC(), nil
}

func scanFollowingRow(scanner interface{ Scan(...any) error }) (FollowingItem, time.Time, error) {
	var item FollowingItem
	var avatarURL string
	var followedAt time.Time
	var presenceStatus string
	var lastOnline time.Time
	var sortableLastOnline time.Time
	if err := scanner.Scan(&item.User.UserID, &item.User.Nickname, &avatarURL, &followedAt, &item.IsMutual, &presenceStatus, &lastOnline, &sortableLastOnline); err != nil {
		return FollowingItem{}, time.Time{}, err
	}
	item.User.AvatarURL = avatarURL
	item.FollowedAt = followedAt.UTC().Format(time.RFC3339)
	ps := UserPresenceStatus(presenceStatus)
	item.PresenceStatus = &ps
	lastOnlineStr := lastOnline.UTC().Format(time.RFC3339)
	item.LastOnline = &lastOnlineStr
	return item, followedAt.UTC(), nil
}

func scanFollowerRow(scanner interface{ Scan(...any) error }) (FollowerItem, time.Time, error) {
	var item FollowerItem
	var avatarURL string
	var followedAt time.Time
	var presenceStatus string
	var lastOnline time.Time
	var sortableLastOnline time.Time
	if err := scanner.Scan(&item.User.UserID, &item.User.Nickname, &avatarURL, &followedAt, &item.IsMutual, &presenceStatus, &lastOnline, &sortableLastOnline); err != nil {
		return FollowerItem{}, time.Time{}, err
	}
	item.User.AvatarURL = avatarURL
	item.FollowedAt = followedAt.UTC().Format(time.RFC3339)
	ps := UserPresenceStatus(presenceStatus)
	item.PresenceStatus = &ps
	lastOnlineStr := lastOnline.UTC().Format(time.RFC3339)
	item.LastOnline = &lastOnlineStr
	return item, followedAt.UTC(), nil
}

func scanUserGroupRow(scanner interface{ Scan(...any) error }) (grouptypes.GroupItem, time.Time, error) {
	var item grouptypes.GroupItem
	var locationName, joinMethod string
	var latitude, longitude float64
	var joinedAt time.Time
	if err := scanner.Scan(&item.Group.GroupID, &item.Group.Name, &item.Group.AvatarURL, &item.Description, &item.HasPassword, &locationName, &latitude, &longitude, &joinMethod, &item.IsSearchable, &item.MaxMembers, &joinedAt); err != nil {
		return grouptypes.GroupItem{}, time.Time{}, err
	}
	item.Location = grouptypes.Location{Point: struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}{Latitude: latitude, Longitude: longitude}, LocationName: locationName}
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
	item.Location = momenttypes.Location{Point: struct {
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
	item.Location = momenttypes.Location{Point: struct {
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
	case limit > 100:
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
	parsed, err := time.Parse(time.RFC3339, decoded.At)
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

func loadUserMomentTags(ctx context.Context, queryable persistence.QueryableDB, momentIDs []string) (map[string][]string, error) {
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

func timePtrToAny(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}

func loadUserProfileTags(ctx context.Context, queryable persistence.QueryableDB, userID string) ([]string, error) {
	rows, err := queryable.Query(ctx, queryListUserProfileTags, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tags := make([]string, 0)
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

// loadUserProfileTagsBatch fetches profile tags for many user IDs in one
// query. It returns a userID -> []tags map; user IDs with no tags are absent
// from the map (callers should treat absence as an empty slice).
func loadUserProfileTagsBatch(ctx context.Context, queryable persistence.QueryableDB, userIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := queryable.Query(ctx, queryListUserProfileTagsBatch, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID, tag string
		if err := rows.Scan(&userID, &tag); err != nil {
			return nil, err
		}
		result[userID] = append(result[userID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
