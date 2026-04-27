package actor

import "context"

type Info struct {
	UserID    string
	Nickname  string
	AvatarURL string
}

type Provider interface {
	GetActorInfo(ctx context.Context, userID string) (Info, error)
}
