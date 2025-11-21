import { BaseService } from './base.service'
import type { BatchItemResult, BatchResultData } from '~/types/api'

/**
 * 批量操作服务
 * ✅ 类型定义已统一从 types/api.ts 导入
 */

// ==================== Request 类型定义 ====================

export interface MarkAsReadBatchItem {
  chat_type: 'private_chat' | 'group_chat'
  chat_id: string
  up_to: string
}

export interface MarkAsReadBatchRequest {
  items: MarkAsReadBatchItem[]
}

export interface RemoveInboxBatchRequest {
  ids: string[]
}

// ==================== 服务类 ====================

export class BatchService extends BaseService {
  protected readonly baseUrl = '/api/v1'

  /**
   * 批量标记消息已读
   */
  async markAsRead(data: MarkAsReadBatchRequest): Promise<BatchResultData> {
    return this.post<BatchResultData>('/protected/messages/mark_as_read_batch', data)
  }

  /**
   * 批量删除通知
   */
  async removeInbox(data: RemoveInboxBatchRequest): Promise<BatchResultData> {
    return this.post<BatchResultData>('/protected/inbox/remove_batch', data)
  }
}

export const batchService = new BatchService()

export function useBatchService() {
  return batchService
}


