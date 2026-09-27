// 通知数据类型与类型配置。

import type { Component } from 'svelte';
import { Heart, UserPlus, MessageCircle, Bell, MapPin } from '@lucide/svelte';

import type { NotifyPayload } from '@/api/types';

export interface NotifyViewItem {
	id: string;
	type: string;
	read: boolean;
	time: string;
	user: { id: string; name: string; avatar: string };
	text: string;
	momentImage?: string;
	/** 入群邀请通知携带的 invite_id（payload.invite_id），供接受/拒绝操作使用。 */
	inviteId?: string;
	/** 点击通知行的跳转目标（payload 上下文推导：瞬间/用户/群详情）；无则不跳转。 */
	href?: string;
}

/**
 * 按 payload 上下文推导通知行点击的跳转目标（统一口径，全类型一致）：
 * 瞬间类 → 瞬间详情；关注类 → actor 主页；群类 → 群详情；其余不跳转。
 * type 为后端原始类型（非语义归一化键）。
 */
export function resolveNotifHref(
	type: string,
	payload: Partial<NotifyPayload> | undefined,
	actorId: string
): string | undefined {
	const p = (payload ?? {}) as NotifyPayload;
	switch (type) {
		case 'nearby_new_moment':
		case 'moment_commented':
		case 'comment_reply':
		case 'mentioned':
			return p.moment_id ? `/moments/${p.moment_id}` : undefined;
		case 'reaction_added':
			return p.target_type === 'moment' && p.target_id ? `/moments/${p.target_id}` : undefined;
		case 'new_follower':
		case 'mutual_follow':
			return actorId ? `/users/${actorId}` : undefined;
		case 'group_invite_received':
		case 'group_application_received':
		case 'group_application_accepted':
		case 'group_application_rejected':
		case 'group_mentioned':
			return p.group_id ? `/groups/${p.group_id}` : undefined;
		default:
			return undefined;
	}
}

// 通知类型语义 → design token 色族档位（主色 + 淡底/淡边框，参照主题 -bg/-border 定义）。
// 数据层只提供 type 语义键，视觉色全部由本映射表指向 token，不在组件里派生。
export const TYPE_CONFIG: Record<string, { icon: Component; color: string; bg: string; border: string; label: string }> = {
	// 点赞语义 = reaction_added(契约已删 moment_liked)。
	reaction: { icon: Heart, color: 'var(--cst-status-danger)', bg: 'var(--cst-status-danger-bg)', border: 'var(--cst-status-danger-border)', label: '点赞' },
	follow: { icon: UserPlus, color: 'var(--cst-accent-default)', bg: 'var(--cst-accent-soft-bg)', border: 'var(--cst-accent-border-soft)', label: '关注' },
	comment: { icon: MessageCircle, color: 'var(--cst-marker-user)', bg: 'var(--cst-marker-user-bg)', border: 'var(--cst-marker-user-border)', label: '评论' },
	nearby: { icon: MapPin, color: 'var(--cst-status-online)', bg: 'var(--cst-status-online-bg)', border: 'var(--cst-status-online-border)', label: '附近' },
	event: { icon: Bell, color: 'var(--cst-status-warning)', bg: 'var(--cst-status-warning-bg)', border: 'var(--cst-status-warning-border)', label: '活动' }
};

/**
 * 将后端 NotifyType 字符串归一化为 TYPE_CONFIG 的语义键。
 *
 * 通知列表 / 桌面 Rail / 计数 / 过滤都以语义键（reaction/follow/comment/nearby/event）
 * 为口径；后端返回的是 `reaction_added`、`new_follower` 等原始 type，若不归一化，
 * NotifyRow 会全部回退到 reaction 图标、分类计数恒为 0、Rail 分区也永远为空。
 */
const NOTIFY_TYPE_TO_CATEGORY: Record<string, string> = {
	reaction_added: 'reaction',
	new_follower: 'follow',
	mutual_follow: 'follow',
	moment_commented: 'comment',
	comment_reply: 'comment',
	mentioned: 'comment',
	group_mentioned: 'comment',
	nearby_new_moment: 'nearby',
	group_invite_received: 'event',
	group_application_received: 'event',
	group_application_accepted: 'event',
	group_application_rejected: 'event',
	report_resolved: 'event',
	report_dismissed: 'event',
	system_announcement: 'event'
};

export function normalizeNotifType(apiType: string): string {
	return NOTIFY_TYPE_TO_CATEGORY[apiType] ?? 'event';
}
