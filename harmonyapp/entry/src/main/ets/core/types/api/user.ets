// ==================== 用户API类型 ====================
import { FollowSource, GeoPoint, Location, PaginationParams, PrivacyLevel, UserPresenceStatus, UserType } from './common';
import { UserInfo, UserItem, UserStats } from './entities';


export type Gender = 'male' | 'female' | 'other';
export type ChatPerm = 'everyone' | 'friends' | 'nobody';
// 10200 失败时的权限拒绝原因（spec api_spec_errors.md → ForbiddenReason 权威定义，12 项）
export type ForbiddenReason =
  | 'not_admin'
  | 'insufficient_admin_permission'
  | 'admin_protected_target'
  | 'not_group_member'
  | 'not_group_admin'
  | 'not_group_owner'
  | 'not_resource_owner'
  | 'privacy_setting'
  | 'blocked_by_user'
  | 'stranger_quota_exceeded'
  | 'banned_in_group';
export type ApplicationStatus = 'pending' | 'accepted' | 'rejected' | 'revoked';

export interface SearchUsersRequest {
  keyword?: string;
  tag?: string;
  radius?: number;
  sort_by?: 'relevance' | 'distance' | 'newest';
  reference_point?: GeoPoint;
  pagination?: PaginationParams;
}

export interface OpenUserDoorRequest {
  user_id: string;
  include_recent_moments?: boolean; // true 时返回主页首屏聚合（spec user.md → open_the_door）
}

// UserFriendsRequest / UserFollowingRequest / UserFollowersRequest 已合并入
// FetchFriendsRequest / FetchFollowingRequest / FetchFollowersRequest（user_id 可选）

export interface FetchUserGroupsRequest {
  user_id: string;
  pagination?: PaginationParams;
}

export interface FetchUserMomentsRequest {
  user_id: string;
  pagination?: PaginationParams;
}

export interface FetchUserReactedMomentsRequest {
  user_id: string;
  pagination?: PaginationParams;
}

// My Profile

export interface AccountInfo {
  user_id: string;
  nickname: string;
  avatar_url?: string;
  user_type: UserType;
  bio?: string;
  tags?: string[];
  gender?: Gender;
  email?: string;
  phone?: string;
  location: Location;
  presence_status: UserPresenceStatus;
  last_online_at: string;
  created_at: string;
  stats: UserStats;
}

export interface UpdateAccountRequest {
  nickname?: string;
  avatar_file_id?: string;
  bio?: string;
  tags?: string[];
  gender?: Gender;
}

export interface UpdateLocationRequest {
  location: Location;
}

export interface UpgradeAccountRequest {
  login_id: string;
  password: string;
}

export interface UserSettings {
  privacy_level: PrivacyLevel;
  location_visible: boolean;
  presence_visible: boolean;
  social_lists_visible: boolean;
  groups_visible: boolean;
  reacted_moments_visible: boolean;
  stats_visible: boolean;
  allow_follow: boolean;
  allow_private_chat: ChatPerm;
  searchable: boolean;
  phone_searchable: boolean;
  nickname_searchable: boolean;
  location_searchable: boolean;
  nearby_moment_notify: boolean;
}

export interface UpdateSettingsRequest {
  settings: Partial<UserSettings>;
}

export interface DeleteAccountRequest {
  confirmation: string;
}

export interface FollowRequest {
  user_id: string;
  active?: boolean;
  source?: FollowSource;
  source_moment_id?: string;
}

export interface BlockRequest {
  user_id: string;
  active?: boolean;
  reason?: string;
}

export interface FetchFriendsRequest {
  user_id?: string;
  pagination?: PaginationParams;
}

export interface FetchFollowingRequest {
  user_id?: string;
  pagination?: PaginationParams;
}

export interface FetchFollowersRequest {
  user_id?: string;
  pagination?: PaginationParams;
}

export interface FetchBlockedRequest {
  pagination?: PaginationParams;
}

export interface SetRemarkRequest {
  user_id: string;
  remark: string;
}

// 用户关系列表项（list_friends / list_following / list_followers 共用，spec user.md → RelatedUserItem）
// user 为 UserItem（presence 隐私受控字段随 user 返回）；followed_back_at 仅 list_friends
export interface RelatedUserItem {
  user: UserItem;
  followed_at: string;
  followed_back_at: string | null; // 对方回关时间（成为好友的时间）；仅 list_friends 返回
  remark: string | null; // @owner-only：仅查询自己时返回
}

export interface BlockedUserItem {
  user: UserInfo;
  blocked_at: string;
  reason: string | null;
}
