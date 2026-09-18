package users

import (
	"context"
	"errors"
	"hotspot/backend-go/internal/platform/persistence"
	"hotspot/backend-go/internal/platform/types"
	"time"

	"github.com/jackc/pgx/v5"
)

func (r *PostgresRepository) Relationship(ctx context.Context, userID, targetUserID string) (Relationship, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return Relationship{}, persistence.ErrNotImplemented
	}
	following, blocked, _, _, _, _, err := r.getRelationFields(ctx, userID, targetUserID)
	if err != nil {
		return Relationship{}, err
	}
	followedBy, blockedReverse, _, _, _, _, err := r.getRelationFields(ctx, targetUserID, userID)
	if err != nil {
		return Relationship{}, err
	}
	return Relationship{IsFollowing: following, IsFollowedBy: followedBy, IsBlocked: blocked, TargetBlocked: blockedReverse}, nil
}

// MutualFriends 一次查询返回双向共同好友总数 + 前 previewLimit 个预览
// （总数经 COUNT(*) OVER 随行，不被 LIMIT 截断）。替代逐用户全量分页好友
// 列表再内存求交的做法（OpenTheDoor 热路径）。
func (r *PostgresRepository) MutualFriends(ctx context.Context, viewerUserID, targetUserID string, previewLimit int) (int, []UserInfo, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return 0, nil, persistence.ErrNotImplemented
	}
	if previewLimit < 0 {
		previewLimit = 0
	}
	// LIMIT 0 会连总数一起截掉：只想要总数时仍取 1 行读 COUNT(*) OVER。
	fetchLimit := previewLimit
	if fetchLimit < 1 {
		fetchLimit = 1
	}
	rows, err := queryable.Query(ctx, queryMutualFriends, viewerUserID, targetUserID, fetchLimit)
	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()
	total := 0
	preview := make([]UserInfo, 0, previewLimit)
	for rows.Next() {
		var userID, nickname, avatarURL string
		if err := rows.Scan(&userID, &nickname, &avatarURL, &total); err != nil {
			return 0, nil, err
		}
		if len(preview) < previewLimit {
			preview = append(preview, UserInfo{UserID: userID, Nickname: nickname, AvatarURL: types.StrPtr(avatarURL)})
		}
	}
	if err := rows.Err(); err != nil {
		return 0, nil, err
	}
	return total, preview, nil
}

// BlockedEitherWay 一次往返解析 a、b 双向拉黑状态（callerBlocked = a 拉黑 b）。
// 发送/typing 等热路径用它替代两次 Relationship 调用（4 条 SQL → 1 条）。
func (r *PostgresRepository) BlockedEitherWay(ctx context.Context, aUserID, bUserID string) (callerBlocked, targetBlocked bool, err error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return false, false, persistence.ErrNotImplemented
	}
	err = queryable.QueryRow(ctx, queryGetBlockedEitherWay, aUserID, bUserID).Scan(&callerBlocked, &targetBlocked)
	if err != nil {
		return false, false, err
	}
	return callerBlocked, targetBlocked, nil
}

