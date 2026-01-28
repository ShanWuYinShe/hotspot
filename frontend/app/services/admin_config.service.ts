import { BaseService } from './base.service'
import { ADMIN_GLOBAL_SETTINGS } from '~/config/api'
import type {
  AdminGlobalSettingsData,
  AdminGetGlobalSettingsRequest,
  UpdateUploadConfigRequest,
  UpdateDefaultsRequest,
  UpdateFeaturesRequest,
  UpdateContactRequest,
  UpdateLimitsRequest,
  UploadConfig,
  ContentDefaults,
  FeatureSwitches,
  ContactInfo,
  AppLimits
} from '~/types/api'

class AdminConfigService extends BaseService {
  async getGlobalSettings(data: AdminGetGlobalSettingsRequest = {}) {
    return this.post<AdminGlobalSettingsData>(ADMIN_GLOBAL_SETTINGS.FETCH, data)
  }

  async updateUploadConfig(data: UpdateUploadConfigRequest) {
    return this.post<{ upload: UploadConfig; updated_at: string; version: string }>(
      ADMIN_GLOBAL_SETTINGS.UPDATE_UPLOAD,
      data
    )
  }

  async updateDefaults(data: UpdateDefaultsRequest) {
    return this.post<{ defaults: ContentDefaults; updated_at: string; version: string }>(
      ADMIN_GLOBAL_SETTINGS.UPDATE_DEFAULTS,
      data
    )
  }

  async updateFeatures(data: UpdateFeaturesRequest) {
    return this.post<{ features: FeatureSwitches; updated_at: string; version: string }>(
      ADMIN_GLOBAL_SETTINGS.UPDATE_FEATURES,
      data
    )
  }

  async updateContact(data: UpdateContactRequest) {
    return this.post<{ contact: ContactInfo; updated_at: string; version: string }>(
      ADMIN_GLOBAL_SETTINGS.UPDATE_CONTACT,
      data
    )
  }

  async updateLimits(data: UpdateLimitsRequest) {
    return this.post<{ limits: AppLimits; updated_at: string; version: string }>(
      ADMIN_GLOBAL_SETTINGS.UPDATE_LIMITS,
      data
    )
  }
}

export const adminConfigService = new AdminConfigService()
export function useAdminConfigService() { return adminConfigService }
