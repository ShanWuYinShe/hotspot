/**
 * 消息服务 - 现代化实现
 * 提供完整的聊天功能
 */

import { BaseService } from './base.service'
import type { 
  SendMessageResponse, 
  ConversationsResponse,
  SendMessageRequest,
  PaginatedResponse 
} from '~/types/api'

export interface CreateConversationRequest {
  type: 'user' | 'group'
  target_id: string
  initial_message?: string
}

export interface MarkReadRequest {
  conversation_id: string
  message_id?: string
}

export class MessagesService extends BaseService {
  protected readonly baseUrl = '/protected/chat'

  /**
   * 获取会话列表
   */
  async getConversations(params?: {
    cursor?: string
    limit?: number
  }): Promise<PaginatedResponse<ConversationsResponse>> {
    return this.paginate<ConversationsResponse>('/conversations', params)
  }

  /**
   * 获取消息历史
   */
  async getMessages(params: {
    conversation_id: string
    cursor?: string
    limit?: number
  }): Promise<PaginatedResponse<SendMessageResponse>> {
    return this.paginate<SendMessageResponse>('/messages', params)
  }

  /**
   * 发送消息
   */
  async sendMessage(data: SendMessageRequest): Promise<SendMessageResponse> {
    return this.post<SendMessageResponse>('/send', data)
  }

  /**
   * 发送文件消息
   */
  async sendFileMessage(
    conversationId: string,
    file: File,
    messageType: 'image' | 'file' = 'file'
  ): Promise<SendMessageResponse> {
    return this.upload<SendMessageResponse>('/send-file', file, {
      conversation_id: conversationId,
      message_type: messageType
    })
  }

  /**
   * 删除消息
   */
  async deleteMessage(messageId: string): Promise<void> {
    return this.delete<void>(`/messages/${messageId}`)
  }

  /**
   * 标记消息已读
   */
  async markAsRead(data: MarkReadRequest): Promise<void> {
    return this.post<void>('/mark-read', data)
  }

  /**
   * 创建会话
   */
  async createConversation(data: CreateConversationRequest): Promise<{ conversation_id: string }> {
    return this.post<{ conversation_id: string }>('/conversations', data)
  }

  /**
   * 获取会话详情
   */
  async getConversationDetails(conversationId: string): Promise<ConversationsResponse> {
    return this.get<ConversationsResponse>(`/conversations/${conversationId}`)
  }

  /**
   * 删除会话
   */
  async deleteConversation(conversationId: string): Promise<void> {
    return this.delete<void>(`/conversations/${conversationId}`)
  }

  /**
   * 静音/取消静音会话
   */
  async toggleMute(conversationId: string, muted: boolean): Promise<void> {
    return this.post<void>('/toggle-mute', {
      conversation_id: conversationId,
      muted
    })
  }

  /**
   * 置顶/取消置顶会话
   */
  async togglePin(conversationId: string, pinned: boolean): Promise<void> {
    return this.post<void>('/toggle-pin', {
      conversation_id: conversationId,
      pinned
    })
  }

  /**
   * 搜索消息
   */
  async searchMessages(params: {
    query: string
    conversation_id?: string
    message_type?: string
    start_date?: string
    end_date?: string
  }): Promise<PaginatedResponse<MessageResponse>> {
    return this.paginate<MessageResponse>('/search', params)
  }

  /**
   * 获取未读消息统计
   */
  async getUnreadStats(): Promise<{
    total_unread: number
    conversations: Array<{
      conversation_id: string
      unread_count: number
    }>
  }> {
    return this.get<{
      total_unread: number
      conversations: Array<{
        conversation_id: string
        unread_count: number
      }>
    }>('/unread-stats')
  }

  /**
   * 批量标记已读
   */
  async batchMarkRead(conversationIds: string[]): Promise<void> {
    return this.post<void>('/batch-mark-read', {
      conversation_ids: conversationIds
    })
  }
}

// 创建服务实例
export const messagesService = new MessagesService()

// 提供 composable 接口
export function useMessagesService() {
  return messagesService
}

// 兼容旧的导出名称
export function useMessageService() {
  return messagesService
}
