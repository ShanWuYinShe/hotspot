/**
 * 位置服务层
 * 处理位置相关的API调用，与后端用户位置API对接
 * 使用 Composition API 格式，提高代码复用性
 */

import { useApiService } from './api.factory'
import type { UpdateLocationRequest, UserActionSuccessResponse } from '~/types/api/user'

// 位置数据接口
export interface LocationData {
  latitude: number
  longitude: number
  location_name: string
  address?: string
  city?: string
  province?: string
  country?: string
  updated_at?: string
}

// 获取当前位置响应
export interface GetCurrentLocationResponse {
  location: LocationData
  status: string
}

/**
 * 位置服务 Composition API
 * 统一管理API服务实例，避免重复调用
 */
export function useLocationService() {
  const apiService = useApiService()

  /**
   * 更新用户位置
   */
  const updateLocation = async (data: UpdateLocationRequest): Promise<UserActionSuccessResponse> => {
    const result = await apiService.safeRequest<UserActionSuccessResponse>('/user/update_location', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 获取当前位置（前端地理位置API）
   */
  const getCurrentLocation = async (): Promise<GetCurrentLocationResponse> => {
    return new Promise((resolve, reject) => {
      if (!navigator.geolocation) {
        reject(new Error('浏览器不支持地理位置API'))
        return
      }

      navigator.geolocation.getCurrentPosition(
        (position) => {
          const { latitude, longitude } = position.coords
          resolve({
            location: {
              latitude,
              longitude,
              location_name: '当前位置',
              updated_at: new Date().toISOString()
            },
            status: 'success'
          })
        },
        (error) => {
          let message = '获取位置失败'
          switch (error.code) {
            case error.PERMISSION_DENIED:
              message = '用户拒绝了位置权限请求'
              break
            case error.POSITION_UNAVAILABLE:
              message = '位置信息不可用'
              break
            case error.TIMEOUT:
              message = '获取位置超时'
              break
          }
          reject(new Error(message))
        },
        {
          enableHighAccuracy: true,
          timeout: 10000,
          maximumAge: 60000
        }
      )
    })
  }

  /**
   * 监听位置变化
   */
  const watchLocation = (callback: (location: LocationData) => void): number => {
    if (!navigator.geolocation) {
      throw new Error('浏览器不支持地理位置API')
    }

    return navigator.geolocation.watchPosition(
      (position) => {
        const { latitude, longitude } = position.coords
        callback({
          latitude,
          longitude,
          location_name: '当前位置',
          updated_at: new Date().toISOString()
        })
      },
      (error) => {
        console.error('位置监听错误:', error)
      },
      {
        enableHighAccuracy: true,
        timeout: 10000,
        maximumAge: 60000
      }
    )
  }

  /**
   * 停止位置监听
   */
  const clearLocationWatch = (watchId: number): void => {
    navigator.geolocation.clearWatch(watchId)
  }

  return {
    updateLocation,
    getCurrentLocation,
    watchLocation,
    clearLocationWatch
  }
} 