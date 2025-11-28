/**
 * 生活瞬间服务
 * 所有API操作统一使用POST方法
 */

import { BaseService } from './base.service'
import type { MomentResponse, PaginatedData } from '~/types/api'

export interface CreateMomentRequest {
  description: string
  location: {
    latitude: number
    longitude: number
    name: string
  }
  tags?: string[]
  images?: string[]
}

export interface UpdateMomentRequest {
  id: string
  description?: string
  images?: string[]
  tags?: string[]
  location?: {
    latitude: number
    longitude: number
    name: string
  }
}

export interface ToggleLikeRequest {
  id: string
  action: 'like' | 'unlike'
}

export interface ToggleFavoriteRequest {
  id: string
  action: 'add' | 'remove'
}

export interface SearchByLocationRequest {
  latitude: number
  longitude: number
  radius?: number
  limit?: number
  cursor?: string
}

export interface SearchByTagsRequest {
  tags: string[]
  limit?: number
  cursor?: string
}

export interface OpenTheDoorRequest {
  id: string
}

export interface GetListRequest {
  user_id?: string
  limit?: number
  cursor?: string
}

export interface DeleteMomentRequest {
  id: string
}

export interface GetLikesRequest {
  id: string
  limit?: number
  cursor?: string
}

export interface GetFavoritesRequest {
  limit?: number
  cursor?: string
}

export interface AddCommentRequest {
  moment_id: string
  content: string
  parent_id?: string
}

export interface GetCommentsRequest {
  moment_id: string
  limit?: number
  cursor?: string
}

export interface DeleteCommentRequest {
  id: string
}

export interface ToggleLikeCommentRequest {
  id: string
  action: 'like' | 'unlike'
}

export class MomentsService extends BaseService {
  protected readonly baseUrl = ''

  /**
   * 创建生活瞬间
   * POST /api/v1/protected/moments/create
   */
  async createMoment(request: CreateMomentRequest): Promise<{ id: string }> {
    return this.post<{ id: string }>('/protected/moments/create', request)
  }

  /**
   * 编辑生活瞬间
   * POST /api/v1/protected/moments/update
   */
  async updateMoment(request: UpdateMomentRequest): Promise<void> {
    return this.post<void>('/protected/moments/update', request)
  }

  /**
   * 删除生活瞬间
   * POST /api/v1/protected/moments/delete
   */
  async deleteMoment(request: DeleteMomentRequest): Promise<void> {
    return this.post<void>('/protected/moments/delete', request)
  }

  /**
   * 点赞/取消点赞生活瞬间
   * POST /api/v1/protected/moments/toggle_like
   */
  async toggleLike(request: ToggleLikeRequest): Promise<void> {
    return this.post<void>('/protected/moments/toggle_like', request)
  }

  /**
   * 点赞生活瞬间（便捷方法）
   * 调用 toggleLike 并指定 action: 'like'
   */
  async likeMoment(momentId: string): Promise<void> {
    return this.toggleLike({ id: momentId, action: 'like' })
  }

  /**
   * 取消点赞生活瞬间（便捷方法）
   * 调用 toggleLike 并指定 action: 'unlike'
   */
  async unlikeMoment(momentId: string): Promise<void> {
    return this.toggleLike({ id: momentId, action: 'unlike' })
  }

  /**
   * 收藏/取消收藏瞬间
   * POST /api/v1/protected/moments/toggle_favorite
   */
  async toggleFavorite(request: ToggleFavoriteRequest): Promise<void> {
    return this.post<void>('/protected/moments/toggle_favorite', request)
  }

  /**
   * 获取收藏列表
   * POST /api/v1/protected/moments/get_favorites
   */
  async getFavorites(request: GetFavoritesRequest = {}): Promise<PaginatedData<MomentResponse>> {
    return this.post<PaginatedData<MomentResponse>>('/protected/moments/get_favorites', request)
  }

  /**
   * 按位置搜索生活瞬间（公开接口）
   * POST /api/v1/public/moments/search_by_location
   */
  async searchByLocation(request: SearchByLocationRequest): Promise<PaginatedData<MomentResponse>> {
    return this.publicPost<PaginatedData<MomentResponse>>('/public/moments/search_by_location', request)
  }

  /**
   * 按标签搜索生活瞬间（公开接口）
   * POST /api/v1/public/moments/search_by_tags
   */
  async searchByTags(request: SearchByTagsRequest): Promise<PaginatedData<MomentResponse>> {
    return this.publicPost<PaginatedData<MomentResponse>>('/public/moments/search_by_tags', request)
  }

  /**
   * 按ID获取生活瞬间详情（公开接口）
   * POST /api/v1/public/moments/open_the_door
   */
  async openTheDoor(request: OpenTheDoorRequest): Promise<MomentResponse> {
    return this.publicPost<MomentResponse>('/public/moments/open_the_door', request)
  }

  /**
   * 获取生活瞬间历史（公开接口）
   * POST /api/v1/public/moments/get_list
   */
  async getList(request: GetListRequest = {}): Promise<PaginatedData<MomentResponse>> {
    return this.publicPost<PaginatedData<MomentResponse>>('/public/moments/get_list', request)
  }

  /**
   * 获取生活瞬间点赞列表（公开接口）
   * POST /api/v1/public/moments/get_likes
   */
  async getLikes(request: GetLikesRequest): Promise<PaginatedData<{
    user: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    liked_at: string
  }>> {
    return this.publicPost('/public/moments/get_likes', request)
  }

  /**
   * 添加评论/回复
   * POST /api/v1/protected/comments/add
   */
  async addComment(request: AddCommentRequest): Promise<{ id: string }> {
    return this.post<{ id: string }>('/protected/comments/add', request)
  }

  /**
   * 获取评论列表（公开接口）
   * POST /api/v1/public/comments/get_list
   */
  async getComments(request: GetCommentsRequest): Promise<PaginatedData<{
    id: string
    user: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    content: string
    parent_id?: string
    reply_to_user?: {
      user_id: string
      nickname: string
      avatar_url?: string
    }
    created_at: string
    likes_count: number
    replies_count: number
  }>> {
    return this.publicPost('/public/comments/get_list', request)
  }

  /**
   * 删除评论
   * POST /api/v1/protected/comments/remove
   */
  async deleteComment(request: DeleteCommentRequest): Promise<void> {
    return this.post<void>('/protected/comments/remove', request)
  }

  /**
   * 点赞/取消点赞评论
   * POST /api/v1/protected/comments/toggle_like
   */
  async toggleLikeComment(request: ToggleLikeCommentRequest): Promise<void> {
    return this.post<void>('/protected/comments/toggle_like', request)
  }

  /**
   * 上传动态图片（使用FormData）
   */
  async uploadImages(files: File[]): Promise<{ image_urls: string[] }> {
    const formData = new FormData()
    files.forEach((file, index) => {
      formData.append(`images[${index}]`, file)
    })

    const builder = this.http.post(
      this.buildUrl('/protected/media/upload_file'),
      formData
    ).auth()
    // 让浏览器自动设置 Content-Type
    
    const response = await builder.send<{ image_urls: string[] }>()
    return response.data
  }

  /**
   * 获取附近动态（别名方法，调用 searchByLocation）
   */
  async getNearbyMoments(request: SearchByLocationRequest): Promise<PaginatedData<MomentResponse>> {
    return this.searchByLocation(request)
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
