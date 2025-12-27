import { BaseService } from './base.service'
import { ADMIN_CONFIG } from '~/config/api'
import type {
  PaginatedData,
  CreateConfigRequest,
  CreateConfigData,
  ConfigDetailRequest,
  ConfigItem,
  ConfigListRequest,
  UpdateConfigRequest,
  DeleteConfigRequest,
  ConfigHistoryRequest,
  ConfigHistoryItem,
  ConfigChangesRequest
} from '~/types/api'

class AdminConfigService extends BaseService {
  async create(data: CreateConfigRequest) {
    return this.post<CreateConfigData>(ADMIN_CONFIG.CREATE, data)
  }

  async detail(data: ConfigDetailRequest) {
    return this.post<ConfigItem>(ADMIN_CONFIG.DETAIL, data)
  }

  async list(data: ConfigListRequest = {}) {
    return this.post<PaginatedData<ConfigItem>>(ADMIN_CONFIG.LIST, data)
  }

  async update(data: UpdateConfigRequest) {
    return this.post<void>(ADMIN_CONFIG.UPDATE, data)
  }

  async delete(data: DeleteConfigRequest) {
    return this.post<void>(ADMIN_CONFIG.DELETE, data)
  }

  async history(data: ConfigHistoryRequest) {
    return this.post<PaginatedData<ConfigHistoryItem>>(ADMIN_CONFIG.HISTORY, data)
  }

  async changes(data: ConfigChangesRequest = {}) {
    return this.post<PaginatedData<ConfigHistoryItem>>(ADMIN_CONFIG.CHANGES, data)
  }
}

export const adminConfigService = new AdminConfigService()
export function useAdminConfigService() { return adminConfigService }

