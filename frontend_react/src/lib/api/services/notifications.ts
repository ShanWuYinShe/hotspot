import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { NotificationItem, PaginatedData, Subscriptions } from '@/types/api';

export const NotificationService = {
  list: (params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<NotificationItem>>(ENDPOINTS.NOTIFICATIONS.LIST, params ?? {}),

  unreadCount: () =>
    apiClient.post<{ total: number }>(ENDPOINTS.NOTIFICATIONS.UNREAD_COUNT, {}),

  markRead: (params?: { notification_ids?: string[] }) =>
    apiClient.post(ENDPOINTS.NOTIFICATIONS.MARK_READ, params ?? {}),
};

export const SubscriptionService = {
  get: () =>
    apiClient.post<Subscriptions>(ENDPOINTS.SUBSCRIPTIONS.ME, {}),

  update: (data: Partial<Subscriptions>) =>
    apiClient.post<Subscriptions>(ENDPOINTS.SUBSCRIPTIONS.UPDATE, data),
};
