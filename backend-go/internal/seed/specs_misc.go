package seed

import (
	"strings"
	"time"
)

// specs_misc.go — 分享/举报/通知/管理端规格（seed-spec §3.5-§3.8）。

// 分享码：shares 表 share_code 的契约要求（api_spec_share.md「分享码规则」）
// 为 12 位、字符集 [A-HJ-NP-Z2-9]（排除 I/O/0/1）。seed-spec 里的 'abc'/'def'
// 只有 3 位，会被 share/open_the_door 的参数校验拒绝（10000），因此这里改用
// 合法且可读的固定码；QA 页面 /share/<code> 直接使用下面两个值。
const (
	shareCodeMomentSeed = "SEEDSUNSET23"
	shareCodeUserSeed   = "SEEDUSERPAGE"
)

// seedShares 返回 2 条分享（规格 §3.6）：moment_seed_01 与 user_seed_lumos。
// 备注：shares 表迁移 006 无 view_count 列（规格文档提到「含 view_count 等」
// 与表结构不符），以迁移为准不写入该列。
func seedShares() []shareSpec {
	return []shareSpec{
		{
			ShareID: "share_seed_1", ShareCode: shareCodeMomentSeed, CreatorID: "user_seed_lumos",
			ShareType: "moment", Payload: map[string]any{"shared_moment_id": "moment_seed_01"},
			CreatedAgo: 20 * oneHour,
		},
		{
			ShareID: "share_seed_2", ShareCode: shareCodeUserSeed, CreatorID: "user_seed_admin",
			ShareType: "user", Payload: map[string]any{"shared_user_id": "user_seed_lumos"},
			CreatedAgo: 5 * oneHour,
		},
	}
}

// seedReports 返回 4 条举报（规格 §3.7）：目标 user/moment/group/message，
// 状态覆盖 pending/resolved/dismissed（reports.status 真实枚举见 spec
// api_spec_types.md：'pending' | 'resolved' | 'dismissed'；规格文档里写的
// handled/rejected 不是有效值，以契约为准）。
func seedReports() []reportSpec {
	return []reportSpec{
		{
			ReportID: "report_seed_1", CreatorID: "user_seed_lumos", TargetType: "user", TargetID: "user_seed_banned",
			Reason: "harassment", Description: "该账号在私聊中反复发送骚扰信息。", Status: "pending",
			CreatedAgo: 6 * oneHour,
		},
		{
			ReportID: "report_seed_2", CreatorID: "user_seed_admin", TargetType: "moment", TargetID: "moment_seed_07",
			Reason: "spam", Description: "疑似广告/引流内容。", Status: "resolved",
			CreatedAgo: 2 * oneDay, HandledAgo: oneDay, HandledBy: "user_seed_admin", HandleNote: "已删除内容并警告作者。",
		},
		{
			ReportID: "report_seed_3", CreatorID: "user_seed_lumos", TargetType: "group", TargetID: "group_seed_3",
			Reason: "other", Description: "群介绍信息不完整。", Status: "dismissed",
			CreatedAgo: 3 * oneDay, HandledAgo: 2 * oneDay, HandledBy: "user_seed_admin", HandleNote: "经复核无违规。",
		},
		{
			ReportID: "report_seed_4", CreatorID: "user_seed_admin", TargetType: "message", TargetID: "chat_seed_marco_03",
			Reason: "other", Description: "私聊消息内容疑似违规。", Status: "pending",
			CreatedAgo: 4 * oneHour,
			MessageSnapshot: func(baseTime time.Time) map[string]any {
				return map[string]any{
					"chat_type":        "private",
					"target_chat_id":   "user_seed_lumos", // 相对发送者（marco）的会话对方
					"message_type":     "text",
					"snapshot_content": "好，暗房还缺显影液。",
					"snapshot_media":   nil,
					"sent_at":          baseTime.Add(-6 * oneHour).Format(time.RFC3339),
				}
			},
		},
	}
}

// payloadContext 是通知 payload 构建所需的跨行引用（昵称/头像/摘要/群名）。
type payloadContext struct {
	baseTime        time.Time
	users           map[string]userSpec
	groups          map[string]groupSpec
	moments         map[string]momentSpec
	userAvatarURLs  map[string]string // user_id → avatar_url（无头像为 ""）
	groupAvatarURLs map[string]string // group_id → avatar_url
}

