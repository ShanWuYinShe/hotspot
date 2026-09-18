package groupmessages

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/idgen"
	"time"

	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) Send(ctx context.Context, req SendGroupMessageRequest) (MessageSentResult, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return MessageSentResult{}, persistence.ErrNotImplemented
	}
	now := time.Now().UTC()
	messageID, err := idgen.WithPrefix("gm_")
	if err != nil {
		return MessageSentResult{}, err
	}
	var senderID interface{}
	if req.SenderUserID != "" {
		senderID = req.SenderUserID
	}
	// 载荷列可空：空串按 NULL 落库，History 侧 COALESCE 还原。
	var shareCode, fileID, replyTo interface{}
	if req.ShareCode != "" {
		shareCode = req.ShareCode
	}
	if req.FileID != "" {
		fileID = req.FileID
	}
	if req.ReplyToMessageID != "" {
		replyTo = req.ReplyToMessageID
	}
	// sent_at 纳秒精度：秒级截断会让同秒消息的排序退化为随机 message_id 兜底，
	// 破坏 History 顺序与已读游标语义（私聊侧同为 RFC3339Nano）。
	sentAt := now.Format(time.RFC3339Nano)
	mentions := req.Mentions
	if mentions == nil {
		mentions = []string{}
	}
	if err := queryable.Exec(ctx, queryInsertGroupMessage, messageID, req.GroupID, senderID, req.Type, req.Content, shareCode, fileID, replyTo, mentions, sentAt); err != nil {
		return MessageSentResult{}, err
	}
	return MessageSentResult{MessageID: messageID, SentAt: sentAt}, nil
}

func (r *PostgresRepository) Update(ctx context.Context, req UpdateGroupMessageRequest) (MessageEventMeta, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	groupID, senderUserID, msgType, isRecalled, sentAt, err := r.lookupMessageMetaFull(ctx, req.MessageID)
	if err != nil {
		return MessageEventMeta{}, err
	}
	if isRecalled {
		return MessageEventMeta{}, ErrMessageRecalled
	}
	if msgType != "text" {
		return MessageEventMeta{}, ErrMessageNotEditable
	}
	// Owner check lives in the service layer via LookupMessageMeta; the SQL
	// `sender_user_id = $2` guard is defense-in-depth (only the sender row can
	// be touched even if the service check is bypassed). 0-row hit on the
	// guarded UPDATE means either a concurrent recall (ErrMessageRecalled) or
	// a non-sender caller (ErrPermissionDenied) — distinguish by the sender
	// we already looked up.
	var updated string
	if err := queryable.QueryRow(ctx, queryUpdateGroupMessage, req.MessageID, req.UserID, req.Content).Scan(&updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if senderUserID != "" && senderUserID != req.UserID {
				return MessageEventMeta{}, ErrPermissionDenied
			}
			return MessageEventMeta{}, ErrMessageRecalled
		}
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{GroupID: groupID, MessageID: req.MessageID, SentAt: sentAt, Content: req.Content}, nil
}

func (r *PostgresRepository) Recall(ctx context.Context, window time.Duration, req RecallGroupMessageRequest) (MessageEventMeta, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	groupID, senderUserID, _, isRecalled, sentAt, err := r.lookupMessageMetaFull(ctx, req.MessageID)
	if err != nil {
		return MessageEventMeta{}, err
	}
	if isRecalled {
		return MessageEventMeta{}, ErrMessageRecalled
	}
	sentTime, parseErr := time.Parse(time.RFC3339Nano, sentAt)
	if parseErr != nil {
		// fail-closed：解析异常的时间窗不得静默放行超时撤回（安全/产品约束）。
		return MessageEventMeta{}, ErrMessageRecallWindow
	}
	// window<=0：撤回禁用（spec chat.md → message_recall_window_seconds 0=禁用）。
	if window <= 0 || time.Since(sentTime) > window {
		return MessageEventMeta{}, ErrMessageRecallWindow
	}
	// Owner check lives in the service layer via LookupMessageMeta; the SQL
	// `sender_user_id = $2` guard is defense-in-depth. 0-row hit on the
	// guarded UPDATE means either a concurrent recall (ErrMessageRecalled) or
	// a non-sender caller (ErrPermissionDenied) — distinguish by the sender
	// we already looked up.
	var updated string
	if err := queryable.QueryRow(ctx, queryRecallGroupMessage, req.MessageID, req.UserID).Scan(&updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			if senderUserID != "" && senderUserID != req.UserID {
				return MessageEventMeta{}, ErrPermissionDenied
			}
			return MessageEventMeta{}, ErrMessageRecalled
		}
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{GroupID: groupID, MessageID: req.MessageID, SentAt: sentAt}, nil
}

