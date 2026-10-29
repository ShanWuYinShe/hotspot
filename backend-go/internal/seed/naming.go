package seed

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// naming.go — seed 数据的固定 ID 约定与媒体命名规则。
//
// 所有 seed 行 ID 使用固定前缀（便于 --reset 按前缀精确清理，且重复执行时
// 命中同一个主键实现幂等）；媒体 file_id / storage_key 的算法刻意与 media
// 模块保持一致，否则写入的 media_upload_sessions 行会被 media 的读取查询
// 判定为「未完成/不存在」，表现为头像 404、瞬间图不显示。
const (
	userIDPrefix               = "user_seed_"
	momentIDPrefix             = "moment_seed_"
	commentIDPrefix            = "comment_seed_"
	groupIDPrefix              = "group_seed_"
	groupMessageIDPrefix       = "gmsg_seed_"
	chatMessageIDPrefix        = "chat_seed_"
	notifyIDPrefix             = "notify_seed_"
	shareIDPrefix              = "share_seed_"
	reportIDPrefix             = "report_seed_"
	uploadIDPrefix             = "upload_seed_"
	inviteIDPrefix             = "invite_seed_"
	applicationIDPrefix        = "application_seed_"
	groupAnnouncementIDPrefix  = "ann_seed_"
	systemAnnouncementIDPrefix = "sysann_seed_"
	adminRoleIDPrefix          = "role_seed_"
	adminAuditIDPrefix         = "audit_seed_"

	// avatarPublicPathPrefix 与 media.avatarPublicPathPrefix 一致
	// （media/service_avatar.go）：users/groups 的 avatar_url 存稳定代理路径，
	// 由 GET /public/v1/media/avatars/{file_id} 匿名 302 到签名读 URL。
	avatarPublicPathPrefix = "/public/v1/media/avatars/"

	// seedJPEGMimeType 是全部素材的 MIME（devdata/seed/media 下均为 JPEG）。
	seedJPEGMimeType = "image/jpeg"
)

// buildFileID 复写 media.buildFileID（media/service.go）：
//
//	file_ + md5(userID + ":" + uploadID) 的 hex
//
// media 包内该函数未导出，无法直接复用；此处复写并加单测锁定规则。
// 若 media 侧调整算法，seed 写入的行会与 complete_upload 产生的行口径不一致，
// 届时两处必须同步修改。
func buildFileID(userID, uploadID string) string {
	sum := md5.Sum([]byte(userID + ":" + uploadID))
	return fmt.Sprintf("file_%x", sum[:])
}

// storageKeyFor 复写 media.storageKeyFor（media/service.go）：
//
//	uploads/<user_id>/<upload_id>.<ext>
//
// ext 取文件名小写扩展名，缺省 bin。seed 素材固定为 .jpg。
func storageKeyFor(userID, uploadID, fileName string) string {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(fileName), "."))
	if ext == "" {
		ext = "bin"
	}
	return fmt.Sprintf("uploads/%s/%s.%s", userID, uploadID, ext)
}

// avatarURL 返回 users/groups.avatar_url 的稳定代理路径形态。
func avatarURL(fileID string) string {
	return avatarPublicPathPrefix + fileID
}

// chatKey 复写 privatechats.key（privatechats/repo_postgres.go orderedChatUsers
// + repo_memory.go key）：私聊 chat_key = 两个 user_id 字典序用 ":" 连接。
// 例如 user_seed_lumos + user_seed_sophia →
// "user_seed_lumos:user_seed_sophia"。
func chatKey(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}

// chatUsers 返回私聊双方在 private_chats 中的 (user_a_id, user_b_id) 列值：
// 与 chat_key 同口径，按 user_id 字典序排列。
func chatUsers(a, b string) (string, string) {
	if a < b {
		return a, b
	}
	return b, a
}

// twoDigits 把 1–99 的序号格式化为两位十进制（如 1 → "01"），
// 固定 ID（moment_seed_01 / gmsg_seed_01 等）与测试断言共用同一格式。
func twoDigits(n int) string {
	return fmt.Sprintf("%02d", n)
}

// sortedStrings 返回排序后的副本（不改写入参），用于稳定生成角色/权限等数组值。
func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
