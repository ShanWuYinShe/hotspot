/**
 * 通知服务
 */

import { post, paginate } from '../client';
import { NOTIFICATIONS } from '../endpoints';
import type {
  NotificationItem,
  UnreadCountResult,
  PaginationParams,
  NotificationType
} from '$lib/types/api';

export const notificationsService = {
  /**
   * 获取通知列表（游标分页）
   */
  list(params?: { type?: NotificationType; pagination?: PaginationParams }) {
    return paginate<NotificationItem>(NOTIFICATIONS.LIST, params);
  },

  /**
   * 获取未读通知数
   */
  unreadCount() {
    return post<UnreadCountResult>(NOTIFICATIONS.UNREAD_COUNT, {});
  },

  /**
   * 批量标记已读（spec: 1-100 个 ID）
   */
  markAsRead(data: { notification_ids: string[] }) {
    return post<void>(NOTIFICATIONS.MARK_AS_READ, data);
  },

  /**
   * 标记全部已读
   */
  markAllRead() {
    return post<void>(NOTIFICATIONS.MARK_ALL_READ, {});
  }
};
