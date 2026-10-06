// 自动拆分自 client.ts（refactor/api-client-split）：按业务域组织的端点包装。
// 传输核心（request/刷新/幂等键/双版本 pick）见 ../core.ts。
import { request, newIdempotencyKey, pick } from '../core';
import { Endpoints } from '@/api/endpoints';
import type {
	CommentItem,
	PaginatedData,
	SetReactionRequest,
	CreateCommentRequest,
	DeleteCommentRequest,
	PaginationParams,
} from '@/api/types';

// Reactions
export async function setReaction(params: SetReactionRequest)         { return request<void>(Endpoints.Reactions.Set, params); }

// Comments
export async function createComment(params: CreateCommentRequest)       { return request<CommentItem>(Endpoints.Comments.Create, params, { 'Idempotency-Key': newIdempotencyKey() }); }
// spec: api_spec_comment.md → comments/delete（仅评论作者；软删除——顶层删后子评论保留，展示占位）
export async function deleteComment(params: DeleteCommentRequest)       { return request<void>(Endpoints.Comments.Delete, params); }

// Comments
export async function listComments(momentId: string, pagination?: PaginationParams) {
  return request<PaginatedData<CommentItem>>(pick(Endpoints.Comments.List, Endpoints.Public.Comments.List), { moment_id: momentId, pagination });
}
// spec: api_spec_comment.md → comments/fetch_replies（子评论列表，时间正序）
export async function listCommentReplies(commentId: string, pagination?: PaginationParams) {
  return request<PaginatedData<CommentItem>>(pick(Endpoints.Comments.ListReplies, Endpoints.Public.Comments.ListReplies), { comment_id: commentId, pagination });
}
