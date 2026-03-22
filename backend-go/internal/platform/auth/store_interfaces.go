package auth

type SessionStore interface {
	Save(SessionRecord) error
	Touch(sessionID string) error
	ListActiveByUser(userID string) ([]SessionRecord, error)
	Revoke(sessionID string) (bool, error)
	RevokeOthers(userID, currentSessionID string) (int, error)
	Get(sessionID string) (SessionRecord, bool, error)
}

type RefreshTokenStore interface {
	Save(session RefreshSession) error
	Get(token string) (RefreshSession, bool, error)
	Rotate(oldToken string, current RefreshSession, next RefreshSession) error
}
