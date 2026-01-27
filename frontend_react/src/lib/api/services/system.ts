import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { HealthStatus } from '@/types/api';

export const SystemService = {
  health: () => apiClient.post<HealthStatus>(ENDPOINTS.SYSTEM.HEALTH),
};
