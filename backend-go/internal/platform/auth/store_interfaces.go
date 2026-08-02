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

// BanChecker reports whether a user account is currently banned. It is
// implemented by the admin module's repository and injected into the auth
// Service at bootstrap so RequiredAuth can reject banned users with 10201
// (spec/api_spec_errors.md). nil means no ban checking is configured.
type BanChecker interface {
	IsBanned(ctx context.Context, userID string) (bool, error)
}
