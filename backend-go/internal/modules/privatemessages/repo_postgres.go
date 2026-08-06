package privatemessages

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

// txQueryable returns the transaction-bound Queryable when one is in effect
// (real pgx tx via RunInTransaction), otherwise falls back to the shared pool.
// The fallback matters for tests, where RunInTransaction detects a stub backend
// and runs fn without injecting a tx.
func (r *PostgresRepository) txQueryable(ctx context.Context) (persistence.Queryable, bool) {
	if q, ok := persistence.QueryableFromContext(ctx); ok {
		return q, true
	}
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, false
	}
	q, ok := r.factory.DB.(persistence.QueryableDB)
	return q, ok
}

func (r *PostgresRepository) Send(ctx context.Context, senderID, targetUserID string, req SendPrivateMessageRequest) (MessageSent, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MessageSent{}, persistence.ErrNotImplemented
	}
	if _, ok := r.factory.DB.(persistence.QueryableDB); !ok {
		return MessageSent{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	messageID := "pm_" + now.Format("20060102150405.000000000")
	chatKey := conversationKey(senderID, targetUserID)
	userAID, userBID := orderedUsers(senderID, targetUserID)
	// Wrap the chat upsert and the message insert in a single transaction so a
	// failure of the second statement cannot leave an empty conversation row
	// behind. The reply-to validation read runs inside the tx too (it sees the
	// freshly upserted chat and is consistent with it).
	return MessageSent{MessageID: messageID, SentAt: now.Format(time.RFC3339Nano)}, persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := r.txQueryable(txCtx)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := queryable.Exec(txCtx, queryInsertPrivateChat, chatKey, userAID, userBID, now.Format(time.RFC3339Nano)); err != nil {
			return err
		}
		if req.ReplyToMessageID != "" {
			meta, err := r.getMessageMetaImpl(txCtx, req.ReplyToMessageID)
			if err != nil || meta.ChatKey != chatKey {
				return ErrMessageNotFound
			}
		}
		return queryable.Exec(txCtx, queryInsertPrivateMessage, messageID, chatKey, senderID, req.Type, req.Content, stringOrNil(req.FileID), req.ShareCode, stringOrNil(req.ReplyToMessageID), now.Format(time.RFC3339Nano))
	})
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
	return MessageEventMeta{TargetUserID: otherUserID(meta.ChatKey, userID), MessageID: req.MessageID, SentAt: meta.SentAt, Content: req.Content}, nil
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
	var sentAt time.Time
	var messageID string
	if err := queryable.QueryRow(ctx, queryPrivateMessageReadStateByMessage, chatKey, req.UpToMessageID).Scan(&sentAt, &messageID); err != nil {
		return MessageEventMeta{}, ErrMessageNotFound
	}
	if err := queryable.Exec(ctx, queryUpsertPrivateMessageReadState, chatKey, userID, messageID, sentAt); err != nil {
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{TargetUserID: req.TargetUserID, MessageID: messageID, SentAt: sentAt.Format(time.RFC3339Nano)}, nil
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
	// Prefer the tx-bound queryable so the reply-to read is consistent with
	// the surrounding transaction; fall back to the pool otherwise.
	queryable, ok := persistence.QueryableFromContext(ctx)
	if !ok {
		queryable, ok = r.factory.DB.(persistence.QueryableDB)
	}
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

func (r *PostgresRepository) GetMessageForReply(ctx context.Context, messageID, viewerUserID string) (map[string]any, bool, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, false, persistence.ErrNotImplemented
	}
	var senderUserID, nickname, content, shareCode string
	err := queryable.QueryRow(ctx, queryGetPrivateMessageForReply, messageID, viewerUserID).Scan(&senderUserID, &nickname, &content, &shareCode)
	if err != nil {
		return nil, false, nil
	}
	result := map[string]any{
		"message_id": messageID,
		"user_id":    senderUserID,
		"nickname":   nickname,
		"content":    content,
	}
	if shareCode != "" {
		result["share_code"] = shareCode
	}
	return result, true, nil
}

func (r *PostgresRepository) GetMediaForMessage(ctx context.Context, fileID, ownerUserID string) (map[string]any, bool, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, false, persistence.ErrNotImplemented
	}
	var fileIDFound, storageKey, fileName, mimeType string
	var fileSize int
	err := queryable.QueryRow(ctx, queryGetMediaFileForMessage, fileID, ownerUserID).Scan(&fileIDFound, &storageKey, &fileName, &fileSize, &mimeType)
	if err != nil {
		return nil, false, nil
	}
	result := map[string]any{
		"url":       storageKey,
		"file_name": fileName,
		"file_size": fileSize,
		"mime_type": mimeType,
	}
	return result, true, nil
}

func orderedUsers(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

func (r *PostgresRepository) MessageExists(ctx context.Context, messageID string) (bool, string, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, "", persistence.ErrNotImplemented
	}
	var senderUserID string
	err := queryable.QueryRow(ctx, queryPrivateMessageExists, messageID).Scan(&senderUserID)
	if err != nil {
		return false, "", nil
	}
	return true, senderUserID, nil
}

func (r *PostgresRepository) CountMessagesByTarget(ctx context.Context, senderID string, since time.Time) (map[string]int, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryCountPrivateMessagesByTarget, senderID, since.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var targetUserID string
		var count int
		if err := rows.Scan(&targetUserID, &count); err != nil {
			return nil, err
		}
		counts[targetUserID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}
