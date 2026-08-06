package feed

import (
	"context"
	"sort"
	"strings"
	"time"

	usersmodule "hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
)

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
