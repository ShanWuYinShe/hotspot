/**
 * 群聊消息服务
 */

import { post, generateIdempotencyKey } from '../client';
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
    return post<PaginatedData<GroupMessageItem>>(ENDPOINTS.GROUP_MESSAGES.HISTORY(groupId), {
      group_chat_id: groupId,
      ...(params ?? {}),
    });
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
    return post<void>(ENDPOINTS.GROUP_MESSAGES.UPDATE(messageId), { message_id: messageId, content }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 撤回消息
   */
  async recallMessage(messageId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_MESSAGES.RECALL(messageId), { message_id: messageId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 删除消息
   */
  async deleteMessage(messageId: string, forEveryone?: boolean): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_MESSAGES.DELETE(messageId), {
      message_id: messageId,
      ...(forEveryone ? { for_everyone: true } : {}),
    });
  },

  /**
   * 标记已读
   */
  async markRead(groupId: string, upToMessageId?: string): Promise<ApiResponse<void>> {
    return post<void>(
      ENDPOINTS.GROUP_MESSAGES.MARK_READ(groupId),
      {
        group_chat_id: groupId,
        ...(upToMessageId ? { up_to_message_id: upToMessageId } : {}),
      },
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
    return post<PaginatedData<MessageReadItem>>(ENDPOINTS.GROUP_MESSAGES.READ_STATUS(messageId), {
      message_id: messageId,
      ...(params ?? {}),
    });
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
    return post<PaginatedData<MessageReactionItem>>(ENDPOINTS.GROUP_MESSAGES.REACTIONS(messageId), {
      message_id: messageId,
      ...(params ?? {}),
    });
  },

  /**
   * 添加反应
   */
  async addReaction(messageId: string, type: ReactionType): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_MESSAGES.ADD_REACTION(messageId), { message_id: messageId, type }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 移除反应
   */
  async removeReaction(messageId: string, type: ReactionType): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.GROUP_MESSAGES.REMOVE_REACTION(messageId, type), {
      message_id: messageId,
      type,
    });
  },
};
