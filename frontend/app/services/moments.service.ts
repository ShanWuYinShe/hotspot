/**
 * 动态服务 - 现代化实现
 * 提供完整的动态管理功能
 */

import { BaseService } from './base.service'
import type { 
  MomentResponse, 
  PaginatedResponse 
} from '~/types/api'

export interface CreateMomentRequest {
  content: string
  images?: string[]
  location_name?: string
  latitude?: number
  longitude?: number
  tags?: string[]
  visibility: 'public' | 'friends' | 'private'
}

export interface UpdateMomentRequest {
  content?: string
  location_name?: string
  latitude?: number
  longitude?: number
  tags?: string[]
  visibility?: 'public' | 'friends' | 'private'
}

export interface MomentComment {
  comment_id: string
  moment_id: string
  user_id: string
  user_name: string
  user_avatar?: string
  content: string
  created_at: string
  updated_at: string
}

export class MomentsService extends BaseService {
  protected readonly baseUrl = '/protected/moments'

  /**
   * 获取附近动态
   */
  async getNearbyMoments(params: {
    latitude: number
    longitude: number
    radius?: number
    cursor?: string
    limit?: number
  }): Promise<PaginatedResponse<MomentResponse>> {
    // 使用公开接口
    const http = useHttp()
    return http.get<PaginatedResponse<MomentResponse>>('/public/moments/nearby', {
      query: params
    })
  }

  /**
   * 获取关注的动态
   */
  async getFollowingMoments(params?: {
    cursor?: string
    limit?: number
  }): Promise<PaginatedResponse<MomentResponse>> {
    return this.paginate<MomentResponse>('/following', params)
  }

  /**
   * 获取我的动态
   */
  async getMyMoments(params?: {
    cursor?: string
    limit?: number
  }): Promise<PaginatedResponse<MomentResponse>> {
    return this.paginate<MomentResponse>('/mine', params)
  }

  /**
   * 获取用户动态
   */
  async getUserMoments(
    userId: string,
    params?: {
      cursor?: string
      limit?: number
    }
  ): Promise<PaginatedResponse<MomentResponse>> {
    // 使用公开接口
    const http = useHttp()
    return http.get<PaginatedResponse<MomentResponse>>(`/public/moments/user/${userId}`, {
      query: params
    })
  }

  /**
   * 获取动态详情
   */
  async getMomentDetails(momentId: string): Promise<MomentResponse> {
    // 使用公开接口
    const http = useHttp()
    return http.get<MomentResponse>(`/public/moments/${momentId}`)
  }

  /**
   * 创建动态
   */
  async createMoment(data: CreateMomentRequest): Promise<{ moment_id: string }> {
    return this.post<{ moment_id: string }>('/create', data)
  }

  /**
   * 更新动态
   */
  async updateMoment(momentId: string, data: UpdateMomentRequest): Promise<MomentResponse> {
    return this.put<MomentResponse>(`/${momentId}`, data)
  }

  /**
   * 删除动态
   */
  async deleteMoment(momentId: string): Promise<void> {
    return this.delete<void>(`/${momentId}`)
  }

  /**
   * 上传动态图片
   */
  async uploadImages(files: File[]): Promise<{ image_urls: string[] }> {
    const formData = new FormData()
    files.forEach((file, index) => {
      formData.append(`images[${index}]`, file)
    })

    return this.http.post<{ image_urls: string[] }>(this.buildUrl('/upload-images'), formData, {
      auth: true,
      headers: {
        // 让浏览器自动设置 Content-Type
      }
    })
  }

  /**
   * 点赞动态
   */
  async likeMoment(momentId: string): Promise<void> {
    return this.post<void>(`/like/${momentId}`)
  }

  /**
   * 取消点赞
   */
  async unlikeMoment(momentId: string): Promise<void> {
    return this.delete<void>(`/like/${momentId}`)
  }

  /**
   * 获取动态点赞列表
   */
  async getMomentLikes(
    momentId: string,
    params?: {
      cursor?: string
      limit?: number
    }
  ): Promise<PaginatedResponse<{
    user_id: string
    user_name: string
    user_avatar?: string
    liked_at: string
  }>> {
    return this.paginate<{
      user_id: string
      user_name: string
      user_avatar?: string
      liked_at: string
    }>(`/likes/${momentId}`, params)
  }

