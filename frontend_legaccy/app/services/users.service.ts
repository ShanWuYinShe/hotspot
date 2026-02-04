import { BaseService } from './base.service'
import { USER } from '~/config/api'
import type {
  AuthData,
  UserSettingsData,
  MyProfileData,
  UserStatsData,
  UserInfo,
  PaginatedData,
  UserItem,
  UserDetailData,
  UserRelation,
  SearchUsersByKeywordRequest,
  SearchUsersByTagRequest,
  SearchUsersByLocationRequest,
  UserOpenTheDoorRequest,
  GetRelationshipRequest,
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
  // ============ 搜索 API ============

  async searchByKeyword(request: SearchUsersByKeywordRequest) {
    return this.post<PaginatedData<UserItem>>(USER.SEARCH_BY_KEYWORD, request)
  }

  async searchByTag(request: SearchUsersByTagRequest) {
    return this.post<PaginatedData<UserItem>>(USER.SEARCH_BY_TAG, request)
  }

  async searchByLocation(request: SearchUsersByLocationRequest) {
    return this.post<PaginatedData<UserItem>>(USER.SEARCH_BY_LOCATION, request)
  }

  // 获取用户资料（通过 ID 获取指定用户的资料，包含在线状态）
  async openTheDoor(request: UserOpenTheDoorRequest) {
    return this.post<UserDetailData>(USER.OPEN_THE_DOOR, request)
  }

  // ============ 指定用户 API ============

  async getRelationship(request: GetRelationshipRequest) {
    return this.post<UserRelation>(USER.RELATIONSHIP, request)
  }

  // 获取指定用户统计数据
  async getStats(request: UserStatsRequest) {
    return this.post<UserStatsData>(USER.STATS, request)
  }

  // 获取指定用户好友列表
  async getUserFriendsList(request: UserFriendsRequest) {
    return this.post<PaginatedData<UserInfo>>(USER.FRIENDS, request)
  }

  // 获取指定用户关注列表
  async getUserFollowingList(request: UserFollowingRequest) {
    return this.post<PaginatedData<UserInfo>>(USER.FOLLOWING, request)
  }

  // 获取指定用户粉丝列表
  async getUserFollowersList(request: UserFollowersRequest) {
    return this.post<PaginatedData<UserInfo>>(USER.FOLLOWERS, request)
  }

  // ============ 当前用户 API（/me/ 路径）============

  async getMe() {
    return this.post<MyProfileData>(USER.ME.PROFILE, {})
  }

  // 获取当前用户统计数据
  async getMyStats() {
    return this.post<UserStatsData>(USER.ME.STATS, {})
  }

  async updateLocation(request: UpdateLocationRequest) {
    return this.post<void>(USER.ME.LOCATION, request)
  }

  async updateProfile(request: UpdateProfileRequest) {
    return this.post<MyProfileData>(USER.ME.PROFILE_UPDATE, request)
  }

  async upgradeAccount(request: UpgradeAccountRequest) {
    return this.post<AuthData>(USER.ME.UPGRADE, request)
  }

  async getSettings() {
    return this.post<UserSettingsData>(USER.ME.SETTINGS, {})
  }

  async updateSettings(request: UpdateSettingsRequest) {
    return this.post<void>(USER.ME.SETTINGS_UPDATE, request)
  }

  async deleteAccount(request: DeleteAccountRequest) {
    return this.post<void>(USER.ME.DELETE, request)
  }

  // ============ 关注关系 API ============

  async follow(request: FollowRequest) {
    return this.post<UserRelation>(USER.ME.FOLLOW, request)
  }

  async unfollow(request: UnfollowRequest) {
    return this.post<UserRelation>(USER.ME.UNFOLLOW, request)
  }

  async block(request: BlockRequest) {
    return this.post<UserRelation>(USER.ME.BLOCK, request)
  }

  async unblock(request: UnblockRequest) {
    return this.post<UserRelation>(USER.ME.UNBLOCK, request)
  }

  async getFriends(request: MyFriendsRequest = {}) {
    return this.post<PaginatedData<FriendListItem>>(USER.ME.FRIENDS, request)
  }

  async getFollowing(request: MyFollowingRequest = {}) {
    return this.post<PaginatedData<FollowingItem>>(USER.ME.FOLLOWING, request)
  }

  async getFollowers(request: MyFollowersRequest = {}) {
    return this.post<PaginatedData<FollowerItem>>(USER.ME.FOLLOWERS, request)
  }

  async getBlockedUsers(request: MyBlockedRequest = {}) {
    return this.post<PaginatedData<BlockedUserItem>>(USER.ME.BLOCKED, request)
  }

  async setRemark(request: SetRemarkRequest) {
    return this.post<void>(USER.ME.REMARK, request)
  }
}

export const userService = new UserService()
export function useUsersService() { return userService }
