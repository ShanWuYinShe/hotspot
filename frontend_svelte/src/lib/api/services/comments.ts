/**
 * 评论服务
 */

import { post, paginate, generateIdempotencyKey } from '../client';
import { COMMENTS } from '../endpoints';
import type { CommentItem, LikeItem, PaginationParams } from '$lib/types/api';

export const commentsService = {
	/**
	 * 获取评论列表
	 */
	getComments(params: { moment_id: string; pagination?: PaginationParams }) {
		return paginate<CommentItem>(COMMENTS.LIST, params);
	},

	/**
	 * 获取评论回复
	 */
	getReplies(params: { comment_id: string; pagination?: PaginationParams }) {
		return paginate<CommentItem>(COMMENTS.LIST_REPLIES, params);
	},

	/**
	 * 获取评论点赞列表
	 */
	getLikes(params: { comment_id: string; pagination?: PaginationParams }) {
		return paginate<LikeItem>(COMMENTS.LIST_LIKES, params);
	},

	/**
	 * 创建评论
	 */
	create(data: {
		moment_id: string;
		content: string;
		parent_id?: string;
		mentions?: string[];
	}) {
		return post<CommentItem>(COMMENTS.CREATE, data, { idempotencyKey: generateIdempotencyKey() });
	},

	/**
	 * 删除评论
	 */
	delete(data: { comment_id: string }) {
		return post<void>(COMMENTS.DELETE, data);
	},

	/**
	 * 点赞评论
	 */
	like(data: { comment_id: string }) {
		return post<void>(COMMENTS.ME.LIKE, data);
	},

	/**
	 * 取消点赞评论
	 */
	unlike(data: { comment_id: string }) {
		return post<void>(COMMENTS.ME.UNLIKE, data);
	}
};
