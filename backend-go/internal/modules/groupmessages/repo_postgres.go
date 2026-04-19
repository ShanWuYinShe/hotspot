package groupmessages

import (
	"context"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Send(ctx context.Context, req SendGroupMessageRequest) (MessageSent, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MessageSent{}, persistence.ErrNotImplemented
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

func (r *PostgresRepository) Update(ctx context.Context, req UpdateGroupMessageRequest) (MessageEventMeta, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	groupID, sentAt, err := r.lookupMessageMeta(ctx, req.MessageID)
	if err != nil {
		return MessageEventMeta{}, err
	}
	if err := queryable.Exec(ctx, queryUpdateGroupMessage, req.MessageID, req.Content); err != nil {
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{GroupID: groupID, MessageID: req.MessageID, SentAt: sentAt}, nil
}

func (r *PostgresRepository) Recall(ctx context.Context, req RecallGroupMessageRequest) (MessageEventMeta, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	groupID, sentAt, err := r.lookupMessageMeta(ctx, req.MessageID)
	if err != nil {
		return MessageEventMeta{}, err
	}
	if err := queryable.Exec(ctx, queryRecallGroupMessage, req.MessageID); err != nil {
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{GroupID: groupID, MessageID: req.MessageID, SentAt: sentAt}, nil
}

func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, req MarkGroupMessagesReadRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
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
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkAllGroupMessagesRead, userID)
}

func (r *PostgresRepository) lookupMessageMeta(ctx context.Context, messageID string) (string, string, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return "", "", persistence.ErrNotImplemented
	}
	var groupID string
	var sentAt string
	if err := queryable.QueryRow(ctx, "SELECT group_id, TO_CHAR(sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"') FROM group_messages WHERE message_id = $1", messageID).Scan(&groupID, &sentAt); err != nil {
		return "", "", ErrMessageNotFound
	}
	return groupID, sentAt, nil
}
