import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { AnnouncementCreated, AnnouncementSeverity, PaginatedData, SystemAnnouncementItem } from '@/types/api';

export const AdminAnnouncementService = {
  list: (params?: { cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<SystemAnnouncementItem>>(ENDPOINTS.ADMIN.ANNOUNCEMENTS.LIST, params ?? {}),

  create: (data: {
    title: string;
    content: string;
    severity: AnnouncementSeverity;
    action_url?: string;
    send_push?: boolean;
  }) => apiClient.post<AnnouncementCreated>(ENDPOINTS.ADMIN.ANNOUNCEMENTS.CREATE, data),

  delete: (data: { announcement_id: string }) =>
    apiClient.post(ENDPOINTS.ADMIN.ANNOUNCEMENTS.DELETE, data),
};
