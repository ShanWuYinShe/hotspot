package auth

type SessionStore interface {
	Save(SessionRecord)
	Touch(sessionID string)
	ListActiveByUser(userID string) []SessionRecord
	Revoke(sessionID string) bool
	RevokeOthers(userID, currentSessionID string) int
	Get(sessionID string) (SessionRecord, bool)
}

type RefreshTokenStore interface {
	Save(session RefreshSession)
	Get(token string) (RefreshSession, bool)
	Rotate(oldToken string, current RefreshSession, next RefreshSession)
}
