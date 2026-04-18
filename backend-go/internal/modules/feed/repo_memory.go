package feed

import "context"

type InMemoryRepository struct {
	dismissed map[string]struct{}
	targets   map[string]struct{}
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		dismissed: make(map[string]struct{}),
		targets:   make(map[string]struct{}),
	}
}

func (r *InMemoryRepository) RegisterTestTarget(targetType, targetID string) {
	r.targets[targetType+":"+targetID] = struct{}{}
}

func (r *InMemoryRepository) ListMoments(_ context.Context, _ momentsQuery) (PaginatedMoments, error) {
	return PaginatedMoments{}, nil
}

func (r *InMemoryRepository) ListGroups(_ context.Context, _ groupsQuery) (PaginatedGroups, error) {
	return PaginatedGroups{}, nil
}

func (r *InMemoryRepository) ListUsers(_ context.Context, _ usersQuery) (PaginatedUsers, error) {
	return PaginatedUsers{}, nil
}

func (r *InMemoryRepository) ListTrending(_ context.Context, _ string) (TrendingBundle, error) {
	return TrendingBundle{}, nil
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
	if _, ok := r.targets[targetType+":"+targetID]; ok {
		return true
	}
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
