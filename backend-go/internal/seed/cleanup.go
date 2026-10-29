package seed

import (
	"context"
	"fmt"

	"hotspot/backend-go/internal/platform/persistence"
)

// cleanup.go — --reset 模式：按固定前缀物理删除 seed 数据。
//
// 删除顺序遵循外键依赖（先删引用方，最后删 users），并额外删除「非 seed 用户
// 产生、但依附于 seed 实体」的行（例如开发者给 seed 瞬间点的赞、发的评论）：
//   - 有 FK 的（comment/moment_images 等）随父行级联删除；
//   - 无 FK 的（reactions 按 target_id、group_bans 等）显式按前缀/目标 ID 删除。
//
// 不写 schema_migrations（迁移版本表只由 migrate 包维护，见 backend-go/AGENTS.md）。

// cleanup 删除全部 seed 数据，返回删除前库内 seed 媒体对象的 storage_key
// （供调用方随后删除对象存储里的对象）。整个删除在调用方的事务内执行。
func cleanup(ctx context.Context, q persistence.Queryable) ([]string, error) {
	storageKeys, err := listSeedStorageKeys(ctx, q)
	if err != nil {
		return nil, err
	}
	for _, statement := range resetStatements {
		if err := q.Exec(ctx, statement.sql); err != nil {
			return nil, fmt.Errorf("清理 %s 失败: %w", statement.description, err)
		}
	}
	return storageKeys, nil
}

// listSeedStorageKeys 读取当前库内 seed 上传会话的 storage_key（--reset 时同步删对象）。
func listSeedStorageKeys(ctx context.Context, q persistence.Queryable) ([]string, error) {
	rows, err := q.Query(ctx, "SELECT storage_key FROM media_upload_sessions WHERE upload_id LIKE '"+uploadIDPrefix+"%'")
	if err != nil {
		return nil, fmt.Errorf("查询 seed 媒体对象失败: %w", err)
	}
	defer rows.Close()
	keys := make([]string, 0, 64)
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("读取 seed 媒体对象失败: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历 seed 媒体对象失败: %w", err)
	}
	return keys, nil
}

// resetStatement 是一条清理语句及其中文说明（错误信息定位用）。
type resetStatement struct {
	description string
	sql         string
}

