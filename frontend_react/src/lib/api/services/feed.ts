/**
 * 信息流服务
 */

import { get, post, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  MomentItem,
  FeedGroupItem,
  FeedUserItem,
  FeedTrendingData,
  FeedSource,
} from '@/types/api';

export interface FeedMomentsParams {
  source?: FeedSource;
  radius?: number;
  limit?: number;
  cursor?: string;
  [key: string]: string | number | boolean | undefined;
}

export interface FeedGroupsParams {
  radius?: number;
  limit?: number;
  cursor?: string;
  [key: string]: string | number | boolean | undefined;
}

export interface FeedUsersParams {
  radius?: number;
  limit?: number;
  cursor?: string;
  [key: string]: string | number | boolean | undefined;
}

export interface FeedFeedbackParams {
  content_type: 'moment' | 'group' | 'user';
  content_id: string;
  action: 'hide' | 'not_interested' | 'report';
  reason?: string;
}

export const feedService = {
  /**
   * 获取瞬间信息流
   */
  async getMoments(params?: FeedMomentsParams): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return get<PaginatedData<MomentItem>>(ENDPOINTS.FEED.MOMENTS, params);
  },

  /**
   * 获取群组推荐
   */
  async getGroups(params?: FeedGroupsParams): Promise<ApiResponse<PaginatedData<FeedGroupItem>>> {
    return get<PaginatedData<FeedGroupItem>>(ENDPOINTS.FEED.GROUPS, params);
  },

  /**
   * 获取用户推荐
   */
  async getUsers(params?: FeedUsersParams): Promise<ApiResponse<PaginatedData<FeedUserItem>>> {
    return get<PaginatedData<FeedUserItem>>(ENDPOINTS.FEED.USERS, params);
  },

  /**
   * 获取热门内容
   */
  async getTrending(limit?: number): Promise<ApiResponse<FeedTrendingData>> {
    return get<FeedTrendingData>(ENDPOINTS.FEED.TRENDING, limit ? { limit } : undefined);
  },

  /**
   * 提交信息流反馈
   */
  async submitFeedback(data: FeedFeedbackParams): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.FEED.FEEDBACK, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },
};
