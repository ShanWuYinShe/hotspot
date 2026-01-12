import { BaseService } from './base.service'
import { MAP } from '~/config/api'
import type { MapRenderRequest, MapRenderData } from '~/types/api'

class MapService extends BaseService {
  /**
   * 地图聚合与渲染接口
   * 根据 zoom 和 viewport 动态返回聚合点(Cluster)或实体点(Entity)
   */
  async render(request: MapRenderRequest) {
    return this.publicPost<MapRenderData>(MAP.RENDER, request)
  }
}

let mapService: MapService | null = null
export const useMapService = () => {
  if (!mapService) mapService = new MapService()
  return mapService
}
