/**
 * 评论服务
 */

import { post, paginate } from '../client';
import { COMMENTS } from '../endpoints';
import type { CommentListItem, CommentLikeItem } from '$lib/types/api';

export const commentsService = {
	/**
	 * 获取评论列表
	 */
	getComments(params: { moment_id: string; limit?: number; cursor?: string }) {
		return paginate<CommentListItem>(COMMENTS.LIST, params);
	},

	/**
	 * 获取评论回复
	 */
	getReplies(params: { comment_id: string; limit?: number; cursor?: string }) {
		return paginate<CommentListItem>(COMMENTS.REPLIES, params);
	},

	/**
	 * 获取评论点赞列表
	 */
	getLikes(params: { comment_id: string; limit?: number; cursor?: string }) {
		return paginate<CommentLikeItem>(COMMENTS.LIKES, params);
	},

	/**
	 * 创建评论
	 */
	create(data: {
		moment_id: string;
		content: string;
		parent_id?: string;
		mentions?: number[];
	}) {
		return post<CommentListItem>(COMMENTS.CREATE, data);
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
		return post<void>(COMMENTS.LIKE, data);
	},

	/**
	 * 取消点赞评论
	 */
	unlike(data: { comment_id: string }) {
		return post<void>(COMMENTS.UNLIKE, data);
	}
};