// resetStatements 是 --reset 的删除语句，顺序即执行顺序（FK 安全）。
var resetStatements = []resetStatement{
	{
		description: "reactions（按 seed 用户或 seed 目标）",
		sql: `DELETE FROM reactions
WHERE user_id LIKE '` + userIDPrefix + `%'
   OR (target_type = 'moment' AND target_id LIKE '` + momentIDPrefix + `%')
   OR (target_type = 'comment' AND target_id LIKE '` + commentIDPrefix + `%')
   OR (target_type = 'message' AND (target_id LIKE '` + chatMessageIDPrefix + `%' OR target_id LIKE '` + groupMessageIDPrefix + `%'))`,
	},
	{
		description: "comment_likes",
		sql: `DELETE FROM comment_likes
WHERE user_id LIKE '` + userIDPrefix + `%' OR comment_id LIKE '` + commentIDPrefix + `%'`,
	},
	{
		description: "comments",
		sql: `DELETE FROM comments
WHERE user_id LIKE '` + userIDPrefix + `%'
   OR comment_id LIKE '` + commentIDPrefix + `%'
   OR moment_id LIKE '` + momentIDPrefix + `%'`,
	},
	{
		description: "moment_likes",
		sql: `DELETE FROM moment_likes
WHERE user_id LIKE '` + userIDPrefix + `%' OR moment_id LIKE '` + momentIDPrefix + `%'`,
	},
	{
		description: "moment_favorites",
		sql: `DELETE FROM moment_favorites
WHERE user_id LIKE '` + userIDPrefix + `%' OR moment_id LIKE '` + momentIDPrefix + `%'`,
	},
	{
		description: "moment_mentions",
		sql: `DELETE FROM moment_mentions
WHERE moment_id LIKE '` + momentIDPrefix + `%' OR mentioned_user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "moment_tags",
		sql:         `DELETE FROM moment_tags WHERE moment_id LIKE '` + momentIDPrefix + `%'`,
	},
	{
		description: "moment_images",
		sql:         `DELETE FROM moment_images WHERE moment_id LIKE '` + momentIDPrefix + `%'`,
	},
	{
		description: "moments",
		sql: `DELETE FROM moments
WHERE moment_id LIKE '` + momentIDPrefix + `%' OR user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "group_message_read_state",
		sql: `DELETE FROM group_message_read_state
WHERE group_id LIKE '` + groupIDPrefix + `%' OR user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "group_messages",
		sql: `DELETE FROM group_messages
WHERE group_id LIKE '` + groupIDPrefix + `%'
   OR message_id LIKE '` + groupMessageIDPrefix + `%'
   OR sender_user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "group_bans",
		sql: `DELETE FROM group_bans
WHERE group_id LIKE '` + groupIDPrefix + `%' OR user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "group_invites",
		sql: `DELETE FROM group_invites
WHERE invite_id LIKE '` + inviteIDPrefix + `%'
   OR group_id LIKE '` + groupIDPrefix + `%'
   OR inviter_user_id LIKE '` + userIDPrefix + `%'
   OR invitee_user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "group_applications",
		sql: `DELETE FROM group_applications
WHERE application_id LIKE '` + applicationIDPrefix + `%'
   OR group_id LIKE '` + groupIDPrefix + `%'
   OR applicant_user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "group_announcements",
		sql: `DELETE FROM group_announcements
WHERE announcement_id LIKE '` + groupAnnouncementIDPrefix + `%'
   OR group_id LIKE '` + groupIDPrefix + `%'
   OR created_by_user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "group_members",
		sql: `DELETE FROM group_members
WHERE group_id LIKE '` + groupIDPrefix + `%' OR user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "groups",
		sql: `DELETE FROM groups
WHERE group_id LIKE '` + groupIDPrefix + `%' OR owner_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "private_chat_read_state",
		sql: `DELETE FROM private_chat_read_state
WHERE user_id LIKE '` + userIDPrefix + `%' OR chat_key LIKE '%` + userIDPrefix + `%'`,
	},
	{
		description: "private_chat_user_state",
		sql: `DELETE FROM private_chat_user_state
WHERE user_id LIKE '` + userIDPrefix + `%' OR chat_key LIKE '%` + userIDPrefix + `%'`,
	},
	{
		description: "private_messages",
		sql: `DELETE FROM private_messages
WHERE message_id LIKE '` + chatMessageIDPrefix + `%'
   OR sender_user_id LIKE '` + userIDPrefix + `%'
   OR chat_key LIKE '%` + userIDPrefix + `%'`,
	},
	{
		description: "private_chats",
		sql: `DELETE FROM private_chats
WHERE user_a_id LIKE '` + userIDPrefix + `%' OR user_b_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "notifications",
		sql: `DELETE FROM notifications
WHERE notification_id LIKE '` + notifyIDPrefix + `%' OR user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "shares",
		sql: `DELETE FROM shares
WHERE share_id LIKE '` + shareIDPrefix + `%' OR creator_user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "reports",
		sql: `DELETE FROM reports
WHERE report_id LIKE '` + reportIDPrefix + `%' OR creator_user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "admin_user_roles",
		sql: `DELETE FROM admin_user_roles
WHERE user_id LIKE '` + userIDPrefix + `%' OR role_id LIKE '` + adminRoleIDPrefix + `%'`,
	},
	{
		description: "admin_users",
		sql:         `DELETE FROM admin_users WHERE user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "admin_role_permissions",
		sql:         `DELETE FROM admin_role_permissions WHERE role_id LIKE '` + adminRoleIDPrefix + `%'`,
	},
	{
		description: "admin_roles",
		sql:         `DELETE FROM admin_roles WHERE role_id LIKE '` + adminRoleIDPrefix + `%' OR name IN ('内容审核员', '只读观察员')`,
	},
	{
		description: "admin_audit_logs",
		sql: `DELETE FROM admin_audit_logs
WHERE log_id LIKE '` + adminAuditIDPrefix + `%' OR operator_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "admin_blocked_tags",
		sql:         `DELETE FROM admin_blocked_tags WHERE name IN ('广告', '引战')`,
	},
	{
		description: "system_announcements",
		sql: `DELETE FROM system_announcements
WHERE announcement_id LIKE '` + systemAnnouncementIDPrefix + `%' OR created_by_user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "app_configs（仅 seed 写过的 default 行）",
		sql:         `DELETE FROM app_configs WHERE config_id = 'default' AND updated_by_user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "media_upload_sessions",
		sql: `DELETE FROM media_upload_sessions
WHERE upload_id LIKE '` + uploadIDPrefix + `%' OR user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "user_profile_tags",
		sql:         `DELETE FROM user_profile_tags WHERE user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "user_relations",
		sql: `DELETE FROM user_relations
WHERE user_id LIKE '` + userIDPrefix + `%' OR target_user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "user_settings",
		sql:         `DELETE FROM user_settings WHERE user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "auth_accounts",
		sql:         `DELETE FROM auth_accounts WHERE user_id LIKE '` + userIDPrefix + `%' OR login_id LIKE '%@hotspot.local'`,
	},
	{
		description: "auth_sessions",
		sql:         `DELETE FROM auth_sessions WHERE user_id LIKE '` + userIDPrefix + `%'`,
	},
	{
		description: "users（最后删，FK 级联兜底）",
		sql:         `DELETE FROM users WHERE user_id LIKE '` + userIDPrefix + `%'`,
	},
}
