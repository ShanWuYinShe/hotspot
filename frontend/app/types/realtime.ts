/**
 * WebSocket 实时消息类型定义
 * 根据需求文档统一的WebSocket消息类型系统
 * 与后端unified_message_types.rs保持完全一致
 * 
 * 重要说明：
 * - 所有消息类型必须与后端保持严格一致
 * - 字段名称和类型必须完全匹配
 * - 枚举值必须与后端定义相同
 */

// ============================================================================
// 基础类型定义
// ============================================================================

export type UserStatus = 'Online' | 'Away' | 'Busy' | 'Offline'
export type MessagePriority = 'low' | 'normal' | 'high' | 'critical'

// 位置更新
export interface LocationUpdate {
  latitude: number
  longitude: number
  location_name?: string
  accuracy?: number
}

// ============================================================================
// 消息载荷数据结构 - 与后端unified_message_types.rs保持一致
// ============================================================================

// 媒体附件
export interface MediaAttachment {
  media_id: string
  media_type: string
  file_url: string
  thumbnail_url?: string
  file_name?: string
  file_size?: number
  width?: number
  height?: number
  duration?: number
}

// 消息载荷
export interface MessagePayload {
  message_id: string
  group_id: string
  sender_id: string
  sender_nickname: string
  sender_avatar?: string
  content: string
  message_type: string
  reply_to_message_id?: string
  mentioned_users: string[]
  created_at: string
  edited_at?: string
  latitude?: number
  longitude?: number
  media_attachments: MediaAttachment[]
}

// 生活瞬间载荷
export interface MomentPayload {
  moment_id: string
  user_id: string
  nickname: string
  user_avatar?: string
  description: string
  latitude: number
  longitude: number
  location_name: string
  images: string[]
  tags: string[]
  likes_count: number
  comments_count: number
  created_at: string
}

// 评论载荷
export interface CommentPayload {
  comment_id: string
  moment_id: string
  user_id: string
  nickname: string
  user_avatar?: string
  content: string
  parent_comment_id?: string
  replies_count: number
  created_at: string
}

// 通知载荷
export interface NotificationPayload {
  notification_id: string
  notification_type: string
  title: string
  content: string
  data: Record<string, any>
  created_at: string
  expires_at?: string
}

// ============================================================================
// 服务器到客户端的消息类型 - 与后端unified_message_types.rs保持一致
// ============================================================================