// userInfo 生成 spec UserInfo（api_spec_types.md）：avatar_url 为 nullable。
func (c payloadContext) userInfo(userID string) map[string]any {
	user, ok := c.users[userID]
	if !ok {
		return map[string]any{"user_id": userID, "nickname": userID, "avatar_url": nil}
	}
	info := map[string]any{"user_id": user.UserID, "nickname": user.Nickname}
	if url := c.userAvatarURLs[userID]; url != "" {
		info["avatar_url"] = url
	} else {
		info["avatar_url"] = nil
	}
	return info
}

// momentSummary 生成 spec MomentSummary。cover_url 需要短时签名 URL（由
// media 服务在读取时生成），seed 的静态 payload 无法预签名，置 null
// （契约允许 nullable，客户端按占位/纯文字卡片渲染）。
func (c payloadContext) momentSummary(momentID string) map[string]any {
	moment, ok := c.moments[momentID]
	if !ok {
		return map[string]any{"moment_id": momentID, "description_preview": "", "cover_url": nil}
	}
	return map[string]any{
		"moment_id":           moment.MomentID,
		"description_preview": previewText(moment.Description, 60),
		"cover_url":           nil,
	}
}

// groupNameAndAvatar 返回群名与群头像 URL（nullable）。
func (c payloadContext) groupNameAndAvatar(groupID string) (string, any) {
	group, ok := c.groups[groupID]
	if !ok {
		return groupID, nil
	}
	if avatar := c.groupAvatarURLs[groupID]; avatar != "" {
		return group.Name, avatar
	}
	return group.Name, nil
}

// previewText 把文本截断为至多 maxRunes 个字符（按 rune，避免中文被截半）。
func previewText(text string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "…"
}

