/**
 * 消息服务
 * 所有API操作统一使用POST方法
 */

import { BaseService } from './base.service'
import type { PaginatedData } from '~/types/api'

// ============ 私聊管理 ============

export interface CreatePrivateChatRequest {
  target_user_id: string
  initial_message?: string
}

export interface GetPrivateChatsRequest {
  limit?: number
  cursor?: string
  include_archived?: boolean
}

export interface OpenPrivateChatDoorRequest {
  id: string  // private_chat_id
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

// ============ 私聊消息 ============

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

// ============ 群聊消息 ============

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

// ============ 消息通用 ============

export interface MarkAllMessagesReadRequest {}

export class MessagesService extends BaseService {
  protected readonly baseUrl = ''

  // ============ 私聊管理 API ============

  /**
   * 发起私聊
   * POST /api/v1/protected/private_chats/create
   */
  async createPrivateChat(request: CreatePrivateChatRequest): Promise<{ private_chat_id: string }> {
    return this.post<{ private_chat_id: string }>('/protected/private_chats/create', request)
  }

  /**
   * 获取私聊列表
   * POST /api/v1/protected/private_chats/get_list
   */
  async getPrivateChats(request: GetPrivateChatsRequest = {}): Promise<PaginatedData<{
    private_chat_id: string
    other_user: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    unread_count: number
    last_message?: {
      id: string
      content: string
      type: 'text' | 'image' | 'video' | 'audio' | 'file'
      sender_id: string
      sent_at: string
    }
    last_activity_at: string
  }>> {
    return this.post('/protected/private_chats/get_list', request)
  }

  /**
   * 获取私聊详情
   * POST /api/v1/protected/private_chats/open_the_door
   */
  async openPrivateChatDoor(request: OpenPrivateChatDoorRequest): Promise<{
    private_chat_id: string
    other_user: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    other_user_status: {
      is_online: boolean
      last_seen?: string
    }
    unread_count: number
    is_muted: boolean
    is_pinned: boolean
    is_archived: boolean
    last_message?: {
      id: string
      content: string
      type: 'text' | 'image' | 'video' | 'audio' | 'file'
      sender_id: string
      sent_at: string
    }
    last_activity_at: string
    created_at: string
  }> {
    return this.post('/protected/private_chats/open_the_door', request)
  }

  /**
   * 更新私聊设置
   * POST /api/v1/protected/private_chats/update_settings
   */
  async updatePrivateChatSettings(request: UpdatePrivateChatSettingsRequest): Promise<void> {
    return this.post<void>('/protected/private_chats/update_settings', request)
  }

  /**
   * 删除私聊
   * POST /api/v1/protected/private_chats/delete
   */
  async deletePrivateChat(request: DeletePrivateChatRequest): Promise<void> {
    return this.post<void>('/protected/private_chats/delete', request)
  }

  // ============ 私聊消息 API ============

  /**
   * 发送私聊消息
   * POST /api/v1/protected/private_messages/send
   */
  async sendPrivateMessage(request: SendPrivateMessageRequest): Promise<{
    id: string
    sent_at: string
  }> {
    return this.post('/protected/private_messages/send', request)
  }

  /**
   * 获取私聊消息历史
   * POST /api/v1/protected/private_messages/get_history
   */
  async getPrivateMessages(request: GetPrivateMessagesRequest): Promise<PaginatedData<{
    id: string
    type: 'text' | 'image' | 'video' | 'audio' | 'file'
    content: string
    sender: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    reply_to?: string
    sent_at: string
    edited_at?: string
    is_recalled: boolean
    recalled_at?: string
  }>> {
    return this.post('/protected/private_messages/get_history', request)
  }

  /**
   * 编辑私聊消息
   * POST /api/v1/protected/private_messages/update
   */
  async updatePrivateMessage(request: UpdatePrivateMessageRequest): Promise<void> {
    return this.post<void>('/protected/private_messages/update', request)
  }

