package auth

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
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

func (s *PostgresRefreshStore) Save(session RefreshSession) error {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Save(session)
	}
	return queryable.Exec(context.Background(), queryUpsertRefreshToken, session.RefreshToken, session.UserID, session.SessionID, emptyToNull(session.RotatedTo), zeroTimeToPtr(session.GraceUntil), session.ExpiresAt)
}

func (s *PostgresRefreshStore) Rotate(oldToken string, current RefreshSession, next RefreshSession) error {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Rotate(oldToken, current, next)
	}
	graceUntil := time.Now().UTC().Add(30 * time.Second)
	return queryable.Exec(context.Background(), queryRotateRefreshToken, oldToken, next.RefreshToken, graceUntil, next.UserID, next.SessionID, next.ExpiresAt)
}

func (s *PostgresRefreshStore) Get(token string) (RefreshSession, bool, error) {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Get(token)
	}
	var session RefreshSession
	var rotatedTo *string
	var graceUntil *time.Time
	if err := queryable.QueryRow(context.Background(), queryGetRefreshToken, token).Scan(&session.UserID, &session.SessionID, &session.RefreshToken, &rotatedTo, &graceUntil, &session.ExpiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RefreshSession{}, false, nil
		}
		return RefreshSession{}, false, err
	}
	if rotatedTo != nil {
		session.RotatedTo = *rotatedTo
	}
	if graceUntil != nil {
		session.GraceUntil = *graceUntil
	}
	return session, true, nil
}

func (s *PostgresRefreshStore) queryable() (persistence.QueryableDB, bool) {
	if s.factory == nil || s.factory.DB == nil || s.factory.Config.PostgresDSN == "" {
		return nil, false
	}
	queryable, ok := s.factory.DB.(persistence.QueryableDB)
	return queryable, ok
}

func zeroTimeToPtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	v := value.UTC()
	return &v
}

func emptyToNull(value string) *string {
	if value == "" {
		return nil
	}
	v := value
	return &v
}
