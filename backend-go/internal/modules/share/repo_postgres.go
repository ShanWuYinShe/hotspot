package share

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

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateShareRequest) ShareCreated {
	return r.fallback.Create(ctx, userID, req)
}
func (r *PostgresRepository) Revoke(ctx context.Context, userID, shareID string) bool {
	return r.fallback.Revoke(ctx, userID, shareID)
}
