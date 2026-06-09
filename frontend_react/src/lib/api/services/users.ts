import {
  post,
  paginate,
  generateIdempotencyKey,
  type RequestOptions,
} from "../client";
import { USERS } from "../endpoints";
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
  VerificationSent,
} from "@/lib/types/api";

export const usersService = {
  search(
    params: {
      keyword?: string;
      tag?: string;
      radius?: number;
      sort_by?: "relevance" | "distance" | "newest";
      reference_point: GeoPoint;
      pagination?: PaginationParams;
    },
    options?: RequestOptions,
  ) {
    return paginate<UserItem>(USERS.SEARCH, params, options);
  },

  getUser(data: { user_id: string }) {
    return post<UserItem>(USERS.OPEN_THE_DOOR, data);
  },

  getFriends(params: { user_id?: string; pagination?: PaginationParams }) {
    return paginate<FriendItem>(USERS.LIST_FRIENDS, params);
  },

  getFollowing(params: { user_id?: string; pagination?: PaginationParams }) {
    return paginate<FollowingItem>(USERS.LIST_FOLLOWING, params);
  },

  getFollowers(params: { user_id?: string; pagination?: PaginationParams }) {
    return paginate<FollowerItem>(USERS.LIST_FOLLOWERS, params);
  },

  getGroups(params: { user_id: string; pagination?: PaginationParams }) {
    return paginate<GroupItem>(USERS.LIST_GROUPS, params);
  },

  getMoments(params: { user_id: string; pagination?: PaginationParams }) {
    return paginate<MomentItem>(USERS.LIST_MOMENTS, params);
  },

  getLikedMoments(params: { user_id: string; pagination?: PaginationParams }) {
    return paginate<MomentItem>(USERS.LIST_LIKED_MOMENTS, params);
  },

  getMyProfile() {
    return post<MyProfile>(USERS.ME.PROFILE_GET);
  },

  updateMyProfile(data: {
    nickname?: string;
    avatar_file_id?: string;
    bio?: string;
    tags?: string[];
    gender?: Gender;
  }) {
    return post<void>(USERS.ME.PROFILE_UPDATE, data);
  },

  updateLocation(data: { location: Location }) {
    return post<void>(USERS.ME.LOCATION_UPDATE, data);
  },

  upgradeAccount(data: { login_id: string; password: string }) {
    return post<void>(USERS.ME.UPGRADE, data);
  },

  getMySettings() {
    return post<UserSettings>(USERS.ME.SETTINGS_GET);
  },

  updateMySettings(data: Partial<UserSettings>) {
    return post<void>(USERS.ME.SETTINGS_UPDATE, { settings: data });
  },

  deleteAccount(data: { confirmation: string }) {
    return post<void>(USERS.ME.DELETE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  getBlocked(params?: { pagination?: PaginationParams }) {
    return paginate<BlockedUserItem>(USERS.ME.LIST_BLOCKED, params);
  },

  follow(data: {
    user_id: string;
    source?: FollowSource;
    source_moment_id?: string;
  }) {
    return post<void>(USERS.ME.FOLLOW, data);
  },

  unfollow(data: { user_id: string }) {
    return post<void>(USERS.ME.UNFOLLOW, data);
  },

  block(data: { user_id: string; reason?: string }) {
    return post<void>(USERS.ME.BLOCK, data);
  },

  unblock(data: { user_id: string }) {
    return post<void>(USERS.ME.UNBLOCK, data);
  },

  setRemark(data: { user_id: string; remark: string }) {
    return post<void>(USERS.ME.SET_REMARK, data);
  },

  requestEmailChange(data: { new_email: string; password: string }) {
    return post<VerificationSent>(USERS.ME.EMAIL_REQUEST_CHANGE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  confirmEmailChange(data: { new_email: string; verification_code: string }) {
    return post<void>(USERS.ME.EMAIL_CONFIRM_CHANGE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  requestPhoneChange(data: { new_phone: string; password: string }) {
    return post<VerificationSent>(USERS.ME.PHONE_REQUEST_CHANGE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  confirmPhoneChange(data: { new_phone: string; verification_code: string }) {
    return post<void>(USERS.ME.PHONE_CONFIRM_CHANGE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },
};
