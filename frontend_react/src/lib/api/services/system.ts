/**
 * 系统服务
 */

import { get, post } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  PaginatedData,
  HealthCheckData,
  GlobalSettings,
  AnnouncementItem,
  TagData,
  TagSuggestItem,
} from '@/types/api';

export interface FetchGlobalSettingsParams {
  context: {
    os: 'ios' | 'android' | 'web' | 'harmony';
    app_version: string;
    device_id?: string;
    language?: string;
  };
  current_version?: string;
}

export interface FetchSettingsResponse {
  action: 'update' | 'no_change' | 'force_update';
  config?: GlobalSettings;
  latest_version: string;
}

export const systemService = {
  /**
   * 健康检查
   */
  async healthCheck(): Promise<ApiResponse<HealthCheckData>> {
    return get<HealthCheckData>(ENDPOINTS.SYSTEM.HEALTH, undefined, {
      requireAuth: false,
    });
  },

  /**
   * 获取全局配置
   */
  async fetchGlobalSettings(params: FetchGlobalSettingsParams): Promise<ApiResponse<FetchSettingsResponse>> {
    return post<FetchSettingsResponse>(ENDPOINTS.SYSTEM.GLOBAL_SETTINGS, params, {
      requireAuth: false,
    });
  },

  /**
   * 获取公告列表
   */
  async getAnnouncements(params?: { limit?: number; cursor?: string }): Promise<ApiResponse<PaginatedData<AnnouncementItem>>> {
    return get<PaginatedData<AnnouncementItem>>(ENDPOINTS.ANNOUNCEMENTS.LIST, params, {
      requireAuth: false,
    });
  },

  /**
   * 获取公告详情
   */
  async getAnnouncementDetail(announcementId: string): Promise<ApiResponse<AnnouncementItem>> {
    return get<AnnouncementItem>(ENDPOINTS.ANNOUNCEMENTS.DETAIL(announcementId), undefined, {
      requireAuth: false,
    });
  },

  /**
   * 获取热门标签
   */
  async getTrendingTags(params?: { limit?: number; timeframe?: 'daily' | 'weekly' | 'monthly' }): Promise<ApiResponse<{ items: TagData[] }>> {
    return get<{ items: TagData[] }>(ENDPOINTS.TAGS.TRENDING, params, {
      requireAuth: false,
    });
  },

  /**
   * 获取标签建议
   */
  async suggestTags(prefix: string, limit?: number): Promise<ApiResponse<{ items: TagSuggestItem[] }>> {
    return get<{ items: TagSuggestItem[] }>(ENDPOINTS.TAGS.SUGGEST, { prefix, limit }, {
      requireAuth: false,
    });
  },
};
