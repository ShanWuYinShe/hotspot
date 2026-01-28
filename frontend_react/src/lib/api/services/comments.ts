/**
 * 评论服务
 * 评论相关 API
 */

import { apiRequestPublic, apiRequestIdempotent } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type {
  ApiResponse,
  PaginatedData,
  CommentListItem,
} from '@/types/api';

interface ListCommentsParams {
  moment_id: string;
  limit?: number;
  cursor?: string;
}

export async function getComments(
  params: ListCommentsParams
): Promise<ApiResponse<PaginatedData<CommentListItem>>> {
  return apiRequestPublic<PaginatedData<CommentListItem>>(
    ENDPOINTS.COMMENTS.LIST,
    params
  );
}

interface CreateCommentParams {
  moment_id: string;
  content: string;
  parent_id?: string;
  mentions?: number[];
}

export async function createComment(
  params: CreateCommentParams
): Promise<ApiResponse<{ comment_id: string }>> {
  return apiRequestIdempotent<{ comment_id: string }>(
    ENDPOINTS.COMMENTS.CREATE,
    params
  );
}

export async function likeComment(
  commentId: string
): Promise<ApiResponse<void>> {
  return apiRequestIdempotent<void>(ENDPOINTS.COMMENTS.LIKE, {
    comment_id: commentId,
  });
}
