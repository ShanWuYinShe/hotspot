import { post, paginate, generateIdempotencyKey } from "../client";
import { COMMENTS } from "../endpoints";
import type { CommentItem, LikeItem, PaginationParams } from "@/lib/types/api";

export const commentsService = {
  getComments(params: { moment_id: string; pagination?: PaginationParams }) {
    return paginate<CommentItem>(COMMENTS.LIST, params);
  },

  getReplies(params: { comment_id: string; pagination?: PaginationParams }) {
    return paginate<CommentItem>(COMMENTS.LIST_REPLIES, params);
  },

  getLikes(params: { comment_id: string; pagination?: PaginationParams }) {
    return paginate<LikeItem>(COMMENTS.LIST_LIKES, params);
  },

  create(data: {
    moment_id: string;
    content: string;
    parent_id?: string;
    mentions?: string[];
  }) {
    return post<CommentItem>(COMMENTS.CREATE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  delete(data: { comment_id: string }) {
    return post<void>(COMMENTS.DELETE, data);
  },

  like(data: { comment_id: string }) {
    return post<void>(COMMENTS.ME.LIKE, data);
  },

  unlike(data: { comment_id: string }) {
    return post<void>(COMMENTS.ME.UNLIKE, data);
  },
};
