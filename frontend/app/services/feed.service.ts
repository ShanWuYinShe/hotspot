import { BaseService } from './base.service'
import type { FeedItem, FeedSystemInfo, PaginatedData } from '~/types/api'

/**
 * 信息流服务
 * ✅ 类型定义已统一从 types/api.ts 导入
 */

// ==================== Request 类型定义 ====================

export interface ListFeedRequest {
  channel: 'following' | 'nearby' | 'recommended'
  limit?: number
  cursor?: string
}

export interface FeedFeedbackRequest {
  item_id: string
  action: 'hide' | 'not_interested' | 'clicked' | 'viewed'
}

// ==================== 服务类 ====================

export class FeedService extends BaseService {
  protected readonly baseUrl = ''

  /**
   * 获取信息流
   */
  async getList(params: ListFeedRequest): Promise<PaginatedData<FeedItem>> {
    return this.post<PaginatedData<FeedItem>>('/protected/feed/get_list', params)
  }

  /**
   * 反馈信息流条目
   */
  async feedback(data: FeedFeedbackRequest): Promise<void> {
    return this.post<void>('/protected/feed/feedback', data)
  }
}

export const feedService = new FeedService()

export function useFeedService() {
  return feedService
}


