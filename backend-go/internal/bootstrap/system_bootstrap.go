package bootstrap

import (
	"context"

	"hotspot/backend-go/internal/modules/notify"
	"hotspot/backend-go/internal/modules/system"
	"hotspot/backend-go/internal/modules/users"
	platformauth "hotspot/backend-go/internal/platform/auth"
	"hotspot/backend-go/internal/platform/observability"
)

// notifyUnreadCounter 是未读摘要所需的 notify 能力（只暴露用到的一个方法）：
// 装配层单测可直接注入桩，无需构造完整 notify.Service。
type notifyUnreadCounter interface {
	UnreadCount(ctx context.Context) (notify.UnreadSummaryResult, error)
}

type systemBootstrapResolver struct {
	users  *users.Service
	notify notifyUnreadCounter
}

func (r systemBootstrapResolver) ResolveProfile(ctx context.Context) (any, bool) {
	principal, ok := platformauth.PrincipalFromContext(ctx)
	if !ok {
		return nil, false
	}
	profile, err := r.users.GetMyProfile(ctx)
	if err != nil {
		return nil, false
	}
	_ = principal.UserID
	return profile, true
}

// ResolveUnreadSummary 汇总冷启动未读摘要（spec api_spec_system.md →
// BootstrapData.unread_summary）：通知部分与 notify/unread_summary 同口径；
// 聊天未读暂无低成本聚合入口（见内联说明），综合角标仍按 spec 公式求和。
func (r systemBootstrapResolver) ResolveUnreadSummary(ctx context.Context) system.UnreadSummary {
	result, err := r.notify.UnreadCount(ctx)
	if err != nil {
		// 降级为全零但保留 6 个分类键（notify_by_type 恒为对象，见 unreadByType）。
		observability.FromContext(ctx).Warn("system.bootstrap unread summary query failed", "error", err)
		result = notify.UnreadSummaryResult{}
	}
	summary := system.UnreadSummary{
		NotifyTotal:  result.Total,
		NotifyByType: unreadByType(result.ByType),
		// 聊天未读（private_chat_unread_count / group_chat_unread_count）当前
		// 恒为 0：backend-go/AGENTS.md「未读计数（聊天）」约定服务端不维护独立
		// 聊天未读字段（由客户端基于 WS 增量维护），且 privatechats 与
		// groupmessages 都没有「全部会话未读总数」的低成本聚合入口——现有实现
		// 只在分页会话列表里按已读游标逐会话现算，做全量聚合需在各自 repo 新增
		// SUM 查询并暴露跨模块 API，超出本聚合端点「不引入新数据」的定位。
		// 字段与公式仍按 spec 保留：后续接入聚合入口时只需给这两个字段赋值即可。
	}
	summary.TotalBadge = totalBadge(summary.NotifyTotal, summary.PrivateChatUnreadCount, summary.GroupChatUnreadCount)
	return summary
}

// unreadByType 把通知未读分类投影为 spec 的 Record<NotifyCategory, number>。
// 固定输出 api_spec_types.md 定义的 6 个 NotifyCategory 键：缺键会让客户端按
// 分类渲染角标时静默丢数据（审查 C5 此前只写 4 键，report/nearby 丢失）。
func unreadByType(counts notify.UnreadCountByType) map[string]int {
	return map[string]int{
		"social":      counts.Social,
		"interaction": counts.Interaction,
		"group":       counts.Group,
		"report":      counts.Report,
		"nearby":      counts.Nearby,
		"system":      counts.System,
	}
}

// totalBadge 按 spec api_spec_system.md 的公式计算综合角标：
// notify_total + private_chat_unread_count + group_chat_unread_count。
// F10：同一内容可同时计入通知与聊天两域（如 group_mentioned），角标是各域
// 之和而非互斥未读条数，客户端仅用于展示、不做对账。
func totalBadge(notifyTotal, privateChatUnread, groupChatUnread int) int {
	return notifyTotal + privateChatUnread + groupChatUnread
}
