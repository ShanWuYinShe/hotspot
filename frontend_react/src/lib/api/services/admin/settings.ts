import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import {
  AdminSettings,
  AppLimits,
  ContentDefaults,
  FeatureSwitches,
  GroupSettings,
  UploadConfig,
  ContactInfo,
} from '@/types/api';

export const AdminSettingsService = {
  fetch: () => apiClient.post<AdminSettings>(ENDPOINTS.ADMIN.SETTINGS.FETCH),

  updateUpload: (data: Partial<UploadConfig>) =>
    apiClient.post<{ upload: UploadConfig; updated_at: string }>(ENDPOINTS.ADMIN.SETTINGS.UPDATE_UPLOAD, data),

  updateDefaults: (data: {
    default_location_name?: string;
    default_group_avatar?: string;
    default_user_avatar?: string;
    default_group_settings?: Partial<GroupSettings>;
  }) =>
    apiClient.post<{ defaults: ContentDefaults; updated_at: string }>(
      ENDPOINTS.ADMIN.SETTINGS.UPDATE_DEFAULTS,
      data,
    ),

  updateFeatures: (data: Partial<FeatureSwitches>) =>
    apiClient.post<{ features: FeatureSwitches; updated_at: string }>(ENDPOINTS.ADMIN.SETTINGS.UPDATE_FEATURES, data),

  updateContact: (data: Partial<ContactInfo>) =>
    apiClient.post<{ contact: ContactInfo; updated_at: string }>(ENDPOINTS.ADMIN.SETTINGS.UPDATE_CONTACT, data),

  updateLimits: (data: Partial<AppLimits>) =>
    apiClient.post<{ limits: AppLimits; updated_at: string }>(ENDPOINTS.ADMIN.SETTINGS.UPDATE_LIMITS, data),
};
