/**
 * 地理编码服务 - 使用高德地图 SDK
 */

import { waitForAMap, loadAMapPlugin } from '~/plugins/01.amap.client'

export interface POI {
  name: string
  type: string
  address: string
  distance: number
}

export interface AOI {
  name: string
  type: string
  area: string
}

export interface GeocodeResult {
  formatted_address: string
  province?: string
  city?: string
  district?: string
  street?: string
  township?: string
  streetNumber?: string
  neighborhood?: string
  building?: string
  pois?: POI[]
  aois?: AOI[]
}

class GeocodingService {
  private geocoder: AMap.Geocoder | null = null

  private async getGeocoder(): Promise<AMap.Geocoder> {
    if (this.geocoder) return this.geocoder

    await waitForAMap()
    await loadAMapPlugin('AMap.Geocoder')
    this.geocoder = new AMap.Geocoder()
    return this.geocoder
  }

  /**
   * 逆地理编码：经纬度 → 地址
   */
  async reverseGeocode(latitude: number, longitude: number): Promise<GeocodeResult> {
    try {
      const geocoder = await this.getGeocoder()

      return new Promise((resolve) => {
        geocoder.getAddress([longitude, latitude], (status: string, result: AMap.GeocodeResult) => {
          if (status === 'complete' && result.regeocode) {
            const { regeocode } = result
            const addr = regeocode.addressComponent
            // city 可能是 string 或 string[]
            const city = Array.isArray(addr?.city) ? addr.city[0] : addr?.city
            
            // 提取POI信息
            const pois = regeocode.pois?.map(poi => ({
              name: poi.name,
              type: poi.type,
              address: poi.address,
              distance: poi.distance
            }))
            
            // 提取AOI信息
            const aois = regeocode.aois?.map(aoi => ({
              name: aoi.name,
              type: aoi.type,
              area: aoi.area
            }))
            
            resolve({
              formatted_address: regeocode.formattedAddress,
              province: addr?.province,
              city,
              district: addr?.district,
              street: addr?.street,
              township: addr?.township,
              streetNumber: addr?.streetNumber,
              neighborhood: addr?.neighborhood,
              building: addr?.building,
              pois,
              aois
            })
          } else {
            resolve({ formatted_address: '未知位置' })
          }
        })
      })
    } catch (error) {
      console.error('[Geocoding] 逆地理编码失败:', error)
      return { formatted_address: '未知位置' }
    }
  }

  /**
   * 获取人类友好的位置名称
   * 优先级：POI（兴趣点，如小区、商铺） > AOI（兴趣区域） > 建筑物 > 地址组件
   */
  async getLocationName(latitude: number, longitude: number): Promise<string> {
    const result = await this.reverseGeocode(latitude, longitude)

    // 1. 优先使用POI（兴趣点），如"XX小区"、"XX商场"、"XX餐厅"等
    if (result.pois && result.pois.length > 0) {
      // 选择距离最近的POI（通常第一个就是最近的）
      const nearestPOI = result.pois[0]
      // 如果距离很近（小于50米），直接使用POI名称
      if (nearestPOI.distance < 50) {
        return nearestPOI.name
      }
      // 如果距离较远，使用"XX附近"的格式
      return `${nearestPOI.name}附近`
    }

    // 2. 其次使用AOI（兴趣区域），如"XX商圈"、"XX园区"等
    if (result.aois && result.aois.length > 0) {
      const nearestAOI = result.aois[0]
      return nearestAOI.name
    }

    // 3. 使用建筑物名称（如果有）
    if (result.building) {
      return result.building
    }

    // 4. 使用小区/社区名称（如果有）
    if (result.neighborhood) {
      return result.neighborhood
    }

    // 5. 降级方案：使用地址组件（区 + 街道/乡镇）
    if (result.district) {
      if (result.street) return `${result.district}${result.street}`
      if (result.township) return `${result.district}${result.township}`
      return result.district
    }

    // 6. 最后返回城市或完整地址
    if (result.city) return result.city as string

    return result.formatted_address
  }

  /**
   * 正向地理编码：地址 → 经纬度
   */
  async geocode(address: string): Promise<{ latitude: number; longitude: number } | null> {
    try {
      const geocoder = await this.getGeocoder()

      return new Promise((resolve) => {
        geocoder.getLocation(address, (status: string, result: AMap.GeocodeResult) => {
          if (status === 'complete' && result.geocodes?.length) {
            const geocode = result.geocodes[0]
            if (geocode && geocode.location) {
              const location = geocode.location
              resolve({
                latitude: location.getLat(),
                longitude: location.getLng()
              })
            } else {
              resolve(null)
            }
          } else {
            resolve(null)
          }
        })
      })
    } catch (error) {
      console.error('[Geocoding] 正向地理编码失败:', error)
      return null
    }
  }
}

export const geocoding = new GeocodingService()
