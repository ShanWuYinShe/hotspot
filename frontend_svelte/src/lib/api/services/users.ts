/**
 * 用户服务
 */

import { post, paginate, generateIdempotencyKey, type RequestOptions } from '../client';
import { USERS } from '../endpoints';
import type {
	UserItem,
	UserSettings,
	MyProfile,
	FriendItem,
	FollowingItem,
	FollowerItem,
	BlockedUserItem,
	Location,
	Gender,
	GroupItem,
	MomentItem,
	FollowSource,
	PaginationParams,
	GeoPoint,
	VerificationSent
} from '$lib/types/api';

export const usersService = {
	// ==================== 搜索 ====================

		/**
		 * 搜索用户（对齐 spec SearchUsersRequest）
		 */
	search(params: {
		keyword?: string;
		tag?: string;
		radius?: number;
		sort_by?: 'relevance' | 'distance' | 'newest';
		reference_point: GeoPoint;
		pagination?: PaginationParams;
	}, options?: RequestOptions) {
		return paginate<UserItem>(USERS.SEARCH, params, options);
	},

	// ==================== 用户信息 ====================

	/**
	 * 获取用户详情
	 */
	getUser(data: { user_id: string }) {
		return post<UserItem>(USERS.OPEN_THE_DOOR, data);
	},

	/**
	 * 获取用户好友列表（user_id 不传时返回当前用户的）
	 */
	getFriends(params: { user_id?: string; pagination?: PaginationParams }) {
		return paginate<FriendItem>(USERS.LIST_FRIENDS, params);
	},

	/**
	 * 获取用户关注列表（user_id 不传时返回当前用户的）
	 */
	getFollowing(params: { user_id?: string; pagination?: PaginationParams }) {
		return paginate<FollowingItem>(USERS.LIST_FOLLOWING, params);
	},

	/**
	 * 获取用户粉丝列表（user_id 不传时返回当前用户的）
	 */
	getFollowers(params: { user_id?: string; pagination?: PaginationParams }) {
		return paginate<FollowerItem>(USERS.LIST_FOLLOWERS, params);
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
		gender?: Gender;
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
	 * 删除账户（SHOULD 幂等，spec: confirmation 必须为 "DELETE_MY_ACCOUNT"）
	 */
	deleteAccount(data: { confirmation: string }) {
		return post<void>(USERS.ME.DELETE, data, { idempotencyKey: generateIdempotencyKey() });
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
	setRemark(data: { user_id: string; remark: string }) {
		return post<void>(USERS.ME.SET_REMARK, data);
	},

	// ==================== 邮箱/手机号变更（⏳ 延期：依赖验证码发送服务） ====================

	/**
	 * 请求变更邮箱（发送验证码到新邮箱）
	 * spec: MUST 幂等，需密码验证身份
	 */
	requestEmailChange(data: { new_email: string; password: string }) {
		return post<VerificationSent>(USERS.ME.EMAIL_REQUEST_CHANGE, data, { idempotencyKey: generateIdempotencyKey() });
	},

	/**
	 * 确认变更邮箱（提交验证码完成变更）
	 * spec: MUST 幂等
	 */
	confirmEmailChange(data: { new_email: string; verification_code: string }) {
		return post<void>(USERS.ME.EMAIL_CONFIRM_CHANGE, data, { idempotencyKey: generateIdempotencyKey() });
	},

	/**
	 * 请求变更手机号（发送验证码到新手机号）
	 * spec: MUST 幂等，需密码验证身份
	 */
	requestPhoneChange(data: { new_phone: string; password: string }) {
		return post<VerificationSent>(USERS.ME.PHONE_REQUEST_CHANGE, data, { idempotencyKey: generateIdempotencyKey() });
	},

	/**
	 * 确认变更手机号（提交验证码完成变更）
	 * spec: MUST 幂等
	 */
	confirmPhoneChange(data: { new_phone: string; verification_code: string }) {
		return post<void>(USERS.ME.PHONE_CONFIRM_CHANGE, data, { idempotencyKey: generateIdempotencyKey() });
	}
};
