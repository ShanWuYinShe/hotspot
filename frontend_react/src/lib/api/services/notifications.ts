/**
 * 通知服务
 */

import { get, post, put, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  NotificationItem,
  SubscriptionsData,
  DeviceTokenItem,
  DevicePlatform,
} from '@/types/api';

export const notificationsService = {
  // ============================================================================
  // 通知列表
  // ============================================================================

  /**
   * 获取通知列表
   */
  async getNotifications(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<NotificationItem>>> {
    return get<PaginatedData<NotificationItem>>(ENDPOINTS.NOTIFICATIONS.LIST, params);
  },

  /**
   * 标记通知已读
   */
  async markRead(notificationIds: string[]): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.NOTIFICATIONS.MARK_READ, { notification_ids: notificationIds }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 标记全部已读
   */
  async markAllRead(): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.NOTIFICATIONS.MARK_ALL_READ, undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 获取未读数量
   */
  async getUnreadCount(): Promise<ApiResponse<{ count: number }>> {
    return get<{ count: number }>(ENDPOINTS.NOTIFICATIONS.UNREAD_COUNT);
  },

  // ============================================================================
  // 订阅设置
  // ============================================================================

  /**
   * 获取订阅设置
   */
  async getSubscriptions(): Promise<ApiResponse<SubscriptionsData>> {
    return get<SubscriptionsData>(ENDPOINTS.SUBSCRIPTIONS.GET);
  },

  /**
   * 更新订阅设置
   */
  async updateSubscriptions(data: Partial<SubscriptionsData>): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.SUBSCRIPTIONS.UPDATE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 设备令牌
  // ============================================================================

  /**
   * 注册设备令牌
   */
  async registerDeviceToken(platform: DevicePlatform, deviceToken: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.DEVICE_TOKENS.REGISTER, { platform, device_token: deviceToken }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 注销设备令牌
   */
  async unregisterDeviceToken(platform: DevicePlatform, deviceToken: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.DEVICE_TOKENS.UNREGISTER, { platform, device_token: deviceToken });
  },

  /**
   * 获取设备令牌列表
   */
  async getDeviceTokens(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<DeviceTokenItem>>> {
    return get<PaginatedData<DeviceTokenItem>>(ENDPOINTS.DEVICE_TOKENS.LIST, params);
  },
};
