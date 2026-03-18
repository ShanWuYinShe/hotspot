package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"backend-go/internal/domain"
)

// SessionRepository 封装会话相关持久化操作。
type SessionRepository interface {
	CreateSession(ctx context.Context, s *domain.Session) error
	ListSessionsByUser(ctx context.Context, userID string) ([]domain.Session, error)
	RevokeSession(ctx context.Context, userID, sessionID string) error
	RevokeAllExcept(ctx context.Context, userID, currentSessionID string) (int64, error)
}

type sessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) SessionRepository {
	return &sessionRepository{pool: pool}
}

func (r *sessionRepository) CreateSession(ctx context.Context, s *domain.Session) error {
	const q = `
INSERT INTO auth_sessions (session_id, user_id, device_type, device_name, ip_address, ip_location, created_at, last_active_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`
	_, err := r.pool.Exec(ctx, q, s.SessionID, s.UserID, s.DeviceType, s.DeviceName, s.IPAddress, s.IPLocation, s.CreatedAt, s.LastActiveAt)
	return err
}

func (r *sessionRepository) ListSessionsByUser(ctx context.Context, userID string) ([]domain.Session, error) {
	const q = `
SELECT session_id, user_id, device_type, device_name, ip_address, ip_location, created_at, last_active_at, revoked_at
FROM auth_sessions
WHERE user_id = $1
ORDER BY last_active_at DESC, session_id ASC
`
	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Session
	for rows.Next() {
		var s domain.Session
		if err := rows.Scan(&s.SessionID, &s.UserID, &s.DeviceType, &s.DeviceName, &s.IPAddress, &s.IPLocation, &s.CreatedAt, &s.LastActiveAt, &s.RevokedAt); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *sessionRepository) RevokeSession(ctx context.Context, userID, sessionID string) error {
	const q = `
UPDATE auth_sessions
SET revoked_at = NOW()
WHERE user_id = $1 AND session_id = $2 AND revoked_at IS NULL
`
	_, err := r.pool.Exec(ctx, q, userID, sessionID)
	return err
}

func (r *sessionRepository) RevokeAllExcept(ctx context.Context, userID, currentSessionID string) (int64, error) {
	const q = `
UPDATE auth_sessions
SET revoked_at = NOW()
WHERE user_id = $1 AND session_id <> $2 AND revoked_at IS NULL
`
	cmd, err := r.pool.Exec(ctx, q, userID, currentSessionID)
	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}
