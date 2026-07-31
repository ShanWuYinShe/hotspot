// 通知数据类型与类型配置（迁移自 React NotificationsPage.tsx）。

import type { Component } from 'svelte';
import { Heart, UserPlus, MessageCircle, Bell, MapPin } from '@lucide/svelte';

export interface Notif {
	id: string;
	type: string;
	read: boolean;
	time: string;
	user: { name: string; avatar: string };
	text: string;
	postImage?: string;
}

export const TYPE_CONFIG: Record<string, { icon: Component; color: string; label: string }> = {
	like: { icon: Heart, color: 'var(--semantic-hot)', label: '点赞' },
	follow: { icon: UserPlus, color: 'var(--neu-accent-primary)', label: '关注' },
	comment: { icon: MessageCircle, color: 'var(--map-user)', label: '评论' },
	nearby: { icon: MapPin, color: 'var(--semantic-online)', label: '附近' },
	event: { icon: Bell, color: 'var(--semantic-warning)', label: '活动' }
};
