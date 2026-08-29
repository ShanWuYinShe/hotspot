// ==================== 基础类型 ====================
import { CommentItem, MomentItem, UserInfo } from './entities';


export interface ApiResponse<T = void> {
  code: number;
  message?: string;
  data?: T;
}

// 通用分页容器（spec types.md：next_cursor 恒在，null = 没有更多数据）
export interface PaginatedData<T> {
  items: T[];
  next_cursor: string | null;
}

export interface PaginationParams {
  limit?: number;
  cursor?: string;
}

// ==================== 通用类型定义 (Domain Primitives) ====================

// 消息类型枚举
export type MessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'share' | 'system';
export type SendMessageType = 'text' | 'image' | 'video' | 'audio' | 'file' | 'share';

// 用户类型枚举
export type UserType = 'temp' | 'normal';

// 隐私级别
export type PrivacyLevel = 'public' | 'private';

// 角色
export type GroupRole = 'owner' | 'admin' | 'member';
export type SettableGroupRole = 'admin' | 'member';

// 客户端类型
export type ClientType = 'web' | 'app';

// 设备标识（spec: DeviceId = UUID v4），持久化、驱动单设备单会话踢人语义
export type DeviceId = string;

// 用户在线状态
export type UserPresenceStatus = 'online' | 'offline';

// 举报状态
export type ReportStatus = 'pending' | 'resolved' | 'dismissed';

// 举报目标类型
export type ReportTargetType = 'user' | 'moment' | 'comment' | 'message' | 'group';

// 举报处理操作
export type ReportAction = 'resolve' | 'dismiss';

export type ReportReason =
  | 'spam'
  | 'harassment'
  | 'violence'
  | 'hate_speech'
  | 'sexual_content'
  | 'misinformation'
  | 'impersonation'
  | 'self_harm'
  | 'other';

// 群组准入方式（叠加 requires_approval 决定是否需审核）
export type GroupJoinMethod = 'direct' | 'password' | 'invite_only';

// 公告严重程度 (spec: AnnounceSeverity)
export type AnnounceSeverity = 'info' | 'warning' | 'error';

// spec Reaction: reactions 聚合 (MomentItem/CommentItem/MessageItem 共用)
export interface Reaction {
  emoji: string;
  count: number;
  reacted_by_me: boolean;
}

// spec ReactionItem: reactions/fetch 返回的反应者列表项
export interface ReactionItem {
  user: UserInfo;
  emoji: string;
  reacted_at: string;
}

// 关注来源
export type FollowSource = 'moment' | 'search' | 'recommendation' | 'profile' | 'group';

// 推荐理由类型
export type RecommendationType = 'friend_liked' | 'nearby' | 'trending' | 'following' | 'tag_match';

// 账号状态（系统管理员视角）
export type UserAccountStatus = 'normal' | 'banned';

// ==================== 位置与地理信息 ====================

// 基础坐标点
export interface GeoPoint {
  latitude: number;
  longitude: number;
}

// 命名位置 (用于 Request)
export interface Location {
  point: GeoPoint;
  location_name: string;
}

// 相对位置 (用于 Response)
export interface RelativeLocation {
  point: GeoPoint;
  location_name: string;
  distance: number | null;
  bearing_degrees: number | null;
}

// 视口边界
export interface Bounds {
  north: number;
  south: number;
  east: number;
  west: number;
}

export interface MapViewport {
  bounds: Bounds;
}

// ==================== 媒体资源 ====================

export interface MomentImage {
  url: string;
  thumbnail_url: string | null;
  width: number | null;
  height: number | null;
  mime_type: string | null;
}

export interface MessageMedia {
  url: string;
  thumbnail_url: string | null;
  file_name: string | null;
  file_size: number | null;
  duration: number | null;
  width: number | null;
  height: number | null;
  mime_type: string | null;
}
