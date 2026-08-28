package bootstrap

import (
	"context"

	"hotspot/backend-go/internal/modules/groups"
	"hotspot/backend-go/internal/modules/users"
	"hotspot/backend-go/internal/platform/actor"
)

type realtimeUserService struct{ svc *users.Service }

type realtimeUserRepo struct{ repo users.Repository }

type realtimeGroupRepo struct{ repo groups.Repository }

func (a realtimeUserService) OpenTheDoor(ctx context.Context, userID string) (string, bool, error) {
	item, ok, err := a.svc.OpenTheDoor(ctx, userID)
	if !ok || err != nil || item.PresenceStatus == nil {
		return "", ok, err
	}
	return string(*item.PresenceStatus), ok, nil
}

func (a realtimeUserService) GetActorInfo(ctx context.Context, userID string) (actor.Info, error) {
	return a.svc.GetActorInfo(ctx, userID)
}

func (a realtimeUserRepo) GetSettings(ctx context.Context, userID string) (bool, string, bool, error) {
	settings, ok, err := a.repo.GetSettings(ctx, userID)
	return settings.PresenceVisible, string(settings.AllowPrivateChat), ok, err
}

func (a realtimeUserRepo) Relationship(ctx context.Context, userID, targetUserID string) (bool, bool, bool, error) {
	relation, err := a.repo.Relationship(ctx, userID, targetUserID)
	return relation.IsFollowing, relation.IsFollowedBy, relation.IsBlocked, err
}

func (a realtimeUserRepo) BlockedEitherWay(ctx context.Context, aUserID, bUserID string) (callerBlocked, targetBlocked bool, err error) {
	return a.repo.BlockedEitherWay(ctx, aUserID, bUserID)
}

func (a realtimeGroupRepo) ListMembers(ctx context.Context, groupID string, limit int, cursor string) ([]string, string, error) {
	items, nextCursor, err := a.repo.ListMembers(ctx, groupID, nil, limit, cursor)
	if err != nil {
		return nil, "", err
	}
	members := make([]string, 0, len(items))
	for _, item := range items {
		members = append(members, item.User.UserID)
	}
	return members, nextCursor, nil
}

func (a realtimeGroupRepo) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	return a.repo.IsMember(ctx, groupID, userID)
}
