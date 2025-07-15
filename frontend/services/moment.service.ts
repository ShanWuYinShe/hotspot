/**
 * 时刻动态服务层 (原签到服务，重命名为更贴切的"时刻/动态")
 * 处理用户位置分享、动态发布相关的API调用，与后端checkin API对接
 * 使用 Composition API 格式，提高代码复用性
 */

import { useApiService } from './api.factory'
import type {
  CreateCheckinRequest,
  CreateCheckinResponse,
  DeleteCheckinRequest,
  DeleteCheckinResponse,
  SearchCheckInByIdRequest,
  SearchCheckinsByLocationRequest,
  SearchCheckinByLocationResponse,
  UserCheckinsHistoryRequest,
  UserCheckinsHistoryResponse,
  SearchCheckinsByTagsRequest,
  SearchCheckinsByTagsResponse,
  LikeCheckinRequest,
  LikeCheckinResponse,
  UnlikeCheckinRequest,
  UnlikeCheckinResponse,
  CheckInInfo
} from '~/types/api/checkin'

/**
 * 时刻动态服务 Composition API
 * 统一管理API服务实例，避免重复调用
 */
export function useMomentService() {
  const apiService = useApiService()

  /**
   * 创建时刻动态
   */
  const createMoment = async (data: CreateCheckinRequest): Promise<CreateCheckinResponse> => {
    const result = await apiService.safeRequest<CreateCheckinResponse>('/checkin/create', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 获取时刻详情
   */
  const getMomentDetails = async (params: SearchCheckInByIdRequest): Promise<CheckInInfo> => {
    const result = await apiService.safeRequest<CheckInInfo>('/checkin/search_by_id', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 删除时刻动态
   */
  const deleteMoment = async (data: DeleteCheckinRequest): Promise<DeleteCheckinResponse> => {
    const result = await apiService.safeRequest<DeleteCheckinResponse>('/checkin/delete', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 获取附近时刻
   */
  const getNearbyMoments = async (params: SearchCheckinsByLocationRequest): Promise<SearchCheckinByLocationResponse> => {
    const result = await apiService.safeRequest<SearchCheckinByLocationResponse>('/checkin/search_by_location', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 获取用户时刻历史
   */
  const getUserMoments = async (params: UserCheckinsHistoryRequest): Promise<UserCheckinsHistoryResponse> => {
    const result = await apiService.safeRequest<UserCheckinsHistoryResponse>('/checkin/history', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 按标签搜索时刻
   */
  const searchMomentsByTags = async (params: SearchCheckinsByTagsRequest): Promise<SearchCheckinsByTagsResponse> => {
    const result = await apiService.safeRequest<SearchCheckinsByTagsResponse>('/checkin/search_by_tags', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 点赞时刻
   */
  const likeMoment = async (data: LikeCheckinRequest): Promise<LikeCheckinResponse> => {
    const result = await apiService.safeRequest<LikeCheckinResponse>('/checkin/like', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 取消点赞时刻
   */
  const unlikeMoment = async (data: UnlikeCheckinRequest): Promise<UnlikeCheckinResponse> => {
    const result = await apiService.safeRequest<UnlikeCheckinResponse>('/checkin/unlike', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 切换点赞状态
   */
  const toggleMomentLike = async (data: LikeCheckinRequest): Promise<{ liked: boolean; like_count: number }> => {
    try {
      const response = await likeMoment(data)
      return {
        liked: true,
        like_count: response.likes_count
      }
    } catch (error) {
      const response = await unlikeMoment(data)
      return {
        liked: false,
        like_count: response.likes_count
      }
    }
  }

  return {
    createMoment,
    getMomentDetails,
    deleteMoment,
    getNearbyMoments,
    getUserMoments,
    searchMomentsByTags,
    likeMoment,
    unlikeMoment,
    toggleMomentLike
  }
}