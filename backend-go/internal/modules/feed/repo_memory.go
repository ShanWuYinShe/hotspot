package feed

import "context"

type InMemoryRepository struct {
	dismissed map[string]struct{}
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{dismissed: make(map[string]struct{})}
}

func (r *InMemoryRepository) ListMoments(_ context.Context, _ momentsQuery) (PaginatedMoments, error) {
	return PaginatedMoments{Items: []MomentItem{{
		MomentID:      "moment_feed_1",
		Description:   "feed moment",
		Tags:          []string{"hot"},
		CreatedAt:     "2026-03-01T00:00:00Z",
		LikesCount:    1,
		CommentsCount: 0,
		SharesCount:   0,
		User:          UserInfo{UserID: "user_feed_1", Nickname: "feed user"},
		Location: RelativeLocation{
			Point:        GeoPoint{Latitude: 0, Longitude: 0},
			LocationName: "stub",
		},
	}}}, nil
}

func (r *InMemoryRepository) ListGroups(_ context.Context, _ groupsQuery) (PaginatedGroups, error) {
	return PaginatedGroups{Items: []GroupItem{{
		Group: GroupInfo{GroupID: "group_feed_1", Name: "feed group", MembersCount: 1},
		Location: RelativeLocation{
			Point:        GeoPoint{Latitude: 0, Longitude: 0},
			LocationName: "stub",
		},
	}}}, nil
}

func (r *InMemoryRepository) ListUsers(_ context.Context, _ usersQuery) (PaginatedUsers, error) {
	return PaginatedUsers{Items: []UserItem{{
		User: UserInfo{UserID: "user_feed_1", Nickname: "feed user"},
		Location: &RelativeLocation{
			Point:        GeoPoint{Latitude: 0, Longitude: 0},
			LocationName: "stub",
		},
	}}}, nil
}

func (r *InMemoryRepository) ListTrending(_ context.Context, _ string) (TrendingBundle, error) {
	return TrendingBundle{
		TrendingTags: []TrendingTagItem{{Name: "hot", UsageCount: 1, Trend: "rising"}},
		HotMoments: []MomentItem{{
			MomentID:      "moment_hot_1",
			Description:   "hot moment",
			Tags:          []string{"hot"},
			CreatedAt:     "2026-03-01T00:00:00Z",
			LikesCount:    1,
			CommentsCount: 0,
			SharesCount:   0,
			User:          UserInfo{UserID: "user_feed_1", Nickname: "feed user"},
			Location: RelativeLocation{
				Point:        GeoPoint{Latitude: 0, Longitude: 0},
				LocationName: "stub",
			},
		}},
	}, nil
}

func (r *InMemoryRepository) Dismiss(_ context.Context, cmd dismissCommand) error {
	if normalizeDismissTargetType(cmd.TargetType) == "" || cmd.TargetID == "" || cmd.ViewerUserID == "" {
		return ErrInvalidFeedRequest
	}
	if !r.dismissTargetExists(cmd.TargetType, cmd.TargetID) {
		return ErrDismissTargetNotFound
	}
	key := cmd.ViewerUserID + ":" + cmd.TargetType + ":" + cmd.TargetID
	r.dismissed[key] = struct{}{}
	return nil
}

func (r *InMemoryRepository) dismissTargetExists(targetType, targetID string) bool {
	switch targetType {
	case "moment":
		return targetID == "moment_feed_1" || targetID == "moment_hot_1"
	case "user":
		return targetID == "user_feed_1"
	case "group":
		return targetID == "group_feed_1"
	default:
		return false
	}
}
