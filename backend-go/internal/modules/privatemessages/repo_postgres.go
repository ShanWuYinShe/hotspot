package private_messages

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory, _ Repository) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Send(ctx context.Context, senderID, targetUserID string, req SendPrivateMessageRequest) (MessageSent, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MessageSent{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MessageSent{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	messageID := "pm_" + now.Format("20060102150405.000000000")
	chatKey := conversationKey(senderID, targetUserID)
	userAID, userBID := orderedUsers(senderID, targetUserID)
	if err := queryable.Exec(ctx, queryInsertPrivateChat, chatKey, userAID, userBID, now.Format(time.RFC3339Nano)); err != nil {
		return MessageSent{}, err
	}
	if err := queryable.Exec(ctx, queryInsertPrivateMessage, messageID, chatKey, senderID, req.Type, req.Content, req.ShareCode, now.Format(time.RFC3339Nano)); err != nil {
		return MessageSent{}, err
	}
	return MessageSent{MessageID: messageID, SentAt: now.Format(time.RFC3339Nano)}, nil
}
func (r *PostgresRepository) Update(ctx context.Context, userID string, req UpdatePrivateMessageRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdatePrivateMessage, req.MessageID, req.Content)
}
func (r *PostgresRepository) Recall(ctx context.Context, userID string, req RecallPrivateMessageRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryRecallPrivateMessage, req.MessageID)
}
func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, req MarkPrivateMessagesReadRequest) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	chatKey := conversationKey(userID, req.TargetUserID)
	if req.UpToMessageID == "" {
		return queryable.Exec(ctx, queryMarkAllPrivateMessagesReadForChat, chatKey, userID)
	}
	var sentAt string
	var messageID string
	if err := queryable.QueryRow(ctx, queryPrivateMessageReadStateByMessage, chatKey, req.UpToMessageID).Scan(&sentAt, &messageID); err != nil {
		return ErrMessageNotFound
	}
	return queryable.Exec(ctx, queryUpsertPrivateMessageReadState, chatKey, userID, messageID, sentAt)
}
func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryMarkAllPrivateMessagesRead, userID)
}

func orderedUsers(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}
