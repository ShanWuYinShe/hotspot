package seed

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"hotspot/backend-go/internal/platform/rbac"
)

// data_misc_test.go — 通知/分享/举报/管理端/媒体行一致性。

func TestNotifications(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	rows := rowsOfTable(t, plan, "notifications")
	columns := columnsOfTable(t, plan, "notifications")
	userIndex := columnIndex(t, columns, "user_id")
	typeIndex := columnIndex(t, columns, "type")
	readIndex := columnIndex(t, columns, "is_read")
	payloadIndex := columnIndex(t, columns, "payload")

	if len(rows) != 9 {
		t.Fatalf("通知数 = %d，期望 9", len(rows))
	}
	unread := 0
	types := map[string]bool{}
	for _, row := range rows {
		if mustString(t, row[userIndex]) != "user_seed_lumos" {
			t.Errorf("通知应全部发给 lumos，实际发给 %v", row[userIndex])
		}
		notifyType := mustString(t, row[typeIndex])
		types[notifyType] = true
		if !mustBool(t, row[readIndex]) {
			unread++
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(mustString(t, row[payloadIndex])), &payload); err != nil {
			t.Fatalf("通知 payload 不是合法 JSON: %v", err)
		}
		if payload["type"] != notifyType {
			t.Errorf("通知 %s payload.type = %v 与 type 不一致", row[0], payload["type"])
		}
		if err := validateNotifyPayload(notifyType, payload); err != nil {
			t.Errorf("通知 %v payload 校验失败: %v", row[0], err)
		}
	}
	if unread != 4 {
		t.Errorf("未读通知数 = %d，期望 4", unread)
	}
	for _, wantType := range []string{
		"new_follower", "mutual_follow", "reaction_added", "moment_commented",
		"comment_reply", "group_invite_received", "group_application_rejected",
		"nearby_new_moment", "system_announcement",
	} {
		if !types[wantType] {
			t.Errorf("缺少通知类型 %s", wantType)
		}
	}
}

// validateNotifyPayload 按 spec api_spec_notify.md 的联合类型校验必备字段。
func validateNotifyPayload(notifyType string, payload map[string]any) error {
	var required []string
	switch notifyType {
	case "new_follower":
		required = []string{"actor", "source", "source_moment_id"}
	case "mutual_follow":
		required = []string{"actor"}
	case "moment_commented":
		required = []string{"actor", "moment_id", "moment_summary", "comment_id", "comment_preview"}
	case "comment_reply":
		required = []string{"actor", "comment_id", "replied_comment_preview", "reply_id", "reply_content_preview", "moment_id", "moment_summary"}
	case "reaction_added":
		required = []string{"actor", "target_type", "target_id", "emoji"}
	case "group_invite_received":
		required = []string{"actor", "group_id", "group_name", "group_avatar_url", "invite_id", "message"}
	case "group_application_rejected":
		required = []string{"group_id", "group_name", "application_id", "reason"}
	case "nearby_new_moment":
		required = []string{"actor", "moment_id", "moment_summary", "distance"}
	case "system_announcement":
		required = []string{"announcement_id", "title", "severity"}
	default:
		return stringError("未知通知类型 " + notifyType)
	}
	for _, key := range required {
		if _, ok := payload[key]; !ok {
			return stringError("缺少字段 " + key)
		}
	}
	return nil
}

// stringError 是测试用的小错误类型。
type stringError string

func (e stringError) Error() string { return string(e) }

