/**
 * 消息状态存储 - 纯状态管理
 * 
 * ✅ 职责：管理消息相关的运行时状态
 * ❌ 不包含：API 调用（使用 service）
 */

import { defineStore } from 'pinia'
import type { UserInfo } from '~/types/api'
import type {
  PrivateMessageNewEvent,
  PrivateMessageEditedEvent,
  PrivateMessageRecalledEvent,
  PrivateMessageDeletedEvent,
  PrivateMessageReadEvent,
  PrivateMessageReactionEvent,
  GroupMessageNewEvent,
  GroupMessageEditedEvent,
  GroupMessageRecalledEvent,
  GroupMessageDeletedEvent,
  GroupMessageReadEvent,
  GroupMessageReactionEvent,
  GroupUserTypingEvent,
  PrivateUserTypingEvent,
  UserOfflineEvent,
  UserOnlineEvent
} from '~/types/websocket'

// 消息类型
export interface Message {
  id: string
  type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'system'
  content?: string
  sender: UserInfo
  reply_to_message_id?: string
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
    type: Message['type']
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
  // 用户在线状态缓存（用于“事件早到/会话晚到”的一致性问题）
  const _userPresence = ref<Map<number, boolean>>(new Map())

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
    list.forEach((conv) => {
      // 把已知在线状态合并到会话上（避免 UI 显示“明明在线但不亮”）
      if (conv.chat_type === 'private' && conv.target_user_id != null) {
        const cached = _userPresence.value.get(conv.target_user_id)
        if (cached !== undefined) conv = { ...conv, is_online: cached }
      }
      _conversations.value.set(conv.chat_id, conv)
    })
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
        // 用不可变更新确保 Map 内值变更能触发依赖更新（避免“看过仍未读”的 UI 卡住）
        _conversations.value.set(chatId, { ...conversation, unread_count: 0 })
      }
    }
  }

  const updateUnreadCount = (chatId: string, count: number) => {
    const conversation = _conversations.value.get(chatId)
    if (conversation) _conversations.value.set(chatId, { ...conversation, unread_count: count })
  }

  const updateConversation = (chatId: string, updates: Partial<Conversation>) => {
    const conversation = _conversations.value.get(chatId)
    if (conversation) {
      _conversations.value.set(chatId, { ...conversation, ...updates })
    }
  }

  const setDraft = (chatId: string, draft: string) => {
    const conversation = _conversations.value.get(chatId)
    if (conversation) _conversations.value.set(chatId, { ...conversation, draft })
  }

  const setTyping = (chatId: string, typing: boolean) => {
    const conversation = _conversations.value.get(chatId)
    if (conversation) _conversations.value.set(chatId, { ...conversation, typing })
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
    if (index !== -1) {
      const updated = { ...list[index], ...updates }
      // 如果 updates 中显式传递了 undefined 的 id，需要确保 id 不被覆盖（理论上不应发生，但为了安全）
      if (!updated.id) updated.id = messageId
      list[index] = updated as Message
    }
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
      _conversations.value.set(chatId, {
        ...conversation,
        last_message: {
          id: message.id,
          content: message.content || '',
          type: message.type,
          sender_id: message.sender.user_id,
          sent_at: message.sent_at
        },
        last_activity_at: message.sent_at
      })
    }
  }

  // ============================================================================
  // WebSocket event handlers (single source of truth)
  // ============================================================================

  // Helper: Find private chat ID by partner user ID
  const findPrivateChatId = (partnerUserId: number): string | undefined => {
    for (const [chatId, conv] of _conversations.value.entries()) {
      if (conv.chat_type === 'private' && conv.target_user_id === partnerUserId) {
        return chatId
      }
    }
    return undefined
  }

  // Helper: Find chat ID containing a specific message (slow scan)
  const findChatIdByMessageId = (messageId: string): string | undefined => {
    for (const [chatId, msgs] of _messages.value.entries()) {
      if (msgs.some(m => m.id === messageId)) {
        return chatId
      }
    }
    return undefined
  }

  // ------ Private Messages ------

  const handlePrivateMessageNew = (
    payload: PrivateMessageNewEvent['payload'],
    currentUserId: number | undefined
  ) => {
    if (!currentUserId) return

    // Determine the partner ID
    const isFromMe = payload.sender.user_id === currentUserId
    const partnerId = isFromMe ? payload.target_user_id : payload.sender.user_id
    
    const chatId = findPrivateChatId(partnerId)
    if (!chatId) return // Chat not loaded, ignore or fetch list?

    const message: Message = {
      id: payload.message_id,
      type: payload.type as Message['type'],
      content: payload.content || '',
      sender: {
        user_id: payload.sender.user_id,
        nickname: payload.sender.nickname,
        avatar_url: payload.sender.avatar_url,
        user_type: payload.sender.user_type
      },
      reply_to_message_id: payload.reply_to_message_id || undefined,
      // mentions: undefined, // Private messages usually don't have mentions
      sent_at: payload.sent_at,
      is_recalled: false,
      // Mapping media fields from nested object to flattened fields
      media_url: payload.media?.url,
      thumbnail_url: payload.media?.thumbnail_url,
      file_name: payload.media?.file_name,
      file_size: payload.media?.file_size,
      duration: payload.media?.duration,
      width: payload.media?.width,
      height: payload.media?.height,
    }

    appendMessage(chatId, message)

    const conv = _conversations.value.get(chatId)
    if (conv) {
      const isCurrent = _currentConversationId.value === chatId
      if (!isCurrent && !isFromMe) {
        _conversations.value.set(chatId, {
          ...conv,
          unread_count: (conv.unread_count || 0) + 1
        })
      }
    }
  }

  const handlePrivateMessageEdited = (payload: PrivateMessageEditedEvent['payload']) => {
    const chatId = findChatIdByMessageId(payload.message_id)
    if (chatId) {
      updateMessage(chatId, payload.message_id, {
        content: payload.content,
        edited_at: payload.edited_at
      })
    }
  }

  const handlePrivateMessageRecalled = (payload: PrivateMessageRecalledEvent['payload']) => {
    const chatId = findChatIdByMessageId(payload.message_id)
    if (chatId) {
      updateMessage(chatId, payload.message_id, {
        is_recalled: true,
        recalled_by: payload.recalled_by,
        recalled_at: payload.recalled_at
      })
    }
  }

  const handlePrivateMessageDeleted = (payload: PrivateMessageDeletedEvent['payload']) => {
    const chatId = findChatIdByMessageId(payload.message_id)
    if (chatId) {
      removeMessage(chatId, payload.message_id)
    }
  }

  const handlePrivateMessageRead = (payload: PrivateMessageReadEvent['payload'], currentUserId: number | undefined) => {
    // Logic for marking messages as read (e.g. updating 'read' status of messages)
    // Currently store doesn't track per-message read status for private chats extensively
  }

  const handlePrivateMessageReaction = (payload: PrivateMessageReactionEvent['payload']) => {
    // Reaction handling
  }

  // ------ Group Messages ------

  const handleGroupMessageNew = (
    payload: GroupMessageNewEvent['payload'],
    currentUserId: number | undefined
  ) => {
    const chatId = payload.group_chat_id // group_chat_id IS the chatId
    const message: Message = {
      id: payload.message_id,
      type: payload.type as Message['type'],
      content: payload.content || '',
      sender: {
        user_id: payload.sender.user_id,
        nickname: payload.sender.nickname,
        avatar_url: payload.sender.avatar_url,
        user_type: payload.sender.user_type
      },
      reply_to_message_id: payload.reply_to_message_id || undefined,
      mentions: payload.mentions || undefined,
      sent_at: payload.sent_at,
      is_recalled: false,
      
      // Mapping media fields
      media_url: payload.media?.url,
      thumbnail_url: payload.media?.thumbnail_url,
      file_name: payload.media?.file_name,
      file_size: payload.media?.file_size,
      duration: payload.media?.duration,
      width: payload.media?.width,
      height: payload.media?.height,
    }

    appendMessage(chatId, message)

    const conv = _conversations.value.get(chatId)
    if (conv) {
      const isCurrent = _currentConversationId.value === chatId
      const isFromMe = currentUserId != null && payload.sender.user_id === currentUserId
      if (!isCurrent && !isFromMe) {
        _conversations.value.set(chatId, {
          ...conv,
          unread_count: (conv.unread_count || 0) + 1
        })
      }
    }
  }

  const handleGroupMessageEdited = (payload: GroupMessageEditedEvent['payload']) => {
    updateMessage(payload.group_chat_id, payload.message_id, {
      content: payload.content,
      edited_at: payload.edited_at
    })
  }

  const handleGroupMessageRecalled = (payload: GroupMessageRecalledEvent['payload']) => {
    updateMessage(payload.group_chat_id, payload.message_id, {
      is_recalled: true,
      recalled_by: payload.recalled_by,
      recalled_at: payload.recalled_at
    })
  }

  const handleGroupMessageDeleted = (payload: GroupMessageDeletedEvent['payload']) => {
    removeMessage(payload.group_chat_id, payload.message_id)
  }

  const handleGroupMessageRead = (_payload: GroupMessageReadEvent['payload'], _currentUserId: number | undefined) => {
    // Read status handling
  }

  const handleGroupMessageReaction = (_payload: GroupMessageReactionEvent['payload']) => {
    // Reaction handling
  }

  const handlePrivateUserTyping = (payload: PrivateUserTypingEvent['payload']) => {
    // 收到 user_id 给 target_user_id 发送的正在输入
    // 如果我是 target_user_id，说明 user_id 在对我输入
    // 需要找到对应的私聊会话
    for (const [chatId, conv] of _conversations.value.entries()) {
      if (conv.chat_type === 'private' && conv.target_user_id === payload.user_id) {
        setTyping(chatId, payload.is_typing)
        break
      }
    }
  }

  const handleGroupUserTyping = (payload: GroupUserTypingEvent['payload'], currentUserId: number | undefined) => {
    if (currentUserId != null && payload.user_id === currentUserId) return
    setTyping(payload.group_chat_id, payload.is_typing)
  }

  const handleUserOnline = (payload: UserOnlineEvent['payload']) => {
    _userPresence.value.set(payload.user_id, true)
    for (const [chatId, conv] of _conversations.value.entries()) {
      if (conv.chat_type === 'private' && conv.target_user_id === payload.user_id) {
        // 用不可变更新确保 UI 响应（Map 内值直接 mutation 不会触发依赖）
        _conversations.value.set(chatId, { ...conv, is_online: true })
      }
    }
  }

  const handleUserOffline = (payload: UserOfflineEvent['payload']) => {
    _userPresence.value.set(payload.user_id, false)
    for (const [chatId, conv] of _conversations.value.entries()) {
      if (conv.chat_type === 'private' && conv.target_user_id === payload.user_id) {
        _conversations.value.set(chatId, { ...conv, is_online: false })
      }
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
    clearAll,

    // WS handlers
    handlePrivateMessageNew,
    handlePrivateMessageEdited,
    handlePrivateMessageRecalled,
    handlePrivateMessageDeleted,
    handlePrivateMessageRead,
    handlePrivateMessageReaction,
    handleGroupMessageNew,
    handleGroupMessageEdited,
    handleGroupMessageRecalled,
    handleGroupMessageDeleted,
    handleGroupMessageRead,
    handleGroupMessageReaction,
    handlePrivateUserTyping,
    handleGroupUserTyping,
    handleUserOnline,
    handleUserOffline
  }
}, {
  // 显式声明：此 store 不需要持久化
  persist: false
})

