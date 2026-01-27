import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { AdminUserDetail, AdminUserItem, PaginatedData, UserAccountStatus, UserType } from '@/types/api';

export const AdminUserService = {
  search: (params: {
    cursor?: string;
    limit?: number;
    user_id?: number;
    nickname?: string;
    login_id?: string;
    status?: UserAccountStatus;
    user_type?: UserType;
  }) => apiClient.post<PaginatedData<AdminUserItem>>(ENDPOINTS.ADMIN.USERS.SEARCH, params),

  detail: (userId: number) =>
    apiClient.post<AdminUserDetail>(ENDPOINTS.ADMIN.USERS.DETAIL, { user_id: userId }),

  ban: (data: { user_id: number; reason?: string; duration_days?: number }) =>
    apiClient.post(ENDPOINTS.ADMIN.USERS.BAN, data),

  unban: (data: { user_id: number }) =>
    apiClient.post(ENDPOINTS.ADMIN.USERS.UNBAN, data),
};
