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
  ConfigChangesRequest,
  AdminGlobalSettingsData,
  AdminGetGlobalSettingsRequest,
  UpdateUploadConfigRequest,
  UpdateDefaultsRequest,
  UpdateFeaturesRequest,
  UpdateContactRequest,
  UpdateLimitsRequest,
  GlobalSettings,
  UploadConfig,
  ContentDefaults,
  FeatureSwitches,
  ContactInfo,
  AppLimits
} from '~/types/api'

class AdminConfigService extends BaseService {
  // ============ New Strongly Typed Global Settings API ============

  async getGlobalSettings(data: AdminGetGlobalSettingsRequest = {}) {
    return this.post<AdminGlobalSettingsData>(ADMIN_CONFIG.ADMIN_FETCH, data)
  }

  async updateUploadConfig(data: UpdateUploadConfigRequest) {
    return this.post<{ upload: UploadConfig; updated_at: string; version: string }>(
      ADMIN_CONFIG.ADMIN_UPDATE_UPLOAD,
      data
    )
  }

  async updateDefaults(data: UpdateDefaultsRequest) {
    return this.post<{ defaults: ContentDefaults; updated_at: string; version: string }>(
      ADMIN_CONFIG.ADMIN_UPDATE_DEFAULTS,
      data
    )
  }

  async updateFeatures(data: UpdateFeaturesRequest) {
    return this.post<{ features: FeatureSwitches; updated_at: string; version: string }>(
      ADMIN_CONFIG.ADMIN_UPDATE_FEATURES,
      data
    )
  }

  async updateContact(data: UpdateContactRequest) {
    return this.post<{ contact: ContactInfo; updated_at: string; version: string }>(
      ADMIN_CONFIG.ADMIN_UPDATE_CONTACT,
      data
    )
  }

  async updateLimits(data: UpdateLimitsRequest) {
    return this.post<{ limits: AppLimits; updated_at: string; version: string }>(
      ADMIN_CONFIG.ADMIN_UPDATE_LIMITS,
      data
    )
  }

  // ============ Old Generic KV API (Deprecated/Compat) ============

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
