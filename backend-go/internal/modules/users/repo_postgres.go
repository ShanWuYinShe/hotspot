package users

import (
	"context"
	"fmt"
	"time"

	grouptypes "hotspot/backend-go/internal/modules/groups"
	momenttypes "hotspot/backend-go/internal/modules/moments"
	"hotspot/backend-go/internal/platform/geo"
	"hotspot/backend-go/internal/platform/persistence"
)

type PostgresRepository struct {
	factory *persistence.Factory
}

func NewPostgresRepository(factory *persistence.Factory) *PostgresRepository {
	return &PostgresRepository{factory: factory}
}

func (r *PostgresRepository) PutProfile(ctx context.Context, profile MyProfile) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	if _, ok := r.factory.DB.(persistence.QueryableDB); !ok {
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
		queryable, ok := queryableForTx(txCtx, r.factory)
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
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
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
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpsertUserSettings, userID, settings.PrivacyLevel, settings.LocationVisible, settings.PresenceVisible, settings.SocialListsVisible, settings.GroupsVisible, settings.ReactedMomentsVisible, settings.StatsVisible, settings.AllowFollow, settings.AllowPrivateChat, settings.Searchable, settings.PhoneSearchable, settings.NicknameSearchable, settings.LocationSearchable, settings.NearbyMomentNotify)
}
func (r *PostgresRepository) Follow(ctx context.Context, userID, targetUserID string, source FollowSource, sourceMomentID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	_, blocked, remark, _, blockedAt, blockReason, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, true, blocked, remark, nowRFC3339(), blockedAt, blockReason)
}
func (r *PostgresRepository) Unfollow(ctx context.Context, userID, targetUserID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	_, blocked, remark, _, blockedAt, blockReason, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, false, blocked, remark, nil, blockedAt, blockReason)
}
func (r *PostgresRepository) Block(ctx context.Context, userID, targetUserID, reason string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	following, _, remark, followedAt, _, _, _ := r.getRelationFields(ctx, userID, targetUserID)
	blockedAt := nowRFC3339()
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, true, remark, followedAt, blockedAt, reason)
}
func (r *PostgresRepository) Unblock(ctx context.Context, userID, targetUserID string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	following, _, remark, followedAt, _, _, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, false, remark, followedAt, nil, "")
}
func (r *PostgresRepository) SetRemark(ctx context.Context, userID, targetUserID, remark string) error {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return persistence.ErrNotImplemented
	}
	following, blocked, _, followedAt, blockedAt, blockReason, _ := r.getRelationFields(ctx, userID, targetUserID)
	return queryable.Exec(ctx, queryUpsertUserRelation, userID, targetUserID, following, blocked, remark, followedAt, blockedAt, blockReason)
}

func (r *PostgresRepository) ListBlocked(ctx context.Context, userID string, limit int, cursor string) (PaginatedBlockedUsers, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedBlockedUsers{}, persistence.ErrNotImplemented
	}
	return listBlockedUsersPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListReactedMoments(ctx context.Context, userID string, limit int, cursor, viewerUserID string) (momenttypes.PaginatedMoments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return momenttypes.PaginatedMoments{}, persistence.ErrNotImplemented
	}
	return listReactedMomentsPostgres(ctx, r.factory, userID, limit, cursor, viewerUserID)
}

func (r *PostgresRepository) ListFriends(ctx context.Context, userID string, limit int, cursor string) (PaginatedFriends, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedFriends{}, persistence.ErrNotImplemented
	}
	return listFriendsPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListFollowing(ctx context.Context, userID string, limit int, cursor string) (PaginatedFollowing, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedFollowing{}, persistence.ErrNotImplemented
	}
	return listFollowingPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListFollowers(ctx context.Context, userID string, limit int, cursor string) (PaginatedFollowers, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return PaginatedFollowers{}, persistence.ErrNotImplemented
	}
	return listFollowersPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListGroups(ctx context.Context, userID string, limit int, cursor string) (grouptypes.PaginatedGroups, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return grouptypes.PaginatedGroups{}, persistence.ErrNotImplemented
	}
	return listUserGroupsPostgres(ctx, r.factory, userID, limit, cursor)
}

func (r *PostgresRepository) ListMoments(ctx context.Context, userID string, limit int, cursor string, includePrivate bool) (momenttypes.PaginatedMoments, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return momenttypes.PaginatedMoments{}, persistence.ErrNotImplemented
	}
	return listUserMomentsPostgres(ctx, r.factory, userID, limit, cursor, includePrivate)
}

func (r *PostgresRepository) CommonGroupCount(ctx context.Context, viewerUserID, targetUserID string) (int, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return 0, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
	if !ok {
		return 0, persistence.ErrNotImplemented
	}
	var count int
	if err := queryable.QueryRow(ctx, queryCountCommonGroups, viewerUserID, targetUserID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *PostgresRepository) UserExists(ctx context.Context, userID string) (bool, error) {
	if r.factory == nil || r.factory.DB == nil || r.factory.Config.PostgresDSN == "" {
		return false, persistence.ErrNotImplemented
	}
	queryable, ok := r.factory.DB.(persistence.QueryableDB)
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
func queryableForTx(ctx context.Context, factory *persistence.Factory) (persistence.Queryable, bool) {
	if q, ok := persistence.QueryableFromContext(ctx); ok {
		return q, true
	}
	if factory == nil || factory.DB == nil {
		return nil, false
	}
	q, ok := factory.DB.(persistence.QueryableDB)
	return q, ok
}
