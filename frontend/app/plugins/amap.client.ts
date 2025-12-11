/**
 * 高德地图 JS API 2.0 插件
 * 
 * 安全密钥配置（根据环境自动选择）：
 * - 开发环境：使用 securityJsCode 明文方式（方便调试）
 * - 生产环境：使用 serviceHost 代理服务器转发（安全）
 * 
 * 参考：https://lbs.amap.com/api/javascript-api-v2/guide/abc/jscode
 */

// 全局 AMap 加载状态 Promise
let amapLoadPromise: Promise<void> | null = null

/**
 * 等待 AMap SDK 加载完成
 */
export function waitForAMap(): Promise<void> {
  if (typeof window === 'undefined') {
    return Promise.reject(new Error('仅支持客户端'))
  }
  
  if (window.AMap) {
    return Promise.resolve()
  }
  
  if (amapLoadPromise) {
    return amapLoadPromise
  }
  
  // 如果还没开始加载，等待加载
  return new Promise((resolve, reject) => {
    let attempts = 0
    const check = setInterval(() => {
      attempts++
      if (window.AMap) {
        clearInterval(check)
        resolve()
      } else if (attempts > 100) {
        clearInterval(check)
        reject(new Error('AMap SDK 加载超时'))
      }
    }, 100)
  })
}

/**
 * 加载 AMap 插件
 */
export function loadAMapPlugin(plugins: string | string[]): Promise<void> {
  return new Promise((resolve, reject) => {
    waitForAMap()
      .then(() => {
        const pluginList = Array.isArray(plugins) ? plugins : [plugins]
        AMap.plugin(pluginList, () => {
          console.log('[AMap] 插件加载成功:', pluginList)
          resolve()
        })
      })
      .catch(reject)
  })
}

export default defineNuxtPlugin(() => {
  const config = useRuntimeConfig()
  const amapKey = config.public.amapKey
  const amapSecurityCode = config.public.amapSecurityCode

  if (!amapKey) {
    console.warn('[AMap] 未配置 NUXT_PUBLIC_AMAP_KEY，地图功能不可用')
    return
  }

  // 根据环境配置安全密钥（必须在加载脚本前配置）
  if (import.meta.dev) {
    // 开发环境：明文方式（需配置 NUXT_PUBLIC_AMAP_SECURITY_CODE）
    if (amapSecurityCode) {
      window._AMapSecurityConfig = {
        securityJsCode: amapSecurityCode as string
      }
      console.log('[AMap] 开发模式：使用明文安全密钥')
    } else {
      console.warn('[AMap] 开发模式：未配置 NUXT_PUBLIC_AMAP_SECURITY_CODE')
    }
  } else {
    // 生产环境：代理服务器转发
    window._AMapSecurityConfig = {
      serviceHost: '/_AMapService'
    }
    console.log('[AMap] 生产模式：使用代理服务器')
  }

  // 加载 SDK
  amapLoadPromise = new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.src = `https://webapi.amap.com/maps?v=2.0&key=${amapKey}`
    script.async = true
    script.onload = () => {
      console.log('[AMap] SDK 加载成功')
      // SDK 加载后预加载常用插件
      loadAMapPlugin(['AMap.Geocoder', 'AMap.Geolocation'])
        .then(resolve)
        .catch(() => {
          // 插件加载失败不阻塞，只打印警告
          console.warn('[AMap] 插件预加载失败')
          resolve()
        })
    }
    script.onerror = () => {
      console.error('[AMap] SDK 加载失败')
      reject(new Error('AMap SDK 加载失败'))
    }
    document.head.appendChild(script)
  })
})

// 类型声明
declare global {
  interface Window {
    AMap: typeof AMap
    _AMapSecurityConfig?: {
      securityJsCode?: string
      serviceHost?: string
    }
  }

  namespace AMap {
    class Map {
      constructor(container: string | HTMLElement, options?: MapOptions)
      destroy(): void
      getCenter(): LngLat
      setCenter(center: [number, number]): void
      setZoom(zoom: number): void
      getZoom(): number
      setZoomAndCenter(zoom: number, center: [number, number]): void
      add(overlays: Marker | Marker[]): void
      remove(overlays: Marker | Marker[]): void
      clearMap(): void
      on(event: string, handler: (e: unknown) => void): void
    }

    class Marker {
      constructor(options?: MarkerOptions)
      setPosition(position: [number, number]): void
      getPosition(): LngLat
      on(event: string, handler: (e: unknown) => void): void
      setExtData(data: unknown): void
      getExtData(): unknown
    }

    class LngLat {
      getLng(): number
      getLat(): number
    }

    interface MapOptions {
      center?: [number, number]
      zoom?: number
      viewMode?: '2D' | '3D'
      zooms?: [number, number]
      mapStyle?: string
    }

    interface MarkerOptions {
      position?: [number, number]
      title?: string
      extData?: unknown
    }

    class Geocoder {
      constructor(options?: { city?: string })
      getAddress(
        location: [number, number],
        callback: (status: string, result: GeocodeResult) => void
      ): void
      getLocation(
        address: string,
        callback: (status: string, result: GeocodeResult) => void
      ): void
    }

    interface GeocodeResult {
      info: string
      regeocode?: {
        formattedAddress: string
        addressComponent?: {
          province?: string
          city?: string | string[]
          district?: string
          street?: string
          township?: string
        }
      }
      geocodes?: Array<{
        location: LngLat
        formattedAddress: string
      }>
    }

    // 高德地图定位插件
    class Geolocation {
      constructor(options?: GeolocationOptions)
      getCurrentPosition(
        callback: (status: string, result: GeolocationResult) => void
      ): void
      watchPosition(): number
      clearWatch(watchId: number): void
    }

    interface GeolocationOptions {
      /** 是否使用高精度定位，默认 true */
      enableHighAccuracy?: boolean
      /** 超时时间，默认 5000ms */
      timeout?: number
      /** 定位按钮的排放位置 */
      buttonPosition?: string
      /** 是否使用坐标偏移，取值 true:自动偏移坐标 */
      convert?: boolean
      /** 是否显示定位按钮 */
      showButton?: boolean
      /** 是否显示定位精度圆 */
      showCircle?: boolean
      /** 是否显示定位点 */
      showMarker?: boolean
      /** 定位精度圆样式 */
      circleOptions?: object
      /** 定位点样式 */
      markerOptions?: object
      /** 定位超时时间 */
      maximumAge?: number
      /** 移动端是否优先使用H5定位 */
      GeoLocationFirst?: boolean
      /** 没有权限时是否使用IP定位 */
      noIpLocate?: number
      /** GPS定位是否使用3D加速 */
      useNative?: boolean
      /** 定位精度 */
      extensions?: 'all' | 'base'
    }

    interface GeolocationResult {
      /** 定位结果 */
      position: LngLat
      /** 精度范围，单位：米 */
      accuracy: number
      /** 定位类型 */
      location_type: 'ip' | 'html5' | 'sdk' | 'ipcity'
      /** 定位成功的消息 */
      message: string
      /** 是否为模拟定位 */
      isConverted: boolean
      /** 状态信息 */
      info: string
      /** 地址信息 */
      formattedAddress?: string
      /** 地址组件 */
      addressComponent?: {
        province?: string
        city?: string
        district?: string
        street?: string
        township?: string
        streetNumber?: string
        neighborhood?: string
        building?: string
      }
    }

    function plugin(name: string | string[], callback: () => void): void
  }
}




