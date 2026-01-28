/**
 * 标签服务
 * 标签相关 API
 */

import { apiRequestPublic } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import type { ApiResponse, TagTrend, TagTimeframe } from '@/types/api';

interface TrendingTag {
  tag: string;
  count: number;
  trend: TagTrend;
}

interface TrendingTagsParams {
  timeframe?: TagTimeframe;
  limit?: number;
}

interface SuggestTagsParams {
  prefix: string;
  limit?: number;
}

/**
 * 获取热门标签
 */
export async function getTrendingTags(
  params: TrendingTagsParams = {}
): Promise<ApiResponse<TrendingTag[]>> {
  return apiRequestPublic<TrendingTag[]>(
    ENDPOINTS.TAGS.TRENDING,
    params
  );
}

/**
 * 获取标签建议
 */
export async function suggestTags(
  params: SuggestTagsParams
): Promise<ApiResponse<string[]>> {
  return apiRequestPublic<string[]>(
    ENDPOINTS.TAGS.SUGGEST,
    params
  );
}
