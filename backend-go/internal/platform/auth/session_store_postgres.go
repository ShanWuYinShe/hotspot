package auth

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
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

func (s *PostgresSessionStore) Save(session SessionRecord)   { s.fallback.Save(session) }
func (s *PostgresSessionStore) Touch(sessionID string)       { s.fallback.Touch(sessionID) }
func (s *PostgresSessionStore) Revoke(sessionID string) bool { return s.fallback.Revoke(sessionID) }
func (s *PostgresSessionStore) RevokeOthers(userID, currentSessionID string) int {
	return s.fallback.RevokeOthers(userID, currentSessionID)
}

func (s *PostgresSessionStore) Get(sessionID string) (SessionRecord, bool) {
	queryable, ok := s.factory.DB.(persistence.QueryableDB)
	if !ok {
		return s.fallback.Get(sessionID)
	}
	var record SessionRecord
	var revokedAt *time.Time
	if err := queryable.QueryRow(context.Background(), queryGetSession, sessionID).Scan(&record.SessionID, &record.UserID, &record.CreatedAt, &record.LastActiveAt, &revokedAt); err != nil {
		return s.fallback.Get(sessionID)
	}
	record.RevokedAt = revokedAt
	return record, true
}

func (s *PostgresSessionStore) ListActiveByUser(userID string) []SessionRecord {
	queryable, ok := s.factory.DB.(persistence.QueryableDB)
	if !ok {
		return s.fallback.ListActiveByUser(userID)
	}
	rows, err := queryable.Query(context.Background(), queryListActiveSessionsByUser, userID)
	if err != nil {
		return s.fallback.ListActiveByUser(userID)
	}
	defer rows.Close()
	items := []SessionRecord{}
	for rows.Next() {
		var record SessionRecord
		var revokedAt *time.Time
		if err := rows.Scan(&record.SessionID, &record.UserID, &record.CreatedAt, &record.LastActiveAt, &revokedAt); err != nil {
			return s.fallback.ListActiveByUser(userID)
		}
		record.RevokedAt = revokedAt
		items = append(items, record)
	}
	if len(items) == 0 {
		return s.fallback.ListActiveByUser(userID)
	}
	return items
}
