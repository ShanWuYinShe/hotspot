/**
 * 地图服务
 */

import { post } from '../client';
import { ENDPOINTS } from '../endpoints';
import type {
  ApiResponse,
  MapRenderRequest,
  MapRenderData,
} from '@/types/api';

export const mapService = {
  /**
   * 获取地图渲染数据
   */
  async render(params: MapRenderRequest): Promise<ApiResponse<MapRenderData>> {
    return post<MapRenderData>(ENDPOINTS.MAP.RENDER, params, {
      requireAuth: false,
    });
  },
};
