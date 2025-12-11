/**
 * 地理编码服务 - 使用高德地图 SDK
 */

import { waitForAMap, loadAMapPlugin } from '~/plugins/amap.client'

export interface GeocodeResult {
  formatted_address: string
  province?: string
  city?: string
  district?: string
  street?: string
  township?: string
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
            resolve({
              formatted_address: regeocode.formattedAddress,
              province: addr?.province,
              city,
              district: addr?.district,
              street: addr?.street,
              township: addr?.township
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
   * 获取简短位置名称
   */
  async getLocationName(latitude: number, longitude: number): Promise<string> {
    const result = await this.reverseGeocode(latitude, longitude)

    // 优先返回：区 + 街道/乡镇
    if (result.district) {
      if (result.street) return `${result.district}${result.street}`
      if (result.township) return `${result.district}${result.township}`
      return result.district
    }

    // 其次返回城市
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
            const location = result.geocodes[0].location
            resolve({
              latitude: location.getLat(),
              longitude: location.getLng()
            })
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