// seedNotifications 返回 9 条通知，全部发给 lumos（规格 §3.5）。
// 未读 4 条：new_follower / reaction_added / moment_commented / group_invite_received，
// notify/unread_summary 由 notifications 表实时聚合，与列表天然同源。
func seedNotifications() []notifySpec {
	return []notifySpec{
		{
			NotifyID: "notify_seed_01", UserID: "user_seed_lumos", Type: "new_follower",
			IsRead: false, CreatedAgo: 30 * oneMinute,
			Payload: func(c payloadContext) map[string]any {
				return map[string]any{
					"type":             "new_follower",
					"actor":            c.userInfo("user_seed_lin"),
					"source":           "profile",
					"source_moment_id": nil,
				}
			},
		},
		{
			NotifyID: "notify_seed_02", UserID: "user_seed_lumos", Type: "mutual_follow",
			IsRead: true, CreatedAgo: 2 * oneHour,
			Payload: func(c payloadContext) map[string]any {
				return map[string]any{"type": "mutual_follow", "actor": c.userInfo("user_seed_sophia")}
			},
		},
		{
			NotifyID: "notify_seed_03", UserID: "user_seed_lumos", Type: "reaction_added",
			IsRead: false, CreatedAgo: 3 * oneHour,
			Payload: func(c payloadContext) map[string]any {
				return map[string]any{
					"type":        "reaction_added",
					"actor":       c.userInfo("user_seed_marco"),
					"target_type": "moment",
					"target_id":   "moment_seed_01",
					"emoji":       emojiHeart,
				}
			},
		},
		{
			NotifyID: "notify_seed_04", UserID: "user_seed_lumos", Type: "moment_commented",
			IsRead: false, CreatedAgo: 5 * oneHour,
			Payload: func(c payloadContext) map[string]any {
				return map[string]any{
					"type":            "moment_commented",
					"actor":           c.userInfo("user_seed_sophia"),
					"moment_id":       "moment_seed_01",
					"moment_summary":  c.momentSummary("moment_seed_01"),
					"comment_id":      "comment_seed_01_01",
					"comment_preview": "光影真的好，求机位！",
				}
			},
		},
		{
			NotifyID: "notify_seed_05", UserID: "user_seed_lumos", Type: "comment_reply",
			IsRead: true, CreatedAgo: 8 * oneHour,
			Payload: func(c payloadContext) map[string]any {
				return map[string]any{
					"type":                    "comment_reply",
					"actor":                   c.userInfo("user_seed_marco"),
					"comment_id":              "comment_seed_01_01",
					"replied_comment_preview": "光影真的好，求机位！",
					"reply_id":                "comment_seed_01_02",
					"reply_content_preview":   "云龙湖北岸那个木栈道。",
					"moment_id":               "moment_seed_01",
					"moment_summary":          c.momentSummary("moment_seed_01"),
				}
			},
		},
		{
			NotifyID: "notify_seed_06", UserID: "user_seed_lumos", Type: "group_invite_received",
			IsRead: false, CreatedAgo: oneDay,
			Payload: func(c payloadContext) map[string]any {
				name, avatar := c.groupNameAndAvatar("group_seed_4")
				message := "夜读角本周读《城与不确定的墙》，来吗？"
				return map[string]any{
					"type":             "group_invite_received",
					"actor":            c.userInfo("user_seed_chen"),
					"group_id":         "group_seed_4",
					"group_name":       name,
					"group_avatar_url": avatar,
					"invite_id":        "invite_seed_1",
					"message":          message,
				}
			},
		},
		{
			NotifyID: "notify_seed_07", UserID: "user_seed_lumos", Type: "group_application_rejected",
			IsRead: true, CreatedAgo: 2 * oneDay,
			Payload: func(c payloadContext) map[string]any {
				name, _ := c.groupNameAndAvatar("group_seed_4")
				return map[string]any{
					"type":           "group_application_rejected",
					"group_id":       "group_seed_4",
					"group_name":     name,
					"application_id": "application_seed_2",
					"reason":         "本期读书会名额已满",
				}
			},
		},
		{
			NotifyID: "notify_seed_08", UserID: "user_seed_lumos", Type: "nearby_new_moment",
			IsRead: true, CreatedAgo: 3 * oneDay,
			Payload: func(c payloadContext) map[string]any {
				return map[string]any{
					"type":           "nearby_new_moment",
					"actor":          c.userInfo("user_seed_mika"),
					"moment_id":      "moment_seed_14",
					"moment_summary": c.momentSummary("moment_seed_14"),
					"distance":       520,
				}
			},
		},
		{
			NotifyID: "notify_seed_09", UserID: "user_seed_lumos", Type: "system_announcement",
			IsRead: true, CreatedAgo: 5 * oneDay,
			Payload: func(c payloadContext) map[string]any {
				return map[string]any{
					"type":            "system_announcement",
					"announcement_id": "sysann_seed_1",
					"title":           "社区公约更新提醒",
					"severity":        "info",
				}
			},
		},
	}
}

// seedAdminRoles 返回 2 个管理端角色（规格 §3.8）。
// 权限 key 以 platform/rbac 常量为准（如 admin.reports.read），不是规格文档
// 里的简写形式（reports:read）。
func seedAdminRoles() []adminRoleSpec {
	return []adminRoleSpec{
		{
			RoleID: adminRoleIDPrefix + "moderator", Name: "内容审核员",
			Description: "审核举报、治理标签（seed 数据）",
			Permissions: []string{
				"admin.reports.read", "admin.reports.write",
				"admin.tags.read", "admin.tags.write",
				"admin.users.read", "admin.groups.read", "admin.audit.read",
			},
			CreatedAgo: 50 * oneDay,
		},
		{
			RoleID: adminRoleIDPrefix + "viewer", Name: "只读观察员",
			Description: "只读查看各管理页面（seed 数据）",
			Permissions: []string{
				"admin.reports.read", "admin.app_config.read", "admin.tags.read",
				"admin.announcements.read", "admin.users.read", "admin.groups.read",
				"admin.audit.read", "admin.stats.read",
			},
			CreatedAgo: 50 * oneDay,
		},
	}
}

// seedAdminUsers 返回管理端授权：admin 全权限（grant_all）+ viewer 角色；
// lumos 不授权（普通用户视角，用于验证非管理员访问管理页被拒）。
func seedAdminUsers() []adminUserSpec {
	return []adminUserSpec{
		{UserID: "user_seed_admin", GrantAll: true, RoleIDs: []string{adminRoleIDPrefix + "viewer"}, GrantedAgo: 100 * oneDay},
	}
}

