/**
 * 签到服务层
 * 处理用户签到相关的API调用，与后端checkin API对接
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
 * 签到服务 Composition API
 * 统一管理API服务实例，避免重复调用
 */
export function useCheckinService() {
  const apiService = useApiService()

  /**
   * 创建签到
   */
  const createCheckin = async (data: CreateCheckinRequest): Promise<CreateCheckinResponse> => {
    const result = await apiService.safeRequest<CreateCheckinResponse>('/checkin/create', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 获取签到详情
   */
  const getCheckinDetails = async (params: SearchCheckInByIdRequest): Promise<CheckInInfo> => {
    const result = await apiService.safeRequest<CheckInInfo>('/checkin/search_by_id', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 删除签到
   */
  const deleteCheckin = async (data: DeleteCheckinRequest): Promise<DeleteCheckinResponse> => {
    const result = await apiService.safeRequest<DeleteCheckinResponse>('/checkin/delete', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 按位置搜索签到
   */
  const getNearbyCheckins = async (params: SearchCheckinsByLocationRequest): Promise<SearchCheckinByLocationResponse> => {
    const result = await apiService.safeRequest<SearchCheckinByLocationResponse>('/checkin/search_by_location', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 获取用户签到历史
   */
  const getUserCheckins = async (params: UserCheckinsHistoryRequest): Promise<UserCheckinsHistoryResponse> => {
    const result = await apiService.safeRequest<UserCheckinsHistoryResponse>('/checkin/history', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 按标签搜索签到
   */
  const searchCheckinsByTags = async (params: SearchCheckinsByTagsRequest): Promise<SearchCheckinsByTagsResponse> => {
    const result = await apiService.safeRequest<SearchCheckinsByTagsResponse>('/checkin/search_by_tags', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 点赞签到
   */
  const likeCheckin = async (data: LikeCheckinRequest): Promise<LikeCheckinResponse> => {
    const result = await apiService.safeRequest<LikeCheckinResponse>('/checkin/like', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 取消点赞签到
   */
  const unlikeCheckin = async (data: UnlikeCheckinRequest): Promise<UnlikeCheckinResponse> => {
    const result = await apiService.safeRequest<UnlikeCheckinResponse>('/checkin/unlike', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  return {
    createCheckin,
    getCheckinDetails,
    deleteCheckin,
    getNearbyCheckins,
    getUserCheckins,
    searchCheckinsByTags,
    likeCheckin,
    unlikeCheckin
  }
}