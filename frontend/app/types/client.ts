/**
 * 客户端本地扩展类型
 * 
 * 这些类型仅用于前端组件和逻辑，不是 API 文档定义的类型
 */

import type { 
  UserInfo, 
  GroupInfo, 
  UserRelationship, 
  GeoData, 
  GroupSettings, 
  GroupSearchItem, 
  PrivacyLevel, 
  UserSearchItem, 
  MomentSearchItem,
  UserType,
  MessageType as ApiMessageType
} from './api'

// ============================================================================
// 用户存储类型（客户端本地）
// ============================================================================

/**
 * 用户详情扩展类型（前端组件使用）
 * 组合自 UserSearchItem 并添加前端需要的字段
 */
export interface UserDetailData extends UserSearchItem {
  user_type?: UserType;
  created_at?: string;
  last_active_at?: string;
  relationship?: UserRelationship;
  geo: GeoData; // 覆盖基类，要求必填
}

// ============================================================================
// 地理位置类型
// ============================================================================

/**
 * 地理坐标（客户端使用）
 */
export interface Coordinates {
  latitude: number;
  longitude: number;
  accuracy?: number;
  timestamp?: number;
}

// ============================================================================
// 组件渲染类型
// ============================================================================

/**
 * 群组详情扩展类型（前端组件使用）
 * 组合 GroupSearchItem + GroupSettings 的客户端扩展
 * API 规范中不存在此类型，仅用于前端组件渲染
 */
export interface GroupDetailData extends GroupSearchItem {
  settings: GroupSettings;
  created_at: string;
  last_activity_at?: string;
}

/**
 * 群组卡片渲染类型（前端组件使用）
 * 合并了 GroupDetailData、GroupSearchItem 和 JoinedGroupItem 的字段
 * 用于 GroupCard 等需要渲染不同来源群组数据的组件
 */
export interface GroupCardData extends GroupDetailData {
  // 来自 JoinedGroupItem
  my_role?: 'owner' | 'admin' | 'member'; // 当前用户角色
  is_member?: boolean;      // 当前用户是否为成员
  unread_count?: number;    // 未读消息数
}

/**
 * 动态卡片渲染类型（前端组件使用）
 * 扩展 MomentSearchItem，添加用户关系和点赞状态
 */
export interface MomentCardData extends MomentSearchItem {
  // 展平结构，避免重新定义 user 类型
  relationship?: Partial<UserRelationship>; // 与动态作者的关系
  liked_by_me?: boolean;
}

// ============================================================================
// 用户设置本地扩展类型
// ============================================================================

/**
 * 前端完整用户设置（组合 API 设置 + 本地设置）
 */
export interface UserSettings {
  // 通知设置
  notifications: {
    push_enabled: boolean;
    email_enabled: boolean;
    sms_enabled: boolean;
    message_notifications: boolean;
    group_notifications: boolean;
    new_follower: boolean;
    mutual_follow: boolean;
    direct_message_toast: boolean;
    group_message_toast: boolean;
    system_notification_toast: boolean;
    moment_notifications: boolean;
    comment_notifications: boolean;
  };
  // 隐私设置
  privacy: {
    privacy_level: PrivacyLevel;
    location_sharing: boolean;
    online_status: boolean;
    show_in_search: boolean;
    allow_private_chat: boolean;
  };
  // 外观设置（前端本地）
  appearance: {
    language: string;
    animations: boolean;
    compact_mode: boolean;
    high_contrast: boolean;
  };
  // 偏好设置（前端本地）
  preferences: {
    language: string;
    distance_unit: string;
    time_format: string;
  };
  // 账户设置
  account: {
    auto_save: boolean;
    backup_enabled: boolean;
    nickname?: string;
    bio?: string;
    email?: string;
    phone?: string;
    avatar_url?: string;
  };
}

// ============================================================================
// 通用客户端类型
// ============================================================================

/**
 * 消息类型枚举（客户端使用）
 * 扩展 API 消息类型，添加客户端特有的 location 类型
 */
export type ClientMessageType = ApiMessageType | 'location';

/**
 * 基础成功响应
 */
export interface BasicSuccess {
  success: boolean;
  message?: string;
}

/**
 * 未读消息统计（客户端使用）
 */
export interface UnreadCount {
  total_unread: number;
  conversation_unread: number;
  group_unread: number;
}

/**
 * 上传配置（客户端使用）
 */
export interface UploadConfig {
  max_file_size: number;
  allowed_types: string[];
  max_files_per_batch: number;
}

// ============================================================================
// 通知扩展类型
// ============================================================================

/**
 * 通知数据（客户端扩展）
 */
export interface NotificationData {
  follower?: UserInfo;
  is_mutual?: boolean;
  group?: GroupInfo;
  inviter?: UserInfo;
  applicant?: UserInfo;
  moment_id?: string;
  comment_id?: string;
  message_id?: string;
  [key: string]: string | number | boolean | UserInfo | GroupInfo | undefined;
}
