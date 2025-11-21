/**
 * 用户服务 - 符合API规范的实现
 * 所有API操作统一使用POST方法
 * 基于 api_spec.mdc 第3.2节 用户API规范
 */

import { BaseService } from './base.service'
import type {
  UserResponse,
  UserSettings,
  PaginatedResponse,
  DiscoverUsersRequest,
  UpdateProfileRequest,
  UpdateLocationRequest,
  UpdateSettingsRequest,
  DeleteAccountRequest,
  SendFriendRequestRequest,
  AcceptFriendRequestRequest,
  RejectFriendRequestRequest,
  RemoveFriendRequest,
  FollowUserRequest,
  UnfollowUserRequest,
  BlockUserRequest,
  UnblockUserRequest,
  ListFriendsRequest,
  GetBlockedUsersRequest,
  OpenTheDoorRequest,
  GetStatsRequest
} from '~/types/api'

// ✅ 所有类型已从 ~/types/api 导入，移除本地定义

export class UserService extends BaseService {
  protected readonly baseUrl = '/api/v1'

  /**
   * 发现用户（公开接口）
   * POST /api/v1/public/user/discover
   */
  async discoverUsers(request: DiscoverUsersRequest): Promise<PaginatedResponse<UserResponse>> {
    return this.publicPost<PaginatedResponse<UserResponse>>('/public/user/discover', request)
  }

  /**
   * 获取当前用户资料
   * POST /api/v1/protected/user/me
   */
  async getMe(): Promise<UserResponse> {
    return this.post<UserResponse>('/protected/user/me', {})
  }

  /**
   * 获取其他用户资料
   * POST /api/v1/protected/user/open_the_door
   */
  async openTheDoor(request: OpenTheDoorRequest): Promise<UserResponse> {
    return this.post<UserResponse>('/protected/user/open_the_door', request)
  }

  /**
   * 获取统计数据
   * POST /api/v1/protected/user/get_stats
   */
  async getStats(request: GetStatsRequest = {}): Promise<{
    moments_count: number
    friends_count: number
    followers_count: number
    following_count: number
    groups_count: number
  }> {
    return this.post('/protected/user/get_stats', request)
  }

  /**
   * 更新用户位置
   * POST /api/v1/protected/user/update_location
   */
  async updateLocation(request: UpdateLocationRequest): Promise<void> {
    return this.post<void>('/protected/user/update_location', request)
  }

  /**
   * 更新用户资料
   * POST /api/v1/protected/user/update_profile
   */
  async updateProfile(request: UpdateProfileRequest): Promise<void> {
    return this.post<void>('/protected/user/update_profile', request)
  }

  /**
   * 获取用户设置
   * POST /api/v1/protected/user/get_settings
   */
  async getSettings(): Promise<UserSettings> {
    return this.post<UserSettings>('/protected/user/get_settings', {})
  }

  /**
   * 更新用户设置
   * POST /api/v1/protected/user/update_settings
   */
  async updateSettings(request: UpdateSettingsRequest): Promise<void> {
    return this.post<void>('/protected/user/update_settings', request)
  }

  /**
   * 删除账户
   * POST /api/v1/protected/user/delete_account
   */
  async deleteAccount(request: DeleteAccountRequest): Promise<void> {
    return this.post<void>('/protected/user/delete_account', request)
  }

  /**
   * 发送好友申请
   * POST /api/v1/protected/user/send_friend_request
   */
  async sendFriendRequest(request: SendFriendRequestRequest): Promise<{ id: string }> {
    return this.post<{ id: string }>('/protected/user/send_friend_request', request)
  }

  /**
   * 接受好友申请
   * POST /api/v1/protected/user/accept_friend_request
   */
  async acceptFriendRequest(request: AcceptFriendRequestRequest): Promise<void> {
    return this.post<void>('/protected/user/accept_friend_request', request)
  }

  /**
   * 拒绝好友申请
   * POST /api/v1/protected/user/reject_friend_request
   */
  async rejectFriendRequest(request: RejectFriendRequestRequest): Promise<void> {
    return this.post<void>('/protected/user/reject_friend_request', request)
  }

  /**
   * 删除好友
   * POST /api/v1/protected/user/remove_friend
   */
  async removeFriend(request: RemoveFriendRequest): Promise<void> {
    return this.post<void>('/protected/user/remove_friend', request)
  }

  /**
   * 关注用户
   * POST /api/v1/protected/user/follow
   */
  async followUser(request: FollowUserRequest): Promise<void> {
    return this.post<void>('/protected/user/follow', request)
  }

  /**
   * 取消关注
   * POST /api/v1/protected/user/unfollow
   */
  async unfollowUser(request: UnfollowUserRequest): Promise<void> {
    return this.post<void>('/protected/user/unfollow', request)
  }

  /**
   * 拉黑用户
   * POST /api/v1/protected/user/block_user
   */
  async blockUser(request: BlockUserRequest): Promise<void> {
    return this.post<void>('/protected/user/block_user', request)
  }

  /**
   * 解除拉黑
   * POST /api/v1/protected/user/unblock_user
   */
  async unblockUser(request: UnblockUserRequest): Promise<void> {
    return this.post<void>('/protected/user/unblock_user', request)
  }

  /**
   * 获取好友列表
   * POST /api/v1/protected/user/get_friends
   */
  async getFriends(request: ListFriendsRequest = {}): Promise<PaginatedResponse<UserResponse>> {
    return this.post<PaginatedResponse<UserResponse>>('/protected/user/get_friends', request)
  }

  /**
   * 获取拉黑列表
   * POST /api/v1/protected/user/get_blocked_users
   */
  async getBlockedUsers(request: GetBlockedUsersRequest = {}): Promise<PaginatedResponse<UserResponse>> {
    return this.post<PaginatedResponse<UserResponse>>('/protected/user/get_blocked_users', request)
  }

  /**
   * 上传头像（保留，使用FormData）
   */
  async uploadAvatar(file: File): Promise<{ avatar_url: string }> {
    return this.upload<{ avatar_url: string }>('/protected/media/upload_file', file, {
      category: 'avatar'
    })
  }

  /**
   * 获取附近用户（别名方法，调用 discoverUsers）
   */
  async getNearbyUsers(request: DiscoverUsersRequest): Promise<PaginatedResponse<UserResponse>> {
    return this.discoverUsers(request)
  }

  /**
   * 搜索用户（别名方法，调用 discoverUsers）
   */
  async searchUsers(request: DiscoverUsersRequest): Promise<PaginatedResponse<UserResponse>> {
    return this.discoverUsers(request)
  }
}

// 创建服务实例
export const userService = new UserService()

// 提供 composable 接口
export function useUsersService() {
  return userService
}

// 兼容旧的导出名称
export function useUserService() {
  return userService
}