// TestShares 分享码契约（12 位、字符集 [A-HJ-NP-Z2-9]）与目标 payload。
func TestShares(t *testing.T) {
	data := buildSeedData()
	if len(data.shares) != 2 {
		t.Fatalf("分享数 = %d，期望 2", len(data.shares))
	}
	codePattern := regexp.MustCompile("^[A-HJ-NP-Z2-9]{12}$")
	codes := map[string]bool{}
	for _, share := range data.shares {
		if !codePattern.MatchString(share.ShareCode) {
			t.Errorf("分享码 %q 不符合契约（12 位、字符集 [A-HJ-NP-Z2-9]）", share.ShareCode)
		}
		if codes[share.ShareCode] {
			t.Errorf("分享码重复: %s", share.ShareCode)
		}
		codes[share.ShareCode] = true
		switch share.ShareType {
		case "moment":
			if share.Payload["shared_moment_id"] != "moment_seed_01" {
				t.Errorf("瞬间分享目标错误: %v", share.Payload)
			}
		case "user":
			if share.Payload["shared_user_id"] != "user_seed_lumos" {
				t.Errorf("用户分享目标错误: %v", share.Payload)
			}
		default:
			t.Errorf("未预期的分享类型: %s", share.ShareType)
		}
	}
	if !codes[shareCodeMomentSeed] || !codes[shareCodeUserSeed] {
		t.Errorf("缺少 QA 分享码（%s / %s）", shareCodeMomentSeed, shareCodeUserSeed)
	}
}

// TestReports 举报类型/状态/消息快照。
func TestReports(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	rows := rowsOfTable(t, plan, "reports")
	columns := columnsOfTable(t, plan, "reports")
	targetTypeIndex := columnIndex(t, columns, "target_type")
	statusIndex := columnIndex(t, columns, "status")
	snapshotIndex := columnIndex(t, columns, "message_snapshot")

	if len(rows) != 4 {
		t.Fatalf("举报数 = %d，期望 4", len(rows))
	}
	validStatus := map[string]bool{"pending": true, "resolved": true, "dismissed": true}
	targetTypes := map[string]bool{}
	statuses := map[string]bool{}
	for _, row := range rows {
		targetType := mustString(t, row[targetTypeIndex])
		targetTypes[targetType] = true
		status := mustString(t, row[statusIndex])
		statuses[status] = true
		if !validStatus[status] {
			t.Errorf("举报状态 %q 不在契约枚举内", status)
		}
		if targetType == "message" {
			raw, ok := row[snapshotIndex].(string)
			if !ok || raw == "" {
				t.Fatalf("message 举报缺少 message_snapshot")
			}
			var snapshot map[string]any
			if err := json.Unmarshal([]byte(raw), &snapshot); err != nil {
				t.Fatalf("message_snapshot 不是合法 JSON: %v", err)
			}
			for _, key := range []string{"chat_type", "target_chat_id", "message_type", "snapshot_content", "snapshot_media", "sent_at"} {
				if _, ok := snapshot[key]; !ok {
					t.Errorf("message_snapshot 缺少字段 %s", key)
				}
			}
			sentAt, ok := snapshot["sent_at"].(string)
			if !ok {
				t.Errorf("message_snapshot.sent_at 类型 = %T", snapshot["sent_at"])
			} else if _, err := time.Parse(time.RFC3339, sentAt); err != nil {
				t.Errorf("message_snapshot.sent_at 不是 RFC3339: %v", err)
			}
		} else if row[snapshotIndex] != nil {
			t.Errorf("非 message 举报不应写 message_snapshot")
		}
	}
	for _, wantType := range []string{"user", "moment", "group", "message"} {
		if !targetTypes[wantType] {
			t.Errorf("举报目标类型缺少 %s", wantType)
		}
	}
	for _, wantStatus := range []string{"pending", "resolved", "dismissed"} {
		if !statuses[wantStatus] {
			t.Errorf("举报状态缺少 %s", wantStatus)
		}
	}
}

