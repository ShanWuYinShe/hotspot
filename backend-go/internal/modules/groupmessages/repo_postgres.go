package groupmessages

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/idgen"
	"time"

	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
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
	messageID, err := idgen.WithPrefix("gm_")
	if err != nil {
		return MessageSent{}, err
	}
	var senderID interface{}
	if req.SenderUserID != "" {
		senderID = req.SenderUserID
	}
	if err := queryable.Exec(ctx, queryInsertGroupMessage, messageID, req.GroupID, senderID, req.Type, req.Content, now.Format(time.RFC3339)); err != nil {
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
	groupID, _, msgType, isRecalled, sentAt, err := r.lookupMessageMetaFull(ctx, req.MessageID)
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
	// be touched even if the service check is bypassed).
	if err := queryable.Exec(ctx, queryUpdateGroupMessage, req.MessageID, req.UserID, req.Content); err != nil {
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{GroupID: groupID, MessageID: req.MessageID, SentAt: sentAt, Content: req.Content}, nil
}

func (r *PostgresRepository) Recall(ctx context.Context, req RecallGroupMessageRequest) (MessageEventMeta, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	groupID, _, _, isRecalled, sentAt, err := r.lookupMessageMetaFull(ctx, req.MessageID)
	if err != nil {
		return MessageEventMeta{}, err
	}
	if isRecalled {
		return MessageEventMeta{}, ErrMessageRecalled
	}
	sentTime, parseErr := time.Parse(time.RFC3339Nano, sentAt)
	if parseErr == nil && time.Since(sentTime) > 2*time.Minute {
		return MessageEventMeta{}, ErrMessageRecallWindow
	}
	// Owner check lives in the service layer via LookupMessageMeta; the SQL
	// `sender_user_id = $2` guard is defense-in-depth.
	if err := queryable.Exec(ctx, queryRecallGroupMessage, req.MessageID, req.UserID); err != nil {
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{GroupID: groupID, MessageID: req.MessageID, SentAt: sentAt}, nil
}

// RecallAsAdmin recalls a group message regardless of sender, with no
// recall-window enforcement (admin path, spec api_spec_group.md).
func (r *PostgresRepository) RecallAsAdmin(ctx context.Context, messageID string) (MessageEventMeta, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return MessageEventMeta{}, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
	if err := queryable.Exec(ctx, queryRecallGroupMessageAsAdmin, messageID); err != nil {
		return MessageEventMeta{}, err
	}
	return MessageEventMeta{GroupID: groupID, MessageID: messageID, SentAt: sentAt}, nil
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
	var sentAt time.Time
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

func (r *PostgresRepository) lookupMessageMetaFull(ctx context.Context, messageID string) (groupID, senderUserID, msgType string, isRecalled bool, sentAt string, err error) {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return "", "", "", false, "", persistence.ErrNotImplemented
	}
	var recalled bool
	var rawSenderID *string
	if err := queryable.QueryRow(ctx, "SELECT group_id, COALESCE(sender_user_id, ''), type, is_recalled, TO_CHAR(sent_at AT TIME ZONE 'UTC', 'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"') FROM group_messages WHERE message_id = $1", messageID).Scan(&groupID, &rawSenderID, &msgType, &recalled, &sentAt); err != nil {
		return "", "", "", false, "", ErrMessageNotFound
	}
	senderUserID = ""
	if rawSenderID != nil {
		senderUserID = *rawSenderID
	}
	return groupID, senderUserID, msgType, recalled, sentAt, nil
}

func (r *PostgresRepository) IsMember(ctx context.Context, groupID, userID string) (bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return false, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
	queryable, ok := r.getQueryable()
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

func (r *PostgresRepository) GetMessageForReply(ctx context.Context, messageID string) (map[string]any, bool, error) {
	queryable, ok := r.getQueryable()
	if !ok {
		return nil, false, persistence.ErrNotImplemented
	}
	var senderUserID, nickname, content, shareCode string
	err := queryable.QueryRow(ctx, queryGetGroupMessageForReply, messageID).Scan(&senderUserID, &nickname, &content, &shareCode)
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

func (r *PostgresRepository) MessageExists(ctx context.Context, messageID string) (bool, string, error) {
	queryable, ok := r.getQueryable()
	if !ok {
		return false, "", persistence.ErrNotImplemented
	}
	var senderUserID string
	err := queryable.QueryRow(ctx, queryGetGroupMessageForReply, messageID).Scan(&senderUserID, new(string), new(string), new(string))
	if err != nil {
		return false, "", nil
	}
	return true, senderUserID, nil
}

func (r *PostgresRepository) MessageGroupID(ctx context.Context, messageID string) (string, bool, error) {
	queryable, ok := r.getQueryable()
	if !ok {
		return "", false, persistence.ErrNotImplemented
	}
	var groupID string
	err := queryable.QueryRow(ctx, queryGetGroupMessageGroupID, messageID).Scan(&groupID)
	if err != nil {
		return "", false, nil
	}
	return groupID, true, nil
}

func (r *PostgresRepository) GetGroupName(ctx context.Context, groupID string) (string, bool, error) {
	queryable, ok := r.getQueryable()
	if !ok {
		return "", false, persistence.ErrNotImplemented
	}
	var name string
	err := queryable.QueryRow(ctx, queryGetGroupName, groupID).Scan(&name)
	if err != nil {
		return "", false, nil
	}
	return name, true, nil
}

// LookupMessageMeta returns the group + sender metadata used by the service
// layer to authorize Update/Recall (membership + ownership). A NULL
// sender_user_id is returned as "" — the service refuses to mutate such
// messages.
func (r *PostgresRepository) LookupMessageMeta(ctx context.Context, messageID string) (MessageOwnerMeta, bool, error) {
	queryable, ok := r.getQueryable()
	if !ok {
		return MessageOwnerMeta{}, false, persistence.ErrNotImplemented
	}
	var groupID string
	var rawSenderID *string
	err := queryable.QueryRow(ctx, queryLookupMessageOwner, messageID).Scan(&groupID, &rawSenderID)
	if err != nil {
		return MessageOwnerMeta{}, false, nil
	}
	senderUserID := ""
	if rawSenderID != nil {
		senderUserID = *rawSenderID
	}
	return MessageOwnerMeta{GroupID: groupID, SenderUserID: senderUserID}, true, nil
}

func (r *PostgresRepository) getQueryable() (persistence.QueryableDB, bool) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return nil, false
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	return queryable, ok
}

func (r *PostgresRepository) GetMediaForMessage(ctx context.Context, fileID string) (map[string]any, bool, error) {
	return nil, false, nil
}
