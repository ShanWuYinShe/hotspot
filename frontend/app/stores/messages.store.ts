/**
 * 消息状态存储 - 纯状态管理
 * 
 * ✅ 职责：管理消息相关的运行时状态
 * ❌ 不包含：API 调用（使用 service）
 */

import { defineStore } from 'pinia'
import type { UserInfo } from '~/types/api'

// 消息类型
export interface Message {
  id: string
  type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'system'
  content?: string
  sender: UserInfo
  reply_to?: string
  mentions?: number[]
  sent_at: string
  edited_at?: string
  is_recalled: boolean
  recalled_by?: number
  recalled_at?: string
  read_count?: number
  // 媒体消息字段
  media_url?: string
  thumbnail_url?: string
  file_name?: string
  file_size?: number
  duration?: number
  width?: number
  height?: number
  // 本地状态
  sending?: boolean
  failed?: boolean
  optimistic?: boolean
}

// 会话类型
export interface Conversation {
  chat_id: string
  chat_type: 'private' | 'group'
  target_user_id?: number  // 私聊对象的用户ID（仅私聊有效）
  name: string
  avatar_url?: string
  unread_count: number
  last_message?: {
    id: string
    content?: string
    type: string
    sender_id: number
    sent_at: string
  }
  last_activity_at: string
  is_online?: boolean
  draft?: string
  typing?: boolean
}

export const useMessagesStore = defineStore('messages', () => {
  // ============ 内部状态 ============
  const _conversations = ref<Map<string, Conversation>>(new Map())
  const _messages = ref<Map<string, Message[]>>(new Map())
  const _currentConversationId = ref<string | null>(null)
  const _loading = ref(false)

  // ============ Getter ============

  const conversations = computed(() => _conversations.value)
  const messages = computed(() => _messages.value)
  const currentConversationId = computed(() => _currentConversationId.value)
  const loading = computed(() => _loading.value)

  const conversationsList = computed(() => {
    return Array.from(_conversations.value.values())
      .sort((a, b) => new Date(b.last_activity_at).getTime() - new Date(a.last_activity_at).getTime())
  })

  const currentConversation = computed(() => {
    if (!_currentConversationId.value) return null
    return _conversations.value.get(_currentConversationId.value) || null
  })

  const currentMessages = computed(() => {
    if (!_currentConversationId.value) return []
    return _messages.value.get(_currentConversationId.value) || []
  })

  const totalUnreadCount = computed(() => {
    return Array.from(_conversations.value.values())
      .reduce((total, conv) => total + (conv.unread_count || 0), 0)
  })

  // ============ Setter ============

  const setConversations = (list: Conversation[]) => {
    _conversations.value.clear()
    list.forEach(conv => _conversations.value.set(conv.chat_id, conv))
  }

  const upsertConversation = (conversation: Conversation) => {
    _conversations.value.set(conversation.chat_id, conversation)
  }

  const removeConversation = (chatId: string) => {
    _conversations.value.delete(chatId)
    _messages.value.delete(chatId)
  }

  const setCurrentConversation = (chatId: string | null) => {
    _currentConversationId.value = chatId
    if (chatId) {
      const conversation = _conversations.value.get(chatId)
      if (conversation && conversation.unread_count) {
        conversation.unread_count = 0
      }
    }
  }

  const updateUnreadCount = (chatId: string, count: number) => {
    const conversation = _conversations.value.get(chatId)
    if (conversation) conversation.unread_count = count
  }

  const updateConversation = (chatId: string, updates: Partial<Conversation>) => {
    const conversation = _conversations.value.get(chatId)
    if (conversation) {
      Object.assign(conversation, updates)
    }
  }

  const setDraft = (chatId: string, draft: string) => {
    const conversation = _conversations.value.get(chatId)
    if (conversation) conversation.draft = draft
  }

  const setTyping = (chatId: string, typing: boolean) => {
    const conversation = _conversations.value.get(chatId)
    if (conversation) conversation.typing = typing
  }

  const setMessages = (chatId: string, list: Message[]) => {
    _messages.value.set(chatId, [...list])
  }

  const prependMessages = (chatId: string, list: Message[]) => {
    const existing = _messages.value.get(chatId) || []
    _messages.value.set(chatId, [...list, ...existing])
  }

  const appendMessage = (chatId: string, message: Message) => {
    const existing = _messages.value.get(chatId) || []
    // 检查消息是否已存在，避免重复添加
    if (existing.some(m => m.id === message.id)) {
      return
    }
    existing.push(message)
    _messages.value.set(chatId, existing)
    updateConversationLastMessage(chatId, message)
  }

  const updateMessage = (chatId: string, messageId: string, updates: Partial<Message>) => {
    const list = _messages.value.get(chatId) || []
    const index = list.findIndex(m => m.id === messageId)
    if (index !== -1) list[index] = { ...list[index], ...updates }
  }

  const replaceMessage = (chatId: string, oldMessageId: string, newMessage: Message) => {
    const list = _messages.value.get(chatId) || []
    const index = list.findIndex(m => m.id === oldMessageId)
    if (index !== -1) list[index] = newMessage
  }

  const removeMessage = (chatId: string, messageId: string) => {
    const list = _messages.value.get(chatId) || []
    _messages.value.set(chatId, list.filter(m => m.id !== messageId))
  }

  const markMessageSending = (chatId: string, messageId: string) => {
    updateMessage(chatId, messageId, { sending: true, failed: false })
  }

  const markMessageSent = (chatId: string, messageId: string) => {
    updateMessage(chatId, messageId, { sending: false, failed: false, optimistic: false })
  }

  const markMessageFailed = (chatId: string, messageId: string) => {
    updateMessage(chatId, messageId, { sending: false, failed: true })
  }

  const setLoading = (isLoading: boolean) => {
    _loading.value = isLoading
  }

  const updateConversationLastMessage = (chatId: string, message: Message) => {
    const conversation = _conversations.value.get(chatId)
    if (conversation) {
      conversation.last_message = {
        id: message.id,
        content: message.content || '',
        type: message.type,
        sender_id: message.sender.user_id,
        sent_at: message.sent_at
      }
      conversation.last_activity_at = message.sent_at
    }
  }

  const getConversation = (chatId: string) => _conversations.value.get(chatId)
  const getMessages = (chatId: string) => _messages.value.get(chatId) || []

  const clearAll = () => {
    _conversations.value.clear()
    _messages.value.clear()
    _currentConversationId.value = null
  }

  return {
    // Getter
    conversations,
    messages,
    currentConversationId,
    loading,
    conversationsList,
    currentConversation,
    currentMessages,
    totalUnreadCount,

    // Setter
    setConversations,
    upsertConversation,
    removeConversation,
    setCurrentConversation,
    updateUnreadCount,
    updateConversation,
    setDraft,
    setTyping,
    setMessages,
    prependMessages,
    appendMessage,
    updateMessage,
    replaceMessage,
    removeMessage,
    markMessageSending,
    markMessageSent,
    markMessageFailed,
    setLoading,
    getConversation,
    getMessages,
    clearAll
  }
}, {
  // 显式声明：此 store 不需要持久化
  persist: false
})

