package auth

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRefreshStore struct {
	factory  *persistence.Factory
	fallback RefreshTokenStore
}

func NewPostgresRefreshStore(factory *persistence.Factory, fallback RefreshTokenStore) *PostgresRefreshStore {
	if fallback == nil {
		fallback = NewInMemoryRefreshStore()
	}
	return &PostgresRefreshStore{factory: factory, fallback: fallback}
}

func (s *PostgresRefreshStore) Save(session RefreshSession) { s.fallback.Save(session) }
func (s *PostgresRefreshStore) Rotate(oldToken string, current RefreshSession, next RefreshSession) {
	s.fallback.Rotate(oldToken, current, next)
}

func (s *PostgresRefreshStore) Get(token string) (RefreshSession, bool) {
	queryable, ok := s.factory.DB.(persistence.QueryableDB)
	if !ok {
		return s.fallback.Get(token)
	}
	var session RefreshSession
	var rotatedTo string
	var graceUntil *time.Time
	if err := queryable.QueryRow(context.Background(), queryGetRefreshToken, token).Scan(&session.UserID, &session.SessionID, &session.RefreshToken, &rotatedTo, &graceUntil, &session.ExpiresAt); err != nil {
		return s.fallback.Get(token)
	}
	session.RotatedTo = rotatedTo
	if graceUntil != nil {
		session.GraceUntil = *graceUntil
	}
	return session, true
}
