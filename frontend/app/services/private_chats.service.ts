import { BaseService } from './base.service'
import type { 
  PrivateChatLastMessage, 
  PrivateChatListItem, 
  PrivateChatDetailData,
  PaginatedData
} from '~/types/api'

/**
 * 私聊管理服务
 * ✅ 类型定义已统一从 types/api.ts 导入
 */

// ==================== Request 类型定义 ====================

export interface CreatePrivateChatRequest {
  target_user_id: string
  initial_message?: string
}

export interface CreatePrivateChatData {
  private_chat_id: string
}

export interface GetPrivateChatsRequest {
  limit?: number
  cursor?: string
  include_archived?: boolean
}

export interface UpdatePrivateChatSettingsRequest {
  private_chat_id: string
  is_muted?: boolean
  is_pinned?: boolean
  is_archived?: boolean
}

export interface DeletePrivateChatRequest {
  private_chat_id: string
}

// ==================== 服务类 ====================

export class PrivateChatsService extends BaseService {
  protected readonly baseUrl = ''

  /**
   * 发起私聊
   */
  async create(data: CreatePrivateChatRequest): Promise<CreatePrivateChatData> {
    return this.post<CreatePrivateChatData>('/protected/private_chats/create', data)
  }

  /**
   * 获取私聊列表
   */
  async getList(params: GetPrivateChatsRequest = {}): Promise<PaginatedData<PrivateChatListItem>> {
    return this.post<PaginatedData<PrivateChatListItem>>('/protected/private_chats/get_list', params)
  }

  /**
   * 获取私聊详情
   */
  async getDetail(privateChatId: string): Promise<PrivateChatDetailData> {
    return this.post<PrivateChatDetailData>('/protected/private_chats/open_the_door', {
      id: privateChatId
    })
  }

  /**
   * 更新私聊设置
   */
  async updateSettings(data: UpdatePrivateChatSettingsRequest): Promise<void> {
    return this.post<void>('/protected/private_chats/update_settings', data)
  }

  /**
   * 删除私聊
   */
  async delete(data: DeletePrivateChatRequest): Promise<void> {
    return this.post<void>('/protected/private_chats/delete', data)
  }
}

export const privateChatsService = new PrivateChatsService()

export function usePrivateChatsService() {
  return privateChatsService
}


