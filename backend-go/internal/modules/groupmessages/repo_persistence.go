package groupmessages

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
	return &PersistenceRepository{impl: NewPostgresRepository(factory, memory)}
}

func (r *PersistenceRepository) Send(ctx context.Context, req SendGroupMessageRequest) (MessageSent, error) {
	return r.impl.Send(ctx, req)
}

func (r *PersistenceRepository) History(ctx context.Context, groupID string) (PaginatedGroupMessages, error) {
	return r.impl.History(ctx, groupID)
}