// RecallAsAdmin recalls a group message regardless of sender, with no
// recall-window enforcement (admin path, spec api_spec_group.md).
func (r *PostgresRepository) RecallAsAdmin(ctx context.Context, messageID, operatorID string) (MessageEventMeta, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	groupID, _, _, isRecalled, sentAt, err := r.lookupMessageMetaFull(ctx, messageID)
	if err != nil {
		return MessageEventMeta{}, err
	}
	if isRecalled {
		return MessageEventMeta{}, ErrMessageRecalled
	}
	var updated string
	if err := queryable.QueryRow(ctx, queryRecallGroupMessageAsAdminReturning, messageID, operatorID).Scan(&updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MessageEventMeta{}, ErrMessageRecalled
		}
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{GroupID: groupID, MessageID: messageID, SentAt: sentAt}, nil
}

func (r *PostgresRepository) MarkAsRead(ctx context.Context, userID string, req MarkGroupMessagesReadRequest) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if req.UpToMessageID == "" {
		return queryable.Exec(ctx, queryMarkGroupMessagesReadToLatest, userID, req.GroupID)
	}
	var sentAt time.Time
	var messageID string
	if err := queryable.QueryRow(ctx, queryGroupMessageReadStateByMessage, userID, req.GroupID, req.UpToMessageID).Scan(&sentAt, &messageID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMessageNotFound
		}
		return err
	}
	return queryable.Exec(ctx, queryUpsertGroupMessageReadState, req.GroupID, userID, messageID, sentAt)
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string) ([]string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	if err := queryable.Exec(ctx, queryMarkAllGroupMessagesRead, userID); err != nil {
		return nil, err
	}
	rows, err := queryable.Query(ctx, queryUserGroupIDs, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groupIDs := make([]string, 0, 4)
	for rows.Next() {
		var groupID string
		if err := rows.Scan(&groupID); err != nil {
			return nil, err
		}
		groupIDs = append(groupIDs, groupID)
	}
	return groupIDs, rows.Err()
}

func (r *PostgresRepository) GetReadCount(ctx context.Context, groupID string, messageID string) (int, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var count int
	if err := queryable.QueryRow(ctx, queryGroupMessageReadCount, groupID, messageID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// GetReadCounts 批量返回已读成员数（单次往返）；不存在的消息缺席。
func (r *PostgresRepository) GetReadCounts(ctx context.Context, groupID string, messageIDs []string) (map[string]int, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	counts := make(map[string]int, len(messageIDs))
	if len(messageIDs) == 0 {
		return counts, nil
	}
	rows, err := queryable.Query(ctx, queryGroupMessageReadCounts, groupID, messageIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var messageID string
		var count int
		if err := rows.Scan(&messageID, &count); err != nil {
			return nil, err
		}
		counts[messageID] = count
	}
	return counts, rows.Err()
}

func (r *PostgresRepository) LatestMessageID(ctx context.Context, groupID string) (string, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return "", persistence.ErrNotImplemented
	}
	var messageID string
	if err := queryable.QueryRow(ctx, queryLatestGroupMessageID, groupID).Scan(&messageID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrMessageNotFound
		}
		return "", err
	}
	return messageID, nil
}

func (r *PostgresRepository) lookupMessageMetaFull(ctx context.Context, messageID string) (groupID, senderUserID, msgType string, isRecalled bool, sentAt string, err error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return "", "", "", false, "", persistence.ErrNotImplemented
	}
	var recalled bool
	var rawSenderID *string
	if err := queryable.QueryRow(ctx, "SELECT group_id, COALESCE(sender_user_id, ''), type, is_recalled, TO_CHAR(sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"') FROM group_messages WHERE message_id = $1", messageID).Scan(&groupID, &rawSenderID, &msgType, &recalled, &sentAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", "", false, "", ErrMessageNotFound
		}
		return "", "", "", false, "", err
	}
	senderUserID = ""
	if rawSenderID != nil {
		senderUserID = *rawSenderID
	}
	return groupID, senderUserID, msgType, recalled, sentAt, nil
}

