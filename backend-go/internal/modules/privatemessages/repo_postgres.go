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
	if req.ReplyToMessageID != "" {
		meta, err := r.getMessageMeta(ctx, req.ReplyToMessageID)
		if err != nil || meta.ChatKey != chatKey {
			return MessageSent{}, ErrMessageNotFound
		}
	}
	if err := queryable.Exec(ctx, queryInsertPrivateMessage, messageID, chatKey, senderID, req.Type, req.Content, stringOrNil(req.FileID), req.ShareCode, stringOrNil(req.ReplyToMessageID), now.Format(time.RFC3339Nano)); err != nil {
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
	meta, err := r.getMessageMeta(ctx, req.MessageID)
	if err != nil {
		return ErrMessageNotFound
	}
	if meta.SenderUserID != userID {
		return ErrMessagePermissionDenied
	}
	if meta.Type != "text" || meta.IsRecalled {
		return ErrMessageNotEditable
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
	meta, err := r.getMessageMeta(ctx, req.MessageID)
	if err != nil {
		return ErrMessageNotFound
	}
	if meta.SenderUserID != userID {
		return ErrMessagePermissionDenied
	}
	if meta.IsRecalled {
		return ErrMessageNotRecallable
	}
	sentAt, parseErr := time.Parse(time.RFC3339Nano, meta.SentAt)
	if parseErr == nil && time.Since(sentAt) > 2*time.Minute {
		return ErrMessageNotRecallable
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

type privateMessageMeta struct {
	ChatKey      string
	SenderUserID string
	Type         string
	IsRecalled   bool
	SentAt       string
}

func (r *PostgresRepository) getMessageMeta(ctx context.Context, messageID string) (privateMessageMeta, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return privateMessageMeta{}, persistence.ErrNotImplemented
	}
	var meta privateMessageMeta
	if err := queryable.QueryRow(ctx, queryGetPrivateMessageMeta, messageID).Scan(&meta.ChatKey, &meta.SenderUserID, &meta.Type, &meta.IsRecalled, &meta.SentAt); err != nil {
		return privateMessageMeta{}, err
	}
	return meta, nil
}

func stringOrNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func orderedUsers(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}
