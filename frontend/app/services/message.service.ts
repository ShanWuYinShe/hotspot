/**
 * 消息服务 - 最简化版本
 * 保持基本接口，供现有代码调用
 */


import { MESSAGE_ENDPOINTS, CHAT_ENDPOINTS, buildMessageUrl, buildChatUrl } from "~/config/api";
import { DEFAULTS } from "~/config/defaults";

export const useMessageService = () => {
  const api = useApi();

  // 群组消息（与 hotspot.md 对齐：/chat/group/*）
  const sendGroupMessage = async (groupId: string, content: string, messageType: string = "text"): Promise<any> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.GROUP_SEND), {
      group_id: groupId,
      content,
      message_type: messageType,
    }, { auth: true });
  };

  const getGroupMessageHistory = async (groupId: string, cursor?: number): Promise<any> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.GROUP_HISTORY), {
      group_id: groupId,
      cursor: cursor || 0,
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  // 私聊消息
  // 与 hotspot.md 对齐：/chat/user/send
  const sendDirectMessage = async (recipientId: string, content: string, messageType: string = "text"): Promise<any> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.USER_SEND), {
      recipient_id: recipientId,
      content,
      message_type: messageType,
    }, { auth: true });
  };

  // 与 hotspot.md 对齐：/chat/user/history
  const getDirectMessageHistory = async (
    params: { conversationId?: string; recipientId?: string },
    cursor?: number,
  ): Promise<any> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.USER_HISTORY), {
      conversation_id: params.conversationId,
      recipient_id: params.recipientId,
      cursor: cursor || 0,
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  // 消息状态管理（与 hotspot.md 对齐）
  const markAsRead = async (
    id: string,
    type: 'user' | 'group' = 'user',
    lastReadMessageId?: string,
  ): Promise<any> => {
    if (type === 'group') {
      return await api.post(
        buildChatUrl(CHAT_ENDPOINTS.GROUP_MARK_READ),
        { group_id: id, last_read_message_id: lastReadMessageId },
        { auth: true },
      );
    }
    return await api.post(
      buildChatUrl(CHAT_ENDPOINTS.USER_MARK_READ),
      { conversation_id: id, last_read_message_id: lastReadMessageId },
      { auth: true },
    );
  };

  const deleteMessage = async (messageId: string, type: 'user' | 'group' = 'group'): Promise<any> => {
    if (type === 'group') {
      return await api.post(buildChatUrl(CHAT_ENDPOINTS.GROUP_DELETE), { 
        message_id: messageId 
      }, { auth: true });
    }
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.USER_DELETE), { 
      message_id: messageId 
    }, { auth: true });
  };

  // 搜索
  const searchMessages = async (data: any): Promise<any> => {
    return await api.post(buildMessageUrl('/search'), {
      query: data.query,
      group_id: data.group_id,
      user_id: data.user_id,
      message_type: data.message_type,
      cursor: data.cursor || 0,
      limit: data.limit || DEFAULTS.pageSize,
    }, { auth: true });
  };

  // 会话管理（与 hotspot.md 对齐）
  const getUserUnreadCount = async (conversationId?: string): Promise<any> => {
    return await api.post(
      buildChatUrl(CHAT_ENDPOINTS.USER_UNREAD_COUNT),
      conversationId ? { conversation_id: conversationId } : {},
      { auth: true },
    );
  };

  const getGroupUnreadCount = async (groupId: string): Promise<any> => {
    return await api.post(
      buildChatUrl(CHAT_ENDPOINTS.GROUP_UNREAD_COUNT),
      { group_id: groupId },
      { auth: true },
    );
  };

  const getConversations = async (cursor?: number): Promise<any> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.CONVERSATIONS), {
      cursor: cursor || 0,
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
