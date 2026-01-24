'use server';

/**
 * Comments Server Actions
 * 处理评论相关的服务端操作
 */

import { revalidateTag } from 'next/cache';
import { serverApi, isServerSuccess } from '@/lib/api/server';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type { CreateCommentRequest } from '@/types/api';

/**
 * 创建评论
 */
export async function createComment(
  momentId: string,
  data: {
    content: string;
    parent_id?: string;
    reply_to_user_id?: number;
  }
) {
  const payload: CreateCommentRequest = {
    moment_id: momentId,
    content: data.content,
    parent_id: data.parent_id,
    mentions: data.reply_to_user_id ? [data.reply_to_user_id] : undefined,
  };
  const response = await serverApi.post(ENDPOINTS.COMMENTS.CREATE, payload, {});
  if (isServerSuccess(response)) {
    revalidateTag(`moment-${momentId}`, {});
    revalidateTag(`comments-${momentId}`, {});
  }
  return response;
}

/**
 * 删除评论
 */
export async function deleteComment(momentId: string, commentId: string) {
  const response = await serverApi.post(ENDPOINTS.COMMENTS.DELETE(commentId), { comment_id: commentId }, {});
  if (isServerSuccess(response)) {
    revalidateTag(`moment-${momentId}`, {});
    revalidateTag(`comments-${momentId}`, {});
  }
  return response;
}

/**
 * 点赞评论
 */
export async function likeComment(commentId: string) {
  const response = await serverApi.post(ENDPOINTS.COMMENTS.LIKE(commentId), { comment_id: commentId }, {});
  if (isServerSuccess(response)) {
    revalidateTag(`comment-${commentId}`, {});
  }
  return response;
}

export async function unlikeComment(commentId: string) {
  const response = await serverApi.post(ENDPOINTS.COMMENTS.UNLIKE(commentId), { comment_id: commentId }, {});
  if (isServerSuccess(response)) {
    revalidateTag(`comment-${commentId}`, {});
  }
  return response;
}
