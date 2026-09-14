package users

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) PutProfile(ctx context.Context, profile MyProfile) error {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return persistence.ErrNotImplemented
	}
	// Wrap the profile upsert, the tag delete and the tag (re)insert in a single
	// transaction. Previously a failure mid-loop left the user with all tags
	// deleted and only some re-inserted.
	s2CellID := geo.LeafCellID(geo.Point{Latitude: profile.Location.Point.Latitude, Longitude: profile.Location.Point.Longitude})
	tags := profile.Tags
	if tags == nil {
		tags = []string{}
	}
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := queryable.Exec(txCtx, queryUpsertUserProfile, profile.UserID, profile.Nickname, profile.AvatarURL, profile.UserType, profile.Bio, profile.Gender, profile.Email, profile.Phone, profile.Location.LocationName, profile.Location.Point.Latitude, profile.Location.Point.Longitude, profile.PresenceStatus, profile.LastOnline, profile.CreatedAt, s2CellID); err != nil {
			return err
		}
		if err := queryable.Exec(txCtx, queryReplaceUserProfileTags, profile.UserID); err != nil {
			return err
		}
		if len(tags) == 0 {
			return nil
		}
		return queryable.Exec(txCtx, queryInsertUserProfileTagsBatch, profile.UserID, tags)
	})
}

// DeleteProfile 账户删除匿名化（spec api_spec_user.md「数据保留策略」）：
// users 行保留（user_id 关联不断，已发布内容因不触发级联而保留），
// 昵称/头像等个人信息替换为"已注销用户"占位并标记 is_deleted；
// 个人数据（登录凭证/会话/设置/关系/收藏/通知/私聊记录/上传会话）同一事务清理。
func (r *PostgresRepository) DeleteProfile(ctx context.Context, userID string) error {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return persistence.ErrNotImplemented
	}
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		for _, stmt := range []struct {
			sql string
			arg any
		}{
			{queryAnonymizeUserProfile, userID},
			{queryDeleteAuthAccountByUserID, userID},
			{queryRevokeAllUserSessions, userID},
			{queryDeleteUserSettingsByUserID, userID},
			{queryDeleteUserRelations, userID},
			{queryDeleteUserProfileTagsByUserID, userID},
			{queryDeleteUserFavorites, userID},
			{queryDeleteUserNotifications, userID},
			{queryDeleteUserPrivateChats, userID},
			{queryDeleteUserUploadSessions, userID},
		} {
			if err := r.execInTx(txCtx, stmt.sql, stmt.arg); err != nil {
				return fmt.Errorf("users.DeleteProfile %q: %w", stmt.sql, err)
			}
		}
		return nil
	})
}

func (r *PostgresRepository) execInTx(ctx context.Context, sql string, arg any) error {
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if q, ok := persistence.QueryableFromContext(ctx); ok {
		return q.Exec(ctx, sql, arg)
	}
	return queryable.Exec(ctx, sql, arg)
}
func (r *PostgresRepository) PutSettings(ctx context.Context, userID string, settings UserSettings) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpsertUserSettings, userID, settings.PrivacyLevel, settings.LocationVisible, settings.PresenceVisible, settings.SocialListsVisible, settings.GroupsVisible, settings.ReactedMomentsVisible, settings.StatsVisible, settings.AllowFollow, settings.AllowPrivateChat, settings.Searchable, settings.PhoneSearchable, settings.NicknameSearchable, settings.LocationSearchable, settings.NearbyMomentNotify)
}

// Follow 建立关注关系并报告是否发生状态迁移：SQL 带 `WHERE is_following = FALSE`
// 守卫 + RETURNING，已关注时 0 行命中（ErrNoRows）→ changed=false，调用方跳过
// 重复通知（审计 F-12）。
func (r *PostgresRepository) Follow(ctx context.Context, userID, targetUserID string, source FollowSource, sourceMomentID string) (bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var following bool
	err := queryable.QueryRow(ctx, queryUpsertUserRelationFollow, userID, targetUserID, nowRFC3339()).Scan(&following)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// 已处于关注状态：no-op，不重复通知。
			return false, nil
		}
		return false, err
	}
	return true, nil
}
func (r *PostgresRepository) Unfollow(ctx context.Context, userID, targetUserID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpsertUserRelationUnfollow, userID, targetUserID)
}
func (r *PostgresRepository) Block(ctx context.Context, userID, targetUserID, reason string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	blockedAt := nowRFC3339()
	return queryable.Exec(ctx, queryUpsertUserRelationBlock, userID, targetUserID, blockedAt, reason)
}
func (r *PostgresRepository) BlockCascade(ctx context.Context, userID, targetUserID, reason string) error {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return persistence.ErrNotImplemented
	}
	blockedAt := nowRFC3339()
	// 解除双向关注 + 拉黑三步包进同一事务：避免「关注已解除但未拉黑」的中间状态。
	return persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		q, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if err := q.Exec(txCtx, queryUpsertUserRelationUnfollow, userID, targetUserID); err != nil {
			return err
		}
		if err := q.Exec(txCtx, queryUpsertUserRelationUnfollow, targetUserID, userID); err != nil {
			return err
		}
		return q.Exec(txCtx, queryUpsertUserRelationBlock, userID, targetUserID, blockedAt, reason)
	})
}
func (r *PostgresRepository) Unblock(ctx context.Context, userID, targetUserID string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpsertUserRelationUnblock, userID, targetUserID)
}
func (r *PostgresRepository) SetRemark(ctx context.Context, userID, targetUserID, remark string) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpsertUserRelationRemark, userID, targetUserID, remark)
}

// CheckBlockedTags 返回 tags 中命中平台屏蔽标签（admin_blocked_tags）的标签名；空表示无屏蔽标签。
func (r *PostgresRepository) CheckBlockedTags(ctx context.Context, tags []string) ([]string, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return nil, persistence.ErrNotImplemented
	}
	if len(tags) == 0 {
		return nil, nil
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	rows, err := queryable.Query(ctx, queryListBlockedUserTags, tags)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	blocked := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		blocked = append(blocked, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return blocked, nil
}

func (r *PostgresRepository) ListBlocked(ctx context.Context, userID string, limit int, cursor string) (PaginatedBlockedUsers, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return PaginatedBlockedUsers{}, persistence.ErrNotImplemented
	}
	return listBlockedUsersPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListFriends(ctx context.Context, userID string, limit int, cursor string) (PaginatedFriends, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return PaginatedFriends{}, persistence.ErrNotImplemented
	}
	return listFriendsPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListFollowing(ctx context.Context, userID string, limit int, cursor string) (PaginatedFollowing, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return PaginatedFollowing{}, persistence.ErrNotImplemented
	}
	return listFollowingPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListFollowers(ctx context.Context, userID string, limit int, cursor string) (PaginatedFollowers, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return PaginatedFollowers{}, persistence.ErrNotImplemented
	}
	return listFollowersPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var one int
	if err := queryable.QueryRow(ctx, queryUserExists, userID).Scan(&one); err != nil {
		return false, nil
	}
	return true, nil
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// queryableForTx returns the Queryable bound to ctx by RunInTransaction, or
// nil when no transaction is in effect. Mirrors the helper used by groups/
// moments/comments so repo methods transparently join an in-flight tx.
