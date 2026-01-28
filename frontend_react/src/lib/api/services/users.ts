/**
 * 用户服务
 * 用户相关 API
 * 
 * 基于 @api_spec_user.mdc 规范
 */

import { apiRequest, apiRequestPublic, apiRequestIdempotent } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type {
  ApiResponse,
  PaginatedData,
  UserCard,
  Relationship,
  UserStats,
  UserSettings,
  MyCard,
  UserMomentListItem,
  UserInfo,
  GroupSearchItem,
  MomentSearchItem,
  Gender,
} from '@/types/api';

// ============== 用户查询 ==============

/**
 * 获取用户资料详情
 * Token 可选；登录态返回 relation
 * @api POST /api/v1/user/open_the_door
 */
export async function getUser(
  userId: number
): Promise<ApiResponse<UserCard>> {
  return apiRequestPublic<UserCard>(ENDPOINTS.USER.OPEN_THE_DOOR, {
    user_id: userId,
  });
}

/**
 * 获取与指定用户的关系
 * @api POST /api/v1/user/relationship
 */
export async function getRelationship(
  userId: number
): Promise<ApiResponse<Relationship>> {
  return apiRequest<Relationship>(ENDPOINTS.USER.RELATIONSHIP, {
    user_id: userId,
  });
}

/**
 * 获取用户统计数据
 * 隐私受控：受 stats_visible 设置控制
 * @api POST /api/v1/user/stats
 */
export async function getUserStats(
  userId: number
): Promise<ApiResponse<UserStats>> {
  return apiRequestPublic<UserStats>(ENDPOINTS.USER.STATS, {
    user_id: userId,
  });
}

// ============== 用户列表 ==============

interface UserListParams {
  user_id: number;
  limit?: number;
  cursor?: string;
}

/**
 * 获取用户好友列表
 * 隐私受控：受 friends_visible 设置控制
 * @api POST /api/v1/user/friends
 */
export async function getUserFriends(
  params: UserListParams
): Promise<ApiResponse<PaginatedData<UserInfo>>> {
  return apiRequestPublic<PaginatedData<UserInfo>>(
    ENDPOINTS.USER.FRIENDS,
    params
  );
}

/**
 * 获取用户关注列表
 * 隐私受控：受 following_visible 设置控制
 * @api POST /api/v1/user/following
 */
export async function getUserFollowing(
  params: UserListParams
): Promise<ApiResponse<PaginatedData<UserInfo>>> {
  return apiRequestPublic<PaginatedData<UserInfo>>(
    ENDPOINTS.USER.FOLLOWING,
    params
  );
}

/**
 * 获取用户粉丝列表
 * 隐私受控：受 followers_visible 设置控制
 * @api POST /api/v1/user/followers
 */
export async function getUserFollowers(
  params: UserListParams
): Promise<ApiResponse<PaginatedData<UserInfo>>> {
  return apiRequestPublic<PaginatedData<UserInfo>>(
    ENDPOINTS.USER.FOLLOWERS,
    params
  );
}

/**
 * 获取用户群组列表
 * 隐私受控：受 groups_visible 设置控制
 * @api POST /api/v1/user/groups
 */
export async function getUserGroups(
  params: UserListParams
): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
  return apiRequestPublic<PaginatedData<GroupSearchItem>>(
    ENDPOINTS.USER.GROUPS,
    params
  );
}

/**
 * 获取用户点赞瞬间列表
 * 隐私受控：受 liked_moments_visible 设置控制
 * @api POST /api/v1/user/liked_moments
 */
export async function getUserLikedMoments(
  params: UserListParams
): Promise<ApiResponse<PaginatedData<MomentSearchItem>>> {
  return apiRequestPublic<PaginatedData<MomentSearchItem>>(
    ENDPOINTS.USER.LIKED_MOMENTS,
    params
  );
}

/**
 * 获取用户瞬间列表
 * @api POST /api/v1/user/moments
 */
export async function getUserMoments(
  params: UserListParams
): Promise<ApiResponse<PaginatedData<UserMomentListItem>>> {
  return apiRequestPublic<PaginatedData<UserMomentListItem>>(
    ENDPOINTS.USER.MOMENTS,
    params
  );
}

// ============== 当前用户资料 ==============

/**
 * 获取当前用户资料
 * @api POST /api/v1/user/me/profile
 */
export async function getMyProfile(): Promise<ApiResponse<MyCard>> {
  return apiRequest<MyCard>(ENDPOINTS.USER.ME.PROFILE, {});
}

interface UpdateProfileParams {
  nickname?: string;
  bio?: string;
  avatar_file_id?: string;
  tags?: string[];
  gender?: Gender;
}

/**
 * 更新用户资料
 * @api POST /api/v1/user/me/profile/update
 */
export async function updateProfile(
  params: UpdateProfileParams
): Promise<ApiResponse<MyCard>> {
  return apiRequest<MyCard>(ENDPOINTS.USER.ME.PROFILE_UPDATE, params);
}

interface UpdateLocationParams {
  location: {
    latitude: number;
    longitude: number;
    location_name: string;
  };
}

/**
 * 更新用户位置
 * @api POST /api/v1/user/me/location
 */
export async function updateLocation(
  params: UpdateLocationParams
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.USER.ME.LOCATION, params);
}

/**
 * 获取当前用户统计
 * @api POST /api/v1/user/me/stats
 */
export async function getMyStats(): Promise<ApiResponse<UserStats>> {
  return apiRequest<UserStats>(ENDPOINTS.USER.ME.STATS, {});
}

/**
 * 获取当前用户设置
 * @api POST /api/v1/user/me/settings
 */
