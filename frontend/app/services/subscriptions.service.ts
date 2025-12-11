/**
 * 订阅服务 - 管理通知订阅配置
 */
import { BaseService } from './base.service'
import { SUBSCRIPTIONS } from '~/config/api'
import type { SubscriptionsData, UpdateSubscriptionsRequest } from '~/types/api'

class SubscriptionsService extends BaseService {
  /**
   * 获取订阅配置
   */
  async getConfig(): Promise<SubscriptionsData> {
    return this.post<SubscriptionsData>(SUBSCRIPTIONS.GET_CONFIG, {})
  }

  /**
   * 更新订阅配置
   */
  async update(request: UpdateSubscriptionsRequest): Promise<void> {
    return this.post<void>(SUBSCRIPTIONS.UPDATE, request)
  }
}

export const useSubscriptionsService = () => new SubscriptionsService()






