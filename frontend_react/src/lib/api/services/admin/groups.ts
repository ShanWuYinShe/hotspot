import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { AdminGroupDetail, AdminGroupItem, PaginatedData } from '@/types/api';

export const AdminGroupService = {
  search: (params: { cursor?: string; limit?: number; group_chat_id?: string; name?: string; owner_id?: number }) =>
    apiClient.post<PaginatedData<AdminGroupItem>>(ENDPOINTS.ADMIN.GROUPS.SEARCH, params),

  detail: (groupChatId: string) =>
    apiClient.post<AdminGroupDetail>(ENDPOINTS.ADMIN.GROUPS.DETAIL, { group_chat_id: groupChatId }),

  dismiss: (data: { group_chat_id: string; reason?: string }) =>
    apiClient.post(ENDPOINTS.ADMIN.GROUPS.DISMISS, data),
};
