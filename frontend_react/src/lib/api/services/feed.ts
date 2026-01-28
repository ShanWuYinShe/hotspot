/**
 * Feed 服务
 * 信息流相关 API
 * 
 * 基于 @api_spec_feed.mdc 规范
 */

import { apiRequest, apiRequestPublic } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type {
  ApiResponse,
  PaginatedData,
  FeedMomentItem,
  GroupSearchItem,
  UserSearchItem,
  FeedSource,
  FeedFeedbackAction,
  FeedContentType,
  GeoPoint,
} from '@/types/api';

// ============== 信息流 ==============

interface FeedMomentsParams {
  source?: FeedSource;
  reference_point?: GeoPoint;
  radius?: number;
  limit?: number;
  cursor?: string;
}

/**
 * 获取瞬间信息流
 * 需要登录
 * @api POST /api/v1/feed/moments
 */
export async function getMomentsFeed(
  params: FeedMomentsParams = {}
): Promise<ApiResponse<PaginatedData<FeedMomentItem>>> {
  return apiRequest<PaginatedData<FeedMomentItem>>(
    ENDPOINTS.FEED.MOMENTS,
    params
  );
}

/**
 * 获取热门瞬间（公开接口）
 * Token 可选，登录后可见更多内容
 * @api POST /api/v1/feed/moments (source: 'trending')
 */
export async function getTrendingMoments(
  params: { limit?: number; cursor?: string } = {}
): Promise<ApiResponse<PaginatedData<FeedMomentItem>>> {
  // 热门内容通过 source='all' 获取公开内容
  // 注意：根据规范，/feed/moments 需要登录，但可以用 apiRequestPublic 尝试获取公开内容
  return apiRequestPublic<PaginatedData<FeedMomentItem>>(
    ENDPOINTS.FEED.MOMENTS,
    { ...params, source: 'all' as FeedSource }
  );
}

interface FeedGroupsParams {
  reference_point: GeoPoint;
  radius?: number;
  limit?: number;
  cursor?: string;
}

/**
 * 获取附近群组
 * 需要登录
 * @api POST /api/v1/feed/groups
 */
export async function getGroupsFeed(
  params: FeedGroupsParams
): Promise<ApiResponse<PaginatedData<GroupSearchItem>>> {
  return apiRequest<PaginatedData<GroupSearchItem>>(
    ENDPOINTS.FEED.GROUPS,
    params
  );
}

interface FeedUsersParams {
  reference_point: GeoPoint;
  radius?: number;
  limit?: number;
  cursor?: string;
}

/**
 * 获取附近用户
 * 需要登录
 * @api POST /api/v1/feed/users
 */
export async function getUsersFeed(
  params: FeedUsersParams
): Promise<ApiResponse<PaginatedData<UserSearchItem>>> {
  return apiRequest<PaginatedData<UserSearchItem>>(
    ENDPOINTS.FEED.USERS,
    params
  );
}

interface TrendingParams {
  limit?: number;
}

interface TrendingItem {
  content_type: FeedContentType;
  content_id: string;
  score: number;
  preview: {
    title: string;
    subtitle?: string;
    cover_url?: string;
  };
}

/**
 * 获取热门内容
 * 需要登录
 * @api POST /api/v1/feed/trending
 */
export async function getTrending(
  params: TrendingParams = {}
): Promise<ApiResponse<TrendingItem[]>> {
  return apiRequest<TrendingItem[]>(
    ENDPOINTS.FEED.TRENDING,
    params
  );
}

interface FeedbackParams {
  content_type: FeedContentType;
  content_id: string;
  action: FeedFeedbackAction;
  reason?: string;
}

/**
 * 提交内容反馈
 * 需要登录
 * @api POST /api/v1/feed/feedback
 */
export async function submitFeedback(
  params: FeedbackParams
): Promise<ApiResponse<void>> {
  return apiRequest<void>(ENDPOINTS.FEED.FEEDBACK, params);
}
