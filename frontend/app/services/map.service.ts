import { BaseService } from './base.service'
import { MAP } from '~/config/api'
import type { MapClusterRequest, MapClustersData, MapItemsData, MapSearchByBoundsRequest } from '~/types/api'

class MapService extends BaseService {
  async clusterUsers(request: MapClusterRequest) {
    return this.publicPost<MapClustersData>(MAP.USERS.CLUSTER, request)
  }

  async clusterGroups(request: MapClusterRequest) {
    return this.publicPost<MapClustersData>(MAP.GROUPS.CLUSTER, request)
  }

  async clusterMoments(request: MapClusterRequest) {
    return this.publicPost<MapClustersData>(MAP.MOMENTS.CLUSTER, request)
  }

  async searchUsersByBounds(request: MapSearchByBoundsRequest) {
    return this.publicPost<MapItemsData>(MAP.USERS.SEARCH_BY_BOUNDS, request)
  }

  async searchGroupsByBounds(request: MapSearchByBoundsRequest) {
    return this.publicPost<MapItemsData>(MAP.GROUPS.SEARCH_BY_BOUNDS, request)
  }

  async searchMomentsByBounds(request: MapSearchByBoundsRequest) {
    return this.publicPost<MapItemsData>(MAP.MOMENTS.SEARCH_BY_BOUNDS, request)
  }
}

let mapService: MapService | null = null
export const useMapService = () => {
  if (!mapService) mapService = new MapService()
  return mapService
}

