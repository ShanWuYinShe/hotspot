package users

import (
	"context"

	"hotspot/backend-go/internal/platform/actor"
)

func GetActorInfo(ctx context.Context, provider actor.Provider, userID string) actor.Info {
	if provider == nil {
		return actor.Info{UserID: userID, Nickname: userID}
	}
	info, err := provider.GetActorInfo(ctx, userID)
	if err != nil {
		return actor.Info{UserID: userID, Nickname: userID}
	}
	return info
}
