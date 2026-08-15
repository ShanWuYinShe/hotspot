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

func (s *PostgresRefreshStore) Save(ctx context.Context, session RefreshSession) error {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Save(ctx, session)
	}
	return queryable.Exec(ctx, queryUpsertRefreshToken, session.RefreshToken, session.UserID, session.SessionID, emptyToNull(session.RotatedTo), zeroTimeToPtr(session.GraceUntil), session.ExpiresAt)
}

func (s *PostgresRefreshStore) Rotate(ctx context.Context, oldToken string, current RefreshSession, next RefreshSession) error {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Rotate(ctx, oldToken, current, next)
	}
	graceUntil := time.Now().UTC().Add(RefreshRotationGrace)
	var rotated string
	if err := queryable.QueryRow(ctx, queryRotateRefreshToken, oldToken, next.RefreshToken, graceUntil, next.UserID, next.SessionID, next.ExpiresAt).Scan(&rotated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// UPDATE 带 rotated_to IS NULL 守卫后 0 行命中：旧 token 已被并发
			// 请求轮换，本次未获得轮换权。
			return ErrRefreshRotationConflict
		}
		return err
	}
	return nil
}

func (s *PostgresRefreshStore) Get(ctx context.Context, token string) (RefreshSession, bool, error) {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Get(ctx, token)
	}
	var session RefreshSession
	var rotatedTo *string
	var graceUntil *time.Time
	if err := queryable.QueryRow(ctx, queryGetRefreshToken, token).Scan(&session.UserID, &session.SessionID, &session.RefreshToken, &rotatedTo, &graceUntil, &session.ExpiresAt); err != nil {
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
