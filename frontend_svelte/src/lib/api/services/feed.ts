/**
 * 信息流服务
 */

import { post, paginate } from '../client';
import { FEED } from '../endpoints';
import type {
	MomentItem,
	GroupItem,
	UserItem,
	TrendingBundle,
	FeedSource,
	GeoPoint,
	FeedContentType,
	FeedFeedbackAction,
	PaginationParams
} from '$lib/types/api';

export const feedService = {
	/**
	 * 获取瞬间信息流
	 */
	getMoments(params: {
		reference_point: GeoPoint;
		source?: FeedSource;
		radius?: number;
		pagination?: PaginationParams;
	}) {
		return paginate<MomentItem>(FEED.LIST_MOMENTS, params);
	},

	/**
	 * 获取群组推荐
	 */
	getGroups(params: { reference_point: GeoPoint; radius?: number; pagination?: PaginationParams }) {
		return paginate<GroupItem>(FEED.LIST_GROUPS, params);
	},

	/**
	 * 获取用户推荐
	 */
	getUsers(params: { reference_point: GeoPoint; radius?: number; pagination?: PaginationParams }) {
		return paginate<UserItem>(FEED.LIST_USERS, params);
	},

	/**
	 * 获取热门内容
	 */
	getTrending() {
		return post<TrendingBundle>(FEED.LIST_TRENDING);
	},

	/**
	 * 提交信息流反馈
	 */
	submitFeedback(data: {
		content_type: FeedContentType;
		content_id: string;
		action: FeedFeedbackAction;
		reason?: string;
	}) {
		return post<void>(FEED.ME.FEEDBACK, data);
	}
};