func (r *PostgresRepository) FilterMemberIDs(ctx context.Context, groupID string, userIDs []string) ([]string, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryFilterGroupMemberIDs, groupID, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	memberIDs := make([]string, 0, len(userIDs))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		memberIDs = append(memberIDs, id)
	}
	return memberIDs, rows.Err()
}

func (r *PostgresRepository) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var v int
	if err := queryable.QueryRow(ctx, queryCheckGroupMembership, groupID, userID).Scan(&v); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *PostgresRepository) MemberMuteStatus(ctx context.Context, groupID, userID string) (isMuted bool, mutedUntil string, found bool, err error) {
	queryable, ok := r.getQueryable(ctx)
	if !ok {
		return false, "", false, persistence.ErrNotImplemented
	}
	if err := queryable.QueryRow(ctx, queryMemberMuteStatus, groupID, userID).Scan(&isMuted, &mutedUntil); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", false, nil
		}
		return false, "", false, err
	}
	return isMuted, mutedUntil, true, nil
}

func (r *PostgresRepository) GetMessageForReply(ctx context.Context, messageID string) (*MessageBrief, bool, error) {
	queryable, ok := r.getQueryable(ctx)
	if !ok {
		return nil, false, persistence.ErrNotImplemented
	}
	var senderUserID, nickname, avatarURL, msgType, content, shareCode, sentAt string
	err := queryable.QueryRow(ctx, queryGetGroupMessageForReply, messageID).Scan(&senderUserID, &nickname, &avatarURL, &msgType, &content, &shareCode, &sentAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		// 其余错误（连接故障、schema 不匹配）原样上抛：此前统一吞成 not-found，
		// 掩蔽了 share_code 列缺失这类真实故障。
		return nil, false, err
	}
	return &MessageBrief{
		MessageID: messageID,
		Type:      msgType,
		Content:   types.StrPtr(content),
		ShareCode: types.StrPtr(shareCode),
		Sender:    types.UserInfo{UserID: senderUserID, Nickname: nickname, AvatarURL: types.StrPtr(avatarURL)},
		SentAt:    sentAt,
	}, true, nil
}

func (r *PostgresRepository) MessageExists(ctx context.Context, messageID string) (bool, string, error) {
	queryable, ok := r.getQueryable(ctx)
	if !ok {
		return false, "", persistence.ErrNotImplemented
	}
	var senderUserID string
	var ignoreNickname, ignoreAvatar, ignoreType, ignoreContent, ignoreShareCode, ignoreSentAt string
	err := queryable.QueryRow(ctx, queryGetGroupMessageForReply, messageID).Scan(&senderUserID, &ignoreNickname, &ignoreAvatar, &ignoreType, &ignoreContent, &ignoreShareCode, &ignoreSentAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, "", nil
		}
		return false, "", err
	}
	return true, senderUserID, nil
}

