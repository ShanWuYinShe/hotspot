/**
 * WebSocket 实时通信类型定义
 * 严格按照 backend_doc.mdc 中的 WebSocket 消息类型定义
 */

// ============================================================================
// 基础 WebSocket 消息格式
// ============================================================================

export interface WebSocketMessage<T = any> {
  type: string;             // 消息类型标识
  timestamp: number;        // 消息时间戳（Unix时间戳）
  data: T;                  // 消息数据载荷
}

// ============================================================================
// 客户端消息类型（发送到服务端）
// ============================================================================

export type ClientMessageType = 
  | 'authenticate'
  | 'join_room'
  | 'leave_room'
  | 'send_message'
  | 'typing_start'
  | 'typing_stop'
  | 'mark_read'
  | 'ping';

// ============================================================================
// 服务端消息类型（从服务端接收）
// ============================================================================

export type ServerMessageType = 
  | 'connected'
  | 'authenticated'
  | 'error'
  | 'NewMessage'
  | 'MessageRead'
  | 'TypingIndicator'
  | 'UserOnline'
  | 'UserOffline'
  | 'UserLocationUpdate'
  | 'GroupMemberJoined'
  | 'GroupMemberLeft'
  | 'GroupRoleChanged'
  | 'SystemNotification'
  | 'NewMomentNearby'
  | 'pong';

// ============================================================================
// 客户端消息数据类型
// ============================================================================

export interface AuthenticateData {
  token: string;            // JWT 访问令牌
}

export interface JoinRoomData {
  room_type: 'user' | 'group' | 'conversation';
  room_id: string;          // 房间ID（用户ID、群组ID等）
}

export interface LeaveRoomData {
  room_type: 'user' | 'group' | 'conversation';
  room_id: string;          // 房间ID
}

export interface SendMessageData {
  group_id: string;         // 群组ID
  content: string;          // 消息内容
  message_type?: string;    // 消息类型，默认'text'
  reply_to_message_id?: string; // 回复的消息ID
  mentioned_users?: string[]; // 提及的用户ID数组
  latitude?: number;        // 纬度（位置消息）
  longitude?: number;       // 经度（位置消息）
}

export interface TypingData {
  group_id: string;         // 群组ID
}

export interface MarkReadData {
  group_id: string;         // 群组ID
  message_id?: string;      // 最后已读消息ID（可选）
}

// ============================================================================
// 服务端消息数据类型
// ============================================================================

export interface ConnectedData {
  user_id: string;          // 用户ID
  session_id: string;       // 会话ID
  server_time: string;      // 服务器时间（ISO 8601）
}

export interface AuthenticatedData {
  user_id: string;          // 用户ID
  nickname: string;         // 用户昵称
  authenticated_at: string; // 认证时间（ISO 8601）
}

export interface ErrorData {
  code: number;             // 错误码
  message: string;          // 错误消息
  details?: any;            // 错误详情
}

// ============================================================================
// 实时消息类型
// ============================================================================

export interface NewMessageData {
  message_id: string;       // 消息ID
  group_id: string;         // 群组ID（统一架构）
  group_type: string;       // 群组类型：'public_chat'|'private_chat'
  sender_id?: string;       // 发送者ID（系统消息可为null）
  sender_name?: string;     // 发送者昵称（系统消息可为null）
  sender_avatar?: string;   // 发送者头像URL
  content: string;          // 消息内容
  message_type: string;     // 消息类型：'text'|'image'|'location'|'file'|'system'
  system_event_type?: string; // 系统事件类型（仅系统消息）
  reply_to_message_id?: string; // 回复的消息ID
  mentioned_users: string[]; // 提及的用户ID数组
  affected_users: string[]; // 受影响的用户ID数组（系统消息用）
  sent_at: string;          // 发送时间（ISO 8601）
  latitude?: number;        // 纬度（位置消息）
  longitude?: number;       // 经度（位置消息）
  location_name?: string;   // 位置名称（位置消息）
  file_url?: string;        // 文件URL（文件消息）
  file_name?: string;       // 文件名（文件消息）
  file_size?: number;       // 文件大小（文件消息）
}

