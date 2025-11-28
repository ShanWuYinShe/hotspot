/**
 * 系统服务
 * 提供系统健康检查等功能
 */

import { BaseService } from './base.service'
import type { HealthCheckResponse } from '~/types/api'

/**
 * 系统服务类
 */
export class SystemService extends BaseService {
  protected readonly baseUrl = '/public/system'

  /**
   * 系统健康检查
   */
  async healthCheck(): Promise<HealthCheckResponse> {
    return this.publicPost<HealthCheckResponse>('/health_check', {})
  }
}

/**
 * 使用系统服务的 Composable
 */
export function useSystemService() {
  return new SystemService()
}
