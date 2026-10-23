package auth

import (
	"context"
	"errors"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
	err := queryable.Exec(ctx, queryUpsertSession, session.SessionID, session.UserID, session.CreatedAt, session.LastActiveAt, session.RevokedAt, session.DeviceType, session.DeviceName, session.IPAddress, session.IPLocation, session.DeviceID)
	if isDeviceSessionConflict(err) {
		// 同一设备已有活跃会话（迁移 057 的部分唯一索引）：交给上层撤销后重试，
		// 而不是把 23505 当成 500 抛给客户端。
		return ErrDeviceSessionConflict
	}
	return err
}

// isDeviceSessionConflict 判定唯一索引冲突（23505 + 约束名）。只认本迁移的
// 约束名，避免把其它唯一冲突误判成设备会话冲突。
func isDeviceSessionConflict(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == "uq_auth_sessions_active_device"
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
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.RevokeOthers(ctx, userID, currentSessionID)
	}
	// 用 SQL 实际撤销数（RETURNING 行数）作为返回值：两步 List+UPDATE 之间的
	// 新登录/撤销会让预计算的 count 与实际不符（审计口径失真）。
	var revoked int
	if err := queryable.QueryRow(ctx, `
WITH revoked AS (
  UPDATE auth_sessions SET revoked_at = NOW()
  WHERE user_id = $1 AND session_id <> $2 AND revoked_at IS NULL
  RETURNING 1
)
SELECT COUNT(*) FROM revoked`, userID, currentSessionID).Scan(&revoked); err != nil {
		return 0, err
	}
	return revoked, nil
}

func (s *PostgresSessionStore) Get(ctx context.Context, sessionID string) (SessionRecord, bool, error) {
	queryable, ok := s.queryable()
	if !ok {
		return s.fallback.Get(ctx, sessionID)
	}
	var record SessionRecord
	var revokedAt *time.Time
	if err := queryable.QueryRow(ctx, queryGetSession, sessionID).Scan(&record.SessionID, &record.UserID, &record.CreatedAt, &record.LastActiveAt, &revokedAt, &record.DeviceType, &record.DeviceName, &record.IPAddress, &record.IPLocation, &record.DeviceID); err != nil {
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
		if err := rows.Scan(&record.SessionID, &record.UserID, &record.CreatedAt, &record.LastActiveAt, &revokedAt, &record.DeviceType, &record.DeviceName, &record.IPAddress, &record.IPLocation, &record.DeviceID); err != nil {
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
