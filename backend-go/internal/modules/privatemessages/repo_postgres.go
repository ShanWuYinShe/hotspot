package private_messages

import (
	"context"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
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
		meta, err := r.getMessageMetaImpl(ctx, req.ReplyToMessageID)
		if err != nil || meta.ChatKey != chatKey {
			return MessageSent{}, ErrMessageNotFound
		}
	}
	if err := queryable.Exec(ctx, queryInsertPrivateMessage, messageID, chatKey, senderID, req.Type, req.Content, stringOrNil(req.FileID), req.ShareCode, stringOrNil(req.ReplyToMessageID), now.Format(time.RFC3339Nano)); err != nil {
		return MessageSent{}, err
	}
	return MessageSent{MessageID: messageID, SentAt: now.Format(time.RFC3339Nano)}, nil
}
func (r *PostgresRepository) Update(ctx context.Context, userID string, req UpdatePrivateMessageRequest) (MessageEventMeta, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	meta, err := r.getMessageMetaImpl(ctx, req.MessageID)
	if err != nil {
		return MessageEventMeta{}, ErrMessageNotFound
	}
	if meta.SenderUserID != userID {
		return MessageEventMeta{}, ErrMessagePermissionDenied
	}
	if meta.Type != "text" || meta.IsRecalled {
		return MessageEventMeta{}, ErrMessageNotEditable
	}
	if err := queryable.Exec(ctx, queryUpdatePrivateMessage, req.MessageID, req.Content); err != nil {
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{TargetUserID: otherUserID(meta.ChatKey, userID), MessageID: req.MessageID, SentAt: meta.SentAt}, nil
}
func (r *PostgresRepository) Recall(ctx context.Context, userID string, req RecallPrivateMessageRequest) (MessageEventMeta, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	meta, err := r.getMessageMetaImpl(ctx, req.MessageID)
	if err != nil {
		return MessageEventMeta{}, ErrMessageNotFound
	}
	if meta.SenderUserID != userID {
		return MessageEventMeta{}, ErrMessagePermissionDenied
	}
	if meta.IsRecalled {
		return MessageEventMeta{}, ErrMessageNotRecallable
	}
	sentAt, parseErr := time.Parse(time.RFC3339Nano, meta.SentAt)
	if parseErr == nil && time.Since(sentAt) > 2*time.Minute {
		return MessageEventMeta{}, ErrMessageNotRecallable
	}
	if err := queryable.Exec(ctx, queryRecallPrivateMessage, req.MessageID); err != nil {
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{TargetUserID: otherUserID(meta.ChatKey, userID), MessageID: req.MessageID, SentAt: meta.SentAt}, nil
}
func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, req MarkPrivateMessagesReadRequest) (MessageEventMeta, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	chatKey := conversationKey(userID, req.TargetUserID)
	if req.UpToMessageID == "" {
		if err := queryable.Exec(ctx, queryMarkAllPrivateMessagesReadForChat, chatKey, userID); err != nil {
			return MessageEventMeta{}, err
		}
		return MessageEventMeta{TargetUserID: req.TargetUserID}, nil
	}
	var sentAt string
	var messageID string
	if err := queryable.QueryRow(ctx, queryPrivateMessageReadStateByMessage, chatKey, req.UpToMessageID).Scan(&sentAt, &messageID); err != nil {
		return MessageEventMeta{}, ErrMessageNotFound
	}
	if err := queryable.Exec(ctx, queryUpsertPrivateMessageReadState, chatKey, userID, messageID, sentAt); err != nil {
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{TargetUserID: req.TargetUserID, MessageID: messageID, SentAt: sentAt}, nil
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

func (r *PostgresRepository) getMessageMetaImpl(ctx context.Context, messageID string) (privateMessageMeta, error) {
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