// TestAdminSpec 管理端角色/权限/审计/公告/屏蔽标签。
func TestAdminSpec(t *testing.T) {
	data := buildSeedData()
	allowedPermissions := map[string]bool{
		rbac.PermAdminRbacManage: true, rbac.PermAdminReportsRead: true,
		rbac.PermAdminReportsWrite: true, rbac.PermAdminAppConfigRead: true,
		rbac.PermAdminAppConfigWrite: true, rbac.PermAdminTagsRead: true,
		rbac.PermAdminTagsWrite: true, rbac.PermAdminAnnouncementsRead: true,
		rbac.PermAdminAnnouncementsWrite: true, rbac.PermAdminUsersRead: true,
		rbac.PermAdminUsersReadIP: true, rbac.PermAdminUsersWrite: true,
		rbac.PermAdminGroupsRead: true, rbac.PermAdminGroupsWrite: true,
		rbac.PermAdminAuditRead: true, rbac.PermAdminAuditReadIP: true,
		rbac.PermAdminStatsRead: true, rbac.PermAdminSuperadmin: true,
	}
	if len(data.adminRoles) != 2 {
		t.Fatalf("管理角色数 = %d，期望 2", len(data.adminRoles))
	}
	for _, role := range data.adminRoles {
		if role.Name == "" || len(role.Permissions) == 0 {
			t.Errorf("角色 %s 缺少名称/权限", role.RoleID)
		}
		for _, permission := range role.Permissions {
			if !allowedPermissions[permission] {
				t.Errorf("角色 %s 使用了未知权限 key %q", role.RoleID, permission)
			}
		}
	}
	adminGranted, lumosGranted := false, false
	for _, admin := range data.adminUsers {
		switch admin.UserID {
		case "user_seed_admin":
			adminGranted = admin.GrantAll && len(admin.RoleIDs) == 1
		case "user_seed_lumos":
			lumosGranted = true
		}
	}
	if !adminGranted {
		t.Errorf("user_seed_admin 应为 grant_all + viewer 角色")
	}
	if lumosGranted {
		t.Errorf("user_seed_lumos 不应出现在 admin_users（普通用户视角）")
	}

	validAuditActions := map[string]bool{
		"user.ban": true, "user.unban": true, "group.dismiss": true,
		"group.member_mute": true, "group.member_ban": true, "group.member_remove": true,
		"announcement.create": true, "announcement.update": true, "announcement.delete": true,
		"role.create": true, "role.update": true, "role.delete": true,
		"role.set_permissions": true, "admin.award": true, "admin.revoke": true,
		"report.judge": true, "tag.set_ban": true, "config.update": true,
	}
	if len(data.adminAudits) != 6 {
		t.Errorf("审计日志数 = %d，期望 6", len(data.adminAudits))
	}
	for _, log := range data.adminAudits {
		if !validAuditActions[log.Action] {
			t.Errorf("审计 action %q 不在契约枚举内", log.Action)
		}
		if got := log.Details["type"]; got != log.Action {
			t.Errorf("审计 %s details.type = %v，期望与 action 一致", log.LogID, got)
		}
	}
	if len(data.blockedTags) != 2 {
		t.Errorf("屏蔽标签数 = %d，期望 2", len(data.blockedTags))
	}
	if len(data.announcements) != 2 {
		t.Errorf("系统公告数 = %d，期望 2", len(data.announcements))
	}
	severities := map[string]bool{}
	for _, announcement := range data.announcements {
		severities[announcement.Severity] = true
	}
	if !severities["info"] || !severities["warning"] {
		t.Errorf("系统公告应覆盖 info/warning，实际 %v", severities)
	}
}

