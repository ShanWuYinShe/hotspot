package privatemessages

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"hotspot/backend-go/internal/platform/idgen"
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
	return persistence.RepoQueryable(ctx, r.factory)
}

func (r *PostgresRepository) Send(ctx context.Context, senderID, targetUserID string, req SendPrivateMessageRequest) (MessageSentResult, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return MessageSentResult{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	messageID, err := idgen.WithPrefix("pm_")
	if err != nil {
		return MessageSentResult{}, err
	}
	chatKey := conversationKey(senderID, targetUserID)
	userAID, userBID := orderedUsers(senderID, targetUserID)
	// Wrap the chat upsert and the message insert in a single transaction so a
	// failure of the second statement cannot leave an empty conversation row
	// behind. The reply-to validation read runs inside the tx too (it sees the
	// freshly upserted chat and is consistent with it).
	return MessageSentResult{MessageID: messageID, SentAt: now.Format(time.RFC3339Nano)}, persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
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
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
	var updated string
	if err := queryable.QueryRow(ctx, queryUpdatePrivateMessage, req.MessageID, req.Content).Scan(&updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// lookup 与 UPDATE 之间被并发撤回：不得广播假编辑事件，按不可编辑返回。
			return MessageEventMeta{}, ErrMessageNotEditable
		}
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{TargetUserID: otherUserID(meta.ChatKey, userID), MessageID: req.MessageID, SentAt: meta.SentAt, Content: req.Content}, nil
}
func (r *PostgresRepository) Recall(ctx context.Context, userID string, window time.Duration, req RecallPrivateMessageRequest) (MessageEventMeta, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
	// window<=0：撤回禁用（spec chat.md → message_recall_window_seconds 0=禁用）。
	sentAt, parseErr := time.Parse(time.RFC3339Nano, meta.SentAt)
	if parseErr != nil {
		// fail-closed：解析异常的时间窗不得静默放行超时撤回（安全/产品约束）。
		return MessageEventMeta{}, ErrMessageNotRecallable
	}
	if window <= 0 || time.Since(sentAt) > window {
		return MessageEventMeta{}, ErrMessageNotRecallable
	}
	var updated string
	if err := queryable.QueryRow(ctx, queryRecallPrivateMessage, req.MessageID).Scan(&updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 双并发 Recall：后者不得再次广播撤回事件。
			return MessageEventMeta{}, ErrMessageNotRecallable
		}
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{TargetUserID: otherUserID(meta.ChatKey, userID), MessageID: req.MessageID, SentAt: meta.SentAt}, nil
}
func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, req MarkPrivateMessagesReadRequest) (MessageEventMeta, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
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

func (r *PostgresRepository) MessageParticipants(ctx context.Context, messageID string) (string, string, bool, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return "", "", false, persistence.ErrNotImplemented
	}
	var senderUserID, receiverUserID string
	err := queryable.QueryRow(ctx, queryPrivateMessageParticipants, messageID).Scan(&senderUserID, &receiverUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", false, nil
		}
		return "", "", false, err
	}
	return senderUserID, receiverUserID, true, nil
}

func (r *PostgresRepository) MessageExists(ctx context.Context, messageID string) (bool, string, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return false, "", persistence.ErrNotImplemented
	}
	var senderUserID string
	err := queryable.QueryRow(ctx, queryPrivateMessageExists, messageID).Scan(&senderUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, senderUserID, nil
}

// GetMessageReportSnapshot 按 message_id 读取私聊消息真值（举报快照的生成
// 依据，spec api_spec_types.md ReportSnapshot：服务端生成，管理员审核基于
// 快照，不直接查询私聊消息库）。target_chat_id 为相对发送者的会话对方。
func (r *PostgresRepository) GetMessageReportSnapshot(ctx context.Context, messageID string) (MessageReportSnapshot, bool, error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MessageReportSnapshot{}, false, persistence.ErrNotImplemented
	}
	var snapshot MessageReportSnapshot
	err := queryable.QueryRow(ctx, queryGetPrivateMessageSnapshot, messageID).Scan(&snapshot.SenderUserID, &snapshot.TargetUserID, &snapshot.Type, &snapshot.Content, &snapshot.FileID, &snapshot.SentAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MessageReportSnapshot{}, false, nil
		}
		return MessageReportSnapshot{}, false, err
	}
	return snapshot, true, nil
}

func (r *PostgresRepository) CountMessagesByTarget(ctx context.Context, senderID string, since time.Time) (map[string]int, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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

// LockStrangerQuota 取「发送者 + UTC 日期」维度的事务级咨询锁（见
// queryLockStrangerQuota）。锁在事务提交/回滚时自动释放；无事务 ctx 时
// pg_advisory_xact_lock 会话级生效直到连接归还池——调用方必须保证在
// RunInTransaction 事务内调用（service.Send 已保证）。
func (r *PostgresRepository) LockStrangerQuota(ctx context.Context, senderID, dayKey string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	var locked bool
	if err := queryable.QueryRow(ctx, queryLockStrangerQuota, senderID+"\x00"+dayKey).Scan(&locked); err != nil {
		return fmt.Errorf("privatemessages.LockStrangerQuota: %w", err)
	}
	if !locked {
		return fmt.Errorf("privatemessages.LockStrangerQuota: lock not acquired")
	}
	return nil
}
