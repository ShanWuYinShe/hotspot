package notifications

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	memory := NewInMemoryRepository()
	if factory == nil || factory.Config.PostgresDSN == "" {
		return &PersistenceRepository{impl: memory}
	}
	return &PersistenceRepository{impl: NewPostgresRepository(factory, nil)}
}

func (r *PersistenceRepository) Append(ctx context.Context, userID string, item NotificationItem) error {
	return r.impl.Append(ctx, userID, item)
}

func (r *PersistenceRepository) AppendMany(ctx context.Context, userIDs []string, item NotificationItem) error {
	return r.impl.AppendMany(ctx, userIDs, item)
}

func (r *PersistenceRepository) List(ctx context.Context, userID string, req NotificationListRequest) (PaginatedNotifications, error) {
	return r.impl.List(ctx, userID, req)
}

func (r *PersistenceRepository) UnreadCount(ctx context.Context, userID string) (UnreadCountResult, error) {
	return r.impl.UnreadCount(ctx, userID)
}

func (r *PersistenceRepository) MarkAsRead(ctx context.Context, userID string, ids []string) error {
	return r.impl.MarkAsRead(ctx, userID, ids)
}

func (r *PersistenceRepository) MarkAllRead(ctx context.Context, userID string) error {
	return r.impl.MarkAllRead(ctx, userID)
}
