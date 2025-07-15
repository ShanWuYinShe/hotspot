/**
 * 消息服务层
 * 处理消息和对话相关的API调用，与后端message API对接
 * 使用 Composition API 格式，提高代码复用性
 */

import { useApiService } from './api.factory'
import type {
  SendDirectMessageRequest,
  SendDirectMessageResponse,
  SendGroupMessageRequest,
  SendGroupMessageResponse,
  ConversationsRequest,
  ConversationsResponse,
  DirectMessageHistoryRequest,
  DirectMessageHistoryResponse,
  GroupMessageHistoryRequest,
  GroupMessageHistoryResponse,
  MarkDirectMessageReadRequest,
  MarkDirectMessageReadResponse,
  DeleteDirectMessageRequest,
  DeleteGroupMessageRequest,
  DeleteMessageResponse,
  UnreadCountResponse,
  SendTypingIndicatorRequest
} from '~/types/api/message'

/**
 * 消息服务 Composition API
 * 统一管理API服务实例，避免重复调用
 */
export function useMessageService() {
  const apiService = useApiService()

  /**
   * 发送私聊消息
   */
  const sendDirectMessage = async (data: SendDirectMessageRequest): Promise<SendDirectMessageResponse> => {
    const result = await apiService.safeRequest<SendDirectMessageResponse>('/chat/user/send', data, { authRequired: true })
    if (!result) throw new Error('发送消息失败')
    return result
  }

  /**
   * 发送群组消息
   */
  const sendGroupMessage = async (data: SendGroupMessageRequest): Promise<SendGroupMessageResponse> => {
    const result = await apiService.safeRequest<SendGroupMessageResponse>('/chat/group/send', data, { authRequired: true })
    if (!result) throw new Error('发送群组消息失败')
    return result
  }

  /**
   * 获取对话列表
   */
  const getConversations = async (params: ConversationsRequest): Promise<ConversationsResponse> => {
    const result = await apiService.safeRequest<ConversationsResponse>('/chat/user/conversations', params, { authRequired: true })
    if (!result) throw new Error('获取对话列表失败')
    return result
  }

  /**
   * 获取私聊消息历史
   */
  const getDirectMessages = async (params: DirectMessageHistoryRequest): Promise<DirectMessageHistoryResponse> => {
    const result = await apiService.safeRequest<DirectMessageHistoryResponse>('/chat/user/history', params, { authRequired: true })
    if (!result) throw new Error('获取消息历史失败')
    return result
  }

  /**
   * 获取群组消息历史
   */
  const getGroupMessages = async (params: GroupMessageHistoryRequest): Promise<GroupMessageHistoryResponse> => {
    const result = await apiService.safeRequest<GroupMessageHistoryResponse>('/chat/group/history', params, { authRequired: true })
    if (!result) throw new Error('获取群组消息历史失败')
    return result
  }

  /**
   * 标记消息已读
   */
  const markMessagesRead = async (data: MarkDirectMessageReadRequest): Promise<MarkDirectMessageReadResponse> => {
    const result = await apiService.safeRequest<MarkDirectMessageReadResponse>('/chat/user/mark-read', data, { authRequired: true })
    if (!result) throw new Error('标记已读失败')
    return result
  }

  /**
   * 删除私聊消息
   */
  const deleteDirectMessage = async (data: DeleteDirectMessageRequest): Promise<DeleteMessageResponse> => {
    const result = await apiService.safeRequest<DeleteMessageResponse>('/chat/user/delete', data, { authRequired: true })
    if (!result) throw new Error('删除消息失败')
    return result
  }

  /**
   * 删除群组消息
   */
  const deleteGroupMessage = async (data: DeleteGroupMessageRequest): Promise<DeleteMessageResponse> => {
    const result = await apiService.safeRequest<DeleteMessageResponse>('/chat/group/delete', data, { authRequired: true })
    if (!result) throw new Error('删除群组消息失败')
    return result
  }

  /**
   * 获取未读消息数量 (私聊)
   */
  const getUnreadCount = async (): Promise<UnreadCountResponse> => {
    const result = await apiService.safeRequest<UnreadCountResponse>('/chat/user/unread_count', {}, { authRequired: true })
    if (!result) throw new Error('获取未读数量失败')
    return result
  }

  /**
   * 标记群组消息为已读
   */
  const markGroupMessagesRead = async (data: MarkDirectMessageReadRequest): Promise<MarkDirectMessageReadResponse> => {
    const result = await apiService.safeRequest<MarkDirectMessageReadResponse>('/chat/group/mark-read', data, { authRequired: true })
    if (!result) throw new Error('标记群组消息已读失败')
    return result
  }

  /**
   * 获取群组未读消息数量
   */
  const getGroupUnreadCount = async (groupId: string): Promise<UnreadCountResponse> => {
    const result = await apiService.safeRequest<UnreadCountResponse>('/chat/group/unread_count', { group_id: groupId }, { authRequired: true })
    if (!result) throw new Error('获取群组未读数量失败')
    return result
  }

  /**
   * 发送输入状态指示器 (暂时未实现)
   */
  const sendTypingIndicator = async (data: SendTypingIndicatorRequest): Promise<void> => {
    // TODO: 后端需要实现 typing 端点
    console.warn('Typing indicator not implemented on backend')
    return Promise.resolve()
  }

  return {
    sendDirectMessage,
    sendGroupMessage,
    getConversations,
    getDirectMessages,
    getGroupMessages,
    markMessagesRead,
    markGroupMessagesRead,
    deleteDirectMessage,
    deleteGroupMessage,
    getUnreadCount,
    getGroupUnreadCount,
    sendTypingIndicator
  }
}
