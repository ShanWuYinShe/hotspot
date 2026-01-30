/**
 * 通知服务
 */

import { post, paginate } from '../client';
import { NOTIFICATIONS, SUBSCRIPTIONS, DEVICE_TOKENS } from '../endpoints';
import type {
	DevicePlatform,
	DeviceTokenItem,
	NotificationFilterType,
	NotificationItem,
	Subscriptions,
	UnreadCount
} from '$lib/types/api';

export const notificationsService = {
	/**
	 * 获取通知列表
	 */
	getNotifications(params?: {
		type?: NotificationFilterType;
		limit?: number;
		cursor?: string;
	}) {
		return paginate<NotificationItem>(NOTIFICATIONS.LIST, params);
	},

	/**
	 * 获取未读数量
	 */
	getUnreadCount() {
		return post<UnreadCount>(NOTIFICATIONS.UNREAD_COUNT);
	},

	/**
	 * 标记已读
	 */
	markRead(data: { notification_ids?: string[]; type?: NotificationFilterType }) {
		return post<void>(NOTIFICATIONS.MARK_READ, data);
	}
};

export const subscriptionsService = {
	/**
	 * 获取订阅设置
	 */
	getSettings() {
		return post<Subscriptions>(SUBSCRIPTIONS.ME);
	},

	/**
	 * 更新订阅设置
	 */
	updateSettings(data: Partial<Subscriptions>) {
		return post<void>(SUBSCRIPTIONS.UPDATE, data);
	}
};

export const deviceTokensService = {
	/**
	 * 获取设备令牌列表
	 */
	getTokens(params?: { limit?: number; cursor?: string }) {
		return paginate<DeviceTokenItem>(DEVICE_TOKENS.ME, params);
	},

	/**
	 * 注册设备令牌
	 */
	register(data: { device_token: string; platform: DevicePlatform }) {
		return post<void>(DEVICE_TOKENS.REGISTER, data);
	},

	/**
	 * 注销设备令牌
	 */
	unregister(data: { device_token: string; platform: DevicePlatform }) {
		return post<void>(DEVICE_TOKENS.UNREGISTER, data);
	}
};
