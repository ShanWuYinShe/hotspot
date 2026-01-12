import { BaseService } from './base.service'
import { ADMIN_CONFIG } from '~/config/api'
import type { FetchGlobalSettingsRequest, FetchSettingsData } from '~/types/api'

class GlobalSettingsService extends BaseService {
  /**
   * 拉取全局配置 (Remote Config 风格)
   * 客户端提供环境上下文和当前版本，服务端决定是否返回新配置
   */
  async fetch(request: FetchGlobalSettingsRequest) {
    return this.publicPost<FetchSettingsData>(ADMIN_CONFIG.FETCH, request)
  }
}

export const globalSettingsService = new GlobalSettingsService()
export function useGlobalSettingsService() { return globalSettingsService }