// TestMediaRowsMatchAssets 媒体会话行与素材元数据一致（宽度/高度/大小/MIME/md5）。
func TestMediaRowsMatchAssets(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	assets := plan.assets // 计划已加载素材，直接复用避免二次读取文件系统
	rows := rowsOfTable(t, plan, "media_upload_sessions")
	columns := columnsOfTable(t, plan, "media_upload_sessions")
	fileNameIndex := columnIndex(t, columns, "file_name")
	statusIndex := columnIndex(t, columns, "status")
	sizeIndex := columnIndex(t, columns, "final_file_size")
	mimeIndex := columnIndex(t, columns, "final_mime_type")
	widthIndex := columnIndex(t, columns, "width")
	heightIndex := columnIndex(t, columns, "height")
	md5Index := columnIndex(t, columns, "md5")
	if len(rows) != len(plan.media) {
		t.Fatalf("media_upload_sessions 行数 = %d，计划对象数 = %d", len(rows), len(plan.media))
	}
	for index, row := range rows {
		object := plan.media[index]
		asset, ok := assets[mustString(t, row[fileNameIndex])]
		if !ok {
			t.Fatalf("媒体行引用了不存在的素材: %v", row[fileNameIndex])
		}
		if mustString(t, row[statusIndex]) != "completed" {
			t.Errorf("媒体 %s 状态应为 completed", object.UploadID)
		}
		if got := mustInt(t, row[sizeIndex]); got != asset.Size {
			t.Errorf("媒体 %s final_file_size = %d，素材大小 = %d", object.UploadID, got, asset.Size)
		}
		if got := mustString(t, row[mimeIndex]); got != seedJPEGMimeType {
			t.Errorf("媒体 %s final_mime_type = %q", object.UploadID, got)
		}
		if got := mustInt(t, row[widthIndex]); got != asset.Width {
			t.Errorf("媒体 %s width = %d，素材 width = %d", object.UploadID, got, asset.Width)
		}
		if got := mustInt(t, row[heightIndex]); got != asset.Height {
			t.Errorf("媒体 %s height = %d，素材 height = %d", object.UploadID, got, asset.Height)
		}
		if got := mustString(t, row[md5Index]); got != asset.MD5 {
			t.Errorf("媒体 %s md5 与素材内容不一致", object.UploadID)
		}
	}
}

// TestAvatarURLsMatchMedia 用户/群头像 avatar_url 必须指向真实 file_id 的代理路径。
func TestAvatarURLsMatchMedia(t *testing.T) {
	plan := mustBuildPlan(t, testPlanOptions())
	userAvatarFileID := map[string]string{}
	groupFileIDByGroup := map[string]string{}
	for _, object := range plan.media {
		switch object.Category {
		case "avatar":
			userAvatarFileID[object.UserID] = object.FileID
		case "group_avatar":
			slug := strings.TrimPrefix(object.UploadID, uploadIDPrefix+"group_")
			groupFileIDByGroup[groupIDPrefix+slug] = object.FileID
		}
	}

	usersColumns := columnsOfTable(t, plan, "users")
	userIDIndex := columnIndex(t, usersColumns, "user_id")
	avatarIndex := columnIndex(t, usersColumns, "avatar_url")
	for _, row := range rowsOfTable(t, plan, "users") {
		userID := mustString(t, row[userIDIndex])
		fileID, hasAvatar := userAvatarFileID[userID]
		if !hasAvatar {
			if row[avatarIndex] != nil {
				t.Errorf("用户 %s 无头像素材但 avatar_url 非空: %v", userID, row[avatarIndex])
			}
			continue
		}
		if got := mustString(t, row[avatarIndex]); got != avatarURL(fileID) {
			t.Errorf("用户 %s avatar_url = %q，期望 %q", userID, got, avatarURL(fileID))
		}
	}

	groupsColumns := columnsOfTable(t, plan, "groups")
	groupIDIndex := columnIndex(t, groupsColumns, "group_id")
	groupAvatarIndex := columnIndex(t, groupsColumns, "avatar_url")
	for _, row := range rowsOfTable(t, plan, "groups") {
		groupID := mustString(t, row[groupIDIndex])
		fileID, ok := groupFileIDByGroup[groupID]
		if !ok {
			t.Fatalf("群 %s 缺少群头像媒体", groupID)
		}
		if got := mustString(t, row[groupAvatarIndex]); got != avatarURL(fileID) {
			t.Errorf("群 %s avatar_url = %q，期望 %q", groupID, got, avatarURL(fileID))
		}
	}
}
