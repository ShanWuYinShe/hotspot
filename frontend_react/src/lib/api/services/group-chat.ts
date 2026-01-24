/**
 * 群聊消息服务
 */

import { get, post, put, del, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  GroupMessageItem,
  SendGroupMessageRequest,
  MessageReactionItem,
  MessageReadItem,
  ReactionType,
} from '@/types/api';

export const groupChatService = {
  // ============================================================================
  // 消息管理
  // ============================================================================

  /**
   * 获取消息历史
   */
  async getHistory(
    groupId: string,
    params?: { limit?: number; cursor?: string; include_system?: boolean }
  ): Promise<ApiResponse<PaginatedData<GroupMessageItem>>> {
    return get<PaginatedData<GroupMessageItem>>(ENDPOINTS.GROUP_MESSAGES.HISTORY(groupId), params);
  },

  /**
   * 发送消息
   */
  async sendMessage(data: SendGroupMessageRequest): Promise<ApiResponse<{ message_id: string }>> {
    return post<{ message_id: string }>(ENDPOINTS.GROUP_MESSAGES.SEND, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 编辑消息
   */
  async editMessage(messageId: string, content: string): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.GROUP_MESSAGES.UPDATE(messageId), { content }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 撤回消息
   */
  async recallMessage(messageId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_MESSAGES.RECALL(messageId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 删除消息
   */
  async deleteMessage(messageId: string, forEveryone?: boolean): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.GROUP_MESSAGES.DELETE(messageId), {
      params: forEveryone ? { for_everyone: true } : undefined,
    });
  },

  /**
   * 标记已读
   */
  async markRead(groupId: string, upToMessageId?: string): Promise<ApiResponse<void>> {
    return post<void>(
      ENDPOINTS.GROUP_MESSAGES.MARK_READ(groupId),
      upToMessageId ? { up_to_message_id: upToMessageId } : undefined,
      { idempotencyKey: generateIdempotencyKey() }
    );
  },

  /**
   * 获取消息已读状态
   */
  async getReadStatus(
    messageId: string,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<MessageReadItem>>> {
    return get<PaginatedData<MessageReadItem>>(ENDPOINTS.GROUP_MESSAGES.READ_STATUS(messageId), params);
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
    return get<PaginatedData<MessageReactionItem>>(ENDPOINTS.GROUP_MESSAGES.REACTIONS(messageId), params);
  },

  /**
   * 添加反应
   */
  async addReaction(messageId: string, type: ReactionType): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_MESSAGES.ADD_REACTION(messageId), { type }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 移除反应
   */
  async removeReaction(messageId: string, type: ReactionType): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.GROUP_MESSAGES.REMOVE_REACTION(messageId, type));
  },
};