func (r *PostgresRepository) MessageGroupID(ctx context.Context, messageID string) (string, bool, error) {
	queryable, ok := r.getQueryable(ctx)
	if !ok {
		return "", false, persistence.ErrNotImplemented
	}
	var groupID string
	err := queryable.QueryRow(ctx, queryGetGroupMessageGroupID, messageID).Scan(&groupID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return groupID, true, nil
}

func (r *PostgresRepository) GetGroupName(ctx context.Context, groupID string) (string, bool, error) {
	queryable, ok := r.getQueryable(ctx)
	if !ok {
		return "", false, persistence.ErrNotImplemented
	}
	var name string
	err := queryable.QueryRow(ctx, queryGetGroupName, groupID).Scan(&name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	return name, true, nil
}

// LookupMessageMeta returns the group + sender metadata used by the service
// layer to authorize Update/Recall (membership + ownership). A NULL
// sender_user_id is returned as "" — the service refuses to mutate such
// messages.
func (r *PostgresRepository) LookupMessageMeta(ctx context.Context, messageID string) (MessageOwnerMeta, bool, error) {
	queryable, ok := r.getQueryable(ctx)
	if !ok {
		return MessageOwnerMeta{}, false, persistence.ErrNotImplemented
	}
	var groupID string
	var rawSenderID *string
	err := queryable.QueryRow(ctx, queryLookupMessageOwner, messageID).Scan(&groupID, &rawSenderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MessageOwnerMeta{}, false, nil
		}
		return MessageOwnerMeta{}, false, err
	}
	senderUserID := ""
	if rawSenderID != nil {
		senderUserID = *rawSenderID
	}
	return MessageOwnerMeta{GroupID: groupID, SenderUserID: senderUserID}, true, nil
}

// GetMessageReportSnapshot 按 message_id 读取群消息真值（举报快照的生成依据，
// spec api_spec_types.md ReportSnapshot：服务端生成，管理员审核基于快照）。
func (r *PostgresRepository) GetMessageReportSnapshot(ctx context.Context, messageID string) (MessageReportSnapshot, bool, error) {
	queryable, ok := r.getQueryable(ctx)
	if !ok {
		return MessageReportSnapshot{}, false, persistence.ErrNotImplemented
	}
	var snapshot MessageReportSnapshot
	err := queryable.QueryRow(ctx, queryGetGroupMessageSnapshot, messageID).Scan(&snapshot.GroupID, &snapshot.Type, &snapshot.Content, &snapshot.FileID, &snapshot.SentAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return MessageReportSnapshot{}, false, nil
		}
		return MessageReportSnapshot{}, false, err
	}
	return snapshot, true, nil
}

func (r *PostgresRepository) getQueryable(ctx context.Context) (persistence.Queryable, bool) {
	return persistence.RepoQueryable(ctx, r.factory)
}

// GetMediaForMessage 取聊天媒体元数据（WS media 负载）。owner 过滤确保只能
// 引用自己上传的 completed 文件（与私聊侧语义一致）。
func (r *PostgresRepository) GetMediaForMessage(ctx context.Context, fileID, ownerUserID string) (map[string]any, bool, error) {
	queryable, ok := r.getQueryable(ctx)
	if !ok {
		return nil, false, persistence.ErrNotImplemented
	}
	var fileIDFound, storageKey, fileName, mimeType, thumbnailURL string
	var fileSize int
	var width, height, duration *int
	err := queryable.QueryRow(ctx, queryGetGroupMediaFileForMessage, fileID, ownerUserID).Scan(&fileIDFound, &storageKey, &fileName, &fileSize, &mimeType, &thumbnailURL, &width, &height, &duration)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	// 键集合与 History media 回填一致（spec MessageMedia 8 键恒定 nullable）。
	return map[string]any{
		"url":           storageKey,
		"thumbnail_url": types.StrPtr(thumbnailURL),
		"width":         width,
		"height":        height,
		"mime_type":     types.StrPtr(mimeType),
		"file_name":     types.StrPtr(fileName),
		"file_size":     fileSize,
		"duration":      duration,
	}, true, nil
}
