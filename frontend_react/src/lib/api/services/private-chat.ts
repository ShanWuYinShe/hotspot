/**
 * 私聊服务
 */

import { post, generateIdempotencyKey } from '../client';
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
    return post<PaginatedData<PrivateChatListItem>>(ENDPOINTS.PRIVATE_CHATS.LIST, params ?? {});
  },

  /**
   * 获取私聊会话详情
   */
  async getChatDetail(userId: number): Promise<ApiResponse<PrivateChatDetailData>> {
    return post<PrivateChatDetailData>(ENDPOINTS.PRIVATE_CHATS.DETAIL(userId), { target_user_id: userId });
  },

  /**
   * 更新私聊设置
   */
  async updateSettings(
    userId: number,
    settings: { is_muted?: boolean; is_pinned?: boolean }
  ): Promise<ApiResponse<void>> {
    return post<void>(
      ENDPOINTS.PRIVATE_CHATS.SETTINGS(userId),
      { target_user_id: userId, ...settings },
      {
      idempotencyKey: generateIdempotencyKey(),
      }
    );
  },

  /**
   * 删除私聊会话
   */
  async deleteChat(userId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.PRIVATE_CHATS.DELETE(userId), { target_user_id: userId });
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
    return post<PaginatedData<PrivateMessageItem>>(ENDPOINTS.PRIVATE_MESSAGES.HISTORY(userId), {
      target_user_id: userId,
      ...(params ?? {}),
    });
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
    return post<void>(ENDPOINTS.PRIVATE_MESSAGES.UPDATE(messageId), { message_id: messageId, content }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 撤回消息
   */
  async recallMessage(messageId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.PRIVATE_MESSAGES.RECALL(messageId), { message_id: messageId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 删除消息
   */
  async deleteMessage(messageId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.PRIVATE_MESSAGES.DELETE(messageId), { message_id: messageId });
  },

  /**
   * 标记已读
   */
  async markRead(userId: number, upToMessageId?: string): Promise<ApiResponse<void>> {
    return post<void>(
      ENDPOINTS.PRIVATE_MESSAGES.MARK_READ(userId),
      {
        target_user_id: userId,
        ...(upToMessageId ? { up_to_message_id: upToMessageId } : {}),
      },
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
    return post<PaginatedData<MessageReactionItem>>(ENDPOINTS.PRIVATE_MESSAGES.REACTIONS(messageId), {
      message_id: messageId,
      ...(params ?? {}),
    });
  },

  /**
   * 添加反应
   */
  async addReaction(messageId: string, type: ReactionType): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.PRIVATE_MESSAGES.ADD_REACTION(messageId), { message_id: messageId, type }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 移除反应
   */
  async removeReaction(messageId: string, type: ReactionType): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.PRIVATE_MESSAGES.REMOVE_REACTION(messageId, type), {
      message_id: messageId,
      type,
    });
  },
};
