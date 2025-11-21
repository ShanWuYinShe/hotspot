/**
 * 系统服务
 * 提供系统健康检查等功能
 * 
 * 符合 API 规范 3.14 节
 */

import { BaseService } from './base.service'
import type { HealthCheckData } from '~/types/api'

/**
 * 系统服务类
 */
export class SystemService extends BaseService {
  constructor() {
    super('/api/v1/public/system')
  }

  /**
   * 系统健康检查
   * 3.14.1. POST /api/v1/public/system/health_check - 系统健康检查
   */
  async healthCheck(): Promise<HealthCheckData> {
    return this.publicPost<HealthCheckData>('/health_check', {})
  }
}

/**
 * 使用系统服务的 Composable
 */
export function useSystemService() {
  return new SystemService()
}
