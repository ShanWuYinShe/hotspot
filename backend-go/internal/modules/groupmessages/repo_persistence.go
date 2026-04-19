package groupmessages

import (
	"context"

	"hotspot/backend-go/internal/platform/persistence"
)

type PersistenceRepository struct{ impl Repository }

func NewPersistenceRepository(factory *persistence.Factory) *PersistenceRepository {
	return &PersistenceRepository{impl: NewPostgresRepository(factory)}
}

func (r *PersistenceRepository) Send(ctx context.Context, req SendGroupMessageRequest) (MessageSent, error) {
	return r.impl.Send(ctx, req)
}

func (r *PersistenceRepository) History(ctx context.Context, groupID string, pagination *PaginationParams) (PaginatedGroupMessages, error) {
	return r.impl.History(ctx, groupID, pagination)
}

func (r *PersistenceRepository) Update(ctx context.Context, req UpdateGroupMessageRequest) (MessageEventMeta, error) {
	return r.impl.Update(ctx, req)
}

func (r *PersistenceRepository) Recall(ctx context.Context, req RecallGroupMessageRequest) (MessageEventMeta, error) {
	return r.impl.Recall(ctx, req)
}

func (r *PersistenceRepository) MarkAsRead(ctx context.Context, userID string, req MarkGroupMessagesReadRequest) error {
	return r.impl.MarkAsRead(ctx, userID, req)
}

func (r *PersistenceRepository) MarkAllRead(ctx context.Context, userID string) error {
	return r.impl.MarkAllRead(ctx, userID)
}
