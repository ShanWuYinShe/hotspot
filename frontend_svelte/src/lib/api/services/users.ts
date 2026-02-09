/**
 * 用户服务
 */

import { post, paginate } from '../client';
import { USERS } from '../endpoints';
import type {
	UserItem,
	UserInfo,
	UserSettings,
	MyProfile,
	FriendItem,
	FollowingItem,
	FollowerItem,
	BlockedUserItem,
	Location,
	GroupItem,
	MomentItem,
	FollowSource,
	PaginationParams,
	SearchBaseParams
} from '$lib/types/api';

export const usersService = {
	// ==================== 搜索 ====================

	/**
	 * 搜索用户（对齐 spec SearchUsersRequest）
	 */
	search(params: {
		search: SearchBaseParams;
		keyword?: string;
		tag?: string;
		radius?: number;
		sort_by?: 'relevance' | 'distance' | 'newest';
	}) {
		return paginate<UserItem>(USERS.SEARCH, params);
	},

	// ==================== 用户信息 ====================

	/**
	 * 获取用户详情
	 */
	getUser(data: { user_id: string }) {
		return post<UserItem>(USERS.OPEN_THE_DOOR, data);
	},

	/**
	 * 获取用户好友列表
	 */
	getFriends(params: { user_id: string; pagination?: PaginationParams }) {
		return paginate<UserInfo>(USERS.LIST_FRIENDS, params);
	},

	/**
	 * 获取用户关注列表
	 */
	getFollowing(params: { user_id: string; pagination?: PaginationParams }) {
		return paginate<UserInfo>(USERS.LIST_FOLLOWING, params);
	},

	/**
	 * 获取用户粉丝列表
	 */
	getFollowers(params: { user_id: string; pagination?: PaginationParams }) {
		return paginate<UserInfo>(USERS.LIST_FOLLOWERS, params);
	},

	/**
	 * 获取用户加入的群组
	 */
	getGroups(params: { user_id: string; pagination?: PaginationParams }) {
		return paginate<GroupItem>(USERS.LIST_GROUPS, params);
	},

	/**
	 * 获取用户的瞬间
	 */
	getMoments(params: { user_id: string; pagination?: PaginationParams }) {
		return paginate<MomentItem>(USERS.LIST_MOMENTS, params);
	},

	/**
	 * 获取用户点赞的瞬间
	 */
	getLikedMoments(params: { user_id: string; pagination?: PaginationParams }) {
		return paginate<MomentItem>(USERS.LIST_LIKED_MOMENTS, params);
	},

	// ==================== 当前用户 ====================

	/**
	 * 获取当前用户资料
	 */
	getMyProfile() {
		return post<MyProfile>(USERS.ME.PROFILE_GET);
	},

	/**
	 * 更新当前用户资料
	 */
	updateMyProfile(data: {
		nickname?: string;
		avatar_file_id?: string;
		bio?: string;
		tags?: string[];
		gender?: 'male' | 'female' | 'apache_helicopter';
	}) {
		return post<void>(USERS.ME.PROFILE_UPDATE, data);
	},

	/**
	 * 更新位置
	 */
	updateLocation(data: { location: Location }) {
		return post<void>(USERS.ME.LOCATION_UPDATE, data);
	},

	/**
	 * 升级账户（访客转正式）
	 */
	upgradeAccount(data: { login_id: string; password: string }) {
		return post<void>(USERS.ME.UPGRADE, data);
	},

	/**
	 * 获取当前用户设置
	 */
	getMySettings() {
		return post<UserSettings>(USERS.ME.SETTINGS_GET);
	},

	/**
	 * 更新当前用户设置（spec: { settings: Partial<UserSettings> }）
	 */
	updateMySettings(data: Partial<UserSettings>) {
		return post<void>(USERS.ME.SETTINGS_UPDATE, { settings: data });
	},

	/**
	 * 删除账户（spec: confirmation 必须为 "DELETE_MY_ACCOUNT"）
	 */
	deleteAccount(data: { confirmation: string }) {
		return post<void>(USERS.ME.DELETE, data);
	},

	/**
	 * 获取我的好友列表
	 */
	getMyFriends(params?: { pagination?: PaginationParams }) {
		return paginate<FriendItem>(USERS.ME.LIST_FRIENDS, params);
	},

	/**
	 * 获取我的关注列表
	 */
	getMyFollowing(params?: { pagination?: PaginationParams }) {
		return paginate<FollowingItem>(USERS.ME.LIST_FOLLOWING, params);
	},

	/**
	 * 获取我的粉丝列表
	 */
	getMyFollowers(params?: { pagination?: PaginationParams }) {
		return paginate<FollowerItem>(USERS.ME.LIST_FOLLOWERS, params);
	},

	/**
	 * 获取我的黑名单
	 */
	getBlocked(params?: { pagination?: PaginationParams }) {
		return paginate<BlockedUserItem>(USERS.ME.LIST_BLOCKED, params);
	},

	// ==================== 关注操作 ====================

	/**
	 * 关注用户
	 */
	follow(data: { user_id: string; source?: FollowSource; source_moment_id?: string }) {
		return post<void>(USERS.ME.FOLLOW, data);
	},

	/**
	 * 取消关注
	 */
	unfollow(data: { user_id: string }) {
		return post<void>(USERS.ME.UNFOLLOW, data);
	},

	/**
	 * 拉黑用户
	 */
	block(data: { user_id: string; reason?: string }) {
		return post<void>(USERS.ME.BLOCK, data);
	},

	/**
	 * 取消拉黑
	 */
	unblock(data: { user_id: string }) {
		return post<void>(USERS.ME.UNBLOCK, data);
	},

	/**
	 * 设置备注
	 */
	setRemark(data: { user_id: string; remark?: string }) {
		return post<void>(USERS.ME.SET_REMARK, data);
	}
};
