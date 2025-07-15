/**
 * 群组业务类型定义
 * 基于后端API响应的群组信息结构
 */

// 群组信息（来自后端API）
export interface GroupInfo {
  group_id: string // UUID
  name: string
  description?: string
  owner_id: number
  creator_name: string
  created_at: string
  last_active_at: string
  latitude: number
  longitude: number
  member_count: number
  distance: number
  location_name: string
  is_password_required: boolean
  is_member: boolean
  user_role: string
  avatar_url?: string
}

// 群组成员信息（来自后端API）
export interface GroupMember {
  id: number
  user_id: number
  nickname: string
  last_active_at: string
  role: string
  join_time: string
  avatar_url?: string
  is_muted?: boolean
}
