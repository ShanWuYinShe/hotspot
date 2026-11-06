package notify

// bucket 映射通知类型到未读分类桶（内联内存实现与 PG 读实现共用同一口径；
// 内存仓储实现已迁入测试构建，P0-2）。
func bucket(notifyType string) string {
	switch notifyType {
	case "new_follower", "mutual_follow":
		return "social"
	case "moment_commented", "comment_reply", "mentioned", "reaction_added":
		return "interaction"
	// 只列 spec NotifyType union 内的 group 类类型（api_spec_types.md 的
	// NotifyCategory 映射段）。此前还包含 group_invite_revoked / _rejected /
	// _accepted / group_application_cancelled —— 这些既不在契约 union 内，也没有
	// 任何落库方（邀请/申请过期只广播 WS 事件），属死分支，已删（审查 P2-6）。
	case "group_invite_received", "group_application_received", "group_application_accepted", "group_application_rejected":
		return "group"
	case "report_resolved", "report_dismissed":
		return "report"
	case "nearby_new_moment":
		return "nearby"
	default:
		return "system"
	}
}
