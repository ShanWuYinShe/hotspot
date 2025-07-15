/**
 * 消息状态管理 Store
 * 使用 Pinia 管理消息、对话和系统通知状态
 * 与后端 message API 对接，确保数据一致性
 */

import { defineStore } from 'pinia'
import { ref, computed, readonly } from 'vue'
import { useMessageService } from '~/services/message.service'
import type {
  SendDirectMessageRequest,
  SendGroupMessageRequest,
  DirectMessageHistoryItem,
  GroupMessageHistoryItem,
  ConversationItem,
  ConversationsRequest,
  DirectMessageHistoryRequest,
  GroupMessageHistoryRequest,
  MarkDirectMessageReadRequest,
  DeleteDirectMessageRequest,
  DeleteGroupMessageRequest,
  SystemNotificationData
} from '~/types/api/message'

// 为了类型简化，定义本地类型别名
type DirectMessage = DirectMessageHistoryItem
type GroupMessage = GroupMessageHistoryItem
type Conversation = ConversationItem
type SystemNotification = SystemNotificationData

export const useMessageStore = defineStore('message', () => {
  // ==================== 状态定义 ====================
  
  // 对话列表
  const conversations = ref<Conversation[]>([])
  
  // 私聊消息列表（按conversation_id分组）
  const directMessages = ref<Map<string, DirectMessage[]>>(new Map())
  
  // 群组消息列表（按group_id分组）
  const groupMessages = ref<Map<string, GroupMessage[]>>(new Map())
  
  // 系统通知列表
  const systemNotifications = ref<SystemNotification[]>([])
  
  // 未读消息数量
  const unreadCounts = ref<Map<string, number>>(new Map())
  
  // 加载状态
  const isLoading = ref(false)
  const isLoadingHistory = ref(false)
  const isSending = ref(false)
  
  // 错误状态
  const error = ref<string | null>(null)
  
  // ==================== 计算属性 ====================
  
  // 总未读消息数
  const totalUnreadCount = computed(() => {
    return Array.from(unreadCounts.value.values()).reduce((sum, count) => sum + count, 0)
  })
  
  // 是否有未读消息
  const hasUnreadMessages = computed(() => totalUnreadCount.value > 0)
  
  // 按时间排序的对话列表
  const sortedConversations = computed(() => {
    return [...conversations.value].sort((a, b) => {
      const aTime = new Date(a.last_message_time).getTime()
      const bTime = new Date(b.last_message_time).getTime()
      return bTime - aTime
    })
  })
  
  // ==================== Service 实例 ====================
  
  const messageService = useMessageService()
  
  // ==================== Actions ====================
  
  /**
   * 获取对话列表
   */
  async function fetchConversations(params: ConversationsRequest = { limit: 50 }) {
    isLoading.value = true
    error.value = null
    
    try {
      const response = await messageService.getConversations(params)
      
      conversations.value = response.items
      
      // 更新未读数量
      response.items.forEach(conversation => {
        if (conversation.unread_count > 0) {
          unreadCounts.value.set(conversation.conversation_id, conversation.unread_count)
        }
      })
      
      return response
    } catch (err: any) {
      const errorMessage = err.message || '获取对话列表失败'
      error.value = errorMessage
      throw new Error(errorMessage)
    } finally {
      isLoading.value = false
    }
  }
  
  /**
   * 获取私聊消息历史
   */
  async function fetchDirectMessageHistory(params: DirectMessageHistoryRequest) {
    isLoadingHistory.value = true
    error.value = null
    
    try {
      const response = await messageService.getDirectMessages(params)
      const conversationId = params.conversation_id
      
      if (!conversationId) {
        throw new Error('conversation_id is required')
      }
      
      directMessages.value.set(conversationId, response.items)
      
      return response
    } catch (err: any) {
      const errorMessage = err.message || '获取私聊历史失败'
      error.value = errorMessage
      throw new Error(errorMessage)
    } finally {
      isLoadingHistory.value = false
    }
  }
  
  /**
   * 获取群组消息历史
   */
  async function fetchGroupMessageHistory(params: GroupMessageHistoryRequest) {
    isLoadingHistory.value = true
    error.value = null
    
    try {
      const response = await messageService.getGroupMessages(params)
      const groupId = params.group_id
      
      groupMessages.value.set(groupId, response.items)
      
      return response
    } catch (err: any) {
      const errorMessage = err.message || '获取群组消息历史失败'
      error.value = errorMessage
      throw new Error(errorMessage)
    } finally {
      isLoadingHistory.value = false
    }
  }
  
  /**
   * 发送私聊消息
   */
  async function sendDirectMessage(data: SendDirectMessageRequest) {
    isSending.value = true
    error.value = null
    
    try {
      const response = await messageService.sendDirectMessage(data)
      
      // 立即添加消息到本地状态（乐观更新）
      const newMessage: DirectMessage = {
        message_id: response.message_id,
        conversation_id: response.conversation_id,
        sender_id: response.recipient_id, // 从当前用户角度
        sender_name: '', // 需要从用户store获取
        recipient_id: data.recipient_id,
        content: data.content,
        message_type: data.message_type || 'text',
        sent_at: response.sent_at,
        read_at: undefined,
        latitude: data.latitude || 0,
        longitude: data.longitude || 0
      }
      
      const conversationId = response.conversation_id
      const existing = directMessages.value.get(conversationId) || []
      directMessages.value.set(conversationId, [newMessage, ...existing])
      
      // 更新对话列表中的最后消息
      updateConversationLastMessage(conversationId, newMessage.content, newMessage.sent_at)
      
      return response
    } catch (err: any) {
      const errorMessage = err.message || '发送消息失败'
      error.value = errorMessage
      throw new Error(errorMessage)
    } finally {
      isSending.value = false
    }
  }
  
  /**
   * 发送群组消息
   */
  async function sendGroupMessage(data: SendGroupMessageRequest) {
    isSending.value = true
    error.value = null
    
    try {
      const response = await messageService.sendGroupMessage(data)
      
      // 立即添加消息到本地状态（乐观更新）
      const newMessage: GroupMessage = {
        message_id: response.message_id,
        group_id: response.group_id,
        sender_id: '', // 将由后端提供
        sender_name: '', // 将由后端提供
        content: data.content,
        message_type: data.message_type || 'text',
        sent_at: response.sent_at,
        latitude: data.latitude || 0,
        longitude: data.longitude || 0
      }
      
      const groupId = response.group_id
      const existing = groupMessages.value.get(groupId) || []
      groupMessages.value.set(groupId, [newMessage, ...existing])
      
      return response
    } catch (err: any) {
      const errorMessage = err.message || '发送群组消息失败'
      error.value = errorMessage
      throw new Error(errorMessage)
    } finally {
      isSending.value = false
    }
  }
  
  /**
   * 标记私聊消息为已读
   */
  async function markDirectMessagesAsRead(data: MarkDirectMessageReadRequest) {
    try {
      await messageService.markMessagesRead(data)
      
      // 更新本地未读数量
      unreadCounts.value.set(data.conversation_id, 0)
      
      // 更新对话列表中的未读数量
      const conversation = conversations.value.find(c => c.conversation_id === data.conversation_id)
      if (conversation) {
        conversation.unread_count = 0
      }
      
      // 更新本地消息的已读状态
      const messages = directMessages.value.get(data.conversation_id)
      if (messages) {
        const now = new Date().toISOString()
        messages.forEach(message => {
          if (!message.read_at) {
            message.read_at = now
          }
        })
      }
    } catch (err: any) {
      const errorMessage = err.message || '标记消息已读失败'
      error.value = errorMessage
      throw new Error(errorMessage)
    }
  }
  
  /**
   * 标记群组消息为已读
   */
  async function markGroupMessagesAsRead(groupId: string) {
    try {
      await messageService.markGroupMessagesRead({ conversation_id: groupId })
      
      // 更新本地未读数量
      unreadCounts.value.set(groupId, 0)
    } catch (err: any) {
      const errorMessage = err.message || '标记群组消息已读失败'
      error.value = errorMessage
      throw new Error(errorMessage)
    }
  }
  
  /**
   * 删除私聊消息
   */
  async function deleteDirectMessage(data: DeleteDirectMessageRequest) {
    try {
      await messageService.deleteDirectMessage(data)
      
      // 从本地状态中移除消息
      for (const [conversationId, messages] of directMessages.value.entries()) {
        const filtered = messages.filter(m => m.message_id !== data.message_id)
        if (filtered.length !== messages.length) {
          directMessages.value.set(conversationId, filtered)
          break
        }
      }
    } catch (err: any) {
      const errorMessage = err.message || '删除消息失败'
      error.value = errorMessage
      throw new Error(errorMessage)
    }
  }
  
  /**
   * 删除群组消息
   */
  async function deleteGroupMessage(data: DeleteGroupMessageRequest) {
    try {
      await messageService.deleteGroupMessage(data)
      
      // 从本地状态中移除消息
      for (const [groupId, messages] of groupMessages.value.entries()) {
        const filtered = messages.filter(m => m.message_id !== data.message_id)
        if (filtered.length !== messages.length) {
          groupMessages.value.set(groupId, filtered)
          break
        }
      }
    } catch (err: any) {
      const errorMessage = err.message || '删除群组消息失败'
      error.value = errorMessage
      throw new Error(errorMessage)
    }
  }
  
  /**
   * 获取未读消息数量
   */
  async function fetchUnreadCounts() {
    try {
      const directCount = await messageService.getUnreadCount()
      
      // 更新总未读数量
      unreadCounts.value.set('direct_total', directCount.unread_count)
    } catch (err: any) {
      console.warn('获取未读消息数量失败:', err)
    }
  }
  
  /**
   * 初始化系统通知
   */
  async function initializeSystemNotifications() {
    try {
      // 订阅系统通知WebSocket
      const { useWebSocketNotificationService } = await import('~/services/websocket-notification.service')
      const wsNotificationService = useWebSocketNotificationService()
      
      wsNotificationService.subscribeToWebSocketEvent('system_notification', (notification: any) => {
        // 构建现代化的通知对象
        const systemNotification: SystemNotification = {
          notification_id: notification.notification_id || crypto.randomUUID(),
          notification_type: notification.notification_type || 'system',
          title: notification.title || '系统通知',
          content: notification.content,
          data: notification.data,
          created_at: notification.created_at || new Date().toISOString()
        }
        
        // 添加到系统通知列表
        systemNotifications.value.unshift(systemNotification)
      })
    } catch (err: any) {
      console.warn('初始化系统通知失败:', err)
    }
  }
  
  // ==================== 辅助方法 ====================
  
  /**
   * 更新对话的最后消息信息
   */
  function updateConversationLastMessage(conversationId: string, content: string, sentAt: string) {
    const conversation = conversations.value.find(c => c.conversation_id === conversationId)
    if (conversation) {
      conversation.last_message_preview = content
      conversation.last_message_time = sentAt
    }
  }
  
  /**
   * 获取指定对话的私聊消息
   */
  function getDirectMessagesForConversation(conversationId: string): DirectMessage[] {
    return directMessages.value.get(conversationId) || []
  }
  
  /**
   * 获取指定群组的消息
   */
  function getGroupMessagesForGroup(groupId: string): GroupMessage[] {
    return groupMessages.value.get(groupId) || []
  }
  
  /**
   * 清空所有消息和状态
   */
  function clearAll() {
    conversations.value = []
    directMessages.value.clear()
    groupMessages.value.clear()
    systemNotifications.value = []
    unreadCounts.value.clear()
    error.value = null
  }
  
  // ==================== 返回 Store API ====================
  
  return {
    // 状态
    conversations: readonly(conversations),
    directMessages: readonly(directMessages),
    groupMessages: readonly(groupMessages),
    systemNotifications: readonly(systemNotifications),
    unreadCounts: readonly(unreadCounts),
    isLoading: readonly(isLoading),
    isLoadingHistory: readonly(isLoadingHistory),
    isSending: readonly(isSending),
    error: readonly(error),
    
    // 计算属性
    totalUnreadCount,
    hasUnreadMessages,
    sortedConversations,
    
    // Actions
    fetchConversations,
    fetchDirectMessageHistory,
    fetchGroupMessageHistory,
    sendDirectMessage,
    sendGroupMessage,
    markDirectMessagesAsRead,
    markGroupMessagesAsRead,
    deleteDirectMessage,
    deleteGroupMessage,
    fetchUnreadCounts,
    initializeSystemNotifications,
    
    // 辅助方法
    getDirectMessagesForConversation,
    getGroupMessagesForGroup,
    clearAll
  }
})
