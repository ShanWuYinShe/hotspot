/**
 * 高德地图 JS API 2.0 插件
 *
 * 关键原则：
 * - **不要在应用启动时全局加载 AMap SDK**（会显著拖慢首屏并放大 FOUC）
 * - 只在「地图/选点」等确实需要时再按需加载
 *
 * 安全密钥配置（统一使用代理转发，开发/生产保持一致）：
 * - 通过 Nginx 反向代理固定前缀 `/_AMapService` 注入 jscode（securityJsCode）
 * - 前端只设置 `serviceHost`，不再在 runtimeConfig 暴露 securityJsCode
 *
 * 参考：https://lbs.amap.com/api/javascript-api-v2/guide/abc/jscode
 */

// 全局 AMap 加载状态 Promise
let amapLoadPromise: Promise<void> | null = null
let amapSecurityConfigured = false

function configureAMapSecurity(config: ReturnType<typeof useRuntimeConfig>) {
  if (amapSecurityConfigured || typeof window === 'undefined') return

  const amapKey = config.public.amapKey

  if (!amapKey) {
    // 仍然允许 app 启动，只是地图功能不可用
    console.warn('[AMap] 未配置 NUXT_PUBLIC_AMAP_KEY，地图功能不可用')
    amapSecurityConfigured = true
    return
  }

  // 根据环境选择不同的安全配置方式
  const isDev = import.meta.env.DEV

  if (isDev) {
    // 开发环境：直接使用 securityJsCode（方便本地调试）
    const securityCode = (config.public as { amapSecurityCode?: string }).amapSecurityCode
    if (securityCode) {
      window._AMapSecurityConfig = { securityJsCode: securityCode }
    } else {
      console.warn('[AMap] 开发环境未配置 NUXT_PUBLIC_AMAP_SECURITY_CODE，安全验证可能失败')
    }
  } else {
    // 生产环境：通过代理服务器转发安全密钥（jscode）
    // - 默认使用同源 `/_AMapService`（由 Nginx 负责把 jscode 注入到 query）
    // - 如需跨域部署，可通过 NUXT_PUBLIC_AMAP_SERVICE_HOST 显式指定完整域名
    const serviceHost = (config.public as { amapServiceHost?: string }).amapServiceHost
      || `${window.location.origin}/_AMapService`

    window._AMapSecurityConfig = { serviceHost }
  }

  amapSecurityConfigured = true
}

function initAMap(config: ReturnType<typeof useRuntimeConfig>): Promise<void> {
  if (typeof window === 'undefined') {
    return Promise.reject(new Error('仅支持客户端'))
  }

  if (window.AMap) return Promise.resolve()
  if (amapLoadPromise) return amapLoadPromise

  configureAMapSecurity(config)

  const amapKey = config.public.amapKey
  if (!amapKey) {
    return Promise.reject(new Error('未配置 AMap Key'))
  }

  amapLoadPromise = new Promise((resolve, reject) => {
    // 防重复插入
    const existing = document.getElementById('amap-sdk-script') as HTMLScriptElement | null
    if (existing) {
      existing.addEventListener('load', () => resolve())
      existing.addEventListener('error', () => reject(new Error('AMap SDK 加载失败')))
      return
    }

    const script = document.createElement('script')
    script.id = 'amap-sdk-script'
    script.src = `https://webapi.amap.com/maps?v=2.0&key=${amapKey}`
    script.async = true
    script.onload = () => {
      // SDK 加载后预加载常用插件（失败不阻塞）
      loadAMapPlugin(['AMap.Geocoder', 'AMap.Geolocation'])
        .then(resolve)
        .catch(() => resolve())
    }
    script.onerror = () => reject(new Error('AMap SDK 加载失败'))
    document.head.appendChild(script)
  })

  return amapLoadPromise
}

/**
 * 等待 AMap SDK 加载完成（必要时会触发按需加载）
 */
export function waitForAMap(): Promise<void> {
  if (typeof window === 'undefined') {
    return Promise.reject(new Error('仅支持客户端'))
  }

  if (window.AMap) return Promise.resolve()

  // 任何调用方都可能先调用 waitForAMap，所以这里负责触发按需加载
  // 读取 runtimeConfig 必须在 Nuxt 生命周期内调用，因此这里用 useRuntimeConfig()
  return initAMap(useRuntimeConfig())
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
          resolve()
        })
      })
      .catch(reject)
  })
}

export default defineNuxtPlugin(() => {
  // 只做安全配置，不在这里触发 SDK 下载（避免首屏性能/样式加载被拖慢）
  configureAMapSecurity(useRuntimeConfig())
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
          streetNumber?: string
          neighborhood?: string
          building?: string
        }
        pois?: Array<{
          name: string
          type: string
          address: string
          distance: number
          location: LngLat
        }>
        aois?: Array<{
          name: string
          type: string
          area: string
          location: LngLat
        }>
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
      on(event: string, handler: (e: unknown) => void): void
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




