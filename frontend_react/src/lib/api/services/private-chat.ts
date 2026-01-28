/**
 * 私聊服务
 * 私聊相关 API
 * 
 * 基于 @api_spec_chat.mdc 规范
 */

import { apiRequest, apiRequestIdempotent } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type {
  ApiResponse,
  PaginatedData,
  PrivateChatListItem,
  PrivateMessageItem,
  SendMessageType,
  UserInfo,
} from '@/types/api';

// ============== 私聊管理 ==============

interface MyChatsParams {
  limit?: number;
  cursor?: string;
}

/**
 * 获取私聊列表
 * @api POST /api/v1/private_chats/me
 */
export async function getMyChats(
  params: MyChatsParams = {}
): Promise<ApiResponse<PaginatedData<PrivateChatListItem>>> {
  return apiRequest<PaginatedData<PrivateChatListItem>>(
    ENDPOINTS.PRIVATE_CHATS.ME,
    params
  );
}

interface Conversation {
  other_user: UserInfo;
  message_limit?: {
    remaining: number;
    total: number;
    is_unlocked: boolean;
  };
  unread_count: number;
  is_muted: boolean;
  is_pinned: boolean;
  last_activity_at: string;
  created_at: string;
  can_message: boolean;
  recent_messages: {
    items: PrivateMessageItem[];
    has_more: boolean;
    next_cursor?: string;
  };
}

/**
 * 获取私聊详情
 * @api POST /api/v1/private_chats/open_the_door
 */
export async function openChat(
  targetUserId: number,
  recentMessagesLimit?: number
): Promise<ApiResponse<Conversation>> {
  return apiRequest<Conversation>(ENDPOINTS.PRIVATE_CHATS.OPEN_THE_DOOR, {
    target_user_id: targetUserId,
    recent_messages_limit: recentMessagesLimit,
  });
}

interface UpdateChatSettingsParams {
  target_user_id: number;
  is_muted?: boolean;
  is_pinned?: boolean;
}

/**
 * 更新私聊设置
 * @api POST /api/v1/private_chats/update_settings
 */
export async function updateChatSettings(
  params: UpdateChatSettingsParams
): Promise<ApiResponse<Conversation>> {
  return apiRequest<Conversation>(ENDPOINTS.PRIVATE_CHATS.UPDATE_SETTINGS, params);
}

/**
 * 删除私聊会话（仅自己视角）
 * @api POST /api/v1/private_chats/delete
 */
export async function deleteChat(
  targetUserId: number
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.PRIVATE_CHATS.DELETE, {
    target_user_id: targetUserId,
  });
}

// ============== 私聊消息 ==============

interface SendMessageParams {
  target_user_id: number;
  type: SendMessageType;
  content?: string;
  file_id?: string;
  reply_to_message_id?: string;
}

/**
 * 发送私聊消息
 * 需要 Idempotency-Key 请求头
 * @api POST /api/v1/private_messages/send
 */
export async function sendMessage(
  params: SendMessageParams
): Promise<ApiResponse<void>> {
  return apiRequestIdempotent<void>(ENDPOINTS.PRIVATE_MESSAGES.SEND, params);
}

interface MessageHistoryParams {
  target_user_id: number;
  limit?: number;
  cursor?: string;
}

/**
 * 获取私聊消息历史
 * @api POST /api/v1/private_messages/history
 */
export async function getMessageHistory(
  params: MessageHistoryParams
): Promise<ApiResponse<PaginatedData<PrivateMessageItem>>> {
  return apiRequest<PaginatedData<PrivateMessageItem>>(
    ENDPOINTS.PRIVATE_MESSAGES.HISTORY,
    params
  );
}

/**
 * 编辑私聊消息
 * @api POST /api/v1/private_messages/update
 */
export async function updateMessage(
  messageId: string,
  content: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.PRIVATE_MESSAGES.UPDATE, {
    message_id: messageId,
    content,
  });
}

/**
 * 撤回私聊消息
 * @api POST /api/v1/private_messages/recall
 */
export async function recallMessage(
  messageId: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.PRIVATE_MESSAGES.RECALL, {
    message_id: messageId,
  });
}

/**
 * 删除私聊消息（仅自己视角）
 * @api POST /api/v1/private_messages/delete
 */
export async function deleteMessage(
  messageId: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.PRIVATE_MESSAGES.DELETE, {
    message_id: messageId,
  });
}

/**
 * 标记私聊消息已读
 * @api POST /api/v1/private_messages/mark_as_read
 */
export async function markAsRead(
  targetUserId: number,
  upToMessageId?: string
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.PRIVATE_MESSAGES.MARK_AS_READ, {
    target_user_id: targetUserId,
    up_to_message_id: upToMessageId,
  });
}
