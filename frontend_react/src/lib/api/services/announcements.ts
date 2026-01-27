import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { PaginatedData, PaginationParams, SystemAnnouncementItem } from '@/types/api';

export const AnnouncementService = {
  list: (params?: PaginationParams) =>
    apiClient.post<PaginatedData<SystemAnnouncementItem>>(ENDPOINTS.ANNOUNCEMENTS.LIST, params ?? {}),
};
