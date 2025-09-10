/**
 * 消息服务 - Composition API 风格
 * 提供消息相关的所有 API 调用功能
 */


import { CHAT_ENDPOINTS, buildChatUrl } from "~/config/api";
import { DEFAULTS } from "~/config/defaults";
import type { 
  MessageType, 
  Message, 
  PaginatedResponse, 
  ConversationSummary, 
  UnreadCount, 
  BasicSuccess 
} from "~/types";

export const useMessageService = () => {
  const api = useApi();

  // 群组消息（与 hotspot.md 对齐：/chat/group/*）
  const sendGroupMessage = async (groupId: string, content: string, messageType: MessageType = "text"): Promise<Message> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.GROUP_SEND), {
      group_id: groupId,
      content,
      message_type: messageType,
    }, { auth: true });
  };

  const getGroupMessageHistory = async (groupId: string, cursor?: string): Promise<PaginatedResponse<Message>> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.GROUP_HISTORY), {
      group_id: groupId,
      cursor: cursor, // 🔥 修复：首次加载时不传递cursor，让后端处理默认值
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  // 私聊消息
  // 与 hotspot.md 对齐：/chat/user/send
  const sendDirectMessage = async (recipientId: string, content: string, messageType: MessageType = "text"): Promise<Message> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.USER_SEND), {
      recipient_id: recipientId,
      content,
      message_type: messageType,
    }, { auth: true });
  };

  // 与 hotspot.md 对齐：/chat/user/history
  const getDirectMessageHistory = async (
    params: { conversationId?: string; recipientId?: string },
    cursor?: string,
  ): Promise<PaginatedResponse<Message>> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.USER_HISTORY), {
      conversation_id: params.conversationId,
      recipient_id: params.recipientId,
      cursor: cursor, // 🔥 修复：首次加载时不传递cursor，让后端处理默认值
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  // 消息状态管理（与 hotspot.md 对齐）
  const markAsRead = async (
    id: string,
    type: 'user' | 'group' = 'user',
    lastReadMessageId?: string,
  ): Promise<BasicSuccess> => {
    if (type === 'group') {
      return await api.post(
        buildChatUrl(CHAT_ENDPOINTS.PROTECTED.GROUP_MARK_READ),
        { group_id: id, last_read_message_id: lastReadMessageId },
        { auth: true },
      );
    }
    return await api.post(
      buildChatUrl(CHAT_ENDPOINTS.PROTECTED.USER_MARK_READ),
      { conversation_id: id, last_read_message_id: lastReadMessageId },
      { auth: true },
    );
  };

  const deleteMessage = async (messageId: string, type: 'user' | 'group' = 'group'): Promise<BasicSuccess> => {
    if (type === 'group') {
      return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.GROUP_DELETE), { 
        message_id: messageId 
      }, { auth: true });
    }
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.USER_DELETE), { 
      message_id: messageId 
    }, { auth: true });
  };

  // 搜索消息（暂未实现 - hotspot.md 中未定义此端点）
  const searchMessages = async (data: any): Promise<PaginatedResponse<Message>> => {
    throw new Error('消息搜索功能暂未实现，请等待后端API上线');
    // 注意：hotspot.md 7.4.1 中只定义了11个消息端点，不包含搜索功能
  };

  // 会话管理（与 hotspot.md 对齐）
  const getUserUnreadCount = async (conversationId?: string): Promise<UnreadCount> => {
    return await api.post(
      buildChatUrl(CHAT_ENDPOINTS.PROTECTED.USER_UNREAD_COUNT),
      conversationId ? { conversation_id: conversationId } : {},
      { auth: true },
    );
  };

  const getGroupUnreadCount = async (groupId: string): Promise<UnreadCount> => {
    return await api.post(
      buildChatUrl(CHAT_ENDPOINTS.PROTECTED.GROUP_UNREAD_COUNT),
      { group_id: groupId },
      { auth: true },
    );
  };

  const getConversations = async (cursor?: string): Promise<PaginatedResponse<ConversationSummary>> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.CONVERSATIONS), {
      cursor: cursor ? parseInt(cursor) : 0,
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  return {
    // 群组消息
    sendGroupMessage,
    getGroupMessageHistory,
    
    // 私聊消息
    sendDirectMessage,
    getDirectMessageHistory,
    
    // 状态管理
    markAsRead,
    deleteMessage,
    
    // 搜索
    searchMessages,
    
    // 会话管理
    getUserUnreadCount,
    getGroupUnreadCount,
    getConversations,
  };
};
