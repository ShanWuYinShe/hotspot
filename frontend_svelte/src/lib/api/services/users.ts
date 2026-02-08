/**
 * 用户服务
 */

import { post, paginate } from '../client';
import { USER } from '../endpoints';
import type {
	UserCard,
	UserSearchItem,
	UserSuggestions,
	UserInfo,
	UserStats,
	UserSettings,
	MyCard,
	FriendItem,
	FollowingItem,
	FollowerItem,
	FollowRequestItem,
	BlockedUserItem,
	GeoPoint,
	Location,
	GroupSearchItem,
	UserMomentListItem,
	MomentSearchItem,
	FollowSource,
	Relationship
} from '$lib/types/api';

export const usersService = {
	// ==================== 搜索 ====================

	/**
	 * 关键词搜索用户
	 */
	searchByKeyword(params: { reference_point: GeoPoint; kw: string; limit?: number; cursor?: string }) {
		return paginate<UserSearchItem>(USER.SEARCH_BY_KEYWORD, params);
	},

	/**
	 * 用户搜索建议
	 */
	suggest(params: { prefix: string; limit?: number }) {
		return post<UserSuggestions>(USER.SUGGEST, params);
	},

	/**
	 * 标签搜索用户
	 */
	searchByTag(params: { reference_point: GeoPoint; tag: string; limit?: number; cursor?: string }) {
		return paginate<UserSearchItem>(USER.SEARCH_BY_TAG, params);
	},

	/**
	 * 位置搜索用户
	 */
	searchByLocation(params: {
		reference_point: GeoPoint;
		radius: number;
		kw?: string;
		limit?: number;
		cursor?: string;
	}) {
		return paginate<UserSearchItem>(USER.SEARCH_BY_LOCATION, params);
	},

	// ==================== 用户信息 ====================

	/**
	 * 获取用户详情
	 */
	getUser(data: { user_id: number }) {
		return post<UserCard>(USER.OPEN_THE_DOOR, data);
	},

	/**
	 * 获取用户统计
	 */
	getStats(data: { user_id: number }) {
		return post<UserStats>(USER.STATS, data);
	},

	/**
	 * 获取用户好友列表
	 */
	getFriends(params: { user_id: number; limit?: number; cursor?: string }) {
		return paginate<UserInfo>(USER.FRIENDS, params);
	},

	/**
	 * 获取用户关注列表
	 */
	getFollowing(params: { user_id: number; limit?: number; cursor?: string }) {
		return paginate<UserInfo>(USER.FOLLOWING, params);
	},

	/**
	 * 获取用户粉丝列表
	 */
	getFollowers(params: { user_id: number; limit?: number; cursor?: string }) {
		return paginate<UserInfo>(USER.FOLLOWERS, params);
	},

	/**
	 * 获取用户加入的群组
	 */
	getGroups(params: { user_id: number; limit?: number; cursor?: string }) {
		return paginate<GroupSearchItem>(USER.GROUPS, params);
	},

	/**
	 * 获取用户的瞬间
	 */
	getMoments(params: { user_id: number; limit?: number; cursor?: string }) {
		return paginate<UserMomentListItem>(USER.MOMENTS, params);
	},

	/**
	 * 获取用户点赞的瞬间
	 */
	getLikedMoments(params: { user_id: number; limit?: number; cursor?: string }) {
		return paginate<MomentSearchItem>(USER.LIKED_MOMENTS, params);
	},

	// ==================== 当前用户 ====================

	/**
	 * 获取当前用户资料
	 */
	getMyProfile() {
		return post<MyCard>(USER.ME.PROFILE);
	},

	/**
	 * 更新当前用户资料
	 */
	updateMyProfile(data: {
		nickname?: string;
		avatar_file_id?: string;
		bio?: string;
		tags?: string[];
	}) {
		return post<MyCard>(USER.ME.PROFILE_UPDATE, data);
	},

	/**
	 * 更新位置
	 */
	updateLocation(data: { location: Location }) {
		return post<void>(USER.ME.LOCATION, data);
	},

	/**
	 * 升级账户（访客转正式）
	 */
	upgradeAccount(data: { login_id: string; password: string; verify_token?: string }) {
		return post<void>(USER.ME.UPGRADE, data);
	},

	/**
	 * 获取当前用户统计
	 */
	getMyStats() {
		return post<UserStats>(USER.ME.STATS);
	},

	/**
	 * 获取当前用户设置
	 */
	getMySettings() {
		return post<UserSettings>(USER.ME.SETTINGS);
	},

	/**
	 * 更新当前用户设置
	 */
	updateMySettings(data: Partial<UserSettings>) {
		return post<UserSettings>(USER.ME.SETTINGS_UPDATE, data);
	},

	/**
	 * 删除账户
	 */
	deleteAccount(data: { password: string }) {
		return post<void>(USER.ME.DELETE, data);
	},

	/**
	 * 获取我的好友列表
	 */
	getMyFriends(params?: { limit?: number; cursor?: string }) {
		return paginate<FriendItem>(USER.ME.FRIENDS, params);
	},

	/**
	 * 获取我的关注列表
	 */
	getMyFollowing(params?: { limit?: number; cursor?: string }) {
		return paginate<FollowingItem>(USER.ME.FOLLOWING, params);
	},

	/**
	 * 获取我的粉丝列表
	 */
	getMyFollowers(params?: { limit?: number; cursor?: string }) {
		return paginate<FollowerItem>(USER.ME.FOLLOWERS, params);
	},

	/**
	 * 获取我的黑名单
	 */
	getBlocked(params?: { limit?: number; cursor?: string }) {
		return paginate<BlockedUserItem>(USER.ME.BLOCKED, params);
	},

	/**
	 * 获取关注请求
	 */
	getFollowRequests(params?: { limit?: number; cursor?: string }) {
		return paginate<FollowRequestItem>(USER.ME.FOLLOW_REQUESTS, params);
	},

	/**
	 * 处理关注请求
	 */
	handleFollowRequest(data: { request_id: string; action: 'accept' | 'reject' }) {
		return post<void>(USER.ME.FOLLOW_REQUESTS_HANDLE, data);
	},

	// ==================== 关注操作 ====================

	/**
	 * 关注用户
	 */
	follow(data: { user_id: number; source?: FollowSource; source_moment_id?: string }) {
		return post<Relationship>(USER.ME.FOLLOW, data);
	},

	/**
	 * 取消关注
	 */
	unfollow(data: { user_id: number }) {
		return post<Relationship>(USER.ME.UNFOLLOW, data);
	},

	/**
	 * 拉黑用户
	 */
	block(data: { user_id: number }) {
		return post<void>(USER.ME.BLOCK, data);
	},

	/**
	 * 取消拉黑
	 */
	unblock(data: { user_id: number }) {
		return post<void>(USER.ME.UNBLOCK, data);
	},

	/**
	 * 设置备注
	 */
	setRemark(data: { user_id: number; remark: string }) {
		return post<void>(USER.ME.REMARK, data);
	}
};
