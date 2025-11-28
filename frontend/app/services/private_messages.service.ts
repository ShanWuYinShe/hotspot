import { BaseService } from './base.service'
import type { PrivateMessageItem, PaginatedData } from '~/types/api'

/**
 * 私聊消息服务
 * ✅ 类型定义已统一从 types/api.ts 导入
 */

// ==================== Request 类型定义 ====================

export interface SendPrivateMessageRequest {
  private_chat_id: string
  type: 'text' | 'image' | 'video' | 'audio' | 'file'
  content: string
  reply_to?: string
  file_name?: string
  file_size?: number
  thumbnail_url?: string
  duration?: number
}

export interface SendPrivateMessageData {
  id: string
  sent_at: string
}

export interface GetPrivateMessagesRequest {
  private_chat_id: string
  limit?: number
  cursor?: string
}

export interface UpdatePrivateMessageRequest {
  id: string
  content: string
}

export interface RecallPrivateMessageRequest {
  id: string
}

export interface DeletePrivateMessageRequest {
  id: string
}

export interface MarkPrivateMessagesReadRequest {
  private_chat_id: string
  up_to?: string
}

// ==================== 服务类 ====================

export class PrivateMessagesService extends BaseService {
  protected readonly baseUrl = ''

  /**
   * 发送私聊消息
   */
  async send(data: SendPrivateMessageRequest): Promise<SendPrivateMessageData> {
    return this.post<SendPrivateMessageData>('/protected/private_messages/send', data)
  }

  /**
   * 获取私聊消息历史
   */
  async getHistory(params: GetPrivateMessagesRequest): Promise<PaginatedData<PrivateMessageItem>> {
    return this.post<PaginatedData<PrivateMessageItem>>('/protected/private_messages/get_history', params)
  }

  /**
   * 编辑私聊消息
   */
  async update(data: UpdatePrivateMessageRequest): Promise<void> {
    return this.post<void>('/protected/private_messages/update', data)
  }

  /**
   * 撤回私聊消息
   */
  async recall(data: RecallPrivateMessageRequest): Promise<void> {
    return this.post<void>('/protected/private_messages/recall', data)
  }

  /**
   * 删除私聊消息
   */
  async delete(data: DeletePrivateMessageRequest): Promise<void> {
    return this.post<void>('/protected/private_messages/delete', data)
  }

  /**
   * 标记私聊消息已读
   */
  async markAsRead(data: MarkPrivateMessagesReadRequest): Promise<void> {
    return this.post<void>('/protected/private_messages/mark_as_read', data)
  }
}

export const privateMessagesService = new PrivateMessagesService()

export function usePrivateMessagesService() {
  return privateMessagesService
}