// seedAdminAudits 返回 6 条审计日志（不同 action/operator/target，近 7 天）。
func seedAdminAudits() []auditLogSpec {
	operator := "user_seed_admin"
	return []auditLogSpec{
		{
			LogID: "audit_seed_1", Action: "user.ban", OperatorID: operator,
			TargetType: "user", TargetID: "user_seed_banned", TargetName: "已封禁用户",
			Details:    map[string]any{"type": "user.ban", "reason": "发布违规内容（seed）", "duration_days": 7},
			IPLocation: "江苏 徐州", IPAddress: "127.0.0.1", DeviceSummary: "macOS · Chrome",
			CreatedAgo: 3 * oneDay,
		},
		{
			LogID: "audit_seed_2", Action: "report.judge", OperatorID: operator,
			TargetType: "report", TargetID: "report_seed_2", TargetName: "report_seed_2",
			Details:    map[string]any{"type": "report.judge", "report_id": "report_seed_2", "action": "resolve", "note": "已删除内容并警告作者。"},
			CreatedAgo: oneDay,
		},
		{
			LogID: "audit_seed_3", Action: "role.create", OperatorID: operator,
			TargetType: "role", TargetID: adminRoleIDPrefix + "moderator", TargetName: "内容审核员",
			Details: map[string]any{
				"type": "role.create", "role_name": "内容审核员",
				"permissions": []string{"admin.reports.read", "admin.reports.write", "admin.tags.read", "admin.tags.write", "admin.users.read", "admin.groups.read", "admin.audit.read"},
			},
			CreatedAgo: 50 * oneDay,
		},
		{
			LogID: "audit_seed_4", Action: "announcement.create", OperatorID: operator,
			TargetType: "announcement", TargetID: "sysann_seed_1", TargetName: "社区公约更新提醒",
			Details:    map[string]any{"type": "announcement.create", "title": "社区公约更新提醒"},
			CreatedAgo: 3 * oneDay,
		},
		{
			LogID: "audit_seed_5", Action: "tag.set_ban", OperatorID: operator,
			TargetType: "tag", TargetID: "广告", TargetName: "广告",
			Details:    map[string]any{"type": "tag.set_ban", "name": "广告", "active": true},
			CreatedAgo: 7 * oneDay,
		},
		{
			LogID: "audit_seed_6", Action: "config.update", OperatorID: operator,
			TargetType: "config", TargetID: "default", TargetName: "default",
			Details:    map[string]any{"type": "config.update", "config_key": "limits.max_moment_images", "old_value": 9, "new_value": 9},
			CreatedAgo: 5 * oneDay,
		},
	}
}

// seedBlockedTags 返回 2 个被屏蔽标签（moment_tags 匹配到即被过滤）。
func seedBlockedTags() []blockedTagSpec {
	return []blockedTagSpec{
		{Name: "广告", BlockedAgo: 7 * oneDay},
		{Name: "引战", BlockedAgo: 6 * oneDay},
	}
}

// seedSystemAnnouncements 返回 2 条系统公告（info / warning）。
func seedSystemAnnouncements() []systemAnnouncementSpec {
	return []systemAnnouncementSpec{
		{
			AnnouncementID: "sysann_seed_1", Title: "社区公约更新提醒",
			Content:  "为保持社区氛围，我们更新了《社区公约》：禁止发布广告与引战内容，违规内容将被限制展示或封禁账号。生效时间：本周一 00:00。",
			Severity: "info", ViewCount: 128, CreatedAgo: 3 * oneDay,
		},
		{
			AnnouncementID: "sysann_seed_2", Title: "近期系统维护通知",
			Content:  "本周日凌晨 02:00-04:00 进行数据库维护，期间图片上传可能短暂不可用，请提前保存草稿。",
			Severity: "warning", ViewCount: 46, CreatedAgo: 2 * oneDay,
		},
	}
}

// seedAppConfigUpdatedBy 是 app_configs.updated_by_user_id 的固定值。
const seedAppConfigUpdatedBy = "user_seed_admin"

// formatSeedTime 统一 seed 时间里字符串形态（RFC3339，UTC）。
func formatSeedTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
