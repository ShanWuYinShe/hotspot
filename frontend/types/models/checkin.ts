/**
 * 签到业务类型定义
 * 基于后端API响应的签到信息结构
 */

// ==================== 核心业务类型 ====================

// 签到信息（来自后端API）
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

// ==================== 签到统计 ====================

export interface CheckinStats {
  total_checkins: number
  today_checkins: number
  this_week_checkins: number
  this_month_checkins: number
  unique_locations: number
  total_likes_received: number
}

// ==================== 位置信息 ====================

export interface CheckinLocation {
  latitude: number
  longitude: number
  location_name: string
  address?: string
  city?: string
  province?: string
  country?: string
}

// ==================== 签到过滤和搜索 ====================

export interface CheckinFilter {
  date_range?: {
    start: string
    end: string
  }
  location_range?: {
    latitude: number
    longitude: number
    radius: number // 米
  }
  tags?: string[]
  user_ids?: number[]
  has_images?: boolean
  min_likes?: number
}

export interface CheckinSort {
  field: 'created_at' | 'like_count' | 'distance'
  direction: 'asc' | 'desc'
}

// ==================== UI 相关类型 ====================

export interface CheckinDisplayOptions {
  show_location: boolean
  show_tags: boolean
  show_stats: boolean
  enable_like: boolean
  enable_comment: boolean
  enable_share: boolean
}

export interface CheckinFormData {
  description: string
  location: CheckinLocation
  tags: string[]
  images?: File[]
  privacy: 'public' | 'friends' | 'private'
}

// ==================== 地图相关 ====================

export interface CheckinMapMarker {
  checkin_id: string
  position: {
    latitude: number
    longitude: number
  }
  title: string
  description: string
  user_nickname: string
  created_at: string
  like_count: number
  tags: string[]
}
