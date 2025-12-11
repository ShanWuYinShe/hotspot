import { BaseService } from './base.service'
import { DEVICE_TOKENS } from '~/config/api'
import type { PaginatedData } from '~/types/api'

export interface RegisterDeviceTokenRequest {
  platform: 'ios' | 'android' | 'harmony'
  device_token: string
}

export interface UnregisterDeviceTokenRequest {
  platform: 'ios' | 'android' | 'harmony'
  device_token: string
}

export interface DeviceTokenItem {
  id: string
  platform: 'ios' | 'android' | 'harmony'
  device_token: string
  created_at: string
}

class InboxService extends BaseService {
  async registerDeviceToken(data: RegisterDeviceTokenRequest) {
    return this.post<void>(DEVICE_TOKENS.REGISTER, data)
  }

  async unregisterDeviceToken(data: UnregisterDeviceTokenRequest) {
    return this.post<void>(DEVICE_TOKENS.UNREGISTER, data)
  }

  async getDeviceTokens() {
    const result = await this.post<PaginatedData<DeviceTokenItem>>(DEVICE_TOKENS.GET_LIST, {})
    return result.items
  }
}

export const inboxService = new InboxService()
export function useInboxService() { return inboxService }
