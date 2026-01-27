import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import {
  BlockedUserItem,
  FollowRequestItem,
  FollowSource,
  FollowerItem,
  FollowingItem,
  FriendItem,
  MyCard,
  PaginatedData,
  Relationship,
  UserCard,
  UserInfo,
  UserSearchItem,
  UserSettings,
  UserStats,
  UserMomentListItem,
  MomentSearchItem,
  GroupSearchItem,
} from '@/types/api';

export const UserService = {
  searchByKeyword: (params: { kw: string; cursor?: string; limit?: number; reference_point?: { latitude: number; longitude: number } }) =>
    apiClient.post<PaginatedData<UserSearchItem>>(ENDPOINTS.USER.SEARCH_BY_KEYWORD, params),

  searchByTag: (params: { tag: string; cursor?: string; limit?: number; reference_point?: { latitude: number; longitude: number } }) =>
    apiClient.post<PaginatedData<UserSearchItem>>(ENDPOINTS.USER.SEARCH_BY_TAG, params),

  searchByLocation: (params: { radius: number; kw?: string; cursor?: string; limit?: number; reference_point?: { latitude: number; longitude: number } }) =>
    apiClient.post<PaginatedData<UserSearchItem>>(ENDPOINTS.USER.SEARCH_BY_LOCATION, params),

  openTheDoor: (userId: number) =>
    apiClient.post<UserCard>(ENDPOINTS.USER.OPEN_THE_DOOR, { user_id: userId }),

  relationship: (userId: number) =>
    apiClient.post<Relationship>(ENDPOINTS.USER.RELATIONSHIP, { user_id: userId }),

  stats: (userId: number) =>
    apiClient.post<UserStats>(ENDPOINTS.USER.STATS, { user_id: userId }),

  friends: (userId: number, params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<UserInfo>>(ENDPOINTS.USER.FRIENDS, { user_id: userId, ...params }),

  following: (userId: number, params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<UserInfo>>(ENDPOINTS.USER.FOLLOWING, { user_id: userId, ...params }),

  followers: (userId: number, params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<UserInfo>>(ENDPOINTS.USER.FOLLOWERS, { user_id: userId, ...params }),

  groups: (userId: number, params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<GroupSearchItem>>(ENDPOINTS.USER.GROUPS, { user_id: userId, ...params }),

  likedMoments: (userId: number, params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<MomentSearchItem>>(ENDPOINTS.USER.LIKED_MOMENTS, { user_id: userId, ...params }),

  getProfile: () => apiClient.post<MyCard>(ENDPOINTS.USER.ME_PROFILE),

  updateProfile: (data: Partial<MyCard>) =>
    apiClient.post<MyCard>(ENDPOINTS.USER.ME_PROFILE_UPDATE, data),

  updateLocation: (location: { latitude: number; longitude: number; location_name: string }) =>
    apiClient.post(ENDPOINTS.USER.ME_LOCATION, { location }),

  upgrade: (data: { password: string; verify_token?: string }) =>
    apiClient.post(ENDPOINTS.USER.ME_UPGRADE, data),

  myStats: () => apiClient.post<UserStats>(ENDPOINTS.USER.ME_STATS),

  mySettings: () => apiClient.post<UserSettings>(ENDPOINTS.USER.ME_SETTINGS),

  updateMySettings: (data: Partial<UserSettings>) =>
    apiClient.post<UserSettings>(ENDPOINTS.USER.ME_SETTINGS_UPDATE, data),

  deleteAccount: (data: { password: string }) => apiClient.post(ENDPOINTS.USER.ME_DELETE, data),

  follow: (data: { user_id: number; source?: FollowSource; source_moment_id?: string }) =>
    apiClient.post(ENDPOINTS.USER.ME_FOLLOW, data),

  unfollow: (userId: number) => apiClient.post(ENDPOINTS.USER.ME_UNFOLLOW, { user_id: userId }),

  block: (userId: number) => apiClient.post(ENDPOINTS.USER.ME_BLOCK, { user_id: userId }),

  unblock: (userId: number) => apiClient.post(ENDPOINTS.USER.ME_UNBLOCK, { user_id: userId }),

  myFriends: (params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<FriendItem>>(ENDPOINTS.USER.ME_FRIENDS, params ?? {}),

  myFollowing: (params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<FollowingItem>>(ENDPOINTS.USER.ME_FOLLOWING, params ?? {}),

  myFollowers: (params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<FollowerItem>>(ENDPOINTS.USER.ME_FOLLOWERS, params ?? {}),

  myFollowRequests: (params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<FollowRequestItem>>(ENDPOINTS.USER.ME_FOLLOW_REQUESTS, params ?? {}),

  handleFollowRequest: (data: { request_id: string; action: 'accept' | 'reject' }) =>
    apiClient.post(ENDPOINTS.USER.ME_HANDLE_FOLLOW_REQUEST, data),

  myBlocked: (params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<BlockedUserItem>>(ENDPOINTS.USER.ME_BLOCKED, params ?? {}),

  remark: (data: { user_id: number; remark: string }) =>
    apiClient.post(ENDPOINTS.USER.ME_REMARK, data),

  userMoments: (userId: number, params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<UserMomentListItem>>(ENDPOINTS.MOMENTS.USER_MOMENTS, {
      user_id: userId,
      ...params,
    }),
};