  /**
   * 添加评论
   */
  async addComment(momentId: string, content: string): Promise<MomentComment> {
    return this.post<MomentComment>('/comments', {
      moment_id: momentId,
      content
    })
  }

  /**
   * 获取动态评论
   */
  async getMomentComments(
    momentId: string,
    params?: {
      cursor?: string
      limit?: number
    }
  ): Promise<PaginatedResponse<MomentComment>> {
    return this.paginate<MomentComment>(`/comments/${momentId}`, params)
  }

  /**
   * 删除评论
   */
  async deleteComment(commentId: string): Promise<void> {
    return this.delete<void>(`/comments/${commentId}`)
  }

  /**
   * 搜索动态
   */
  async searchMoments(params: {
    query?: string
    tags?: string[]
    latitude?: number
    longitude?: number
    radius?: number
    user_id?: string
    start_date?: string
    end_date?: string
    cursor?: string
    limit?: number
  }): Promise<PaginatedResponse<MomentResponse>> {
    // 使用公开接口
    const http = useHttp()
    return http.get<PaginatedResponse<MomentResponse>>('/public/moments/search', {
      query: params
    })
  }

  /**
   * 获取热门动态
   */
  async getTrendingMoments(params?: {
    time_range?: 'day' | 'week' | 'month'
    cursor?: string
    limit?: number
  }): Promise<PaginatedResponse<MomentResponse>> {
    // 使用公开接口
    const http = useHttp()
    return http.get<PaginatedResponse<MomentResponse>>('/public/moments/trending', {
      query: params
    })
  }

  /**
   * 获取动态统计
   */
  async getMomentStats(momentId: string): Promise<{
    likes_count: number
    comments_count: number
    shares_count: number
    views_count: number
  }> {
    // 使用公开接口
    const http = useHttp()
    return http.get<{
      likes_count: number
      comments_count: number
      shares_count: number
      views_count: number
    }>(`/public/moments/stats/${momentId}`)
  }

  /**
   * 举报动态
   */
  async reportMoment(
    momentId: string, 
    reason: string, 
    description?: string
  ): Promise<void> {
    return this.post<void>('/report', {
      moment_id: momentId,
      reason,
      description
    })
  }

  /**
   * 分享动态
   */
  async shareMoment(momentId: string, platform?: string): Promise<{
    share_url: string
    share_count: number
  }> {
    return this.post<{
      share_url: string
      share_count: number
    }>(`/share/${momentId}`, { platform })
  }

  /**
   * 收藏动态
   */
  async favoriteMoment(momentId: string): Promise<void> {
    return this.post<void>(`/favorite/${momentId}`)
  }

  /**
   * 取消收藏
   */
  async unfavoriteMoment(momentId: string): Promise<void> {
    return this.delete<void>(`/favorite/${momentId}`)
  }

  /**
   * 获取收藏的动态
   */
  async getFavoriteMoments(params?: {
    cursor?: string
    limit?: number
  }): Promise<PaginatedResponse<MomentResponse>> {
    return this.paginate<MomentResponse>('/favorites', params)
  }

  /**
   * 获取动态标签建议
   */
  async getTagSuggestions(query: string): Promise<string[]> {
    const response = await this.get<{ tags: string[] }>('/tag-suggestions', {
      query: { q: query }
    })
    return response.tags
  }

  /**
   * 获取热门标签
   */
  async getTrendingTags(): Promise<Array<{
    tag: string
    count: number
  }>> {
    // 使用公开接口
    const http = useHttp()
    const response = await http.get<{ tags: Array<{ tag: string; count: number }> }>('/public/moments/trending-tags')
    return response.tags
  }
}

// 创建服务实例
export const momentsService = new MomentsService()

// 提供 composable 接口
export function useMomentsService() {
  return momentsService
}

// 兼容旧的导出名称
export function useMomentService() {
  return momentsService
}