export type ServerMessage =
  | { type: 'Connected'; data: { user_id: string; session_id: string } }
  | { type: 'Pong' }
  | { type: 'AuthSuccess'; data: { user_id: string; nickname: string } }
  | { type: 'AuthError'; data: { error: string } }
  | { type: 'NewMessage'; data: { message: MessagePayload } }
  | { 
      type: 'MessageRead'; 
      data: { 
        group_id: string; 
        user_id: string; 
        message_id: string; 
        read_at: string 
      } 
    }
  | { 
      type: 'TypingIndicator'; 
      data: { 
        group_id: string; 
        user_id: string; 
        nickname: string; 
        is_typing: boolean 
      } 
    }
  | { 
      type: 'UserOnline'; 
      data: { 
        user_id: string; 
        nickname: string; 
        status: UserStatus 
      } 
    }
  | { 
      type: 'UserOffline'; 
      data: { 
        user_id: string; 
        nickname: string 
      } 
    }
  | { 
      type: 'UserLocationUpdate'; 
      data: { 
        user_id: string; 
        nickname: string; 
        location: LocationUpdate 
      } 
    }
  | { 
      type: 'GroupMemberJoined'; 
      data: { 
        group_id: string; 
        user_id: string; 
        nickname: string; 
        role: string; 
        joined_at: string 
      } 
    }
  | { 
      type: 'GroupMemberLeft'; 
      data: { 
        group_id: string; 
        user_id: string; 
        nickname: string; 
        left_at: string 
      } 
    }
  | { 
      type: 'GroupRoleChanged'; 
      data: { 
        group_id: string; 
        user_id: string; 
        nickname: string; 
        old_role: string; 
        new_role: string; 
        changed_by: string 
      } 
    }
  | { 
      type: 'GroupInfoUpdated'; 
      data: { 
        group_id: string; 
        updated_fields: Record<string, any>; 
        updated_by: string 
      } 
    }
  | { 
      type: 'NewMomentNearby'; 
      data: { 
        moment: MomentPayload; 
        distance: number 
      } 
    }
  | { 
      type: 'MomentLiked'; 
      data: { 
        moment_id: string; 
        user_id: string; 
        nickname: string; 
        likes_count: number 
      } 
    }
  | { 
      type: 'MomentCommented'; 
      data: { 
        moment_id: string; 
        comment: CommentPayload 
      } 
    }
  | { 
      type: 'SystemNotification'; 
      data: { 
        notification: NotificationPayload 
      } 
    }
  | { 
      type: 'FriendRequestReceived'; 
      data: { 
        from_user_id: string; 
        from_nickname: string; 
        message?: string; 
        request_id: string 
      } 
    }
  | { 
      type: 'FriendRequestAccepted'; 
      data: { 
        user_id: string; 
        nickname: string; 
        request_id: string 
      } 
    }
  | { 
      type: 'ContentReported'; 
      data: { 
        target_type: string; 
        target_id: string; 
        report_id: string; 
        reason: string 
      } 
    }
  | { 
      type: 'Error'; 
      data: { 
        error_code: string; 
        error_message: string; 
        context?: Record<string, any> 
      } 
    }

// ============================================================================
// 客户端到服务器的消息类型 - 与后端unified_message_types.rs保持一致
// ============================================================================

export type ClientMessage =
  | { type: 'Auth'; data: { token: string } }
  | { type: 'Ping' }
  | { type: 'JoinGroup'; data: { group_id: string } }
  | { type: 'LeaveGroup'; data: { group_id: string } }
  | { 
      type: 'SendMessage'; 
      data: {
        group_id: string
        content: string
        message_type: string
        reply_to_message_id?: string
        mentioned_users?: string[]
        latitude?: number
        longitude?: number
      }
    }
  | { 
      type: 'MarkRead'; 
      data: {
        group_id: string
        message_id?: string
      }
    }
  | { 
      type: 'Typing'; 
      data: {
        group_id: string
        is_typing: boolean
      }
    }
  | { 
      type: 'UpdateStatus'; 
      data: {
        status: UserStatus
        location?: LocationUpdate
      }
    }

// ============================================================================
// 前端统一领域模型（供 stores/UI 使用）
// ============================================================================

export interface RealtimeMessage {
  id: string
  group_id: string // 统一消息架构：私聊也通过群组管理
  conversation_id?: string // 兼容字段
  sender_id: string
  sender_name: string
  sender_avatar?: string
  content: string
  message_type: 'text' | 'image' | 'video' | 'audio' | 'file' | 'location' | 'system' // 与后端MessageType枚举保持一致
  media_type?: 'image' | 'video' | 'audio' // 媒体类型
  media_url?: string // 媒体URL
  reply_to_message_id?: string // 回复的消息ID
  created_at: string
  is_read: boolean
}

export interface MessageStatusUpdate {
  message_id: string
  status: 'sent' | 'delivered' | 'read'
  timestamp: string
}

export interface UserStatusUpdate {
  user_id: string
  status: 'online' | 'offline'
  last_seen?: string
}

// ============================================================================
// 连接状态管理
// ============================================================================

export type ConnectionState = 
  | 'Connecting'
  | 'Connected' 
  | 'Authenticated'
  | 'Disconnecting'
  | 'Disconnected'
  | 'Error'

export interface ConnectionInfo {
  connection_id: string
  user_id?: string
  session_id: string
  state: ConnectionState
  connected_at: string
  last_ping_at?: string
  subscribed_groups: string[]
}


