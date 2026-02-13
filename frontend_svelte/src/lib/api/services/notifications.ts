/**
 * 通知 API 服务
 */

import { post } from '$lib/api/client';
import { NOTIFICATIONS } from '$lib/api/endpoints';
import type {
	PaginatedData,
	NotificationItem,
	UnreadCountResult,
	NotificationListRequest
} from '$lib/types/api';

class NotificationsService {
	async list(params: NotificationListRequest = {}) {
		return post<PaginatedData<NotificationItem>>(NOTIFICATIONS.LIST, params);
	}

	async unreadCount() {
		return post<UnreadCountResult>(NOTIFICATIONS.UNREAD_COUNT, {});
	}

	async markAsRead(notificationIds: string[]) {
		return post<null>(NOTIFICATIONS.MARK_AS_READ, {
			notification_ids: notificationIds
		});
	}

	async markAllRead() {
		return post<null>(NOTIFICATIONS.MARK_ALL_READ, {});
	}
}

export const notificationsService = new NotificationsService();
