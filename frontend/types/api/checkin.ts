/**
 * 签到 API 类型定义
 * 与后端 checkin API 严格对应
 */

import type { Pagination, PaginatedResponse } from '~/types/api'

// ==================== 请求类型 ====================

export interface CreateCheckinRequest {
  description: string
  latitude: number
  longitude: number
  location_name: string
  tags?: string[]
}

export interface DeleteCheckinRequest {
  checkin_id: string
}

export interface SearchCheckInByIdRequest {
  checkin_id: string
}

export interface SearchCheckinsByLocationRequest extends Pagination {
  latitude: number
  longitude: number
  radius: number
}

export interface UserCheckinsHistoryRequest extends Pagination {
  user_id?: number
}

export interface SearchCheckinsByTagsRequest extends Pagination {
  tags: string[]
  latitude: number
  longitude: number
}

export interface LikeCheckinRequest {
  checkin_id: string
}

export interface UnlikeCheckinRequest {
  checkin_id: string
}

// ==================== 响应类型 ====================

export interface CreateCheckinResponse {
  checkin_id: string
}

export interface DeleteCheckinResponse {
  success: boolean
}

export interface LikeCheckinResponse {
  success: boolean
  likes_count: number
}

export interface UnlikeCheckinResponse {
  success: boolean
  likes_count: number
}

// ==================== 数据模型 ====================

export interface CheckInInfo {
  id: number
  checkin_id: string
  user_id: number
  nickname: string
  description?: string
  latitude: number
  longitude: number
  location_name: string
  distance: number
  created_at: string
  tags: string[]
  likes_count: number
  comments_count: number
  liked_by_me: boolean
}

// ==================== 分页响应 ====================

export type UserCheckinsHistoryResponse = PaginatedResponse<CheckInInfo>
export type SearchCheckinByLocationResponse = PaginatedResponse<CheckInInfo>
export type SearchCheckinsByTagsResponse = PaginatedResponse<CheckInInfo>
