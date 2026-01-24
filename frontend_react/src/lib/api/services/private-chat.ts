/**
 * 私聊服务
 */

import { get, post, put, del, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  PrivateChatListItem,
  PrivateChatDetailData,
  PrivateMessageItem,
  SendPrivateMessageRequest,
  MessageReactionItem,
  ReactionType,
} from '@/types/api';

export const privateChatService = {
  // ============================================================================
  // 会话管理
  // ============================================================================

  /**
   * 获取私聊会话列表
   */
  async getChats(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<PrivateChatListItem>>> {
    return get<PaginatedData<PrivateChatListItem>>(ENDPOINTS.PRIVATE_CHATS.LIST, params);
  },

  /**
   * 获取私聊会话详情
   */
  async getChatDetail(userId: number): Promise<ApiResponse<PrivateChatDetailData>> {
    return get<PrivateChatDetailData>(ENDPOINTS.PRIVATE_CHATS.DETAIL(userId));
  },

  /**
   * 更新私聊设置
   */
  async updateSettings(
    userId: number,
    settings: { is_muted?: boolean; is_pinned?: boolean }
  ): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.PRIVATE_CHATS.SETTINGS(userId), settings, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 删除私聊会话
   */
  async deleteChat(userId: number): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.PRIVATE_CHATS.DELETE(userId));
  },

  // ============================================================================
  // 消息管理
  // ============================================================================

  /**
   * 获取消息历史
   */
  async getHistory(
    userId: number,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<PrivateMessageItem>>> {
    return get<PaginatedData<PrivateMessageItem>>(ENDPOINTS.PRIVATE_MESSAGES.HISTORY(userId), params);
  },

  /**
   * 发送消息
   */
  async sendMessage(data: SendPrivateMessageRequest): Promise<ApiResponse<{ message_id: string }>> {
    return post<{ message_id: string }>(ENDPOINTS.PRIVATE_MESSAGES.SEND, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 编辑消息
   */
  async editMessage(messageId: string, content: string): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.PRIVATE_MESSAGES.UPDATE(messageId), { content }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 撤回消息
   */
  async recallMessage(messageId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.PRIVATE_MESSAGES.RECALL(messageId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 删除消息
   */
  async deleteMessage(messageId: string): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.PRIVATE_MESSAGES.DELETE(messageId));
  },

  /**
   * 标记已读
   */
  async markRead(userId: number, upToMessageId?: string): Promise<ApiResponse<void>> {
    return post<void>(
      ENDPOINTS.PRIVATE_MESSAGES.MARK_READ(userId),
      upToMessageId ? { up_to_message_id: upToMessageId } : undefined,
      { idempotencyKey: generateIdempotencyKey() }
    );
  },

  // ============================================================================
  // 表情反应
  // ============================================================================

  /**
   * 获取消息反应列表
   */
  async getReactions(
    messageId: string,
    params?: { type?: ReactionType; limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<MessageReactionItem>>> {
    return get<PaginatedData<MessageReactionItem>>(ENDPOINTS.PRIVATE_MESSAGES.REACTIONS(messageId), params);
  },

  /**
   * 添加反应
   */
  async addReaction(messageId: string, type: ReactionType): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.PRIVATE_MESSAGES.ADD_REACTION(messageId), { type }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 移除反应
   */
  async removeReaction(messageId: string, type: ReactionType): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.PRIVATE_MESSAGES.REMOVE_REACTION(messageId, type));
  },
};