// RelationshipBatch resolves the relationship between fromUserID and every
// entry in toUserIDs in two queries (one for the forward edge, one for the
// reverse edge), rather than one query per target. It replaces the N+1 that
// Search previously triggered by calling Relationship once per profile.
func (r *PostgresRepository) RelationshipBatch(ctx context.Context, fromUserID string, toUserIDs []string) (map[string]Relationship, error) {
	result := make(map[string]Relationship, len(toUserIDs))
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return result, persistence.ErrNotImplemented
	}
	if len(toUserIDs) == 0 {
		return result, nil
	}
	// 本方法会在 privatemessages 的事务闭包内被调用：必须复用 ctx 绑定的事务
	// 连接，直取共享池会在事务内写入相关行时读到未提交一致性之外的旧数据。
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return result, persistence.ErrNotImplemented
	}
	// Initialise defaults so missing rows still produce a zero-value Relationship.
	for _, target := range toUserIDs {
		result[target] = Relationship{}
	}
	// Forward edge: am I following / did I block each target?
	forwardRows, err := queryable.Query(ctx, queryListUserRelationsForward, fromUserID, toUserIDs)
	if err != nil {
		return nil, err
	}
	for forwardRows.Next() {
		var targetUserID string
		var following, blocked bool
		if err := forwardRows.Scan(&targetUserID, &following, &blocked); err != nil {
			forwardRows.Close()
			return nil, err
		}
		rel := result[targetUserID]
		rel.IsFollowing = following
		rel.IsBlocked = blocked
		result[targetUserID] = rel
	}
	if err := forwardRows.Err(); err != nil {
		forwardRows.Close()
		return nil, err
	}
	forwardRows.Close()
	// Reverse edge: is each target following me? did each target block me?
	reverseRows, err := queryable.Query(ctx, queryListUserRelationsReverse, fromUserID, toUserIDs)
	if err != nil {
		return nil, err
	}
	for reverseRows.Next() {
		var sourceUserID string
		var following, blocked bool
		if err := reverseRows.Scan(&sourceUserID, &following, &blocked); err != nil {
			reverseRows.Close()
			return nil, err
		}
		rel := result[sourceUserID]
		rel.IsFollowedBy = following
		rel.TargetBlocked = blocked
		result[sourceUserID] = rel
	}
	if err := reverseRows.Err(); err != nil {
		reverseRows.Close()
		return nil, err
	}
	reverseRows.Close()
	return result, nil
}

func (r *PostgresRepository) getRelationFields(ctx context.Context, userID, targetUserID string) (bool, bool, string, any, any, string, error) {
	// 同 RelationshipBatch：可能在事务闭包内被调用，必须复用 ctx 事务连接。
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return false, false, "", nil, nil, "", persistence.ErrNotImplemented
	}
	var following, blocked bool
	var remark string
	var followedAt, blockedAt *time.Time
	var blockReason string
	if err := queryable.QueryRow(ctx, queryGetUserRelation, userID, targetUserID).Scan(&following, &blocked, &remark, &followedAt, &blockedAt, &blockReason); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, false, "", nil, nil, "", nil
		}
		return false, false, "", nil, nil, "", err
	}
	return following, blocked, remark, timePtrToAny(followedAt), timePtrToAny(blockedAt), blockReason, nil
}

func listBlockedUsersPostgres(ctx context.Context, factory *persistence.Factory, userID string, limit int, cursor string) (PaginatedBlockedUsers, error) {
	queryable, ok := factory.DB.(persistence.QueryableDB)
	if !ok {
		return PaginatedBlockedUsers{}, persistence.ErrNotImplemented
	}
	limit = types.NormalizeLimit(limit)
	cursorAt, cursorID, err := decodeUserCursor(cursor)
	if err != nil {
		return PaginatedBlockedUsers{}, ErrInvalidCursor
	}
	rows, err := queryable.Query(ctx, queryListBlockedUsers, userID, cursorAt, cursorID, limit+1)
	if err != nil {
		return PaginatedBlockedUsers{}, err
	}
	defer rows.Close()
	items := make([]BlockedUserItem, 0, limit+1)
	cursors := make([]string, 0, limit+1)
	for rows.Next() {
		var targetUserID, reason string
		var blockedAt time.Time
		if err := rows.Scan(&targetUserID, &blockedAt, &reason); err != nil {
			return PaginatedBlockedUsers{}, err
		}
		items = append(items, BlockedUserItem{User: UserInfo{UserID: targetUserID, Nickname: targetUserID}, BlockedAt: blockedAt.UTC().Format(time.RFC3339), Reason: types.StrPtr(reason)})
		cursors = append(cursors, encodeUserCursor(blockedAt, targetUserID))
		if len(items) == limit+1 {
			c := cursors[limit-1]
			return PaginatedBlockedUsers{Items: items[:limit], NextCursor: &c}, nil
		}
	}
	if err := rows.Err(); err != nil {
		return PaginatedBlockedUsers{}, err
	}
	return PaginatedBlockedUsers{Items: items}, nil
}

func timePtrToAny(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339)
}
