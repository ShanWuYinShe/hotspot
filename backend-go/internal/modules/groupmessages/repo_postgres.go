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

func (r *PostgresRepository) Update(ctx context.Context, req UpdateGroupMessageRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Update(ctx, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdateGroupMessage, req.MessageID, req.Content)
}

func (r *PostgresRepository) Recall(ctx context.Context, req RecallGroupMessageRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.Recall(ctx, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryRecallGroupMessage, req.MessageID)
}

func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, req MarkGroupMessagesReadRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.MarkAsRead(ctx, userID, req)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if req.UpToMessageID == "" {
		return queryable.Exec(ctx, queryMarkGroupMessagesReadToLatest, userID, req.GroupID)
	}
	var sentAt string
	var messageID string
	if err := queryable.QueryRow(ctx, queryGroupMessageReadStateByMessage, userID, req.GroupID, req.UpToMessageID).Scan(&sentAt, &messageID); err != nil {
		return ErrMessageNotFound
	}
	return queryable.Exec(ctx, queryUpsertGroupMessageReadState, req.GroupID, userID, messageID, sentAt)
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return r.fallback.MarkAllRead(ctx, userID)
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkAllGroupMessagesRead, userID)
}
