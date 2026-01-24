/**
 * 用户服务
 */

import { get, post, put, del, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  UserItem,
  UserDetailData,
  UserStatsData,
  MyProfileData,
  UserSettingsData,
  UpdateProfileRequest,
  UpdateSettingsRequest,
  FriendListItem,
  FollowingItem,
  FollowerItem,
  BlockedUserItem,
  GeoPoint,
  Location,
} from '@/types/api';

export interface SearchByKeywordParams {
  kw: string;
  reference_point: GeoPoint;
  limit?: number;
  cursor?: string;
}

export interface SearchByTagParams {
  tag: string;
  reference_point: GeoPoint;
  limit?: number;
  cursor?: string;
}

export interface SearchByLocationParams {
  reference_point: GeoPoint;
  radius: number;
  kw?: string;
  limit?: number;
  cursor?: string;
}

export const userService = {
  // ============================================================================
  // 搜索
  // ============================================================================

  /**
   * 按关键词搜索用户
   */
  async searchByKeyword(params: SearchByKeywordParams): Promise<ApiResponse<PaginatedData<UserItem>>> {
    return post<PaginatedData<UserItem>>(ENDPOINTS.USER.SEARCH_BY_KEYWORD, params, {
      requireAuth: false,
    });
  },

  /**
   * 按标签搜索用户
   */
  async searchByTag(params: SearchByTagParams): Promise<ApiResponse<PaginatedData<UserItem>>> {
    return post<PaginatedData<UserItem>>(ENDPOINTS.USER.SEARCH_BY_TAG, params, {
      requireAuth: false,
    });
  },

  /**
   * 按位置搜索用户
   */
  async searchByLocation(params: SearchByLocationParams): Promise<ApiResponse<PaginatedData<UserItem>>> {
    return post<PaginatedData<UserItem>>(ENDPOINTS.USER.SEARCH_BY_LOCATION, params, {
      requireAuth: false,
    });
  },

  // ============================================================================
  // 用户详情
  // ============================================================================

  /**
   * 获取用户详情
   */
  async getDetail(userId: number, referencePoint?: GeoPoint): Promise<ApiResponse<UserDetailData>> {
    return post<UserDetailData>(
      ENDPOINTS.USER.DETAIL(userId),
      referencePoint ? { reference_point: referencePoint } : undefined,
      { requireAuth: false }
    );
  },

  /**
   * 获取用户统计
   */
  async getStats(userId: number): Promise<ApiResponse<UserStatsData>> {
    return get<UserStatsData>(ENDPOINTS.USER.STATS(userId), undefined, {
      requireAuth: false,
    });
  },

  /**
   * 获取用户好友列表
   */
  async getFriends(
    userId: number,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<FriendListItem>>> {
    return get<PaginatedData<FriendListItem>>(ENDPOINTS.USER.FRIENDS(userId), params, {
      requireAuth: false,
    });
  },

  /**
   * 获取用户关注列表
   */
  async getFollowing(
    userId: number,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<FollowingItem>>> {
    return get<PaginatedData<FollowingItem>>(ENDPOINTS.USER.FOLLOWING(userId), params, {
      requireAuth: false,
    });
  },

  /**
   * 获取用户粉丝列表
   */
  async getFollowers(
    userId: number,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<FollowerItem>>> {
    return get<PaginatedData<FollowerItem>>(ENDPOINTS.USER.FOLLOWERS(userId), params, {
      requireAuth: false,
    });
  },

  // ============================================================================
  // 当前用户
  // ============================================================================

  /**
   * 获取当前用户资料
   */
  async getMyProfile(): Promise<ApiResponse<MyProfileData>> {
    return get<MyProfileData>(ENDPOINTS.USER.ME_PROFILE);
  },

  /**
   * 更新当前用户资料
   */
  async updateProfile(data: UpdateProfileRequest): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.USER.ME_PROFILE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 更新当前用户位置
   */
  async updateLocation(location: Location): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.USER.ME_LOCATION, { location }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 获取当前用户设置
   */
  async getMySettings(): Promise<ApiResponse<UserSettingsData>> {
    return get<UserSettingsData>(ENDPOINTS.USER.ME_SETTINGS);
  },

  /**
   * 更新当前用户设置
   */
  async updateSettings(data: UpdateSettingsRequest): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.USER.ME_SETTINGS, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 获取我的好友列表
   */
  async getMyFriends(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<FriendListItem>>> {
    return get<PaginatedData<FriendListItem>>(ENDPOINTS.USER.ME_FRIENDS, params);
  },

  /**
   * 获取我的关注列表
   */
  async getMyFollowing(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<FollowingItem>>> {
    return get<PaginatedData<FollowingItem>>(ENDPOINTS.USER.ME_FOLLOWING, params);
  },

  /**
   * 获取我的粉丝列表
   */
  async getMyFollowers(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<FollowerItem>>> {
    return get<PaginatedData<FollowerItem>>(ENDPOINTS.USER.ME_FOLLOWERS, params);
  },

  /**
   * 获取我的黑名单
   */
  async getMyBlocked(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<BlockedUserItem>>> {
    return get<PaginatedData<BlockedUserItem>>(ENDPOINTS.USER.ME_BLOCKED, params);
  },

  // ============================================================================
  // 关系操作
  // ============================================================================

  /**
   * 关注用户
   */
  async follow(userId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.USER.FOLLOW(userId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 取消关注
   */
  async unfollow(userId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.USER.UNFOLLOW(userId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 拉黑用户
   */
  async block(userId: number, reason?: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.USER.BLOCK(userId), { reason }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 取消拉黑
   */
  async unblock(userId: number): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.USER.UNBLOCK(userId), undefined, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 设置备注
   */
  async setRemark(userId: number, remark?: string): Promise<ApiResponse<void>> {
    return put<void>(ENDPOINTS.USER.REMARK(userId), { remark }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  // ============================================================================
  // 账号操作
  // ============================================================================

  /**
   * 升级账号（访客升级为正式用户）
   */
  async upgradeAccount(login_id: string, password: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.USER.UPGRADE, { login_id, password }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 删除账号
   */
  async deleteAccount(confirmation: string): Promise<ApiResponse<void>> {
    return del<void>(ENDPOINTS.USER.DELETE, {
      body: { confirmation },
    });
  },
};
