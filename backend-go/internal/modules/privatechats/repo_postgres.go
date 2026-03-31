package privatechats

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory, _ Repository) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) GetOrCreate(ctx context.Context, userID, targetUserID string) (time.Time, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return time.Time{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return time.Time{}, persistence.ErrNotImplemented
	}
	chatKey := key(userID, targetUserID)
	userAID, userBID := orderedChatUsers(userID, targetUserID)
	now := time.Now().UTC()
	if err := queryable.Exec(ctx, queryInsertPrivateChat, chatKey, userAID, userBID, now.Format(time.RFC3339)); err != nil {
		return time.Time{}, err
	}
	var createdAt time.Time
	if err := queryable.QueryRow(ctx, queryGetPrivateChatCreatedAt, chatKey).Scan(&createdAt); err != nil {
		return time.Time{}, err
	}
	return createdAt, nil
}

func orderedChatUsers(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}