  /**
   * 撤回私聊消息
   * POST /api/v1/protected/private_messages/recall
   */
  async recallPrivateMessage(request: RecallPrivateMessageRequest): Promise<void> {
    return this.post<void>('/protected/private_messages/recall', request)
  }

  /**
   * 删除私聊消息
   * POST /api/v1/protected/private_messages/delete
   */
  async deletePrivateMessage(request: DeletePrivateMessageRequest): Promise<void> {
    return this.post<void>('/protected/private_messages/delete', request)
  }

  /**
   * 标记私聊消息已读
   * POST /api/v1/protected/private_messages/mark_as_read
   */
  async markPrivateMessagesRead(request: MarkPrivateMessagesReadRequest): Promise<void> {
    return this.post<void>('/protected/private_messages/mark_as_read', request)
  }

  // ============ 群聊消息 API ============

  /**
   * 发送群聊消息
   * POST /api/v1/protected/group_messages/send
   */
  async sendGroupMessage(request: SendGroupMessageRequest): Promise<{
    id: string
    sent_at: string
  }> {
    return this.post('/protected/group_messages/send', request)
  }

  /**
   * 获取群聊消息历史
   * POST /api/v1/protected/group_messages/get_history
   */
  async getGroupMessages(request: GetGroupMessagesRequest): Promise<PaginatedData<{
    id: string
    type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'system'
    content: string
    sender: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    reply_to?: string
    mentions?: string[]
    sent_at: string
    edited_at?: string
    is_recalled: boolean
    recalled_by?: string
    recalled_at?: string
    read_count?: number
  }>> {
    return this.post('/protected/group_messages/get_history', request)
  }

  /**
   * 编辑群聊消息
   * POST /api/v1/protected/group_messages/update
   */
  async updateGroupMessage(request: UpdateGroupMessageRequest): Promise<void> {
    return this.post<void>('/protected/group_messages/update', request)
  }

  /**
   * 撤回群聊消息
   * POST /api/v1/protected/group_messages/recall
   */
  async recallGroupMessage(request: RecallGroupMessageRequest): Promise<void> {
    return this.post<void>('/protected/group_messages/recall', request)
  }

  /**
   * 删除群聊消息
   * POST /api/v1/protected/group_messages/delete
   */
  async deleteGroupMessage(request: DeleteGroupMessageRequest): Promise<void> {
    return this.post<void>('/protected/group_messages/delete', request)
  }

  /**
   * 标记群聊消息已读
   * POST /api/v1/protected/group_messages/mark_as_read
   */
  async markGroupMessagesRead(request: MarkGroupMessagesReadRequest): Promise<void> {
    return this.post<void>('/protected/group_messages/mark_as_read', request)
  }

  /**
   * 获取消息已读状态
   * POST /api/v1/protected/group_messages/get_read_status
   */
  async getMessageReadStatus(request: GetMessageReadStatusRequest): Promise<PaginatedData<{
    user: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    read_at: string
  }>> {
    return this.post('/protected/group_messages/get_read_status', request)
  }

  /**
   * 添加消息反应
   * POST /api/v1/protected/group_messages/add_reaction
   */
  async addGroupMessageReaction(request: AddGroupMessageReactionRequest): Promise<void> {
    return this.post<void>('/protected/group_messages/add_reaction', request)
  }

  /**
   * 获取消息反应列表
   * POST /api/v1/protected/group_messages/get_reactions
   */
  async getGroupMessageReactions(request: GetGroupMessageReactionsRequest): Promise<PaginatedData<{
    user: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    type: 'like' | 'love' | 'laugh' | 'angry' | 'sad' | 'wow'
    created_at: string
  }>> {
    return this.post('/protected/group_messages/get_reactions', request)
  }

  // ============ 消息通用 API ============

  /**
   * 标记所有消息已读
   * POST /api/v1/protected/messages/mark_all_as_read
   */
  async markAllMessagesRead(request: MarkAllMessagesReadRequest = {}): Promise<void> {
    return this.post<void>('/protected/messages/mark_all_as_read', request)
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
