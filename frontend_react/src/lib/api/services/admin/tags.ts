import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';

export const AdminTagService = {
  delete: (data: { tag_name: string }) => apiClient.post(ENDPOINTS.ADMIN.TAGS.DELETE, data),
};
