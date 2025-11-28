/**
 * 地理编码服务
 * 负责坐标和地址的相互转换
 */

export interface GeocodeResult {
  formatted_address: string
  country?: string
  province?: string
  city?: string
  district?: string
  street?: string
}

export class GeocodingService {
  /**
   * 反向地理编码：坐标 -> 地址名称
   * 优先使用浏览器原生API，失败后返回默认值
   */
  async reverseGeocode(latitude: number, longitude: number): Promise<GeocodeResult> {
    try {
      // 方案1: 使用浏览器原生Geolocation API（如果可用）
      // 注意：浏览器原生API不直接支持反向地理编码，需要外部服务
      
      // 方案2: 为将来集成第三方服务做准备
      // 可以集成：高德地图API、百度地图API、OpenStreetMap等
      
      // 当前实现：返回格式化的坐标作为地址
      return {
        formatted_address: this.formatCoordinates(latitude, longitude),
        country: '中国',
        province: '',
        city: '',
        district: '',
        street: ''
      }
    } catch (error) {
      console.error('反向地理编码失败:', error)
      return {
        formatted_address: this.formatCoordinates(latitude, longitude)
      }
    }
  }

  /**
   * 正向地理编码：地址名称 -> 坐标
   */
  async geocode(address: string): Promise<{ latitude: number; longitude: number } | null> {
    try {
      // 为将来集成第三方服务做准备
      console.warn('正向地理编码功能需要配置第三方地图服务')
      return null
    } catch (error) {
      console.error('地理编码失败:', error)
      return null
    }
  }

  /**
   * 格式化坐标为字符串
   */
  private formatCoordinates(latitude: number, longitude: number): string {
    const latStr = latitude.toFixed(6)
    const lngStr = longitude.toFixed(6)
    const latDir = latitude >= 0 ? 'N' : 'S'
    const lngDir = longitude >= 0 ? 'E' : 'W'
    return `${Math.abs(parseFloat(latStr))}°${latDir}, ${Math.abs(parseFloat(lngStr))}°${lngDir}`
  }

  /**
   * 获取简短的位置描述
   * 基于坐标生成友好的位置名称
   */
  async getLocationName(latitude: number, longitude: number): Promise<string> {
    try {
      const result = await this.reverseGeocode(latitude, longitude)
      
      // 优先使用详细地址
      if (result.city && result.district) {
        return `${result.city} ${result.district}`
      }
      
      if (result.city) {
        return result.city
      }
      
      // 降级为坐标字符串
      return result.formatted_address
    } catch (error) {
      console.error('获取位置名称失败:', error)
      return '未知位置'
    }
  }

  /**
   * 检查是否配置了地理编码服务
   */
  isConfigured(): boolean {
    // 检查环境变量中是否配置了第三方地图服务
    const config = useRuntimeConfig()
    return !!(config.public.mapApiKey || config.public.geocodingApiKey)
  }
}

// 导出单例
export const geocodingService = new GeocodingService()

// 导出组合式函数
export const useGeocodingService = () => geocodingService



