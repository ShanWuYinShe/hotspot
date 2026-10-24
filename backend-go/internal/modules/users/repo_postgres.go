package users

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
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
// 昵称/头像等个人信息替换为"已注销用户"占位并标记 is_deleted，位置与 IP 类
// PII 一并清除；个人数据（登录凭证/会话/设置/关系/收藏/通知/己方私聊视角
// 状态/群成员身份/上传会话）同一事务清理。私聊消息本体保留（对方视角的
// 对话完整性，见 queryDeleteUserChatReadState 注释）。
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
			{queryDeleteUserChatReadState, userID},
			{queryDeleteUserChatUserState, userID},
			{queryDeleteUserGroupMemberships, userID},
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
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	if q, ok := persistence.QueryableFromContext(ctx); ok {
		return q.Exec(ctx, sql, arg)
	}
	return queryable.Exec(ctx, sql, arg)
}

// hasColumns 报告 patch 是否携带任一列级更新。
func (p ProfilePatch) hasColumns() bool {
	return p.Nickname != nil || p.AvatarURL != nil || p.Bio != nil || p.Gender != nil
}

// buildUpdateProfileSQL 构造列级 UPDATE（AGENTS「SQL 构造」：动态列名走
// 固定白名单 if 链，值全部 $N 参数化，禁止用户输入进列名）。
func buildUpdateProfileSQL(userID string, patch ProfilePatch) (string, []any) {
	sets := make([]string, 0, 4)
	args := []any{userID}
	add := func(column string, value any) {
		args = append(args, value)
		sets = append(sets, column+" = $"+strconv.Itoa(len(args)))
	}
	if patch.Nickname != nil {
		add("nickname", *patch.Nickname)
	}
	if patch.AvatarURL != nil {
		add("avatar_url", *patch.AvatarURL)
	}
	if patch.Bio != nil {
		add("bio", *patch.Bio)
	}
	if patch.Gender != nil {
		add("gender", *patch.Gender)
	}
	return "UPDATE users SET " + strings.Join(sets, ", ") + " WHERE user_id = $1 RETURNING 1", args
}

// UpdateProfile 增量更新资料（审查 P1-3）：列级 UPDATE 只写请求携带的列，
// tags 在同一事务内整组替换；行不存在时返回 found=false（调用方走默认
// 建档路径）。tags-only 更新没有列级 UPDATE 可携带 RETURNING，先做一次
// 存在性探测。
func (r *PostgresRepository) UpdateProfile(ctx context.Context, userID string, patch ProfilePatch, tags *[]string) (bool, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return false, persistence.ErrNotImplemented
	}
	if !patch.hasColumns() {
		exists, err := r.UserExists(ctx, userID)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}
	rowMissing := false
	err := persistence.RunInTransaction(ctx, r.factory, func(txCtx context.Context) error {
		queryable, ok := persistence.QueryableForTx(txCtx, r.factory)
		if !ok {
			return persistence.ErrNotImplemented
		}
		if patch.hasColumns() {
			var one int
			updateSQL, updateArgs := buildUpdateProfileSQL(userID, patch)
			if err := queryable.QueryRow(txCtx, updateSQL, updateArgs...).Scan(&one); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					rowMissing = true
					return nil
				}
				return err
			}
		}
		if tags == nil {
			return nil
		}
		if err := queryable.Exec(txCtx, queryReplaceUserProfileTags, userID); err != nil {
			return err
		}
		if len(*tags) == 0 {
			return nil
		}
		return queryable.Exec(txCtx, queryInsertUserProfileTagsBatch, userID, *tags)
	})
	if err != nil {
		return false, err
	}
	return !rowMissing, nil
}

