package auth

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type PostgresSessionStore struct {
	factory  *persistence.Factory
	fallback SessionStore
}

func NewPostgresSessionStore(factory *persistence.Factory, fallback SessionStore) *PostgresSessionStore {
	if fallback == nil {
		fallback = NewInMemorySessionStore()
	}
	return &PostgresSessionStore{factory: factory, fallback: fallback}
}

func (s *PostgresSessionStore) Save(ctx context.Context, session SessionRecord) error {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Save(ctx, session)
	}
	return queryable.Exec(ctx, queryUpsertSession, session.SessionID, session.UserID, session.CreatedAt, session.LastActiveAt, session.RevokedAt, session.DeviceType, session.DeviceName, session.IPAddress, session.IPLocation)
}

func (s *PostgresSessionStore) Touch(ctx context.Context, sessionID string) error {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Touch(ctx, sessionID)
	}
	return queryable.Exec(ctx, queryTouchSession, sessionID)
}

func (s *PostgresSessionStore) Revoke(ctx context.Context, sessionID string) (bool, error) {
	record, ok, err := s.Get(ctx, sessionID)
	if err != nil {
		return false, err
	}
	if !ok || record.RevokedAt != nil {
		return false, nil
	}
	queryable, pg := s.queryable()
	if !pg {
		return s.fallback.Revoke(ctx, sessionID)
	}
	if err := queryable.Exec(ctx, queryRevokeSession, sessionID); err != nil {
		return false, err
	}
	return true, nil
}

func (s *PostgresSessionStore) RevokeOthers(ctx context.Context, userID, currentSessionID string) (int, error) {
	items, err := s.ListActiveByUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, item := range items {
		if item.SessionID == currentSessionID {
			continue
		}
		count++
	}
	if count == 0 {
		return 0, nil
	}
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.RevokeOthers(ctx, userID, currentSessionID)
	}
	if err := queryable.Exec(ctx, queryRevokeOtherSessions, userID, currentSessionID); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *PostgresSessionStore) Get(ctx context.Context, sessionID string) (SessionRecord, bool, error) {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Get(ctx, sessionID)
	}
	var record SessionRecord
	var revokedAt *time.Time
	if err := queryable.QueryRow(ctx, queryGetSession, sessionID).Scan(&record.SessionID, &record.UserID, &record.CreatedAt, &record.LastActiveAt, &revokedAt, &record.DeviceType, &record.DeviceName, &record.IPAddress, &record.IPLocation); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionRecord{}, false, nil
		}
		return SessionRecord{}, false, err
	}
	record.RevokedAt = revokedAt
	return record, true, nil
}

func (s *PostgresSessionStore) ListActiveByUser(ctx context.Context, userID string) ([]SessionRecord, error) {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.ListActiveByUser(ctx, userID)
	}
	rows, err := queryable.Query(ctx, queryListActiveSessionsByUser, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SessionRecord{}
	for rows.Next() {
		var record SessionRecord
		var revokedAt *time.Time
		if err := rows.Scan(&record.SessionID, &record.UserID, &record.CreatedAt, &record.LastActiveAt, &revokedAt, &record.DeviceType, &record.DeviceName, &record.IPAddress, &record.IPLocation); err != nil {
			return nil, err
		}
		record.RevokedAt = revokedAt
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *PostgresSessionStore) queryable() (persistence.QueryableDB, bool) {
	if s.factory == nil || s.factory.DB == nil || s.factory.Config.PostgresDSN == "" {
		return nil, false
	}
	queryable, ok := s.factory.DB.(persistence.QueryableDB)
	return queryable, ok
}
