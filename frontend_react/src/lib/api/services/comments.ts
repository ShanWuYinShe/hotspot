/**
 * 评论服务
 */

import { get, post, del, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  CommentListItem,
  CreateCommentRequest,
  CommentLikeItem,
} from '@/types/api';

export const commentsService = {
  /**
   * 获取瞬间评论列表
   */
  async getComments(
    momentId: string,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return get<PaginatedData<CommentListItem>>(ENDPOINTS.MOMENTS.COMMENTS(momentId), params, {
      requireAuth: false,
    });
  },

  /**
   * 创建评论
   */
  async create(data: CreateCommentRequest): Promise<ApiResponse<CommentListItem>> {
    return post<CommentListItem>(ENDPOINTS.COMMENTS.CREATE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 删除评论
   */
  async delete(commentId: string): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.COMMENTS.DELETE(commentId));
  },

  /**
   * 获取评论回复列表
   */
  async getReplies(
    commentId: string,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
    return get<PaginatedData<CommentListItem>>(ENDPOINTS.COMMENTS.REPLIES(commentId), params, {
      requireAuth: false,
    });
  },

  /**
   * 点赞评论
   */
  async like(commentId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.COMMENTS.LIKE(commentId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 取消点赞评论
   */
  async unlike(commentId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.COMMENTS.UNLIKE(commentId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 获取评论点赞列表
   */
  async getLikes(
    commentId: string,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<CommentLikeItem>>> {
    return get<PaginatedData<CommentLikeItem>>(ENDPOINTS.COMMENTS.LIKES(commentId), params, {
      requireAuth: false,
    });
  },
};