export async function getMySettings(): Promise<ApiResponse<UserSettings>> {
  return apiRequest<UserSettings>(ENDPOINTS.USER.ME.SETTINGS, {});
}

/**
 * 更新用户设置
 * @api POST /api/v1/user/me/settings/update
 */
export async function updateSettings(
  params: Partial<UserSettings>
): Promise<ApiResponse<UserSettings>> {
  return apiRequest<UserSettings>(ENDPOINTS.USER.ME.SETTINGS_UPDATE, params);
}

// ============== 关注操作 ==============

interface FollowParams {
  user_id: number;
  source?: string;
  source_moment_id?: string;
}

/**
 * 关注用户
 * @api POST /api/v1/user/me/follow
 */
export async function followUser(
  params: FollowParams
): Promise<ApiResponse<void>> {
  return apiRequestIdempotent<void>(ENDPOINTS.USER.ME.FOLLOW, params);
}

/**
 * 取消关注
 * @api POST /api/v1/user/me/unfollow
 */
export async function unfollowUser(
  userId: number
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.USER.ME.UNFOLLOW, {
    user_id: userId,
  });
}

interface BlockParams {
  user_id: number;
  reason?: string;
}

/**
 * 拉黑用户
 * @api POST /api/v1/user/me/block
 */
export async function blockUser(
  params: BlockParams
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.USER.ME.BLOCK, params);
}

/**
 * 解除拉黑
 * @api POST /api/v1/user/me/unblock
 */
export async function unblockUser(
  userId: number
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.USER.ME.UNBLOCK, {
    user_id: userId,
  });
}

// ============== 当前用户列表 ==============

interface MyListParams {
  limit?: number;
  cursor?: string;
}

// 好友项
interface FriendItem {
  user: UserInfo;
  is_online?: boolean;
  remark?: string;
  followed_at: string;
  followed_back_at: string;
}

// 关注项
interface FollowingItem {
  user: UserInfo;
  followed_at: string;
  is_mutual: boolean;
  is_online?: boolean;
}

// 粉丝项
interface FollowerItem {
  user: UserInfo;
  followed_at: string;
  is_following_back: boolean;
  is_online?: boolean;
}

// 黑名单项
interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason?: string;
}

/**
 * 获取我的好友列表
 * @api POST /api/v1/user/me/friends
 */
export async function getMyFriends(
  params: MyListParams = {}
): Promise<ApiResponse<PaginatedData<FriendItem>>> {
  return apiRequest<PaginatedData<FriendItem>>(
    ENDPOINTS.USER.ME.FRIENDS,
    params
  );
}

/**
 * 获取我的关注列表
 * @api POST /api/v1/user/me/following
 */
export async function getMyFollowing(
  params: MyListParams = {}
): Promise<ApiResponse<PaginatedData<FollowingItem>>> {
  return apiRequest<PaginatedData<FollowingItem>>(
    ENDPOINTS.USER.ME.FOLLOWING,
    params
  );
}

/**
 * 获取我的粉丝列表
 * @api POST /api/v1/user/me/followers
 */
export async function getMyFollowers(
  params: MyListParams = {}
): Promise<ApiResponse<PaginatedData<FollowerItem>>> {
  return apiRequest<PaginatedData<FollowerItem>>(
    ENDPOINTS.USER.ME.FOLLOWERS,
    params
  );
}

/**
 * 获取我的黑名单
 * @api POST /api/v1/user/me/blocked
 */
export async function getMyBlocked(
  params: MyListParams = {}
): Promise<ApiResponse<PaginatedData<BlockedUserItem>>> {
  return apiRequest<PaginatedData<BlockedUserItem>>(
    ENDPOINTS.USER.ME.BLOCKED,
    params
  );
}

// ============== 关注请求 ==============

interface FollowRequestItem {
  request_id: string;
  requester: UserInfo;
  message?: string;
  requested_at: string;
}

/**
 * 获取待审批关注请求
 * @api POST /api/v1/user/me/follow_requests
 */
export async function getFollowRequests(
  params: MyListParams = {}
): Promise<ApiResponse<PaginatedData<FollowRequestItem>>> {
  return apiRequest<PaginatedData<FollowRequestItem>>(
    ENDPOINTS.USER.ME.FOLLOW_REQUESTS,
    params
  );
}

interface HandleFollowRequestParams {
  request_id: string;
  action: 'accept' | 'reject';
}

/**
 * 处理关注请求
 * @api POST /api/v1/user/me/follow_requests/handle
 */
export async function handleFollowRequest(
  params: HandleFollowRequestParams
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.USER.ME.FOLLOW_REQUESTS_HANDLE, params);
}

// ============== 好友备注 ==============

interface SetRemarkParams {
  user_id: number;
  remark?: string;
}

/**
 * 设置好友备注
 * @api POST /api/v1/user/me/remark
 */
export async function setRemark(
  params: SetRemarkParams
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.USER.ME.REMARK, params);
}

// ============== 访客升级 ==============

interface UpgradeAccountParams {
  login_id: string;
  password: string;
}

/**
 * 访客升级为正式用户
 * @api POST /api/v1/user/me/upgrade
 */
export async function upgradeAccount(
  params: UpgradeAccountParams
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.USER.ME.UPGRADE, params);
}

// ============== 账户删除 ==============

interface DeleteAccountParams {
  confirmation: string;
}

/**
 * 删除账户
 * @api POST /api/v1/user/me/delete
 */
export async function deleteAccount(
  params: DeleteAccountParams
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.USER.ME.DELETE, params);
}
