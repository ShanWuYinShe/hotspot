// 地理位置 composable (客户端)
export const useGeolocation = () => {
  const position = ref<GeolocationPosition | null>(null)
  const error = ref<GeolocationPositionError | null>(null)
  const loading = ref(false)

  const getCurrentPosition = async (options?: PositionOptions): Promise<GeolocationPosition> => {
    return new Promise((resolve, reject) => {
      if (!navigator.geolocation) {
        reject(new Error('浏览器不支持地理位置功能'))
        return
      }

      loading.value = true
      error.value = null

      navigator.geolocation.getCurrentPosition(
        (pos) => {
          position.value = pos
          loading.value = false
          resolve(pos)
        },
        (err) => {
          error.value = err
          loading.value = false
          
          let message = '获取位置失败'
          switch (err.code) {
            case err.PERMISSION_DENIED:
              message = '位置权限被拒绝'
              break
            case err.POSITION_UNAVAILABLE:
              message = '位置信息不可用'
              break
            case err.TIMEOUT:
              message = '获取位置超时'
              break
          }
          
          reject(new Error(message))
        },
        {
          enableHighAccuracy: true,
          timeout: 10000,
          maximumAge: 300000, // 5分钟
          ...options
        }
      )
    })
  }

  const watchPosition = (options?: PositionOptions) => {
    if (!navigator.geolocation) {
      error.value = new GeolocationPositionError()
      return null
    }

    const watchId = navigator.geolocation.watchPosition(
      (pos) => {
        position.value = pos
        error.value = null
      },
      (err) => {
        error.value = err
      },
      {
        enableHighAccuracy: true,
        timeout: 10000,
        maximumAge: 300000,
        ...options
      }
    )

    return watchId
  }

  const clearWatch = (watchId: number) => {
    navigator.geolocation.clearWatch(watchId)
  }

  // 坐标计算属性
  const coordinates = computed(() => {
    if (!position.value?.coords) return null
    return {
      latitude: position.value.coords.latitude,
      longitude: position.value.coords.longitude,
      accuracy: position.value.coords.accuracy,
      altitude: position.value.coords.altitude,
      altitudeAccuracy: position.value.coords.altitudeAccuracy,
      heading: position.value.coords.heading,
      speed: position.value.coords.speed
    }
  })

  // 反向地理编码（模拟实现）
  const reverseGeocode = async (latitude: number, longitude: number): Promise<string> => {
    try {
      // 这里应该调用实际的地理编码服务
      // 暂时返回坐标字符串
      return `${latitude.toFixed(6)}, ${longitude.toFixed(6)}`
    } catch (err) {
      throw new Error('地址解析失败')
    }
  }

  return {
    position: readonly(position),
    coordinates: readonly(coordinates),
    error: readonly(error),
    loading: readonly(loading),
    isLoading: readonly(loading), // 别名
    getCurrentPosition,
    watchPosition,
    clearWatch,
    reverseGeocode
  }
}
