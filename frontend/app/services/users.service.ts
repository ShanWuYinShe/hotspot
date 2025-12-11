import { BaseService } from './base.service'
import { USER, MEDIA } from '~/config/api'
import type {
  UserSettingsData,
  MyProfileData,
  UserStatsData,
  UserInfo,
  PaginatedData,
  UserSearchItem,
  UserRelationship,
  SearchUsersByKeywordRequest,
  SearchUsersByTagRequest,
  SearchUsersByLocationRequest,
  UserOpenTheDoorRequest,
  GetRelationshipRequest,
  UpdateProfileRequest,
  UpdateLocationRequest,
  UpdateSettingsRequest,
  DeleteAccountRequest,
  FollowUserRequest,
  UnfollowUserRequest,
  BlockUserRequest,
  UnblockUserRequest,
  GetFriendsRequest,
  GetFollowingRequest,
  GetFollowersRequest,
  GetBlockedUsersRequest,
  SetFriendRemarkRequest,
  GetUserFriendsListRequest,
  GetUserFollowingListRequest,
  GetUserFollowersListRequest,
  GetUserStatsRequest,
  FriendListItem,
  FollowerItem,
  FollowingItem,
  BlockedUserItem,
  UpgradeAccountRequest
} from '~/types/api'

class UserService extends BaseService {
  // ============ 搜索 API (public) ============

  async searchByKeyword(request: SearchUsersByKeywordRequest) {
    return this.publicPost<PaginatedData<UserSearchItem>>(USER.SEARCH_BY_KEYWORD, request)
  }

  async searchByTag(request: SearchUsersByTagRequest) {
    return this.publicPost<PaginatedData<UserSearchItem>>(USER.SEARCH_BY_TAG, request)
  }

  async searchByLocation(request: SearchUsersByLocationRequest) {
    return this.publicPost<PaginatedData<UserSearchItem>>(USER.SEARCH_BY_LOCATION, request)
  }

  // 获取用户资料（通过 ID 获取指定用户的资料，包含在线状态）
  async openTheDoor(request: UserOpenTheDoorRequest) {
    return this.publicPost<UserSearchItem>(USER.OPEN_THE_DOOR, request)
  }

  // ============ 当前用户 API (protected) ============

  async getMe() {
    return this.post<MyProfileData>(USER.ME, {})
  }

  async getRelationship(request: GetRelationshipRequest) {
    return this.post<UserRelationship>(USER.GET_RELATIONSHIP, request)
  }

  // 获取指定用户统计数据（公开 API）
  async getStats(request: GetUserStatsRequest) {
    return this.post<UserStatsData>(USER.GET_STATS, request)
  }

  // 获取当前用户统计数据（/me/ API）
  async getMyStats() {
    return this.post<UserStatsData>(USER.GET_MY_STATS, {})
  }

  // 获取指定用户好友列表（公开 API）
  async getUserFriendsList(request: GetUserFriendsListRequest) {
    return this.publicPost<PaginatedData<UserInfo>>(USER.GET_USER_FRIENDS_LIST, request)
  }

  // 获取指定用户关注列表（公开 API）
  async getUserFollowingList(request: GetUserFollowingListRequest) {
    return this.publicPost<PaginatedData<UserInfo>>(USER.GET_USER_FOLLOWING_LIST, request)
  }

  // 获取指定用户粉丝列表（公开 API）
  async getUserFollowersList(request: GetUserFollowersListRequest) {
    return this.publicPost<PaginatedData<UserInfo>>(USER.GET_USER_FOLLOWERS_LIST, request)
  }

  async updateLocation(request: UpdateLocationRequest) {
    return this.post<void>(USER.UPDATE_LOCATION, request)
  }

  async updateProfile(request: UpdateProfileRequest) {
    return this.post<void>(USER.UPDATE_PROFILE, request)
  }

  async upgradeAccount(request: UpgradeAccountRequest) {
    return this.post<void>(USER.UPGRADE_ACCOUNT, request)
  }

  async getSettings() {
    return this.post<UserSettingsData>(USER.GET_SETTINGS, {})
  }

  async updateSettings(request: UpdateSettingsRequest) {
    return this.post<void>(USER.UPDATE_SETTINGS, request)
  }

  async deleteAccount(request: DeleteAccountRequest) {
    return this.post<void>(USER.DELETE_ACCOUNT, request)
  }

  // ============ 关注关系 API (protected) ============

  async followUser(request: FollowUserRequest) {
    return this.post<void>(USER.FOLLOW_USER, request)
  }

  async unfollowUser(request: UnfollowUserRequest) {
    return this.post<void>(USER.UNFOLLOW_USER, request)
  }

  async blockUser(request: BlockUserRequest) {
    return this.post<void>(USER.BLOCK_USER, request)
  }

  async unblockUser(request: UnblockUserRequest) {
    return this.post<void>(USER.UNBLOCK_USER, request)
  }

  async getFriends(request: GetFriendsRequest = {}) {
    return this.post<PaginatedData<FriendListItem>>(USER.GET_FRIENDS_LIST, request)
  }

  async getFollowing(request: GetFollowingRequest = {}) {
    return this.post<PaginatedData<FollowingItem>>(USER.GET_FOLLOWING_LIST, request)
  }

  async getFollowers(request: GetFollowersRequest = {}) {
    return this.post<PaginatedData<FollowerItem>>(USER.GET_FOLLOWERS_LIST, request)
  }

  async getBlockedUsers(request: GetBlockedUsersRequest = {}) {
    // TODO: 后端目前返回 UserInfo，未来应返回 BlockedUserItem 包含 blocked_at 等信息
    return this.post<PaginatedData<UserInfo>>(USER.GET_BLOCKED_LIST, request)
  }

  async setFriendRemark(request: SetFriendRemarkRequest) {
    return this.post<void>(USER.SET_FRIEND_REMARK, request)
  }

  // ============ 上传 API ============

  async uploadAvatar(file: File) {
    return this.upload<{ avatar_url: string }>(MEDIA.UPLOAD_FILE, file, { category: 'avatar' })
  }
}

export const userService = new UserService()
export function useUsersService() { return userService }
