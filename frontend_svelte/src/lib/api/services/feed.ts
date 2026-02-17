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
	DismissTargetType,
	GeoPoint,
	PaginationParams
} from '$lib/types/api';

export const feedService = {
	/**
	 * 获取瞬间信息流
	 */
	getMoments(params: {
		reference_point: GeoPoint;
		source: FeedSource;
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
		return post<TrendingBundle>(FEED.LIST_TRENDING, {});
	},

	/**
	 * 标记不感兴趣（⏳ 延期：后端尚未实现）
	 */
	dismiss(data: { target_type: DismissTargetType; target_id: string }) {
		return post<void>(FEED.ME.DISMISS, data);
	}
};
