package moments

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct {
	factory *persistence.Factory
	memory  *InMemoryRepository
}

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	if factory == nil {
		factory = &persistence.Factory{}
	}
	return &PersistenceRepository{factory: factory, memory: NewInMemoryRepository()}
}

func (r *PersistenceRepository) Create(ctx context.Context, userID string, req CreateMomentRequest) MomentItem {
	return r.memory.Create(ctx, userID, req)
}

func (r *PersistenceRepository) IncrementCommentCount(ctx context.Context, momentID string) {
	r.memory.IncrementCommentCount(ctx, momentID)
}
