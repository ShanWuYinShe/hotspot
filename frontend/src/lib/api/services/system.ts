import { post } from '../client';
import { SYSTEM } from '../endpoints';
import type {
  ApiVersionMeta,
  ClientConfig,
  HealthStatus,
  SystemAnnouncementItem
} from '@/lib/types/api';

export const systemService = {
  health() {
    return post<HealthStatus>(SYSTEM.HEALTH, {}, { skipAuth: true });
  },

  getMeta() {
    return post<ApiVersionMeta>(SYSTEM.GET_META, {}, { skipAuth: true });
  },

  getClientConfig() {
    return post<ClientConfig>(SYSTEM.GET_CLIENT_CONFIG, {}, { skipAuth: true });
  },

  getAnnouncements() {
    return post<{ items: SystemAnnouncementItem[] }>(SYSTEM.LIST_ANNOUNCEMENTS, {}, { skipAuth: true });
  }
};