export interface MessageReadData {
  message_id: string;       // 消息ID
  group_id: string;         // 群组ID（统一架构）
  read_by: string;          // 已读用户ID
  read_by_name: string;     // 已读用户昵称
  read_at: string;          // 已读时间（ISO 8601）
}

export interface TypingIndicatorData {
  group_id: string;         // 群组ID（统一架构）
  user_id: string;          // 正在输入的用户ID
  user_name: string;        // 正在输入的用户昵称
  is_typing: boolean;       // 是否正在输入
}

// ============================================================================
// 用户状态类型
// ============================================================================

export interface UserOnlineData {
  user_id: string;          // 用户ID
  nickname: string;         // 用户昵称
  avatar_url?: string;      // 头像URL
  last_seen?: string;       // 最后在线时间（ISO 8601）
}

export interface UserOfflineData {
  user_id: string;          // 用户ID
  nickname: string;         // 用户昵称
  last_seen: string;        // 最后在线时间（ISO 8601）
}

export interface UserLocationUpdateData {
  user_id: string;          // 用户ID
  nickname: string;         // 用户昵称
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name?: string;   // 位置名称
  updated_at: string;       // 更新时间（ISO 8601）
}

// ============================================================================
// 群组事件类型
// ============================================================================

export interface GroupMemberJoinedData {
  group_id: string;         // 群组ID
  group_name: string;       // 群组名称
  user_id: string;          // 新成员ID
  nickname: string;         // 新成员昵称
  avatar_url?: string;      // 新成员头像URL
  user_role: string;        // 成员角色：'member'|'admin'|'owner'
  joined_at: string;        // 加入时间（ISO 8601）
  invited_by?: string;      // 邀请者ID（如果是被邀请）
}

export interface GroupMemberLeftData {
  group_id: string;         // 群组ID
  group_name: string;       // 群组名称
  user_id: string;          // 离开成员ID
  nickname: string;         // 离开成员昵称
  left_at: string;          // 离开时间（ISO 8601）
  reason: string;           // 离开原因：'leave'|'kick'|'ban'
  operated_by?: string;     // 操作者ID（踢出/封禁时）
}

export interface GroupRoleChangedData {
  group_id: string;         // 群组ID
  group_name: string;       // 群组名称
  user_id: string;          // 用户ID
  nickname: string;         // 用户昵称
  old_role: string;         // 原角色：'member'|'admin'|'owner'
  new_role: string;         // 新角色：'member'|'admin'|'owner'
  changed_by: string;       // 操作者ID
  changed_by_name: string;  // 操作者昵称
  changed_at: string;       // 变更时间（ISO 8601）
}

// ============================================================================
// 系统通知类型
// ============================================================================

export interface SystemNotificationData {
  notification_id: string;  // 通知ID
  title: string;            // 通知标题
  content: string;          // 通知内容
  level: string;            // 通知级别：'info'|'warning'|'error'|'success'
  category?: string;        // 通知分类：'system'|'security'|'feature'|'maintenance'
  action_url?: string;      // 操作链接
  action_text?: string;     // 操作按钮文本
  expires_at?: string;      // 过期时间（ISO 8601）
  created_at: string;       // 创建时间（ISO 8601）
}

export interface NewMomentNearbyData {
  moment_id: string;        // 瞬间ID
  user_id: string;          // 发布者ID
  nickname: string;         // 发布者昵称
  user_avatar?: string;     // 发布者头像URL
  description: string;      // 瞬间描述
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name: string;    // 位置名称
  distance: number;         // 距离（米）
  created_at: string;       // 创建时间（ISO 8601）
  tags: string[];           // 标签数组
  images?: string[];        // 图片URL数组
  likes_count: number;      // 点赞数
}

// ============================================================================
// 具体消息类型定义
// ============================================================================

