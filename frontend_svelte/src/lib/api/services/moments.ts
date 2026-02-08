/**
 * 瞬间服务
 */

import { post, paginate } from '../client';
import { MOMENTS } from '../endpoints';
import type {
	MomentCard,
	MomentSearchItem,
	MomentSuggestions,
	MomentLikeItem,
	GeoPoint,
	Location,
	PrivacyLevel,
	UserMomentListItem
} from '$lib/types/api';

export const momentsService = {
	// ==================== 搜索 ====================

	/**
	 * 关键词搜索瞬间
	 */
	searchByKeyword(params: { reference_point: GeoPoint; kw: string; limit?: number; cursor?: string }) {
		return paginate<MomentSearchItem>(MOMENTS.SEARCH_BY_KEYWORD, params);
	},

	/**
	 * 瞬间搜索建议
	 */
	suggest(params: { prefix: string; limit?: number }) {
		return post<MomentSuggestions>(MOMENTS.SUGGEST, params);
	},

	/**
	 * 标签搜索瞬间
	 */
	searchByTag(params: { reference_point: GeoPoint; tag: string; limit?: number; cursor?: string }) {
		return paginate<MomentSearchItem>(MOMENTS.SEARCH_BY_TAG, params);
	},

	/**
	 * 位置搜索瞬间
	 */
	searchByLocation(params: {
		reference_point: GeoPoint;
		radius: number;
		kw?: string;
		limit?: number;
		cursor?: string;
	}) {
		return paginate<MomentSearchItem>(MOMENTS.SEARCH_BY_LOCATION, params);
	},

	// ==================== 瞬间信息 ====================

	/**
	 * 获取瞬间详情
	 */
	getMoment(data: { moment_id: string }) {
		return post<MomentCard>(MOMENTS.OPEN_THE_DOOR, data);
	},

	/**
	 * 获取点赞列表
	 */
	getLikes(params: { moment_id: string; limit?: number; cursor?: string }) {
		return paginate<MomentLikeItem>(MOMENTS.LIKES, params);
	},

	// ==================== 瞬间操作 ====================

	/**
	 * 创建瞬间
	 */
	create(data: {
		description: string;
		location: Location;
		tags?: string[];
		image_file_ids?: string[];
		privacy_level?: PrivacyLevel;
		mentions?: number[];
	}) {
		return post<UserMomentListItem>(MOMENTS.CREATE, data);
	},

	/**
	 * 更新瞬间
	 */
	update(data: {
		moment_id: string;
		description?: string;
		location?: Location;
		tags?: string[];
		image_file_ids?: string[];
		privacy_level?: PrivacyLevel;
		mentions?: number[];
	}) {
		return post<void>(MOMENTS.UPDATE, data);
	},

	/**
	 * 删除瞬间
	 */
	delete(data: { moment_id: string }) {
		return post<void>(MOMENTS.DELETE, data);
	},

	/**
	 * 点赞瞬间
	 */
	like(data: { moment_id: string }) {
		return post<void>(MOMENTS.LIKE, data);
	},

	/**
	 * 取消点赞
	 */
	unlike(data: { moment_id: string }) {
		return post<void>(MOMENTS.UNLIKE, data);
	},

	/**
	 * 收藏瞬间
	 */
	favorite(data: { moment_id: string }) {
		return post<void>(MOMENTS.FAVORITE, data);
	},

	/**
	 * 取消收藏
	 */
	unfavorite(data: { moment_id: string }) {
		return post<void>(MOMENTS.UNFAVORITE, data);
	},

	// ==================== 当前用户 ====================

	/**
	 * 获取我的瞬间列表
	 */
	getMyMoments(params?: { limit?: number; cursor?: string }) {
		return paginate<UserMomentListItem>(MOMENTS.ME.LIST, params);
	},

	/**
	 * 获取我的收藏
	 */
	getMyFavorites(params?: { limit?: number; cursor?: string }) {
		return paginate<MomentCard>(MOMENTS.ME.FAVORITES, params);
	}
};
