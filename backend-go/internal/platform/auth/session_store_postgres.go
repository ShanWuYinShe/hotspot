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

func (s *PostgresSessionStore) Save(session SessionRecord) error {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Save(session)
	}
	return queryable.Exec(context.Background(), queryUpsertSession, session.SessionID, session.UserID, session.CreatedAt, session.LastActiveAt, session.RevokedAt, session.DeviceType, session.DeviceName, session.IPAddress, session.IPLocation)
}

func (s *PostgresSessionStore) Touch(sessionID string) error {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Touch(sessionID)
	}
	return queryable.Exec(context.Background(), queryTouchSession, sessionID)
}

func (s *PostgresSessionStore) Revoke(sessionID string) (bool, error) {
	record, ok, err := s.Get(sessionID)
	if err != nil {
		return false, err
	}
	if !ok || record.RevokedAt != nil {
		return false, nil
	}
	queryable, pg := s.queryable()
	if !pg {
		return s.fallback.Revoke(sessionID)
	}
	if err := queryable.Exec(context.Background(), queryRevokeSession, sessionID); err != nil {
		return false, err
	}
	return true, nil
}

func (s *PostgresSessionStore) RevokeOthers(userID, currentSessionID string) (int, error) {
	items, err := s.ListActiveByUser(userID)
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
		return s.fallback.RevokeOthers(userID, currentSessionID)
	}
	if err := queryable.Exec(context.Background(), queryRevokeOtherSessions, userID, currentSessionID); err != nil {
		return 0, err
	}
	return count, nil
}

func (s *PostgresSessionStore) Get(sessionID string) (SessionRecord, bool, error) {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Get(sessionID)
	}
	var record SessionRecord
	var revokedAt *time.Time
	if err := queryable.QueryRow(context.Background(), queryGetSession, sessionID).Scan(&record.SessionID, &record.UserID, &record.CreatedAt, &record.LastActiveAt, &revokedAt, &record.DeviceType, &record.DeviceName, &record.IPAddress, &record.IPLocation); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionRecord{}, false, nil
		}
		return SessionRecord{}, false, err
	}
	record.RevokedAt = revokedAt
	return record, true, nil
}

func (s *PostgresSessionStore) ListActiveByUser(userID string) ([]SessionRecord, error) {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.ListActiveByUser(userID)
	}
	rows, err := queryable.Query(context.Background(), queryListActiveSessionsByUser, userID)
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
