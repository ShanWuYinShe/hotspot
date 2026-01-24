/**
 * 瞬间服务
 */

import { post, generateIdempotencyKey } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  MomentItem,
  CreateMomentRequest,
  UpdateMomentRequest,
  MomentLikeItem,
  GeoPoint,
} from '@/types/api';

export interface SearchByKeywordParams {
  kw: string;
  reference_point: GeoPoint;
  limit?: number;
  cursor?: string;
}

export interface SearchByTagParams {
  tag: string;
  reference_point: GeoPoint;
  limit?: number;
  cursor?: string;
}

export interface SearchByLocationParams {
  reference_point: GeoPoint;
  radius: number;
  kw?: string;
  limit?: number;
  cursor?: string;
}

export const momentsService = {
  // ============================================================================
  // 搜索
  // ============================================================================

  /**
   * 按关键词搜索瞬间
   */
  async searchByKeyword(params: SearchByKeywordParams): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return post<PaginatedData<MomentItem>>(ENDPOINTS.MOMENTS.SEARCH_BY_KEYWORD, params, {
      requireAuth: false,
    });
  },

  /**
   * 按标签搜索瞬间
   */
  async searchByTag(params: SearchByTagParams): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return post<PaginatedData<MomentItem>>(ENDPOINTS.MOMENTS.SEARCH_BY_TAG, params, {
      requireAuth: false,
    });
  },

  /**
   * 按位置搜索瞬间
   */
  async searchByLocation(params: SearchByLocationParams): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return post<PaginatedData<MomentItem>>(ENDPOINTS.MOMENTS.SEARCH_BY_LOCATION, params, {
      requireAuth: false,
    });
  },

  // ============================================================================
  // CRUD
  // ============================================================================

  /**
   * 获取瞬间详情
   */
  async getDetail(momentId: string, referencePoint?: GeoPoint): Promise<ApiResponse<MomentItem>> {
    return post<MomentItem>(
      ENDPOINTS.MOMENTS.DETAIL(momentId),
      {
        moment_id: momentId,
        ...(referencePoint ? { reference_point: referencePoint } : {}),
      },
      { requireAuth: false }
    );
  },

  /**
   * 创建瞬间
   */
  async create(data: CreateMomentRequest): Promise<ApiResponse<MomentItem>> {
    return post<MomentItem>(ENDPOINTS.MOMENTS.CREATE, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 更新瞬间
   */
  async update(momentId: string, data: Omit<UpdateMomentRequest, 'moment_id'>): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.MOMENTS.UPDATE(momentId), { moment_id: momentId, ...data }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 删除瞬间
   */
  async delete(momentId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.MOMENTS.DELETE(momentId), { moment_id: momentId });
  },

  // ============================================================================
  // 互动
  // ============================================================================

  /**
   * 点赞瞬间
   */
  async like(momentId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.MOMENTS.LIKE(momentId), { moment_id: momentId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 取消点赞
   */
  async unlike(momentId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.MOMENTS.UNLIKE(momentId), { moment_id: momentId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 收藏瞬间
   */
  async favorite(momentId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.MOMENTS.FAVORITE(momentId), { moment_id: momentId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 取消收藏
   */
  async unfavorite(momentId: string): Promise<ApiResponse<void>> {
    return post<void>(ENDPOINTS.MOMENTS.UNFAVORITE(momentId), { moment_id: momentId }, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  /**
   * 获取点赞列表
   */
  async getLikes(
    momentId: string,
    params?: { limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<MomentLikeItem>>> {
    return post<PaginatedData<MomentLikeItem>>(
      ENDPOINTS.MOMENTS.LIKES(momentId),
      { moment_id: momentId, ...(params ?? {}) },
      { requireAuth: false }
    );
  },

  // ============================================================================
  // 用户瞬间
  // ============================================================================

  /**
   * 获取用户瞬间列表
   */
  async getUserMoments(
    userId: number,
    params: { reference_point: GeoPoint; limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return post<PaginatedData<MomentItem>>(ENDPOINTS.USER.MOMENTS(userId), params, {
      requireAuth: false,
    });
  },

  /**
   * 获取我的瞬间列表
   */
  async getMyMoments(
    params: { reference_point: GeoPoint; limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return post<PaginatedData<MomentItem>>(ENDPOINTS.USER.ME_MOMENTS, params);
  },

  /**
   * 获取我的收藏列表
   */
  async getMyFavorites(
    params: { reference_point: GeoPoint; limit?: number; cursor?: string }
  ): Promise<ApiResponse<PaginatedData<MomentItem>>> {
    return post<PaginatedData<MomentItem>>(ENDPOINTS.USER.ME_FAVORITES, params);
  },
};
