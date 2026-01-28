/**
 * 通知服务
 * 通知相关 API
 */

import { apiRequest } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type {
  ApiResponse,
  PaginatedData,
  NotificationItem,
  UnreadCount,
  NotificationFilterType,
} from '@/types/api';

interface NotificationsParams {
  filter?: NotificationFilterType;
  limit?: number;
  cursor?: string;
}

export async function getNotifications(
  params: NotificationsParams = {}
): Promise<ApiResponse<PaginatedData<NotificationItem>>> {
  return apiRequest<PaginatedData<NotificationItem>>(
    ENDPOINTS.NOTIFICATIONS.LIST,
    params
  );
}

export async function getUnreadCount(): Promise<ApiResponse<UnreadCount>> {
  return apiRequest<UnreadCount>(ENDPOINTS.NOTIFICATIONS.UNREAD_COUNT, {});
}

interface MarkNotificationsReadParams {
  notification_ids?: string[];
  mark_all?: boolean;
}

export async function markNotificationsRead(
  params: MarkNotificationsReadParams
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.NOTIFICATIONS.MARK_READ, params);
}
