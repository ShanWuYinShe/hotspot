// ==================== 生活瞬间API类型 ====================
import { GeoPoint, Location, PaginationParams, PrivacyLevel } from './common';


// 全球 keyword/tag 搜索（spec moment.md → moments/search；附近发现走 moments/fetch_nearby，feed 域已废除）
export interface SearchMomentsRequest {
  keyword?: string;
  tag?: string;
  sort_by?: 'relevance' | 'newest' | 'popular';
  pagination?: PaginationParams;
}

export interface OpenMomentDoorRequest {
  moment_id: string;
}

export interface CreateMomentRequest {
  description: string;
  location: Location;
  privacy_level?: PrivacyLevel;
  tags?: string[];
  image_file_ids?: string[];
  mentions?: string[];
}

export interface EditMomentRequest {
  moment_id: string;
  description?: string;
  location?: Location;
  privacy_level?: PrivacyLevel;
  tags?: string[];
  image_file_ids?: string[];
  mentions?: string[];
}

export interface DeleteMomentRequest {
  moment_id: string;
}

export interface FavoriteMomentRequest {
  moment_id: string;
  active?: boolean;
}

export interface FetchCreatedMomentsRequest {
  pagination?: PaginationParams;
}

// 关注动态：我关注的人发布的瞬间（spec moment.md → moments/fetch_following，原 feed/following 迁移）
export interface FetchFollowingMomentsRequest {
  reference_point?: GeoPoint;
  pagination?: PaginationParams;
}

// 附近瞬间发现流（spec moment.md → moments/fetch_nearby，原 feed/nearby_moments 迁移）
export interface FetchNearbyMomentsRequest {
  radius?: number;
  reference_point?: GeoPoint;
  pagination?: PaginationParams;
}

export interface FetchFavoritesRequest {
  pagination?: PaginationParams;
}

// ==================== 评论API类型 ====================

export interface CreateCommentRequest {
  moment_id: string;
  content: string;
  parent_id?: string;
  mentions?: string[];
}

export interface ListCommentRepliesRequest {
  comment_id: string;
  pagination: PaginationParams | null;
}

export interface DeleteCommentRequest {
  comment_id: string;
}

export interface FetchCommentsRequest {
  moment_id: string;
  pagination?: PaginationParams;
}

export interface FetchCommentRepliesRequest {
  comment_id: string;
  pagination?: PaginationParams;
}

// ==================== Reactions API类型 ====================

export type ReactTarget = 'moment' | 'comment' | 'message';

export interface SetReactionRequest {
  target_type: ReactTarget;
  target_id: string;
  emoji: string;
  active: boolean;
}

export interface FetchReactionsRequest {
  target_type: string;
  target_id: string;
  emoji?: string;
  pagination?: PaginationParams;
}
