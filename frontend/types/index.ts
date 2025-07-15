// 统一导出所有类型定义

// API 类型
export * from './api'

// 业务类型
export * from './business/common'
export * from './business/group'
export * from './business/location'
export * from './business/message'
export * from './business/user'

// 新增动态社交相关类型
import type { UserListItem } from './business/user'

export interface Post {
  id: string;
  content: string;
  author: UserListItem;
  images?: string[];
  location?: string;
  mood?: string;
  tags?: string[];
  likes_count: number;
  comments_count: number;
  shares_count?: number;
  isLiked?: boolean;
  isSaved?: boolean;
  comments?: Comment[];
  created_at: string;
  updated_at?: string;
}

export interface Comment {
  id: string;
  content: string;
  author: UserListItem;
  post_id: string;
  parent_id?: string;
  likes_count: number;
  isLiked?: boolean;
  replies?: Comment[];
  created_at: string;
}

export interface UserStats {
  posts_count: number;
  followers_count: number;
  following_count: number;
  likes_received: number;
  groups_count: number;
  moments_count?: number;
}

export interface TrendingTopic {
  id: string;
  name: string;
  posts_count: number;
  growth_rate?: number;
  category?: string;
}
