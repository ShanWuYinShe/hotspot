package notifications

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory  *persistence.Factory
	fallback Repository
}

func NewPostgresRepository(factory *persistence.Factory, fallback Repository) *PostgresRepository {
	if fallback == nil {
		fallback = NewInMemoryRepository()
	}
	return &PostgresRepository{factory: factory, fallback: fallback}
}

func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, ids []string) {
	r.fallback.MarkAsRead(ctx, userID, ids)
}
func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string) {
	r.fallback.MarkAllRead(ctx, userID)
}
