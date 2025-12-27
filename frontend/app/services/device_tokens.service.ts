import { BaseService } from './base.service'
import { DEVICE_TOKENS } from '~/config/api'
import type {
  PaginatedData,
  RegisterDeviceTokenRequest,
  UnregisterDeviceTokenRequest,
  DeviceTokenItem,
  MyDeviceTokensRequest
} from '~/types/api'

class DeviceTokensService extends BaseService {
  async registerDeviceToken(data: RegisterDeviceTokenRequest) {
    return this.post<void>(DEVICE_TOKENS.REGISTER, data)
  }

  async unregisterDeviceToken(data: UnregisterDeviceTokenRequest) {
    return this.post<void>(DEVICE_TOKENS.UNREGISTER, data)
  }

  async getDeviceTokens(request: MyDeviceTokensRequest = {}) {
    return this.post<PaginatedData<DeviceTokenItem>>(DEVICE_TOKENS.ME, request)
  }
}

export const deviceTokensService = new DeviceTokensService()
export function useDeviceTokensService() { return deviceTokensService }

