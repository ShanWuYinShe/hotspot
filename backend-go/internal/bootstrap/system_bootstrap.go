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

// chatUnreadCounter 是未读摘要所需的聊天未读聚合能力（privatechats / groups
// 各注入一个实现）：只暴露用到的一个方法，装配层单测可直接注入桩，避免在
// bootstrap 构造完整业务 Service。
type chatUnreadCounter interface {
	CountUnreadForUser(ctx context.Context) (int, error)
}

type systemBootstrapResolver struct {
	users        *users.Service
	notify       notifyUnreadCounter
	privateChats chatUnreadCounter
	groupChats   chatUnreadCounter
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
// 私聊/群聊未读分别经两个聊天模块的跨会话聚合入口现算（服务端不维护独立
// 未读字段，按各自已读游标动态计算，见 backend-go/AGENTS.md「未读计数（聊天）」），
// 综合角标按 spec 公式对三域求和（F10：各域之和，允许同一内容重复计入）。
func (r systemBootstrapResolver) ResolveUnreadSummary(ctx context.Context) system.UnreadSummary {
	result, err := r.notify.UnreadCount(ctx)
	if err != nil {
		// 降级为全零但保留 6 个分类键（notify_by_type 恒为对象，见 unreadByType）。
		observability.FromContext(ctx).Warn("system.bootstrap unread summary query failed", "error", err)
		result = notify.UnreadSummaryResult{}
	}
	summary := system.UnreadSummary{
		NotifyTotal:            result.Total,
		NotifyByType:           unreadByType(result.ByType),
		PrivateChatUnreadCount: r.chatUnreadCount(ctx, "private_chat", r.privateChats),
		GroupChatUnreadCount:   r.chatUnreadCount(ctx, "group_chat", r.groupChats),
	}
	summary.TotalBadge = totalBadge(summary.NotifyTotal, summary.PrivateChatUnreadCount, summary.GroupChatUnreadCount)
	return summary
}

// chatUnreadCount 取单个聊天域的未读总数：查询失败 WARN 后降级为 0（不连累
// 另一域与 total_badge 公式），provider 未注入（无 DB 装配/装配层单测）同样按
// 0 处理——冷启动摘要始终可返回，不因聊天聚合失败阻塞登录。
func (r systemBootstrapResolver) chatUnreadCount(ctx context.Context, domain string, counter chatUnreadCounter) int {
	if counter == nil {
		return 0
	}
	count, err := counter.CountUnreadForUser(ctx)
	if err != nil {
		observability.FromContext(ctx).Warn("system.bootstrap chat unread query failed", "domain", domain, "error", err)
		return 0
	}
	return count
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
