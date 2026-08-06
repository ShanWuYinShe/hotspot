package feed

import (
	"context"
	"sort"
	"strings"
	"time"

	usersmodule "hotspot/backend-go/internal/modules/users"
	platformcursor "hotspot/backend-go/internal/platform/cursor"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) ListFollowingMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedMoments{}, err
	}
	candidates, err := r.loadFollowingCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedMoments{}, err
	}
	return paginateFollowingMoments(candidates, query.Limit, query.Cursor)
}

func (r *PostgresRepository) ListNearbyMoments(ctx context.Context, query momentsQuery) (PaginatedMoments, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedMoments{}, err
	}
	candidates, err := r.loadNearbyCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedMoments{}, err
	}
	return paginateNearbyMoments(candidates, query.Limit, query.Cursor)
}

func (r *PostgresRepository) ListNearbyGroups(ctx context.Context, query groupsQuery) (PaginatedGroups, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedGroups{}, err
	}
	candidates, err := r.loadGroupCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedGroups{}, err
	}
	return paginateGroups(candidates, query.Limit, query.Cursor)
}

func (r *PostgresRepository) ListNearbyUsers(ctx context.Context, query usersQuery) (PaginatedUsers, error) {
	queryable, err := r.queryable()
	if err != nil {
		return PaginatedUsers{}, err
	}
	candidates, err := r.loadUserCandidates(ctx, queryable, query)
	if err != nil {
		return PaginatedUsers{}, err
	}
	return paginateUsers(candidates, query.Limit, query.Cursor)
}

func (r *PostgresRepository) ListTrending(ctx context.Context, viewerUserID string) (TrendingBundle, error) {
	queryable, err := r.queryable()
	if err != nil {
		return TrendingBundle{}, err
	}
	hotMoments, err := loadTrendingMoments(ctx, queryable, viewerUserID)
	if err != nil {
		return TrendingBundle{}, err
	}
	return TrendingBundle{HotMoments: hotMoments}, nil
}

func (r *PostgresRepository) queryable() (persistence.QueryableDB, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	return queryable, nil
}

func (r *PostgresRepository) FetchMomentImageRows(ctx context.Context, momentIDs []string) (map[string][]momentImageRow, error) {
	queryable, err := r.queryable()
	if err != nil {
		return map[string][]momentImageRow{}, nil
	}
	return FetchMomentImageRows(ctx, queryable, momentIDs)
}

