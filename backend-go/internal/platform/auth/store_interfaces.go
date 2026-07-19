package auth

import "context"

type SessionStore interface {
	Save(ctx context.Context, session SessionRecord) error
	Touch(ctx context.Context, sessionID string) error
	ListActiveByUser(ctx context.Context, userID string) ([]SessionRecord, error)
	Revoke(ctx context.Context, sessionID string) (bool, error)
	RevokeOthers(ctx context.Context, userID, currentSessionID string) (int, error)
	Get(ctx context.Context, sessionID string) (SessionRecord, bool, error)
}

type RefreshTokenStore interface {
	Save(ctx context.Context, session RefreshSession) error
	Get(ctx context.Context, token string) (RefreshSession, bool, error)
	Rotate(ctx context.Context, oldToken string, current RefreshSession, next RefreshSession) error
}
