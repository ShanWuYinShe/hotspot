package private_messages

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

func (r *PostgresRepository) Send(ctx context.Context, senderID, targetUserID string, req SendPrivateMessageRequest) MessageSent {
	return r.fallback.Send(ctx, senderID, targetUserID, req)
}
func (r *PostgresRepository) Update(ctx context.Context, userID string, req UpdatePrivateMessageRequest) error {
	return r.fallback.Update(ctx, userID, req)
}
func (r *PostgresRepository) Recall(ctx context.Context, userID string, req RecallPrivateMessageRequest) error {
	return r.fallback.Recall(ctx, userID, req)
}
func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, req MarkPrivateMessagesReadRequest) error {
	return r.fallback.MarkAsRead(ctx, userID, req)
}
func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string) error {
	return r.fallback.MarkAllRead(ctx, userID)
}
