import { BaseService } from './base.service'
import type { GroupMessageItem, MessageReadItem, MessageReactionItem, PaginatedData, UserInfo } from '~/types/api'

/**
 * 群聊消息服务
 * ✅ 类型定义已统一从 types/api.ts 导入
 */

// ==================== Request 类型定义 ====================

export interface SendGroupMessageRequest {
  group_chat_id: string
  type: 'text' | 'image' | 'video' | 'audio' | 'file'
  content: string
  reply_to?: string
  mentions?: string[]
  file_name?: string
  file_size?: number
  thumbnail_url?: string
  duration?: number
}

export interface SendGroupMessageData {
  id: string
  sent_at: string
}

export interface GetGroupMessagesRequest {
  group_chat_id: string
  limit?: number
  cursor?: string
  include_system?: boolean
}

export interface UpdateGroupMessageRequest {
  id: string
  content: string
}

export interface RecallGroupMessageRequest {
  id: string
}

export interface DeleteGroupMessageRequest {
  id: string
  for_everyone?: boolean
}

export interface MarkGroupMessagesReadRequest {
  group_chat_id: string
  up_to?: string
}

export interface GetMessageReadStatusRequest {
  id: string
  limit?: number
  cursor?: string
}

export interface AddGroupMessageReactionRequest {
  id: string
  type: 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow'
  action: 'add' | 'remove'
}

export interface GetGroupMessageReactionsRequest {
  id: string
  type?: 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow'
  limit?: number
  cursor?: string
}

// ==================== 服务类 ====================

export class GroupMessagesService extends BaseService {
  protected readonly baseUrl = ''

  /**
   * 发送群聊消息
   */
  async send(data: SendGroupMessageRequest): Promise<SendGroupMessageData> {
    return this.post<SendGroupMessageData>('/protected/group_messages/send', data)
  }

  /**
   * 获取群聊消息历史
   */
  async getHistory(params: GetGroupMessagesRequest): Promise<PaginatedData<GroupMessageItem>> {
    return this.post<PaginatedData<GroupMessageItem>>('/protected/group_messages/get_history', params)
  }

  /**
   * 编辑群聊消息
   */
  async update(data: UpdateGroupMessageRequest): Promise<void> {
    return this.post<void>('/protected/group_messages/update', data)
  }

  /**
   * 撤回群聊消息
   */
  async recall(data: RecallGroupMessageRequest): Promise<void> {
    return this.post<void>('/protected/group_messages/recall', data)
  }

  /**
   * 删除群聊消息
   */
  async delete(data: DeleteGroupMessageRequest): Promise<void> {
    return this.post<void>('/protected/group_messages/delete', data)
  }

  /**
   * 标记群聊消息已读
   */
  async markAsRead(data: MarkGroupMessagesReadRequest): Promise<void> {
    return this.post<void>('/protected/group_messages/mark_as_read', data)
  }

  /**
   * 获取消息已读状态
   */
  async getReadStatus(params: GetMessageReadStatusRequest): Promise<PaginatedData<MessageReadItem>> {
    return this.post<PaginatedData<MessageReadItem>>('/protected/group_messages/get_read_status', params)
  }

  /**
   * 添加消息反应
   */
  async addReaction(data: AddGroupMessageReactionRequest): Promise<void> {
    return this.post<void>('/protected/group_messages/add_reaction', data)
  }

  /**
   * 获取消息反应列表
   */
  async getReactions(params: GetGroupMessageReactionsRequest): Promise<PaginatedData<MessageReactionItem>> {
    return this.post<PaginatedData<MessageReactionItem>>('/protected/group_messages/get_reactions', params)
  }
}

export const groupMessagesService = new GroupMessagesService()

export function useGroupMessagesService() {
  return groupMessagesService
}


