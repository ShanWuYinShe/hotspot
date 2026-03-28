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

func (r *PersistenceRepository) Update(ctx context.Context, req UpdateGroupMessageRequest) error {
	return r.impl.Update(ctx, req)
}

func (r *PersistenceRepository) Recall(ctx context.Context, req RecallGroupMessageRequest) error {
	return r.impl.Recall(ctx, req)
}

func (r *PersistenceRepository) MarkAsRead(ctx context.Context, groupID string, upToMessageID string) error {
	return r.impl.MarkAsRead(ctx, groupID, upToMessageID)
}

func (r *PersistenceRepository) MarkAllRead(ctx context.Context) error {
	return r.impl.MarkAllRead(ctx)
}
