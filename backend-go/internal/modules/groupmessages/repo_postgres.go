package groupmessages

import (
	"context"
	"fmt"
	"time"

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

func (r *PostgresRepository) Send(ctx context.Context, req SendGroupMessageRequest) (MessageSent, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Send(ctx, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MessageSent{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	messageID := fmt.Sprintf("gm_%d", now.UnixNano())
	if err := queryable.Exec(ctx, queryInsertGroupMessage, messageID, req.GroupID, nil, req.Type, req.Content, now.Format(time.RFC3339)); err != nil {
		return MessageSent{}, err
	}
	return MessageSent{MessageID: messageID, SentAt: now.Format(time.RFC3339)}, nil
}