// UpdateProfileLocation 只更新位置四列（含 s2_cell_id 重算），不触碰
// presence/last_online 等运行时列（queryUpdatePresenceStatus 是其唯一
// 运行时写点；此前全量回写会把 online 回写成 offline，审查 P1-3）。
func (r *PostgresRepository) UpdateProfileLocation(ctx context.Context, userID string, loc Location) (bool, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	s2CellID := geo.LeafCellID(geo.Point{Latitude: loc.Point.Latitude, Longitude: loc.Point.Longitude})
	var one int
	if err := queryable.QueryRow(ctx, queryUpdateProfileLocation, userID, loc.LocationName, loc.Point.Latitude, loc.Point.Longitude, s2CellID).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// buildUpdateSettingsSQL 构造设置列级 UPDATE（列名固定白名单 if 链，
// 值全部 $N 参数化）。空 patch 返回 ok=false。
func buildUpdateSettingsSQL(userID string, patch PartialUserSettings) (string, []any, bool) {
	sets := make([]string, 0, 14)
	args := []any{userID}
	addStr := func(column string, value *string) {
		if value == nil {
			return
		}
		args = append(args, *value)
		sets = append(sets, column+" = $"+strconv.Itoa(len(args)))
	}
	addBool := func(column string, value *bool) {
		if value == nil {
			return
		}
		args = append(args, *value)
		sets = append(sets, column+" = $"+strconv.Itoa(len(args)))
	}
	addStr("privacy_level", patch.PrivacyLevel)
	addBool("location_visible", patch.LocationVisible)
	addBool("presence_visible", patch.PresenceVisible)
	addBool("social_lists_visible", patch.SocialListsVisible)
	addBool("groups_visible", patch.GroupsVisible)
	addBool("reacted_moments_visible", patch.ReactedMomentsVisible)
	addBool("stats_visible", patch.StatsVisible)
	addBool("allow_follow", patch.AllowFollow)
	if patch.AllowPrivateChat != nil {
		args = append(args, string(*patch.AllowPrivateChat))
		sets = append(sets, "allow_private_chat = $"+strconv.Itoa(len(args)))
	}
	addBool("searchable", patch.Searchable)
	addBool("phone_searchable", patch.PhoneSearchable)
	addBool("nickname_searchable", patch.NicknameSearchable)
	addBool("location_searchable", patch.LocationSearchable)
	addBool("nearby_moment_notify", patch.NearbyMomentNotify)
	if len(sets) == 0 {
		return "", nil, false
	}
	return "UPDATE user_settings SET " + strings.Join(sets, ", ") + " WHERE user_id = $1 RETURNING 1", args, true
}

// UpdateSettings 增量更新设置（审查 P1-3）：只写 patch 携带的列。设置行
// 不存在时返回 found=false（调用方按默认值建档）。空 patch 直接成功
// （无需触碰库，语义与「未提供字段不修改」一致）。
func (r *PostgresRepository) UpdateSettings(ctx context.Context, userID string, patch PartialUserSettings) (bool, error) {
	sql, args, ok := buildUpdateSettingsSQL(userID, patch)
	if !ok {
		return true, nil
	}
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return false, persistence.ErrNotImplemented
	}
	var one int
	if err := queryable.QueryRow(ctx, sql, args...).Scan(&one); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
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

// SetPresenceStatus 更新运行时 presence（WS 连接生命周期调用：上线 online、
// 最后一条连接断开 offline，SQL 内同步刷新 last_online）。挂具体类型而非
// Repository 大接口：仅 platform/realtime 经 bootstrap adapter 消费，
// memory repo 无需实现（realtime 侧 nil writer 时跳过写库）。
func (r *PostgresRepository) SetPresenceStatus(ctx context.Context, userID string, status UserPresenceStatus) error {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return persistence.ErrNotImplemented
	}
	return queryable.Exec(ctx, queryUpdatePresenceStatus, userID, string(status))
}

// CheckBlockedTags 返回 tags 中命中平台屏蔽标签（admin_blocked_tags）的标签名；空表示无屏蔽标签。
func (r *PostgresRepository) CheckBlockedTags(ctx context.Context, tags []string) ([]string, error) {
	if _, ok := persistence.RepoQueryable(ctx, r.factory); !ok {
		return nil, persistence.ErrNotImplemented
	}
	if len(tags) == 0 {
		return nil, nil
	}
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
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
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		// 基础设施错误原样上抛（审查 P2-1）：此前任何错误都被当作
		//「用户不存在」，DB 抖动会让 follow/list_* 大面积返回 404
		// 而非 10600，故障被业务语义掩盖。
		return false, err
	}
	return true, nil
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// queryableForTx returns the Queryable bound to ctx by RunInTransaction, or
// nil when no transaction is in effect. Mirrors the helper used by groups/
// moments/comments so repo methods transparently join an in-flight tx.

// NearbyMomentNotifyTargets 实现见 repo.go 接口注释；SQL 见
// repo_postgres_queries.go queryNearbyMomentNotifyTargets（包围盒预过滤）。
func (r *PostgresRepository) NearbyMomentNotifyTargets(ctx context.Context, centerLat, centerLng float64, radiusMeters int, excludeUserID string) ([]NearbyNotifyTarget, error) {
	queryable, ok := persistence.RepoQueryable(ctx, r.factory)
	if !ok {
		return nil, persistence.ErrNotImplemented
	}
	latDelta := float64(radiusMeters) / 111320.0
	// 中纬度经度收缩按 cos(纬度) 精确收缩（Go 1.21+ 内建 max）。
	lngDelta := float64(radiusMeters) / (111320.0 * max(0.05, math.Cos(centerLat*math.Pi/180.0)))
	rows, err := queryable.Query(ctx, queryNearbyMomentNotifyTargets,
		excludeUserID, centerLat-latDelta, centerLat+latDelta,
		centerLng-lngDelta, centerLng+lngDelta)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	targets := make([]NearbyNotifyTarget, 0)
	for rows.Next() {
		var t NearbyNotifyTarget
		if err := rows.Scan(&t.UserID, &t.Latitude, &t.Longitude); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}
