/**
 * 消息业务逻辑 Composable
 * 职责：处理消息相关的业务逻辑和 API 调用，操作 Messages Store
 */

import type { Message, Conversation } from '~/stores/messages.store'

export function useMessages() {
  const messagesStore = useMessagesStore()
  const authStore = useAuthStore()
  const { execute } = useAsyncOperation()
  
  /**
   * 加载会话列表
   */
  const loadConversations = () => {
    return execute(async () => {
      const http = useHttp()
      const response = await http.get('/protected/messages/conversations', { auth: true })
      
      const conversations = response.data.conversations || []
      messagesStore.setConversations(conversations)
      
      return conversations
    })
  }
  
  /**
   * 获取或创建会话
   */
  const getOrCreateConversation = (targetUserId: string) => {
    return execute(async () => {
      // 先检查本地是否已存在
      const existing = messagesStore.conversationsList
        .find(conv => conv.participants.some(p => p.user_id === targetUserId))
      
      if (existing) {
        return existing
      }
      
      // 创建新会话
      const http = useHttp()
      const response = await http.post('/protected/messages/conversations', {
        target_user_id: targetUserId
      }, { auth: true })
      
      const conversation = response.data.conversation
      messagesStore.upsertConversation(conversation)
      
      return conversation
    })
  }
  
  /**
   * 加载会话消息
   */
  const loadMessages = (conversationId: string, page = 1, pageSize = 50) => {
    return execute(async () => {
      const http = useHttp()
      const response = await http.get(`/protected/messages/conversations/${conversationId}/messages`, {
        params: { page, page_size: pageSize },
        auth: true
      })
      
      const messages = response.data.messages || []
      
      if (page === 1) {
        messagesStore.setMessages(conversationId, messages)
      } else {
        messagesStore.prependMessages(conversationId, messages)
      }
      
      return messages
    })
  }
  
  /**
   * 发送消息
   */
  const sendMessage = async (
    conversationId: string,
    content: string,
    type: 'text' | 'image' | 'file' = 'text'
  ) => {
    // 创建乐观更新的消息
    const optimisticMessage: Message = {
      message_id: `temp-${Date.now()}`,
      conversation_id: conversationId,
      sender_id: authStore.user?.user_id || '',
      content,
      message_type: type,
      sent_at: new Date().toISOString(),
      sending: true,
      optimistic: true
    }
    
    // 立即添加到本地状态
    messagesStore.appendMessage(conversationId, optimisticMessage)
    
    try {
      const http = useHttp()
      const response = await http.post(`/protected/messages/conversations/${conversationId}/messages`, {
        content,
        message_type: type
      }, { auth: true })
      
      const realMessage = response.data.message
      
      // 替换乐观更新的消息
      messagesStore.replaceMessage(conversationId, optimisticMessage.message_id, realMessage)
      
      return realMessage
    } catch (error) {
      // 标记消息发送失败
      messagesStore.markMessageFailed(conversationId, optimisticMessage.message_id)
      throw error
    }
  }
  
  /**
   * 重新发送失败的消息
   */
  const resendMessage = async (conversationId: string, messageId: string) => {
    const messages = messagesStore.getMessages(conversationId)
    const message = messages.find(m => m.message_id === messageId)
    
    if (!message || !message.failed) return
    
    // 重置状态
    messagesStore.markMessageSending(conversationId, messageId)
    
    try {
      const http = useHttp()
      const response = await http.post(`/protected/messages/conversations/${conversationId}/messages`, {
        content: message.content,
        message_type: message.message_type
      }, { auth: true })
      
      const realMessage = response.data.message
      messagesStore.replaceMessage(conversationId, messageId, realMessage)
    } catch (error) {
      messagesStore.markMessageFailed(conversationId, messageId)
      throw error
    }
  }
  
  /**
   * 删除消息
   */
  const deleteMessage = (conversationId: string, messageId: string) => {
    return execute(async () => {
      const http = useHttp()
      await http.delete(`/protected/messages/${messageId}`, { auth: true })
      
      // 从本地状态移除
      messagesStore.removeMessage(conversationId, messageId)
    }, {
      successMessage: '消息已删除'
    })
  }
  
  /**
   * 标记消息为已读
   */
  const markAsRead = async (conversationId: string) => {
    try {
      const http = useHttp()
      await http.put(`/protected/messages/conversations/${conversationId}/read`, {}, { auth: true })
      
      // 更新本地未读计数
      messagesStore.updateUnreadCount(conversationId, 0)
    } catch (error) {
      console.warn('标记已读失败:', error)
    }
  }
  
  /**
   * 设置当前会话
   */
  const setCurrentConversation = (conversationId: string | null) => {
    messagesStore.setCurrentConversation(conversationId)
    
    // 如果切换到新会话，自动标记为已读
    if (conversationId) {
      markAsRead(conversationId)
    }
  }
  
  /**
   * 保存草稿
   */
  const saveDraft = (conversationId: string, content: string) => {
    messagesStore.setDraft(conversationId, content)
  }
  
  /**
   * 清除草稿
   */
  const clearDraft = (conversationId: string) => {
    messagesStore.setDraft(conversationId, '')
  }
  
  // ============ WebSocket 事件处理 ============
  
  /**
   * 接收新消息（由 WebSocket 调用）
   */
  const receiveMessage = (message: Message) => {
    messagesStore.appendMessage(message.conversation_id, message)
    
    // 如果不是当前会话，增加未读计数
    if (messagesStore.currentConversationId !== message.conversation_id) {
      const conversation = messagesStore.getConversation(message.conversation_id)
      if (conversation) {
        const newCount = (conversation.unreadCount || 0) + 1
        messagesStore.updateUnreadCount(message.conversation_id, newCount)
      }
    }
  }
  
  /**
   * 更新消息状态（已读、已送达等）
   */
  const updateMessageStatus = (messageId: string, status: 'delivered' | 'read') => {
    // 这里可以根据需要更新消息状态
    // 目前的 Message 接口没有状态字段，可以根据需要扩展
  }
  
  /**
   * 处理用户输入状态
   */
  const handleTyping = (conversationId: string, isTyping: boolean) => {
    messagesStore.setTyping(conversationId, isTyping)
  }
  
  return {
    // 状态（来自 store）
    conversations: messagesStore.conversations,
    messages: messagesStore.messages,
    conversationsList: messagesStore.conversationsList,
    currentConversation: messagesStore.currentConversation,
    currentMessages: messagesStore.currentMessages,
    totalUnreadCount: messagesStore.totalUnreadCount,
    currentConversationId: messagesStore.currentConversationId,
    
    // 业务方法
    loadConversations,
    getOrCreateConversation,
    loadMessages,
    sendMessage,
    resendMessage,
    deleteMessage,
    markAsRead,
    setCurrentConversation,
    saveDraft,
    clearDraft,
    
    // WebSocket 事件处理
    receiveMessage,
    updateMessageStatus,
    handleTyping
  }
}
