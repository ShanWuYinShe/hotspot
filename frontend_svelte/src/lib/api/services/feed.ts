/**
 * 信息流服务
 */

import { post, paginate } from '../client';
import { FEED, TAGS } from '../endpoints';
import type {
	FeedMomentItem,
	FeedGroupItem,
	FeedUserItem,
	TrendingTag,
	TrendingTags,
	TrendingBundle,
	TagSuggestions,
	SearchSuggestions,
	FeedSource,
	GeoPoint,
	FeedContentType,
	FeedFeedbackAction
} from '$lib/types/api';

export const feedService = {
	/**
	 * 获取瞬间信息流
	 */
	getMoments(params: {
		source: FeedSource;
		reference_point?: GeoPoint;
		limit?: number;
		cursor?: string;
		radius?: number;
	}) {
		return paginate<FeedMomentItem>(FEED.MOMENTS, params);
	},

	/**
	 * 获取群组推荐
	 */
	getGroups(params: { reference_point: GeoPoint; radius?: number; limit?: number; cursor?: string }) {
		return paginate<FeedGroupItem>(FEED.GROUPS, params);
	},

	/**
	 * 获取用户推荐
	 */
	getUsers(params: { reference_point: GeoPoint; radius?: number; limit?: number; cursor?: string }) {
		return paginate<FeedUserItem>(FEED.USERS, params);
	},

	/**
	 * 获取热门内容
	 */
	getTrending(params?: { limit?: number }) {
		return post<TrendingBundle>(FEED.TRENDING, params);
	},

	/**
	 * 综合搜索建议
	 */
	suggest(params: { prefix: string; limit?: number }) {
		return post<SearchSuggestions>(FEED.SUGGEST, params);
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
		return post<void>(FEED.FEEDBACK, data);
	}
};

export const tagsService = {
	/**
	 * 获取热门标签
	 */
	getTrending(params?: {
		timeframe?: 'daily' | 'weekly' | 'monthly';
		limit?: number;
	}) {
		return post<TrendingTags>(TAGS.TRENDING, params);
	},

	/**
	 * 标签搜索建议
	 */
	suggest(params: { prefix: string; limit?: number }) {
		return post<TagSuggestions>(TAGS.SUGGEST, params);
	}
};
