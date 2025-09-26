/**
 * 消息服务 - Composition API 风格
 * 提供消息相关的所有 API 调用功能
 */


import { CHAT_ENDPOINTS, buildChatUrl } from "~/config/api";
import { DEFAULTS } from "~/config/defaults";
import type { 
  MessageType, 
  SendMessageResponse, 
  PaginatedResponse, 
  ConversationsResponse, 
  UnreadCount, 
  BasicSuccess 
} from "~/types/api";

export const useMessageService = () => {
  const api = useApi();

  // 群组消息（使用统一消息接口）
  const sendGroupMessage = async (groupId: string, content: string, messageType: MessageType = "text"): Promise<SendMessageResponse> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.SEND), {
      group_id: groupId,
      content,
      message_type: messageType,
    }, { auth: true });
  };

  const getGroupMessageHistory = async (groupId: string, cursor?: string): Promise<PaginatedResponse<SendMessageResponse>> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.HISTORY), {
      group_id: groupId,
      cursor: cursor, // 游标分页
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  // 私聊消息（使用统一消息接口）
  const sendDirectMessage = async (recipientId: string, content: string, messageType: MessageType = "text"): Promise<SendMessageResponse> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.SEND), {
      recipient_id: recipientId,
      content,
      message_type: messageType,
    }, { auth: true });
  };

  // 获取私聊消息历史（使用统一消息接口）
  const getDirectMessageHistory = async (
    params: { conversationId?: string; recipientId?: string },
    cursor?: string,
  ): Promise<PaginatedResponse<SendMessageResponse>> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.HISTORY), {
      conversation_id: params.conversationId,
      recipient_id: params.recipientId,
      cursor: cursor, // 游标分页
      limit: DEFAULTS.pageSize,
    }, { auth: true });
  };

  // 消息状态管理（使用统一消息接口）
  const markAsRead = async (
    id: string,
    type: 'user' | 'group' = 'user',
    lastReadSendMessageResponseId?: string,
  ): Promise<BasicSuccess> => {
    return await api.post(
      buildChatUrl(CHAT_ENDPOINTS.PROTECTED.MARK_READ),
      { 
        [type === 'group' ? 'group_id' : 'conversation_id']: id, 
        last_read_message_id: lastReadSendMessageResponseId 
      },
      { auth: true },
    );
  };

  const deleteMessage = async (messageId: string, type: 'user' | 'group' = 'group'): Promise<BasicSuccess> => {
    return await api.post(buildChatUrl(CHAT_ENDPOINTS.PROTECTED.DELETE), { 
      message_id: messageId 
    }, { auth: true });
  };

  // 搜索消息（暂未实现 - hotspot.md 中未定义此端点）
  const searchMessages = async (data: any): Promise<PaginatedResponse<SendMessageResponse>> => {
    throw new Error('消息搜索功能暂未实现，请等待后端API上线');
    // 注意：hotspot.md 7.4.1 中只定义了11个消息端点，不包含搜索功能
  };

  // 会话管理（与 hotspot.md 对齐）
  const getUserUnreadCount = async (conversationId?: string): Promise<UnreadCount> => {
    return await api.post(
      buildChatUrl(CHAT_ENDPOINTS.PROTECTED.UNREAD_COUNT),
      conversationId ? { conversation_id: conversationId } : {},
      { auth: true },
    );
  };

  const getGroupUnreadCount = async (groupId: string): Promise<UnreadCount> => {
    return await api.post(
      buildChatUrl(CHAT_ENDPOINTS.PROTECTED.UNREAD_COUNT),
      { group_id: groupId },
      { auth: true },
    );
  };

  const getConversations = async (cursor?: string): Promise<PaginatedResponse<ConversationsResponse>> => {
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
