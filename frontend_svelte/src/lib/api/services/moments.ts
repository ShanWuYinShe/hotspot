/**
 * 瞬间服务
 */

import { post, paginate } from '../client';
import { MOMENTS } from '../endpoints';
import type {
	MomentItem,
	LikeItem,
	Location,
	PrivacyLevel,
	PaginationParams,
	GeoPoint
} from '$lib/types/api';

export const momentsService = {
	// ==================== 搜索 ====================

	/**
	 * 搜索瞬间（对齐 spec SearchMomentsRequest）
	 */
	search(params: {
		keyword?: string;
		tag?: string;
		radius?: number;
		sort_by?: 'relevance' | 'distance' | 'newest' | 'popular';
		reference_point: GeoPoint;
		pagination?: PaginationParams;
	}) {
		return paginate<MomentItem>(MOMENTS.SEARCH, params);
	},

	// ==================== 瞬间信息 ====================

	/**
	 * 获取瞬间详情
	 */
	getMoment(data: { moment_id: string }) {
		return post<MomentItem>(MOMENTS.OPEN_THE_DOOR, data);
	},

	/**
	 * 获取点赞列表
	 */
	getLikes(params: { moment_id: string; pagination?: PaginationParams }) {
		return paginate<LikeItem>(MOMENTS.LIST_LIKES, params);
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
		mentions?: string[];
	}) {
		return post<MomentItem>(MOMENTS.CREATE, data);
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
		mentions?: string[];
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
		return post<void>(MOMENTS.ME.LIKE, data);
	},

	/**
	 * 取消点赞
	 */
	unlike(data: { moment_id: string }) {
		return post<void>(MOMENTS.ME.UNLIKE, data);
	},

	/**
	 * 收藏瞬间
	 */
	favorite(data: { moment_id: string }) {
		return post<void>(MOMENTS.ME.FAVORITE, data);
	},

	/**
	 * 取消收藏
	 */
	unfavorite(data: { moment_id: string }) {
		return post<void>(MOMENTS.ME.UNFAVORITE, data);
	},

	// ==================== 当前用户 ====================

	/**
	 * 获取我的瞬间列表
	 */
	getMyMoments(params?: { pagination?: PaginationParams }) {
		return paginate<MomentItem>(MOMENTS.ME.LIST, params);
	},

	/**
	 * 获取我的收藏
	 */
	getMyFavorites(params?: { pagination?: PaginationParams }) {
		return paginate<MomentItem>(MOMENTS.ME.LIST_FAVORITES, params);
	}
};