// 客户端消息类型
export type ClientMessage = 
  | WebSocketMessage<AuthenticateData> & { type: 'authenticate' }
  | WebSocketMessage<JoinRoomData> & { type: 'join_room' }
  | WebSocketMessage<LeaveRoomData> & { type: 'leave_room' }
  | WebSocketMessage<SendMessageData> & { type: 'send_message' }
  | WebSocketMessage<TypingData> & { type: 'typing_start' }
  | WebSocketMessage<TypingData> & { type: 'typing_stop' }
  | WebSocketMessage<MarkReadData> & { type: 'mark_read' }
  | WebSocketMessage<{}> & { type: 'ping' };

// 服务端消息类型
export type ServerMessage = 
  | WebSocketMessage<ConnectedData> & { type: 'connected' }
  | WebSocketMessage<AuthenticatedData> & { type: 'authenticated' }
  | WebSocketMessage<ErrorData> & { type: 'error' }
  | WebSocketMessage<NewMessageData> & { type: 'NewMessage' }
  | WebSocketMessage<MessageReadData> & { type: 'MessageRead' }
  | WebSocketMessage<TypingIndicatorData> & { type: 'TypingIndicator' }
  | WebSocketMessage<UserOnlineData> & { type: 'UserOnline' }
  | WebSocketMessage<UserOfflineData> & { type: 'UserOffline' }
  | WebSocketMessage<UserLocationUpdateData> & { type: 'UserLocationUpdate' }
  | WebSocketMessage<GroupMemberJoinedData> & { type: 'GroupMemberJoined' }
  | WebSocketMessage<GroupMemberLeftData> & { type: 'GroupMemberLeft' }
  | WebSocketMessage<GroupRoleChangedData> & { type: 'GroupRoleChanged' }
  | WebSocketMessage<SystemNotificationData> & { type: 'SystemNotification' }
  | WebSocketMessage<NewMomentNearbyData> & { type: 'NewMomentNearby' }
  | WebSocketMessage<{}> & { type: 'pong' };

// ============================================================================
// WebSocket 连接状态
// ============================================================================

export type WebSocketConnectionState = 
  | 'connecting'
  | 'connected'
  | 'authenticated'
  | 'disconnected'
  | 'error';

export interface WebSocketConnectionInfo {
  state: WebSocketConnectionState;
  user_id?: string;
  session_id?: string;
  connected_at?: string;
  last_ping?: string;
  error?: ErrorData;
}

// ============================================================================
// 房间类型定义
// ============================================================================

export type RoomType = 'user' | 'group' | 'conversation';

export interface RoomInfo {
  type: RoomType;
  id: string;
  joined_at: string;
  last_activity?: string;
}

// ============================================================================
// WebSocket 事件处理器类型
// ============================================================================

export type WebSocketEventHandler<T = any> = (data: T) => void;

export interface WebSocketEventHandlers {
  connected?: WebSocketEventHandler<ConnectedData>;
  authenticated?: WebSocketEventHandler<AuthenticatedData>;
  error?: WebSocketEventHandler<ErrorData>;
  newMessage?: WebSocketEventHandler<NewMessageData>;
  messageRead?: WebSocketEventHandler<MessageReadData>;
  typingIndicator?: WebSocketEventHandler<TypingIndicatorData>;
  userOnline?: WebSocketEventHandler<UserOnlineData>;
  userOffline?: WebSocketEventHandler<UserOfflineData>;
  userLocationUpdate?: WebSocketEventHandler<UserLocationUpdateData>;
  groupMemberJoined?: WebSocketEventHandler<GroupMemberJoinedData>;
  groupMemberLeft?: WebSocketEventHandler<GroupMemberLeftData>;
  groupRoleChanged?: WebSocketEventHandler<GroupRoleChangedData>;
  systemNotification?: WebSocketEventHandler<SystemNotificationData>;
  newMomentNearby?: WebSocketEventHandler<NewMomentNearbyData>;
  pong?: WebSocketEventHandler<{}>;
}

