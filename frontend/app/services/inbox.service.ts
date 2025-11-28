import { BaseService } from './base.service'
import type { InboxItemData, UnreadCategoryStats, PaginatedData } from '~/types/api'

/**
 * 通知中心服务
 * ✅ 类型定义已统一从 types/api.ts 导入
 */

// ==================== Request 类型定义 ====================

export interface UnreadInboxSummary {
  total: number
  by_category: UnreadCategoryStats
}

export interface ListInboxItemsRequest {
  category?: 'social' | 'system' | 'content'
  types?: string[]
  is_read?: boolean
  limit?: number
  cursor?: string
}

export interface MarkInboxItemsReadRequest {
  ids?: string[]
}

export interface MarkInboxReadData {
  marked_count: number
}

export interface DeleteInboxItemsRequest {
  ids: string[]
}

export interface RegisterDeviceTokenRequest {
  platform: 'ios' | 'android' | 'web'
  device_token: string
}

export interface UnregisterDeviceTokenRequest {
  platform: 'ios' | 'android' | 'web'
  device_token: string
}

// ==================== 服务类 ====================

export class InboxService extends BaseService {
  protected readonly baseUrl = '/protected/inbox'

  /**
   * 获取通知列表
   */
  async getList(params: ListInboxItemsRequest = {}): Promise<PaginatedData<InboxItemData>> {
    return this.post<PaginatedData<InboxItemData>>(`${this.baseUrl}/get_list`, params)
  }

  /**
   * 获取未读通知数
   */
  async getUnreadCount(): Promise<UnreadInboxSummary> {
    return this.post<UnreadInboxSummary>(`${this.baseUrl}/get_unread_count`)
  }

  /**
   * 标记通知已读
   */
  async markAsRead(data: MarkInboxItemsReadRequest): Promise<MarkInboxReadData> {
    return this.post<MarkInboxReadData>(`${this.baseUrl}/mark_all_as_read`, data)
  }

  /**
   * 删除通知
   */
  async remove(data: DeleteInboxItemsRequest): Promise<void> {
    return this.post<void>(`${this.baseUrl}/remove`, data)
  }

  /**
   * 注册设备推送Token
   */
  async registerDeviceToken(data: RegisterDeviceTokenRequest): Promise<void> {
    return this.post<void>(`${this.baseUrl}/register_device_token`, data)
  }

  /**
   * 注销设备推送Token
   */
  async unregisterDeviceToken(data: UnregisterDeviceTokenRequest): Promise<void> {
    return this.post<void>(`${this.baseUrl}/unregister_device_token`, data)
  }
}

export const inboxService = new InboxService()

export function useInboxService() {
  return inboxService
}


