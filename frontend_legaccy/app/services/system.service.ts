import { BaseService } from './base.service'
import { SYSTEM } from '~/config/api'
import type { HealthCheckData } from '~/types/api'

class SystemService extends BaseService {
  async healthCheck() {
    return this.post<HealthCheckData>(SYSTEM.HEALTH, {})
  }
}

export const systemService = new SystemService()
export function useSystemService() { return systemService }
