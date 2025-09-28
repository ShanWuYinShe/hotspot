/**
 * 用户服务 - 现代化实现
 * 提供完整的用户管理功能
 */

import { BaseService } from './base.service'
import type { UserResponse, UserSettings, PaginatedResponse } from '~/types/api'

export interface UserSearchParams {
  query?: string
  latitude?: number
  longitude?: number
  radius?: number
  user_type?: number
  page?: number
  limit?: number
}

export interface UserUpdateData {
  nickname?: string
  bio?: string
  avatar_url?: string
  location_name?: string
  latitude?: number
  longitude?: number
}

export class UserService extends BaseService {
  protected readonly baseUrl = '/protected/user'

  /**
   * 获取用户资料
   */
  async getProfile(userId?: string): Promise<UserResponse> {
    const endpoint = userId ? `/profile/${userId}` : '/profile'
    return this.get<UserResponse>(endpoint)
  }

  /**
   * 更新用户资料
   */
  async updateProfile(data: UserUpdateData): Promise<UserResponse> {
    return this.put<UserResponse>('/profile', data)
  }

  /**
   * 更新用户设置
   */
  async updateSettings(settings: Partial<UserSettings>): Promise<UserSettings> {
    return this.put<UserSettings>('/settings', settings)
  }

  /**
   * 上传头像
   */
  async uploadAvatar(file: File): Promise<{ avatar_url: string }> {
    return this.upload<{ avatar_url: string }>('/avatar', file)
  }

  /**
   * 搜索用户
   */
  async searchUsers(params: UserSearchParams): Promise<PaginatedResponse<UserResponse>> {
    return this.paginate<UserResponse>('/search', params)
  }

  /**
   * 获取附近用户
   */
  async getNearbyUsers(
    latitude: number,
    longitude: number,
    radius: number = 5000
  ): Promise<PaginatedResponse<UserResponse>> {
    return this.get<PaginatedResponse<UserResponse>>('/nearby', {
      query: { latitude, longitude, radius }
    })
  }

  /**
   * 关注用户
   */
  async followUser(userId: string): Promise<void> {
    return this.post<void>(`/follow/${userId}`)
  }

  /**
   * 取消关注
   */
  async unfollowUser(userId: string): Promise<void> {
    return this.delete<void>(`/follow/${userId}`)
  }

  /**
   * 拉黑用户
   */
  async blockUser(userId: string): Promise<void> {
    return this.post<void>(`/block/${userId}`)
  }

  /**
   * 解除拉黑
   */
  async unblockUser(userId: string): Promise<void> {
    return this.delete<void>(`/block/${userId}`)
  }

  /**
   * 获取拉黑列表
   */
  async getBlockedUsers(): Promise<PaginatedResponse<UserResponse>> {
    return this.get<PaginatedResponse<UserResponse>>('/blocked')
  }

  /**
   * 删除账户
   */
  async deleteAccount(): Promise<void> {
    return this.delete<void>('/account')
  }
}

// 创建服务实例
export const userService = new UserService()

// 提供 composable 接口
export function useUsersService() {
  return userService
}

// 兼容旧的导出名称
export function useUserService() {
  return userService
}