func (r *PostgresRepository) loadUserCandidates(ctx context.Context, queryable persistence.QueryableDB, query usersQuery) ([]userCandidate, error) {
	bounds := geo.BoundsAround(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius)
	mins, maxs := geo.CoveringArrays(bounds)
	if len(mins) == 0 {
		return nil, nil
	}
	rows, err := queryable.Query(ctx, queryFeedUsers, mins, maxs, query.ViewerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plan := geo.NewSearchPlan(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius, "distance")
	items := make([]userCandidate, 0)
	for rows.Next() {
		var userID, nickname, avatarURL, bio, locationName, presenceStatus string
		var latitude, longitude float64
		var isFollowing, isFollowedBy bool
		var mutualFriendsCount, commonGroupsCount int
		var lastOnline time.Time
		var tags []string
		var momentsCount, friendsCount, followersCount, followingCount, groupsCount int
		if err := rows.Scan(&userID, &nickname, &avatarURL, &bio, &locationName, &latitude, &longitude, &isFollowing, &isFollowedBy, &mutualFriendsCount, &commonGroupsCount, &presenceStatus, &lastOnline, &tags, &momentsCount, &friendsCount, &followersCount, &followingCount, &groupsCount); err != nil {
			return nil, err
		}
		if userID == query.ViewerUserID {
			continue
		}
		distance, bearing, ok := plan.Match(geo.Point{Latitude: latitude, Longitude: longitude})
		if !ok {
			continue
		}
		d := float64(distance)
		b := float64(bearing)
		item := UserItem{
			User:           UserInfo{UserID: userID, Nickname: nickname, AvatarURL: avatarURL},
			Bio:            strings.TrimSpace(bio),
			Tags:           tags,
			PresenceStatus: presenceStatusPtr(presenceStatus),
			Location: &RelativeLocation{
				Point:          GeoPoint{Latitude: latitude, Longitude: longitude},
				LocationName:   locationName,
				Distance:       &d,
				BearingDegrees: &b,
			},
			Stats: &UserStats{
				MomentsCount:   momentsCount,
				FriendsCount:   friendsCount,
				FollowersCount: followersCount,
				FollowingCount: followingCount,
				GroupsCount:    groupsCount,
			},
			Relation:      &UserRelation{IsFollowing: isFollowing, IsFollowedBy: isFollowedBy, IsBlocked: false},
			SocialContext: &UserSocialContext{MutualFriendsCount: mutualFriendsCount, CommonGroupsCount: commonGroupsCount},
		}
		if !lastOnline.IsZero() {
			lastOnlineStr := lastOnline.UTC().Format(time.RFC3339)
			item.LastOnlineAt = &lastOnlineStr
		}
		items = append(items, userCandidate{Item: item, Distance: distance})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool { return compareUserCandidates(items[i], items[j]) < 0 })
	return items, nil
}

// presenceStatusPtr converts the raw presence column to a pointer, omitting
// empty values (privacy: presence_visible=false hides the field entirely).
func presenceStatusPtr(raw string) *UserPresenceStatus {
	ps := UserPresenceStatus(raw)
	switch ps {
	case usersmodule.UserPresenceOnline, usersmodule.UserPresenceOffline:
		return &ps
	default:
		return nil
	}
}

func (r *PostgresRepository) loadGroupCandidates(ctx context.Context, queryable persistence.QueryableDB, query groupsQuery) ([]groupCandidate, error) {
	bounds := geo.BoundsAround(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius)
	mins, maxs := geo.CoveringArrays(bounds)
	if len(mins) == 0 {
		return nil, nil
	}
	rows, err := queryable.Query(ctx, queryFeedGroups, mins, maxs, query.ViewerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plan := geo.NewSearchPlan(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius, "distance")
	items := make([]groupCandidate, 0)
	for rows.Next() {
		var groupID, name, avatarURL, description, locationName, joinMethod, nextActivity, ownerID, role string
		var latitude, longitude float64
		var hasPassword, isJoined, requiresApproval, isSearchable, isPinned bool
		var membersCount, friendsInGroup, onlineMembersCount, unreadCount, maxMembers int
		var tags []string
		var createdAt, joinedAt, mutedUntil time.Time
		if err := rows.Scan(&groupID, &name, &avatarURL, &description, &hasPassword, &locationName, &latitude, &longitude, &membersCount, &isJoined, &friendsInGroup, &joinMethod, &requiresApproval, &tags, &nextActivity, &isSearchable, &maxMembers, &createdAt, &ownerID, &onlineMembersCount, &role, &joinedAt, &mutedUntil, &isPinned, &unreadCount); err != nil {
			return nil, err
		}
		distance, bearing, ok := plan.Match(geo.Point{Latitude: latitude, Longitude: longitude})
		if !ok {
			continue
		}
		d := float64(distance)
		b := float64(bearing)
		item := GroupItem{
			Group:              GroupInfo{GroupID: groupID, Name: name, AvatarURL: avatarURL, MembersCount: membersCount},
			Description:        strings.TrimSpace(description),
			Tags:               tags,
			JoinMethod:         GroupJoinMethod(joinMethod),
			RequiresApproval:   requiresApproval,
			HasPassword:        hasPassword && isJoined,
			Location:           RelativeLocation{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName, Distance: &d, BearingDegrees: &b},
			FriendsInGroup:     friendsInGroup,
			IsSearchable:       isSearchable,
			MaxMembers:         maxMembers,
			OnlineMembersCount: &onlineMembersCount,
			NextActivity:       strPtr(nextActivity),
			OwnerID:            ownerID,
		}
		if !createdAt.IsZero() {
			createdAtStr := createdAt.UTC().Format(time.RFC3339)
			item.CreatedAt = &createdAtStr
		}
		if isJoined {
			state := &MyGroupState{
				Role:        GroupRole(role),
				IsJoined:    true,
				UnreadCount: unreadCount,
				IsPinned:    isPinned,
			}
			if !joinedAt.IsZero() {
				joinedAtStr := joinedAt.UTC().Format(time.RFC3339)
				state.JoinedAt = &joinedAtStr
			}
			if !mutedUntil.IsZero() {
				mutedUntilStr := mutedUntil.UTC().Format(time.RFC3339)
				state.MutedUntil = &mutedUntilStr
				state.IsMuted = true
			}
			item.MyState = state
		}
		items = append(items, groupCandidate{Item: item, Distance: distance})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(items, func(i, j int) bool { return compareGroupCandidates(items[i], items[j]) < 0 })
	return items, nil
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (r *PostgresRepository) loadFollowingCandidates(ctx context.Context, queryable persistence.QueryableDB, query momentsQuery) ([]momentCandidate, error) {
	rows, err := queryable.Query(ctx, queryFeedMomentsFollowing, query.ViewerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]momentCandidate, 0)
	momentIDs := make([]string, 0)
	for rows.Next() {
		var momentID, userID, nickname, avatarURL, description, locationName string
		var latitude, longitude float64
		var createdAt time.Time
		var likesCount, commentsCount, sharesCount int
		var isLiked, isFavorited bool
		if err := rows.Scan(&momentID, &userID, &nickname, &avatarURL, &description, &createdAt, &likesCount, &commentsCount, &sharesCount, &locationName, &latitude, &longitude, &isLiked, &isFavorited); err != nil {
			return nil, err
		}
		var distanceM int
		var bearingD int
		ref := geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}
		if isValidGeoPoint(query.ReferencePoint) {
			distanceM = geo.DistanceMeters(ref, geo.Point{Latitude: latitude, Longitude: longitude})
			bearingD = geo.BearingDegrees(ref, geo.Point{Latitude: latitude, Longitude: longitude})
		}
		d := float64(distanceM)
		b := float64(bearingD)
		item := MomentItem{
			MomentID:      momentID,
			Description:   description,
			CreatedAt:     createdAt.UTC().Format(time.RFC3339),
			LikesCount:    likesCount,
			CommentsCount: commentsCount,
			SharesCount:   sharesCount,
			User:          UserInfo{UserID: userID, Nickname: nickname, AvatarURL: avatarURL},
			Location:      RelativeLocation{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName, Distance: &d, BearingDegrees: &b},
			MyState:       &MyMomentState{IsFavorited: isFavorited},
		}
		reason := RecommendationContext("following")
		item.Recommendation = &reason
		items = append(items, momentCandidate{Item: item, CreatedAt: createdAt.UTC()})
		momentIDs = append(momentIDs, momentID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].Item.Tags = tagsByMomentID[items[idx].Item.MomentID]
	}
	sort.SliceStable(items, func(i, j int) bool { return compareFollowingCandidates(items[i], items[j]) < 0 })
	return items, nil
}

func (r *PostgresRepository) loadNearbyCandidates(ctx context.Context, queryable persistence.QueryableDB, query momentsQuery) ([]momentCandidate, error) {
	bounds := geo.BoundsAround(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius)
	mins, maxs := geo.CoveringArrays(bounds)
	if len(mins) == 0 {
		return nil, nil
	}
	rows, err := queryable.Query(ctx, queryFeedMomentsNearby, mins, maxs, query.ViewerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plan := geo.NewSearchPlan(geo.Point{Latitude: query.ReferencePoint.Latitude, Longitude: query.ReferencePoint.Longitude}, query.Radius, "relevance")
	items := make([]momentCandidate, 0)
	momentIDs := make([]string, 0)
	for rows.Next() {
		var momentID, userID, nickname, avatarURL, description, locationName string
		var latitude, longitude float64
		var createdAt time.Time
		var likesCount, commentsCount, sharesCount int
		var isLiked, isFavorited bool
		var isFollowing bool
		if err := rows.Scan(&momentID, &userID, &nickname, &avatarURL, &description, &createdAt, &likesCount, &commentsCount, &sharesCount, &locationName, &latitude, &longitude, &isLiked, &isFavorited, &isFollowing); err != nil {
			return nil, err
		}
		distance, bearing, ok := plan.Match(geo.Point{Latitude: latitude, Longitude: longitude})
		if !ok {
			continue
		}
		d := float64(distance)
		b := float64(bearing)
		score := likesCount + commentsCount + sharesCount
		if isFollowing {
			score += 1000
		}
		score += max(0, query.Radius-distance)
		item := MomentItem{
			MomentID:      momentID,
			Description:   description,
			CreatedAt:     createdAt.UTC().Format(time.RFC3339),
			LikesCount:    likesCount,
			CommentsCount: commentsCount,
			SharesCount:   sharesCount,
			User:          UserInfo{UserID: userID, Nickname: nickname, AvatarURL: avatarURL},
			Location:      RelativeLocation{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName, Distance: &d, BearingDegrees: &b},
			MyState:       &MyMomentState{IsFavorited: isFavorited},
		}
		reason := RecommendationContext("nearby")
		item.Recommendation = &reason
		items = append(items, momentCandidate{Item: item, Score: score, Distance: distance, CreatedAt: createdAt.UTC()})
		momentIDs = append(momentIDs, momentID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].Item.Tags = tagsByMomentID[items[idx].Item.MomentID]
	}
	sort.SliceStable(items, func(i, j int) bool { return compareNearbyCandidates(items[i], items[j]) < 0 })
	return items, nil
}

func loadTrendingMoments(ctx context.Context, queryable persistence.QueryableDB, viewerUserID string) ([]MomentItem, error) {
	rows, err := queryable.Query(ctx, queryFeedTrendingHotMoments, viewerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]momentCandidate, 0)
	momentIDs := make([]string, 0)
	for rows.Next() {
		var momentID, userID, nickname, avatarURL, description, locationName string
		var latitude, longitude float64
		var createdAt time.Time
		var likesCount, commentsCount, sharesCount int
		var isLiked, isFavorited bool
		if err := rows.Scan(&momentID, &userID, &nickname, &avatarURL, &description, &createdAt, &likesCount, &commentsCount, &sharesCount, &locationName, &latitude, &longitude, &isLiked, &isFavorited); err != nil {
			return nil, err
		}
		item := MomentItem{
			MomentID:      momentID,
			Description:   description,
			CreatedAt:     createdAt.UTC().Format(time.RFC3339),
			LikesCount:    likesCount,
			CommentsCount: commentsCount,
			SharesCount:   sharesCount,
			User:          UserInfo{UserID: userID, Nickname: nickname, AvatarURL: avatarURL},
			Location:      RelativeLocation{Point: GeoPoint{Latitude: latitude, Longitude: longitude}, LocationName: locationName},
			MyState:       &MyMomentState{IsFavorited: isFavorited},
		}
		reason := RecommendationContext("trending")
		item.Recommendation = &reason
		items = append(items, momentCandidate{Item: item, Score: likesCount + commentsCount + sharesCount, CreatedAt: createdAt.UTC()})
		momentIDs = append(momentIDs, momentID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tagsByMomentID, err := loadMomentTags(ctx, queryable, momentIDs)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].Item.Tags = tagsByMomentID[items[idx].Item.MomentID]
	}
	sort.SliceStable(items, func(i, j int) bool { return compareNearbyCandidates(items[i], items[j]) < 0 })
	out := make([]MomentItem, 0, len(items))
	for _, item := range items {
		out = append(out, item.Item)
	}
	return out, nil
}

func loadMomentTags(ctx context.Context, queryable persistence.QueryableDB, momentIDs []string) (map[string][]string, error) {
	if len(momentIDs) == 0 {
		return map[string][]string{}, nil
	}
	result := make(map[string][]string, len(momentIDs))
	rows, err := queryable.Query(ctx, queryFeedMomentTagsByMomentIDs, momentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var momentID, tag string
		if err := rows.Scan(&momentID, &tag); err != nil {
			return nil, err
		}
		result[momentID] = append(result[momentID], tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func paginateUsers(items []userCandidate, limit int, cursor string) (PaginatedUsers, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedUsers{}, err
	}
	start := findUserCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedUsers{Items: make([]UserItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		resp.NextCursor = encodeFeedCursor(itemCursor{Distance: items[end-1].Distance, ID: items[end-1].Item.User.UserID})
	}
	return resp, nil
}

func paginateGroups(items []groupCandidate, limit int, cursor string) (PaginatedGroups, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedGroups{}, err
	}
	start := findGroupCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedGroups{Items: make([]GroupItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		resp.NextCursor = encodeFeedCursor(itemCursor{Distance: items[end-1].Distance, ID: items[end-1].Item.Group.GroupID})
	}
	return resp, nil
}

func paginateFollowingMoments(items []momentCandidate, limit int, cursor string) (PaginatedMoments, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	start := findFollowingCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedMoments{Items: make([]MomentItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		last := items[end-1]
		resp.NextCursor = encodeFeedCursor(itemCursor{CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339), ID: last.Item.MomentID})
	}
	return resp, nil
}

func paginateNearbyMoments(items []momentCandidate, limit int, cursor string) (PaginatedMoments, error) {
	decoded, err := decodeFeedCursor(cursor)
	if err != nil {
		return PaginatedMoments{}, err
	}
	start := findNearbyCursorStart(items, decoded)
	end := min(len(items), start+limit)
	resp := PaginatedMoments{Items: make([]MomentItem, 0, max(0, end-start))}
	for _, item := range items[start:end] {
		resp.Items = append(resp.Items, item.Item)
	}
	if end < len(items) && end > start {
		last := items[end-1]
		resp.NextCursor = encodeFeedCursor(itemCursor{Score: last.Score, Distance: last.Distance, CreatedAt: last.CreatedAt.UTC().Format(time.RFC3339), ID: last.Item.MomentID})
	}
	return resp, nil
}

// compareUserCandidates 排序规则：距离升序 → 共同好友数降序 → userID 降序
// （ID 大小写只作稳定 tie-break，保证分页键集序与游标一致）。
func compareUserCandidates(left, right userCandidate) int {
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if left.Item.SocialContext != nil && right.Item.SocialContext != nil && left.Item.SocialContext.MutualFriendsCount != right.Item.SocialContext.MutualFriendsCount {
		return right.Item.SocialContext.MutualFriendsCount - left.Item.SocialContext.MutualFriendsCount
	}
	if left.Item.User.UserID > right.Item.User.UserID {
		return -1
	}
	if left.Item.User.UserID < right.Item.User.UserID {
		return 1
	}
	return 0
}

// compareGroupCandidates 排序规则：距离升序 → 群内好友数降序 → groupID 降序。
func compareGroupCandidates(left, right groupCandidate) int {
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if left.Item.FriendsInGroup != right.Item.FriendsInGroup {
		return right.Item.FriendsInGroup - left.Item.FriendsInGroup
	}
	if left.Item.Group.GroupID > right.Item.Group.GroupID {
		return -1
	}
	if left.Item.Group.GroupID < right.Item.Group.GroupID {
		return 1
	}
	return 0
}

// compareFollowingCandidates 排序规则：发布时间降序（最新在前）→ momentID 降序。
func compareFollowingCandidates(left, right momentCandidate) int {
	if !left.CreatedAt.Equal(right.CreatedAt) {
		if left.CreatedAt.After(right.CreatedAt) {
			return -1
		}
		return 1
	}
	if left.Item.MomentID > right.Item.MomentID {
		return -1
	}
	if left.Item.MomentID < right.Item.MomentID {
		return 1
	}
	return 0
}

// compareNearbyCandidates 排序规则：热度评分降序（likes+comments+shares）→
// 距离升序 → 发布时间降序 → momentID 降序。
func compareNearbyCandidates(left, right momentCandidate) int {
	if left.Score != right.Score {
		return right.Score - left.Score
	}
	if left.Distance != right.Distance {
		return left.Distance - right.Distance
	}
	if !left.CreatedAt.Equal(right.CreatedAt) {
		if left.CreatedAt.After(right.CreatedAt) {
			return -1
		}
		return 1
	}
	if left.Item.MomentID > right.Item.MomentID {
		return -1
	}
	if left.Item.MomentID < right.Item.MomentID {
		return 1
	}
	return 0
}

func findUserCursorStart(items []userCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	for idx, item := range items {
		cmp := compareUserCandidates(item, userCandidate{Item: UserItem{User: UserInfo{UserID: cursor.ID}}, Distance: cursor.Distance})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

func findGroupCursorStart(items []groupCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	for idx, item := range items {
		cmp := compareGroupCandidates(item, groupCandidate{Item: GroupItem{Group: GroupInfo{GroupID: cursor.ID}}, Distance: cursor.Distance})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

func findFollowingCursorStart(items []momentCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	parsedAt, _ := time.Parse(time.RFC3339, cursor.CreatedAt)
	for idx, item := range items {
		cmp := compareFollowingCandidates(item, momentCandidate{Item: MomentItem{MomentID: cursor.ID}, CreatedAt: parsedAt.UTC()})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

func findNearbyCursorStart(items []momentCandidate, cursor *itemCursor) int {
	if cursor == nil {
		return 0
	}
	parsedAt, _ := time.Parse(time.RFC3339, cursor.CreatedAt)
	for idx, item := range items {
		cmp := compareNearbyCandidates(item, momentCandidate{Item: MomentItem{MomentID: cursor.ID}, Score: cursor.Score, Distance: cursor.Distance, CreatedAt: parsedAt.UTC()})
		if cmp > 0 {
			return idx
		}
		if cmp == 0 {
			return idx + 1
		}
	}
	return len(items)
}

var feedCursorCodec = platformcursor.NewCodec("feed")

func encodeFeedCursor(cursor itemCursor) string {
	token, err := feedCursorCodec.Encode(cursor)
	if err != nil {
		return ""
	}
	return token
}

func decodeFeedCursor(cursor string) (*itemCursor, error) {
	if cursor == "" {
		return nil, nil
	}
	var decoded itemCursor
	if err := feedCursorCodec.Decode(cursor, &decoded); err != nil {
		return nil, ErrInvalidCursor
	}
	if decoded.ID == "" {
		return nil, ErrInvalidCursor
	}
	if decoded.CreatedAt != "" {
		if _, err := time.Parse(time.RFC3339, decoded.CreatedAt); err != nil {
			return nil, ErrInvalidCursor
		}
	}
	return &decoded, nil
}

var _ Repository = (*PostgresRepository)(nil)
