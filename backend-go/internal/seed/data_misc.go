package seed

import (
	"encoding/json"
	"fmt"
	"time"
)

// data_misc.go — 通知/分享/举报/管理端/系统公告/应用配置表行构建。

func buildNotificationsTable(baseTime time.Time, notifications []notifySpec, payloadCtx payloadContext) (tableData, error) {
	table := tableData{
		Table: "notifications",
		Columns: []column{
			col("notification_id"), col("user_id"), col("type"), col("is_read"),
			col("created_at"), jsonbCol("payload"),
		},
	}
	for _, notification := range notifications {
		payload := notification.Payload(payloadCtx)
		if payload == nil {
			return table, fmt.Errorf("通知 %s 的 payload 为空", notification.NotifyID)
		}
		if got, _ := payload["type"].(string); got != notification.Type {
			return table, fmt.Errorf("通知 %s 的 payload.type=%q 与 type=%q 不一致", notification.NotifyID, got, notification.Type)
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return table, fmt.Errorf("序列化通知 %s payload 失败: %w", notification.NotifyID, err)
		}
		table.addRow(
			notification.NotifyID, notification.UserID, notification.Type, notification.IsRead,
			baseTime.Add(-notification.CreatedAgo), string(encoded),
		)
	}
	return table, nil
}

func buildSharesTable(baseTime time.Time, shares []shareSpec) tableData {
	table := tableData{
		Table: "shares",
		Columns: []column{
			col("share_id"), col("share_code"), col("creator_user_id"), col("share_type"),
			jsonbCol("payload"), col("created_at"), col("expires_at"), col("is_revoked"),
		},
	}
	for _, share := range shares {
		encoded, err := json.Marshal(share.Payload)
		if err != nil {
			panic(fmt.Sprintf("序列化分享 %s payload 失败: %v", share.ShareID, err))
		}
		var expiresAt any
		if share.ExpiresIn > 0 {
			expiresAt = baseTime.Add(share.ExpiresIn)
		}
		table.addRow(
			share.ShareID, share.ShareCode, share.CreatorID, share.ShareType,
			string(encoded), baseTime.Add(-share.CreatedAgo), expiresAt, share.Revoked,
		)
	}
	return table
}

func buildReportsTable(baseTime time.Time, reports []reportSpec) (tableData, error) {
	table := tableData{
		Table: "reports",
		Columns: []column{
			col("report_id"), col("creator_user_id"), col("target_type"), col("target_id"),
			col("reason"), col("description"), col("status"), col("created_at"),
			col("handled_at"), col("handled_by_user_id"), col("handle_note"), jsonbCol("message_snapshot"),
		},
	}
	for _, report := range reports {
		var snapshot any
		if report.MessageSnapshot != nil {
			encoded, err := json.Marshal(report.MessageSnapshot(baseTime))
			if err != nil {
				return table, fmt.Errorf("序列化举报 %s 消息快照失败: %w", report.ReportID, err)
			}
			snapshot = string(encoded)
		}
		var handledAt any
		if report.HandledAgo > 0 {
			handledAt = baseTime.Add(-report.HandledAgo)
		}
		table.addRow(
			report.ReportID, report.CreatorID, report.TargetType, report.TargetID,
			report.Reason, nullableString(report.Description), report.Status, baseTime.Add(-report.CreatedAgo),
			handledAt, nullableString(report.HandledBy), nullableString(report.HandleNote), snapshot,
		)
	}
	return table, nil
}

func buildAdminRolesTable(baseTime time.Time, roles []adminRoleSpec) tableData {
	table := tableData{
		Table: "admin_roles",
		Columns: []column{
			col("role_id"), col("name"), col("description"), col("created_at"), col("updated_at"),
		},
	}
	for _, role := range roles {
		createdAt := baseTime.Add(-role.CreatedAgo)
		table.addRow(role.RoleID, role.Name, nullableString(role.Description), createdAt, createdAt)
	}
	return table
}

func buildAdminRolePermissionsTable(baseTime time.Time, roles []adminRoleSpec) tableData {
	table := tableData{
		Table:   "admin_role_permissions",
		Columns: []column{col("role_id"), col("permission_key"), col("created_at")},
	}
	for _, role := range roles {
		createdAt := baseTime.Add(-role.CreatedAgo)
		for _, permission := range sortedStrings(role.Permissions) {
			table.addRow(role.RoleID, permission, createdAt)
		}
	}
	return table
}

