/**
 * 消息状态存储
 * 职责：消息状态管理 + API 调用
 */

import { defineStore } from 'pinia'
import type { PrivateChatListItem, PrivateMessageItem, GroupMessageItem, UserInfo } from '~/types/api'
import { useMessagesService, type SendPrivateMessageRequest, type SendGroupMessageRequest } from '~/services/messages.service'

// 消息类型（带本地状态）
export interface Message {
  id: string
  type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'system'
  content: string
  sender: UserInfo
  reply_to?: string
  mentions?: string[]
  sent_at: string
  edited_at?: string
  is_recalled: boolean
  recalled_by?: string
  recalled_at?: string
  read_count?: number
  // 本地状态
  sending?: boolean
  failed?: boolean
  optimistic?: boolean
}

// 会话类型（带本地状态）
export interface Conversation {
  chat_id: string
  chat_type: 'private' | 'group'
  name: string
  avatar_url?: string
  unread_count: number
  last_message?: {
    id: string
    content: string
    type: string
    sender_id: string
    sent_at: string
  }
  last_activity_at: string
  // 本地状态
  draft?: string
  typing?: boolean
}

export const useMessagesStore = defineStore('messages', () => {
  // ============ 状态 ============
  const conversations = ref<Map<string, Conversation>>(new Map())
  const messages = ref<Map<string, Message[]>>(new Map())
  const currentConversationId = ref<string | null>(null)
  const loading = ref(false)

  // ============ 计算属性 ============
  const conversationsList = computed(() => {
    return Array.from(conversations.value.values())
      .sort((a, b) => new Date(b.last_activity_at).getTime() - new Date(a.last_activity_at).getTime())
  })
  
  const currentConversation = computed(() => {
    if (!currentConversationId.value) return null
    return conversations.value.get(currentConversationId.value) || null
  })
  
  const currentMessages = computed(() => {
    if (!currentConversationId.value) return []
    return messages.value.get(currentConversationId.value) || []
  })
  
  const totalUnreadCount = computed(() => {
    return Array.from(conversations.value.values())
      .reduce((total, conv) => total + (conv.unread_count || 0), 0)
  })
  
  // ============ 会话状态操作 ============
  
  /**
   * 设置会话列表
   */
  const setConversations = (conversationsList: Conversation[]) => {
    conversations.value.clear()
    conversationsList.forEach(conv => {
      conversations.value.set(conv.chat_id, conv)
    })
  }
  
  /**
   * 添加或更新会话
   */
  const upsertConversation = (conversation: Conversation) => {
    conversations.value.set(conversation.chat_id, conversation)
  }
  
  /**
   * 删除会话
   */
  const removeConversation = (chatId: string) => {
    conversations.value.delete(chatId)
    messages.value.delete(chatId)
  }
  
  /**
   * 设置当前会话
   */
  const setCurrentConversation = (chatId: string | null) => {
    currentConversationId.value = chatId
    
    // 自动标记为已读
    if (chatId) {
      const conversation = conversations.value.get(chatId)
      if (conversation && conversation.unread_count) {
        conversation.unread_count = 0
      }
    }
  }
  
  /**
   * 更新会话未读计数
   */
  const updateUnreadCount = (chatId: string, count: number) => {
    const conversation = conversations.value.get(chatId)
    if (conversation) {
      conversation.unread_count = count
    }
  }
  
  /**
   * 设置会话草稿
   */
  const setDraft = (chatId: string, draft: string) => {
    const conversation = conversations.value.get(chatId)
    if (conversation) {
      conversation.draft = draft
    }
  }
  
  /**
   * 设置会话输入状态
   */
  const setTyping = (chatId: string, typing: boolean) => {
    const conversation = conversations.value.get(chatId)
    if (conversation) {
      conversation.typing = typing
    }
  }
  
  // ============ 消息状态操作 ============
  
  /**
   * 设置会话的消息列表
   */
  const setMessages = (chatId: string, messagesList: Message[]) => {
    messages.value.set(chatId, [...messagesList])
  }
  
  /**
   * 添加消息到会话开头（用于加载历史消息）
   */
  const prependMessages = (chatId: string, messagesList: Message[]) => {
    const existing = messages.value.get(chatId) || []
    messages.value.set(chatId, [...messagesList, ...existing])
  }
  
  /**
   * 添加消息到会话末尾（用于新消息）
   */
  const appendMessage = (chatId: string, message: Message) => {
    const existing = messages.value.get(chatId) || []
    existing.push(message)
    messages.value.set(chatId, existing)
    
    // 更新会话的最后消息信息
    updateConversationLastMessage(chatId, message)
  }
  
  /**
   * 更新消息
   */
  const updateMessage = (chatId: string, messageId: string, updates: Partial<Message>) => {
    const conversationMessages = messages.value.get(chatId) || []
    const index = conversationMessages.findIndex(m => m.id === messageId)
    if (index !== -1) {
      conversationMessages[index] = { ...conversationMessages[index], ...updates }
    }
  }
  
  /**
   * 替换消息（用于乐观更新）
   */
  const replaceMessage = (chatId: string, oldMessageId: string, newMessage: Message) => {
    const conversationMessages = messages.value.get(chatId) || []
    const index = conversationMessages.findIndex(m => m.id === oldMessageId)
    if (index !== -1) {
      conversationMessages[index] = newMessage
    }
  }
  
  /**
   * 删除消息
   */
  const removeMessage = (chatId: string, messageId: string) => {
    const conversationMessages = messages.value.get(chatId) || []
    const filteredMessages = conversationMessages.filter(m => m.id !== messageId)
    messages.value.set(chatId, filteredMessages)
  }
  
  /**
   * 标记消息发送中
   */
  const markMessageSending = (chatId: string, messageId: string) => {
    updateMessage(chatId, messageId, { sending: true, failed: false })
  }
  
  /**
   * 标记消息发送成功
   */
  const markMessageSent = (chatId: string, messageId: string) => {
    updateMessage(chatId, messageId, { sending: false, failed: false, optimistic: false })
  }
  
  /**
   * 标记消息发送失败
   */
  const markMessageFailed = (chatId: string, messageId: string) => {
    updateMessage(chatId, messageId, { sending: false, failed: true })
  }
  
  // ============ 辅助方法 ============
  
  /**
   * 更新会话的最后消息信息
   */
  const updateConversationLastMessage = (chatId: string, message: Message) => {
    const conversation = conversations.value.get(chatId)
    if (conversation) {
      conversation.last_message = {
        id: message.id,
        content: message.content,
        type: message.type,
        sender_id: message.sender.user_id,
        sent_at: message.sent_at
      }
      conversation.last_activity_at = message.sent_at
    }
  }
  
  /**
   * 获取会话
   */
  const getConversation = (chatId: string): Conversation | undefined => {
    return conversations.value.get(chatId)
  }
  
  /**
   * 获取会话消息
   */
  const getMessages = (chatId: string): Message[] => {
    return messages.value.get(chatId) || []
  }
  
  /**
   * 清空所有数据
   */
  const clearAll = () => {
    conversations.value.clear()
    messages.value.clear()
    currentConversationId.value = null
  }

  // ============ API 调用方法 ============

  /**
   * 发送私聊消息
   */
  const sendPrivateMessage = async (chatId: string, data: Omit<SendPrivateMessageRequest, 'private_chat_id'>) => {
    const messagesService = useMessagesService()
    const authStore = useAuthStore()
    
    // 生成临时ID
    const tempId = `temp_${Date.now()}`
    
    // 乐观更新
    const optimisticMessage: Message = {
      id: tempId,
      type: data.type,
      content: data.content,
      sender: {
        user_id: authStore.user?.user_id || '',
        nickname: authStore.user?.nickname || '',
        avatar_url: authStore.user?.avatar_url
      },
      sent_at: new Date().toISOString(),
      is_recalled: false,
      sending: true,
      optimistic: true
    }
    
    appendMessage(chatId, optimisticMessage)
    
    try {
      const response = await messagesService.sendPrivateMessage({
        private_chat_id: chatId,
        ...data
      })
      
      // 用真实消息替换乐观更新的消息
      replaceMessage(chatId, tempId, {
        ...optimisticMessage,
        id: response.id,
        sent_at: response.sent_at,
        sending: false,
        optimistic: false
      })
      
      return response
    } catch (error: any) {
      // 标记为发送失败
      markMessageFailed(chatId, tempId)
      throw error
    }
  }

  /**
   * 发送群聊消息
   */
  const sendGroupMessage = async (chatId: string, data: Omit<SendGroupMessageRequest, 'group_chat_id'>) => {
    const messagesService = useMessagesService()
    const authStore = useAuthStore()
    
    // 生成临时ID
    const tempId = `temp_${Date.now()}`
    
    // 乐观更新
    const optimisticMessage: Message = {
      id: tempId,
      type: data.type,
      content: data.content,
      sender: {
        user_id: authStore.user?.user_id || '',
        nickname: authStore.user?.nickname || '',
        avatar_url: authStore.user?.avatar_url
      },
      mentions: data.mentions,
      sent_at: new Date().toISOString(),
      is_recalled: false,
      sending: true,
      optimistic: true
    }
    
    appendMessage(chatId, optimisticMessage)
    
    try {
      const response = await messagesService.sendGroupMessage({
        group_chat_id: chatId,
        ...data
      })
      
      // 用真实消息替换乐观更新的消息
      replaceMessage(chatId, tempId, {
        ...optimisticMessage,
        id: response.id,
        sent_at: response.sent_at,
        sending: false,
        optimistic: false
      })
      
      return response
    } catch (error: any) {
      // 标记为发送失败
      markMessageFailed(chatId, tempId)
      throw error
    }
  }

  /**
   * 标记私聊消息已读
   */
  const markPrivateMessagesAsRead = async (chatId: string, upTo?: string) => {
    const messagesService = useMessagesService()
    
    try {
      await messagesService.markPrivateMessagesRead({
        private_chat_id: chatId,
        up_to: upTo
      })
      
      // 更新本地未读计数
      updateUnreadCount(chatId, 0)
    } catch (error: any) {
      console.error('标记已读失败:', error)
    }
  }

  /**
   * 标记群聊消息已读
   */
  const markGroupMessagesAsRead = async (chatId: string, upTo?: string) => {
    const messagesService = useMessagesService()
    
    try {
      await messagesService.markGroupMessagesRead({
        group_chat_id: chatId,
        up_to: upTo
      })
      
      // 更新本地未读计数
      updateUnreadCount(chatId, 0)
    } catch (error: any) {
      console.error('标记已读失败:', error)
    }
  }

  /**
   * 撤回消息
   */
  const recallMessage = async (chatId: string, messageId: string, isGroup: boolean) => {
    const messagesService = useMessagesService()
    
    if (isGroup) {
      await messagesService.recallGroupMessage({ id: messageId })
    } else {
      await messagesService.recallPrivateMessage({ id: messageId })
    }
    
    // 更新本地消息状态
    updateMessage(chatId, messageId, {
      is_recalled: true,
      recalled_at: new Date().toISOString()
    })
  }

  /**
   * 删除消息
   */
  const deleteMessage = async (chatId: string, messageId: string, isGroup: boolean, forEveryone = false) => {
    const messagesService = useMessagesService()
    
    if (isGroup) {
      await messagesService.deleteGroupMessage({ id: messageId, for_everyone: forEveryone })
    } else {
      await messagesService.deletePrivateMessage({ id: messageId })
    }
    
    // 从本地删除
    removeMessage(chatId, messageId)
  }

  /**
   * 加载私聊消息历史
   */
  const fetchPrivateMessages = async (chatId: string, cursor?: string) => {
    const messagesService = useMessagesService()
    loading.value = true
    
    try {
      const response = await messagesService.getPrivateMessages({
        private_chat_id: chatId,
        cursor
      })
      
      const messagesList: Message[] = response.items.map(item => ({
        ...item,
        is_recalled: item.is_recalled || false
      }))
      
      if (cursor) {
        prependMessages(chatId, messagesList)
      } else {
        setMessages(chatId, messagesList)
      }
      
      return response.pagination
    } finally {
      loading.value = false
    }
  }

  /**
   * 加载群聊消息历史
   */
  const fetchGroupMessages = async (chatId: string, cursor?: string) => {
    const messagesService = useMessagesService()
    loading.value = true
    
    try {
      const response = await messagesService.getGroupMessages({
        group_chat_id: chatId,
        cursor
      })
      
      const messagesList: Message[] = response.items.map(item => ({
        ...item,
        is_recalled: item.is_recalled || false
      }))
      
      if (cursor) {
        prependMessages(chatId, messagesList)
      } else {
        setMessages(chatId, messagesList)
      }
      
      return response.pagination
    } finally {
      loading.value = false
    }
  }

  return {
    // 状态
    conversations,
    messages,
    currentConversationId,
    loading,
    
    // 计算属性
    conversationsList,
    currentConversation,
    currentMessages,
    totalUnreadCount,

    // 会话操作
    setConversations,
    upsertConversation,
    removeConversation,
    setCurrentConversation,
    updateUnreadCount,
    setDraft,
    setTyping,
    
    // 消息操作
    setMessages,
    prependMessages,
    appendMessage,
    updateMessage,
    replaceMessage,
    removeMessage,
    markMessageSending,
    markMessageSent,
    markMessageFailed,
    
    // 辅助方法
    getConversation,
    getMessages,
    clearAll,

    // API 调用方法
    sendPrivateMessage,
    sendGroupMessage,
    markPrivateMessagesAsRead,
    markGroupMessagesAsRead,
    recallMessage,
    deleteMessage,
    fetchPrivateMessages,
    fetchGroupMessages
  }
})
