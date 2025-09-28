/**
 * 消息状态存储 - 纯状态版本
 * 职责：纯粹的消息状态管理，不包含 API 调用和 WebSocket 管理
 */

import type { MessageResponse, ConversationsResponse } from '~/types/api'

export interface Message extends MessageResponse {
  // 本地状态
  sending?: boolean
  failed?: boolean
  optimistic?: boolean
}

export interface Conversation extends ConversationsResponse {
  // 本地状态
  draft?: string
  typing?: boolean
  unreadCount?: number
}

export const useMessagesStore = defineStore('messages', () => {
  // ============ 状态 ============
  const conversations = ref<Map<string, Conversation>>(new Map())
  const messages = ref<Map<string, Message[]>>(new Map())
  const currentConversationId = ref<string | null>(null)

  // ============ 计算属性 ============
  const conversationsList = computed(() => {
    return Array.from(conversations.value.values())
      .sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime())
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
      .reduce((total, conv) => total + (conv.unreadCount || 0), 0)
  })
  
  // ============ 会话状态操作 ============
  
  /**
   * 设置会话列表
   */
  const setConversations = (conversationsList: Conversation[]) => {
    conversations.value.clear()
    conversationsList.forEach(conv => {
      conversations.value.set(conv.conversation_id, conv)
    })
  }
  
  /**
   * 添加或更新会话
   */
  const upsertConversation = (conversation: Conversation) => {
    conversations.value.set(conversation.conversation_id, conversation)
  }
  
  /**
   * 删除会话
   */
  const removeConversation = (conversationId: string) => {
    conversations.value.delete(conversationId)
    messages.value.delete(conversationId)
  }
  
  /**
   * 设置当前会话
   */
  const setCurrentConversation = (conversationId: string | null) => {
    currentConversationId.value = conversationId
    
    // 自动标记为已读
    if (conversationId) {
      const conversation = conversations.value.get(conversationId)
      if (conversation && conversation.unreadCount) {
        conversation.unreadCount = 0
      }
    }
  }
  
  /**
   * 更新会话未读计数
   */
  const updateUnreadCount = (conversationId: string, count: number) => {
    const conversation = conversations.value.get(conversationId)
    if (conversation) {
      conversation.unreadCount = count
    }
  }
  
  /**
   * 设置会话草稿
   */
  const setDraft = (conversationId: string, draft: string) => {
    const conversation = conversations.value.get(conversationId)
    if (conversation) {
      conversation.draft = draft
    }
  }
  
  /**
   * 设置会话输入状态
   */
  const setTyping = (conversationId: string, typing: boolean) => {
    const conversation = conversations.value.get(conversationId)
    if (conversation) {
      conversation.typing = typing
    }
  }
  
  // ============ 消息状态操作 ============
  
  /**
   * 设置会话的消息列表
   */
  const setMessages = (conversationId: string, messagesList: Message[]) => {
    messages.value.set(conversationId, [...messagesList])
  }
  
  /**
   * 添加消息到会话开头（用于加载历史消息）
   */
  const prependMessages = (conversationId: string, messagesList: Message[]) => {
    const existing = messages.value.get(conversationId) || []
    messages.value.set(conversationId, [...messagesList, ...existing])
  }
  
  /**
   * 添加消息到会话末尾（用于新消息）
   */
  const appendMessage = (conversationId: string, message: Message) => {
    const existing = messages.value.get(conversationId) || []
    existing.push(message)
    messages.value.set(conversationId, existing)
    
    // 更新会话的最后消息信息
    updateConversationLastMessage(conversationId, message)
  }
  
  /**
   * 更新消息
   */
  const updateMessage = (conversationId: string, messageId: string, updates: Partial<Message>) => {
    const conversationMessages = messages.value.get(conversationId) || []
    const index = conversationMessages.findIndex(m => m.message_id === messageId)
    if (index !== -1) {
      conversationMessages[index] = { ...conversationMessages[index], ...updates }
    }
  }
  
  /**
   * 替换消息（用于乐观更新）
   */
  const replaceMessage = (conversationId: string, oldMessageId: string, newMessage: Message) => {
    const conversationMessages = messages.value.get(conversationId) || []
    const index = conversationMessages.findIndex(m => m.message_id === oldMessageId)
    if (index !== -1) {
      conversationMessages[index] = newMessage
    }
  }
  
  /**
   * 删除消息
   */
  const removeMessage = (conversationId: string, messageId: string) => {
    const conversationMessages = messages.value.get(conversationId) || []
    const filteredMessages = conversationMessages.filter(m => m.message_id !== messageId)
    messages.value.set(conversationId, filteredMessages)
  }
  
  /**
   * 标记消息发送中
   */
  const markMessageSending = (conversationId: string, messageId: string) => {
    updateMessage(conversationId, messageId, { sending: true, failed: false })
  }
  
  /**
   * 标记消息发送成功
   */
  const markMessageSent = (conversationId: string, messageId: string) => {
    updateMessage(conversationId, messageId, { sending: false, failed: false, optimistic: false })
  }
  
  /**
   * 标记消息发送失败
   */
  const markMessageFailed = (conversationId: string, messageId: string) => {
    updateMessage(conversationId, messageId, { sending: false, failed: true })
  }
  
  // ============ 辅助方法 ============
  
  /**
   * 更新会话的最后消息信息
   */
  const updateConversationLastMessage = (conversationId: string, message: Message) => {
    const conversation = conversations.value.get(conversationId)
    if (conversation) {
      conversation.last_message = message.content
      conversation.updated_at = message.sent_at
    }
  }
  
  /**
   * 获取会话
   */
  const getConversation = (conversationId: string): Conversation | undefined => {
    return conversations.value.get(conversationId)
  }
  
  /**
   * 获取会话消息
   */
  const getMessages = (conversationId: string): Message[] => {
    return messages.value.get(conversationId) || []
  }
  
  /**
   * 清空所有数据
   */
  const clearAll = () => {
    conversations.value.clear()
    messages.value.clear()
    currentConversationId.value = null
  }

  return {
    // 状态（直接返回，供 Pinia 和插件使用）
    conversations,
    messages,
    currentConversationId,
    
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
    clearAll
  }
})
