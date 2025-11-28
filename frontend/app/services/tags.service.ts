import { BaseService } from './base.service'
import type { TagData } from '~/types/api'

/**
 * 标签服务
 * ✅ 类型定义已统一从 types/api.ts 导入
 */

// ==================== Request 类型定义 ====================

export interface GetTrendingTagsRequest {
  limit?: number
  timeframe?: 'daily' | 'weekly' | 'monthly'
}

export interface TrendingTagsData {
  items: TagData[]
}

export interface SuggestTagsRequest {
  prefix: string
  limit?: number
}

export interface TagSuggestData {
  items: string[]
}

// ==================== 服务类 ====================

export class TagsService extends BaseService {
  protected readonly baseUrl = ''

  /**
   * 获取热门标签
   */
  async getTrending(params: GetTrendingTagsRequest = {}): Promise<TrendingTagsData> {
    return this.publicPost<TrendingTagsData>('/public/tags/get_trending', params)
  }

  /**
   * 标签搜索建议
   */
  async suggest(params: SuggestTagsRequest): Promise<TagSuggestData> {
    return this.publicPost<TagSuggestData>('/public/tags/suggest', params)
  }
}

export const tagsService = new TagsService()

export function useTagsService() {
  return tagsService
}