func buildAdminUsersTable(baseTime time.Time, admins []adminUserSpec) tableData {
	table := tableData{
		Table:   "admin_users",
		Columns: []column{col("user_id"), col("grant_all_permissions"), col("granted_at"), col("revoked_at")},
	}
	for _, admin := range admins {
		table.addRow(admin.UserID, admin.GrantAll, baseTime.Add(-admin.GrantedAgo), any(nil))
	}
	return table
}

func buildAdminUserRolesTable(baseTime time.Time, admins []adminUserSpec) tableData {
	table := tableData{
		Table:   "admin_user_roles",
		Columns: []column{col("user_id"), col("role_id"), col("created_at")},
	}
	for _, admin := range admins {
		grantedAt := baseTime.Add(-admin.GrantedAgo)
		for _, roleID := range sortedStrings(admin.RoleIDs) {
			table.addRow(admin.UserID, roleID, grantedAt)
		}
	}
	return table
}

func buildAdminAuditLogsTable(baseTime time.Time, logs []auditLogSpec, users []userSpec, media *mediaBundle) tableData {
	userByID := make(map[string]userSpec, len(users))
	for _, user := range users {
		userByID[user.UserID] = user
	}
	table := tableData{
		Table: "admin_audit_logs",
		Columns: []column{
			col("log_id"), col("action"), col("operator_id"), col("operator_name"),
			col("operator_avatar_url"), col("target_type"), col("target_id"), col("target_name"),
			jsonbCol("details_json"), col("ip_location"), col("ip_address"), col("device_summary"),
			col("created_at"),
		},
	}
	for _, log := range logs {
		operatorName := log.OperatorID
		if operator, ok := userByID[log.OperatorID]; ok {
			operatorName = operator.Nickname
		}
		var operatorAvatar any
		if fileID, ok := media.userAvatarFileID[log.OperatorID]; ok {
			operatorAvatar = avatarURL(fileID)
		}
		var details any
		if len(log.Details) > 0 {
			encoded, err := json.Marshal(log.Details)
			if err != nil {
				panic(fmt.Sprintf("序列化审计日志 %s details 失败: %v", log.LogID, err))
			}
			details = string(encoded)
		}
		table.addRow(
			log.LogID, log.Action, log.OperatorID, operatorName,
			operatorAvatar, log.TargetType, log.TargetID, nullableString(log.TargetName),
			details, nullableString(log.IPLocation), nullableString(log.IPAddress), nullableString(log.DeviceSummary),
			baseTime.Add(-log.CreatedAgo),
		)
	}
	return table
}

func buildBlockedTagsTable(baseTime time.Time, tags []blockedTagSpec) tableData {
	table := tableData{
		Table:   "admin_blocked_tags",
		Columns: []column{col("name"), col("blocked_at")},
	}
	for _, tag := range tags {
		table.addRow(tag.Name, baseTime.Add(-tag.BlockedAgo))
	}
	return table
}

func buildSystemAnnouncementsTable(baseTime time.Time, announcements []systemAnnouncementSpec) tableData {
	table := tableData{
		Table: "system_announcements",
		Columns: []column{
			col("announcement_id"), col("title"), col("content"), col("severity"),
			col("action_url"), col("status"), col("send_push"), col("push_sent_at"),
			col("view_count"), col("created_by_user_id"), col("created_at"), col("updated_at"),
		},
	}
	for _, announcement := range announcements {
		createdAt := baseTime.Add(-announcement.CreatedAgo)
		table.addRow(
			announcement.AnnouncementID, announcement.Title, announcement.Content, announcement.Severity,
			nullableString(announcement.ActionURL), "published", false, any(nil),
			announcement.ViewCount, "user_seed_admin", createdAt, createdAt,
		)
	}
	return table
}

func buildAppConfigsTable(baseTime time.Time, payloadJSON string) tableData {
	table := tableData{
		Table: "app_configs",
		Columns: []column{
			col("config_id"), jsonbCol("payload_json"), col("updated_at"), col("updated_by_user_id"),
		},
	}
	if payloadJSON == "" {
		return table
	}
	table.addRow("default", payloadJSON, baseTime, seedAppConfigUpdatedBy)
	return table
}

var _ = time.Time{}
