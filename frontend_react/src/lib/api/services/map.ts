import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import { MapCanvas, MapRenderRequest } from '@/types/api';

export const MapService = {
  render: (data: MapRenderRequest) => apiClient.post<MapCanvas>(ENDPOINTS.MAP.RENDER, data),
};
