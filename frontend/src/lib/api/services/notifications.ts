import { post, paginate } from '../client';
import { NOTIFICATIONS } from '../endpoints';
import type {
  NotificationItem,
  UnreadCountResult,
  PaginationParams,
  NotificationType
} from '@/lib/types/api';

export const notificationsService = {
  list(params?: { type?: NotificationType; pagination?: PaginationParams }) {
    return paginate<NotificationItem>(NOTIFICATIONS.LIST, params);
  },

  unreadCount() {
    return post<UnreadCountResult>(NOTIFICATIONS.UNREAD_COUNT, {});
  },

  markAsRead(data: { notification_ids: string[] }) {
    return post<void>(NOTIFICATIONS.MARK_AS_READ, data);
  },

  markAllRead() {
    return post<void>(NOTIFICATIONS.MARK_ALL_READ, {});
  }
};
