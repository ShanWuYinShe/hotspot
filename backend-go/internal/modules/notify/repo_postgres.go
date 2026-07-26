package notifications

import (
	"context"
	"encoding/json"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Append(ctx context.Context, userID string, item NotificationItem) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	item = normalizeNotificationItem(item)
	payloadJSON, err := json.Marshal(item.Payload)
	if err != nil {
		return err
	}
	return queryable.Exec(ctx, queryInsertNotification, item.NotificationID, userID, item.Type, item.IsRead, item.CreatedAt, string(payloadJSON))
}

func (r *PostgresRepository) AppendMany(ctx context.Context, userIDs []string, item NotificationItem) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil
	}
	item = normalizeNotificationItem(item)
	payloadJSON, err := json.Marshal(item.Payload)
	if err != nil {
		return err
	}
	for _, userID := range userIDs {
		if userID == "" {
			continue
		}
		if err := queryable.Exec(ctx, queryInsertNotification, item.NotificationID, userID, item.Type, item.IsRead, item.CreatedAt, string(payloadJSON)); err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, ids []string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkNotificationsRead, userID, ids)
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkAllNotificationsRead, userID)
}
