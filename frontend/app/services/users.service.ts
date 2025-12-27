import { BaseService } from './base.service'
import { USER, MEDIA } from '~/config/api'
import type {
  AuthData,
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
  UserRelationshipRequest,
  UpdateProfileRequest,
  UpdateLocationRequest,
  UpdateSettingsRequest,
  DeleteAccountRequest,
  FollowRequest,
  UnfollowRequest,
  BlockRequest,
  UnblockRequest,
  MyFriendsRequest,
  MyFollowingRequest,
  MyFollowersRequest,
  MyBlockedRequest,
  SetRemarkRequest,
  UserFriendsRequest,
  UserFollowingRequest,
  UserFollowersRequest,
  UserStatsRequest,
  FriendListItem,
  FollowerItem,
  FollowingItem,
  BlockedUserItem,
  UpgradeAccountRequest,
  UploadMediaData
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
    return this.post<MyProfileData>(USER.PROFILE, {})
  }

  async getRelationship(request: UserRelationshipRequest) {
    return this.post<UserRelationship>(USER.RELATIONSHIP, request)
  }

  // 获取指定用户统计数据（公开 API）
  async getStats(request: UserStatsRequest) {
    return this.publicPost<UserStatsData>(USER.STATS, request)
  }

  // 获取当前用户统计数据（/me/ API）
  async getMyStats() {
    return this.post<UserStatsData>(USER.MY_STATS, {})
  }

  // 获取指定用户好友列表（公开 API）
  async getUserFriendsList(request: UserFriendsRequest) {
    return this.publicPost<PaginatedData<UserInfo>>(USER.FRIENDS, request)
  }

  // 获取指定用户关注列表（公开 API）
  async getUserFollowingList(request: UserFollowingRequest) {
    return this.publicPost<PaginatedData<UserInfo>>(USER.FOLLOWING, request)
  }

  // 获取指定用户粉丝列表（公开 API）
  async getUserFollowersList(request: UserFollowersRequest) {
    return this.publicPost<PaginatedData<UserInfo>>(USER.FOLLOWERS, request)
  }

  async updateLocation(request: UpdateLocationRequest) {
    return this.post<void>(USER.LOCATION, request)
  }

  async updateProfile(request: UpdateProfileRequest) {
    return this.post<void>(USER.PROFILE_UPDATE, request)
  }

  async upgradeAccount(request: UpgradeAccountRequest) {
    return this.post<AuthData>(USER.UPGRADE, request)
  }

  async getSettings() {
    return this.post<UserSettingsData>(USER.SETTINGS, {})
  }

  async updateSettings(request: UpdateSettingsRequest) {
    return this.post<void>(USER.SETTINGS_UPDATE, request)
  }

  async deleteAccount(request: DeleteAccountRequest) {
    return this.post<void>(USER.DELETE, request)
  }

  // ============ 关注关系 API (protected) ============

  async follow(request: FollowRequest) {
    return this.post<void>(USER.FOLLOW, request)
  }

  async unfollow(request: UnfollowRequest) {
    return this.post<void>(USER.UNFOLLOW, request)
  }

  async block(request: BlockRequest) {
    return this.post<void>(USER.BLOCK, request)
  }

  async unblock(request: UnblockRequest) {
    return this.post<void>(USER.UNBLOCK, request)
  }

  async getFriends(request: MyFriendsRequest = {}) {
    return this.post<PaginatedData<FriendListItem>>(USER.MY_FRIENDS, request)
  }

  async getFollowing(request: MyFollowingRequest = {}) {
    return this.post<PaginatedData<FollowingItem>>(USER.MY_FOLLOWING, request)
  }

  async getFollowers(request: MyFollowersRequest = {}) {
    return this.post<PaginatedData<FollowerItem>>(USER.MY_FOLLOWERS, request)
  }

  async getBlockedUsers(request: MyBlockedRequest = {}) {
    return this.post<PaginatedData<BlockedUserItem>>(USER.MY_BLOCKED, request)
  }

  async setRemark(request: SetRemarkRequest) {
    return this.post<void>(USER.REMARK, request)
  }

  // ============ 上传 API ============

  async uploadAvatar(file: File) {
    return this.upload<UploadMediaData>(MEDIA.UPLOAD, file, { category: 'avatar' })
  }
}

export const userService = new UserService()
export function useUsersService() { return userService }
