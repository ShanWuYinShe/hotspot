/**
 * 群组相关类型定义
 */

export interface GroupInfo {
  id: string
  name: string
  description?: string
  avatar_url?: string
  member_count: number
  admin_id: string
  created_at: string
  updated_at: string
}

export interface GroupMember {
  id: string
  user_id: string
  group_id: string
  role: 'admin' | 'member'
  nickname?: string
  join_time: string
  avatar_url?: string
}

export * from './chat.types'
