import { BaseService } from './base.service'
import type { CommentListItem, CommentLikeItem, PaginatedData } from '~/types/api'

/**
 * 评论服务
 * ✅ 类型定义已统一从 types/api.ts 导入
 */

// 导出类型别名以保持向后兼容
export type CommentItem = CommentListItem

// ==================== Request 类型定义 ====================

export interface CreateCommentRequest {
  moment_id: string
  content: string
  parent_id?: string
}

export interface CreateCommentData {
  id: string
}

export interface DeleteCommentRequest {
  id: string
}

export interface ListCommentsRequest {
  moment_id: string
  limit?: number
  cursor?: string
}

export interface ListRepliesRequest {
  id: string
  limit?: number
  cursor?: string
}

export interface LikeCommentRequest {
  id: string
  action: 'like' | 'unlike'
}

export interface GetCommentLikesRequest {
  id: string
  limit?: number
  cursor?: string
}

// ==================== 服务类 ====================

export class CommentsService extends BaseService {
  protected readonly baseUrl = ''

  /**
   * 添加评论/回复
   */
  async create(data: CreateCommentRequest): Promise<CreateCommentData> {
    return this.post<CreateCommentData>('/protected/comments/add', data)
  }

  /**
   * 删除评论
   */
  async delete(data: DeleteCommentRequest): Promise<void> {
    return this.post<void>('/protected/comments/remove', data)
  }

  /**
   * 获取评论列表
   */
  async getList(params: ListCommentsRequest): Promise<PaginatedData<CommentListItem>> {
    return this.publicPost<PaginatedData<CommentListItem>>('/public/comments/get_list', params)
  }

  /**
   * 获取子评论列表
   */
  async getReplies(params: ListRepliesRequest): Promise<PaginatedData<CommentListItem>> {
    return this.publicPost<PaginatedData<CommentListItem>>('/public/comments/get_replies', params)
  }

  /**
   * 点赞/取消点赞评论
   */
  async toggleLike(data: LikeCommentRequest): Promise<void> {
    return this.post<void>('/protected/comments/toggle_like', data)
  }

  /**
   * 获取评论点赞列表
   */
  async getLikes(params: GetCommentLikesRequest): Promise<PaginatedData<CommentLikeItem>> {
    return this.publicPost<PaginatedData<CommentLikeItem>>('/public/comments/get_likes', params)
  }
}

export const commentsService = new CommentsService()

export function useCommentsService() {
  return commentsService
}


