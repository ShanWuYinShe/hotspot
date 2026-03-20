package moments

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

func (r *PostgresRepository) Create(ctx context.Context, userID string, req CreateMomentRequest) MomentItem {
	return r.fallback.Create(ctx, userID, req)
}
func (r *PostgresRepository) Get(ctx context.Context, momentID string) (MomentItem, bool) {
	return r.fallback.Get(ctx, momentID)
}
func (r *PostgresRepository) ListMine(ctx context.Context, userID string) PaginatedMoments {
	return r.fallback.ListMine(ctx, userID)
}
func (r *PostgresRepository) IncrementCommentCount(ctx context.Context, momentID string) {
	r.fallback.IncrementCommentCount(ctx, momentID)
}
