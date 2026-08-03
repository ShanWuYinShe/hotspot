// 通知数据类型与类型配置（迁移自旧版 NotificationsPage）。

import type { Component } from 'svelte';
import { Heart, UserPlus, MessageCircle, Bell, MapPin } from '@lucide/svelte';

export interface NotificationItem {
	id: string;
	type: string;
	read: boolean;
	time: string;
	user: { name: string; avatar: string };
	text: string;
	momentImage?: string;
}

export const TYPE_CONFIG: Record<string, { icon: Component; color: string; label: string }> = {
	// 点赞语义 = reaction_added(契约已删 moment_liked)。
	reaction: { icon: Heart, color: 'var(--semantic-hot)', label: '点赞' },
	follow: { icon: UserPlus, color: 'var(--neu-accent-primary)', label: '关注' },
	comment: { icon: MessageCircle, color: 'var(--map-user)', label: '评论' },
	nearby: { icon: MapPin, color: 'var(--semantic-online)', label: '附近' },
	event: { icon: Bell, color: 'var(--semantic-warning)', label: '活动' }
};

/**
 * 将后端 NotifyType 字符串归一化为 TYPE_CONFIG 的语义键。
 *
 * 通知列表 / 桌面 Rail / 计数 / 过滤都以语义键（reaction/follow/comment/nearby/event）
 * 为口径；后端返回的是 `reaction_added`、`new_follower` 等原始 type，若不归一化，
 * NotificationRow 会全部回退到 reaction 图标、分类计数恒为 0、Rail 分区也永远为空。
 */
const NOTIFY_TYPE_TO_CATEGORY: Record<string, string> = {
	reaction_added: 'reaction',
	new_follower: 'follow',
	mutual_follow: 'follow',
	moment_commented: 'comment',
	comment_reply: 'comment',
	mentioned: 'comment',
	nearby_new_moment: 'nearby',
	group_invite_received: 'event',
	group_invite_revoked: 'event',
	group_invite_accepted: 'event',
	group_invite_rejected: 'event',
	group_application_cancelled: 'event',
	group_application_accepted: 'event',
	group_application_rejected: 'event',
	report_resolved: 'event',
	report_dismissed: 'event',
	system_announcement: 'event'
};

export function normalizeNotifType(apiType: string): string {
	return NOTIFY_TYPE_TO_CATEGORY[apiType] ?? 'event';
}
