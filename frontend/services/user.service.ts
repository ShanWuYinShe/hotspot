/**
 * 用户服务层
 * 处理用户相关的API调用，与后端用户API对接
 * 使用 Composition API 格式，提高代码复用性
 */

import { useApiService } from './api.factory'
import type {
  SearchUserByLocationRequest,
  SearchUserByLocationResponse,
  SearchUserByNameRequest,
  SearchUserByNameResponse,
  SearchUserByIdRequest,
  SearchUserByIdResponse,
  UpdateLocationRequest,
  SetPasswordRequest,
  UserActionSuccessResponse
} from '~/types/api/user'

/**
 * 用户服务 Composition API
 * 统一管理API服务实例，避免重复调用
 */
export function useUserService() {
  const apiService = useApiService()

  /**
   * 按位置搜索用户
   */
  const searchUsersByLocation = async (params: SearchUserByLocationRequest): Promise<SearchUserByLocationResponse> => {
    const result = await apiService.safeRequest<SearchUserByLocationResponse>('/user/search_by_location', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 按名称搜索用户
   */
  const searchUsersByName = async (params: SearchUserByNameRequest): Promise<SearchUserByNameResponse> => {
    const result = await apiService.safeRequest<SearchUserByNameResponse>('/user/search_by_name', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 按ID搜索用户
   */
  const searchUsersById = async (params: SearchUserByIdRequest): Promise<SearchUserByIdResponse> => {
    const result = await apiService.safeRequest<SearchUserByIdResponse>('/user/search_by_id', params, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 更新用户位置
   */
  const updateUserLocation = async (data: UpdateLocationRequest): Promise<UserActionSuccessResponse> => {
    const result = await apiService.safeRequest<UserActionSuccessResponse>('/user/update_location', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  /**
   * 设置密码
   */
  const setUserPassword = async (data: SetPasswordRequest): Promise<UserActionSuccessResponse> => {
    const result = await apiService.safeRequest<UserActionSuccessResponse>('/user/set_password', data, { authRequired: true })

    if (!result) throw new Error('请求失败')

    return result
  }

  return {
    searchUsersByLocation,
    searchUsersByName,
    searchUsersById,
    updateUserLocation,
    setUserPassword
  }
}
