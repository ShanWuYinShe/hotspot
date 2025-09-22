/**
 * WebSocket消息类型定义
 * 根据后端文档规范定义的WebSocket消息格式
 */

/**
 * WebSocket消息基础格式
 */
export interface WebSocketMessage {
  type: string;             // 消息类型标识
  timestamp: number;        // 消息时间戳（Unix时间戳）
  data: Record<string, Object>; // 消息数据载荷
}

/**
 * 新消息实时推送数据格式
 */
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

/**
 * 消息已读状态通知数据格式
 */
export interface MessageReadData {
  message_id: string;       // 消息ID
  group_id: string;         // 群组ID（统一架构）
  read_by: string;          // 已读用户ID
  read_by_name: string;     // 已读用户昵称
  read_at: string;          // 已读时间（ISO 8601）
}

/**
 * 输入状态指示器数据格式
 */
export interface TypingIndicatorData {
  group_id: string;         // 群组ID（统一架构）
  user_id: string;          // 正在输入的用户ID
  user_name: string;        // 正在输入的用户昵称
  is_typing: boolean;       // 是否正在输入
}

/**
 * 用户上线通知数据格式
 */
export interface UserOnlineData {
  user_id: string;          // 用户ID
  nickname: string;         // 用户昵称
  avatar_url?: string;      // 头像URL
  last_seen?: string;       // 最后在线时间（ISO 8601）
}

/**
 * 用户下线通知数据格式
 */
export interface UserOfflineData {
  user_id: string;          // 用户ID
  nickname: string;         // 用户昵称
  last_seen: string;        // 最后在线时间（ISO 8601）
}

/**
 * 用户位置更新通知数据格式
 */
export interface UserLocationUpdateData {
  user_id: string;          // 用户ID
  nickname: string;         // 用户昵称
  latitude: number;         // 纬度
  longitude: number;        // 经度
  location_name?: string;   // 位置名称
  updated_at: string;       // 更新时间（ISO 8601）
}

/**
 * 群组新成员加入通知数据格式
 */
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

/**
 * 群组成员离开通知数据格式
 */
export interface GroupMemberLeftData {
  group_id: string;         // 群组ID
  group_name: string;       // 群组名称
  user_id: string;          // 离开成员ID
  nickname: string;         // 离开成员昵称
  left_at: string;          // 离开时间（ISO 8601）
  reason: string;           // 离开原因：'leave'|'kick'|'ban'
  operated_by?: string;     // 操作者ID（踢出/封禁时）
}

/**
 * 群组角色变更通知数据格式
 */
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

/**
 * 系统级通知消息数据格式
 */
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

/**
 * 附近新生活瞬间通知数据格式
 */
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

/**
 * WebSocket消息类型枚举
 */
export enum WebSocketMessageType {
  // 实时消息类型
  NEW_MESSAGE = 'NewMessage',
  MESSAGE_READ = 'MessageRead',
  TYPING_INDICATOR = 'TypingIndicator',
  
  // 用户状态类型
  USER_ONLINE = 'UserOnline',
  USER_OFFLINE = 'UserOffline',
  USER_LOCATION_UPDATE = 'UserLocationUpdate',
  
  // 群组事件类型
  GROUP_MEMBER_JOINED = 'GroupMemberJoined',
  GROUP_MEMBER_LEFT = 'GroupMemberLeft',
  GROUP_ROLE_CHANGED = 'GroupRoleChanged',
  
  // 系统通知类型
  SYSTEM_NOTIFICATION = 'SystemNotification',
  NEW_MOMENT_NEARBY = 'NewMomentNearby'
}

/**
 * 类型检查函数：检查数据是否为有效的 WebSocketMessage
 */
export function isWebSocketMessage(data: Record<string, Object>): boolean {
  return typeof data.type === 'string' && 
         typeof data.timestamp === 'number' && 
         typeof data.data === 'object' && 
         data.data !== null;
}

/**
 * 类型检查函数：检查数据是否为新消息类型
 */
export function isNewMessageData(data: Record<string, Object>): boolean {
  return typeof data.message_id === 'string' && 
         typeof data.group_id === 'string' && 
         typeof data.content === 'string';
}

/**
 * 类型检查函数：检查数据是否为消息已读类型
 */
export function isMessageReadData(data: Record<string, Object>): boolean {
  return typeof data.message_id === 'string' && 
         typeof data.group_id === 'string' && 
         typeof data.read_by === 'string';
}

/**
 * 类型检查函数：检查数据是否为输入状态指示器类型
 */
export function isTypingIndicatorData(data: Record<string, Object>): boolean {
  return typeof data.group_id === 'string' && 
         typeof data.user_id === 'string' && 
         typeof data.is_typing === 'boolean';
}

/**
 * 类型检查函数：检查数据是否为用户上线类型
 */
export function isUserOnlineData(data: Record<string, Object>): boolean {
  return typeof data.user_id === 'string' && 
         typeof data.nickname === 'string';
}

/**
 * 类型检查函数：检查数据是否为用户下线类型
 */
export function isUserOfflineData(data: Record<string, Object>): boolean {
  return typeof data.user_id === 'string' && 
         typeof data.nickname === 'string' && 
         typeof data.last_seen === 'string';
}

/**
 * 类型检查函数：检查数据是否为用户位置更新类型
 */
export function isUserLocationUpdateData(data: Record<string, Object>): boolean {
  return typeof data.user_id === 'string' && 
         typeof data.nickname === 'string' && 
         typeof data.latitude === 'number' && 
         typeof data.longitude === 'number';
}

/**
 * 类型检查函数：检查数据是否为群组成员加入类型
 */
export function isGroupMemberJoinedData(data: Record<string, Object>): boolean {
  return typeof data.group_id === 'string' && 
         typeof data.user_id === 'string' && 
         typeof data.nickname === 'string';
}

/**
 * 类型检查函数：检查数据是否为群组成员离开类型
 */
export function isGroupMemberLeftData(data: Record<string, Object>): boolean {
  return typeof data.group_id === 'string' && 
         typeof data.user_id === 'string' && 
         typeof data.nickname === 'string';
}

/**
 * 类型检查函数：检查数据是否为系统通知类型
 */
export function isSystemNotificationData(data: Record<string, Object>): boolean {
  return typeof data.notification_id === 'string' && 
         typeof data.title === 'string' && 
         typeof data.content === 'string';
}

/**
 * 类型检查函数：检查数据是否为附近新瞬间类型
 */
export function isNewMomentNearbyData(data: Record<string, Object>): boolean {
  return typeof data.moment_id === 'string' && 
         typeof data.user_id === 'string' && 
         typeof data.nickname === 'string';
}

/**
 * WebSocket消息解析结果接口
 */
export interface WebSocketParseResult {
  message: WebSocketMessage | null;
  newMessageData: NewMessageData | null;
  typingIndicatorData: TypingIndicatorData | null;
  messageReadData: MessageReadData | null;
  userOnlineData: UserOnlineData | null;
  userOfflineData: UserOfflineData | null;
  userLocationUpdateData: UserLocationUpdateData | null;
  groupMemberJoinedData: GroupMemberJoinedData | null;
  groupMemberLeftData: GroupMemberLeftData | null;
  systemNotificationData: SystemNotificationData | null;
  newMomentNearbyData: NewMomentNearbyData | null;
}

/**
 * WebSocket消息类型转换工具函数
 */
export class WebSocketMessageConverter {
  
  /**
   * 将Record转换为WebSocketMessage
   */
  static toWebSocketMessage(rawData: Record<string, Object>): WebSocketMessage | null {
    if (isWebSocketMessage(rawData)) {
      const message: WebSocketMessage = {
        type: rawData.type as string,
        timestamp: rawData.timestamp as number,
        data: rawData.data as Record<string, Object>
      };
      return message;
    }
    return null;
  }

  /**
   * 将Record转换为NewMessageData
   */
  static toNewMessageData(data: Record<string, Object>): NewMessageData | null {
    if (isNewMessageData(data)) {
      const messageData: NewMessageData = {
        message_id: data.message_id as string,
        group_id: data.group_id as string,
        group_type: data.group_type as string,
        sender_id: data.sender_id as string,
        sender_name: data.sender_name as string,
        sender_avatar: data.sender_avatar as string,
        content: data.content as string,
        message_type: data.message_type as string,
        system_event_type: data.system_event_type as string,
        reply_to_message_id: data.reply_to_message_id as string,
        mentioned_users: data.mentioned_users as string[],
        affected_users: data.affected_users as string[],
        sent_at: data.sent_at as string,
        latitude: data.latitude as number,
        longitude: data.longitude as number,
        location_name: data.location_name as string,
        file_url: data.file_url as string,
        file_name: data.file_name as string,
        file_size: data.file_size as number
      };
      return messageData;
    }
    return null;
  }

  /**
   * 将Record转换为TypingIndicatorData
   */
  static toTypingIndicatorData(data: Record<string, Object>): TypingIndicatorData | null {
    if (isTypingIndicatorData(data)) {
      const typingData: TypingIndicatorData = {
        group_id: data.group_id as string,
        user_id: data.user_id as string,
        user_name: data.user_name as string,
        is_typing: data.is_typing as boolean
      };
      return typingData;
    }
    return null;
  }

  /**
   * 将Record转换为MessageReadData
   */
  static toMessageReadData(data: Record<string, Object>): MessageReadData | null {
    if (isMessageReadData(data)) {
      const readData: MessageReadData = {
        message_id: data.message_id as string,
        group_id: data.group_id as string,
        read_by: data.read_by as string,
        read_by_name: data.read_by_name as string,
        read_at: data.read_at as string
      };
      return readData;
    }
    return null;
  }

  /**
   * 将Record转换为UserOnlineData
   */
  static toUserOnlineData(data: Record<string, Object>): UserOnlineData | null {
    if (isUserOnlineData(data)) {
      const onlineData: UserOnlineData = {
        user_id: data.user_id as string,
        nickname: data.nickname as string,
        avatar_url: data.avatar_url as string,
        last_seen: data.last_seen as string
      };
      return onlineData;
    }
    return null;
  }

  /**
   * 将Record转换为UserOfflineData
   */
  static toUserOfflineData(data: Record<string, Object>): UserOfflineData | null {
    if (isUserOfflineData(data)) {
      const offlineData: UserOfflineData = {
        user_id: data.user_id as string,
        nickname: data.nickname as string,
        last_seen: data.last_seen as string
      };
      return offlineData;
    }
    return null;
  }

  /**
   * 将Record转换为SystemNotificationData
   */
  static toSystemNotificationData(data: Record<string, Object>): SystemNotificationData | null {
    if (isSystemNotificationData(data)) {
      const notificationData: SystemNotificationData = {
        notification_id: data.notification_id as string,
        title: data.title as string,
        content: data.content as string,
        level: data.level as string,
        category: data.category as string,
        action_url: data.action_url as string,
        action_text: data.action_text as string,
        expires_at: data.expires_at as string,
        created_at: data.created_at as string
      };
      return notificationData;
    }
    return null;
  }

  static toUserLocationUpdateData(data: Record<string, Object>): UserLocationUpdateData | null {
    if (isUserLocationUpdateData(data)) {
      const locationData: UserLocationUpdateData = {
        user_id: data.user_id as string,
        nickname: data.nickname as string,
        latitude: data.latitude as number,
        longitude: data.longitude as number,
        location_name: data.location_name as string,
        updated_at: data.updated_at as string
      };
      return locationData;
    }
    return null;
  }

  static toGroupMemberJoinedData(data: Record<string, Object>): GroupMemberJoinedData | null {
    if (isGroupMemberJoinedData(data)) {
      const joinedData: GroupMemberJoinedData = {
        group_id: data.group_id as string,
        group_name: data.group_name as string,
        user_id: data.user_id as string,
        nickname: data.nickname as string,
        avatar_url: data.avatar_url as string,
        joined_at: data.joined_at as string,
        invited_by: data.invited_by as string,
        user_role: data.user_role as string
      };
      return joinedData;
    }
    return null;
  }

  static toGroupMemberLeftData(data: Record<string, Object>): GroupMemberLeftData | null {
    if (isGroupMemberLeftData(data)) {
      const leftData: GroupMemberLeftData = {
        group_id: data.group_id as string,
        group_name: data.group_name as string,
        user_id: data.user_id as string,
        nickname: data.nickname as string,
        left_at: data.left_at as string,
        reason: data.reason as string
      };
      return leftData;
    }
    return null;
  }

  static toNewMomentNearbyData(data: Record<string, Object>): NewMomentNearbyData | null {
    if (isNewMomentNearbyData(data)) {
      const momentData: NewMomentNearbyData = {
        moment_id: data.moment_id as string,
        user_id: data.user_id as string,
        nickname: data.nickname as string,
        user_avatar: data.user_avatar as string,
        description: data.description as string,
        latitude: data.latitude as number,
        longitude: data.longitude as number,
        location_name: data.location_name as string,
        distance: data.distance as number,
        created_at: data.created_at as string,
        tags: data.tags as string[],
        images: data.images as string[],
        likes_count: data.likes_count as number
      };
      return momentData;
    }
    return null;
  }

  /**
   * 完整的消息解析
   */
  static parseWebSocketMessage(rawData: Record<string, Object>): WebSocketParseResult {
    const message = WebSocketMessageConverter.toWebSocketMessage(rawData);
    if (!message) {
      return {
        message: null,
        newMessageData: null,
        typingIndicatorData: null,
        messageReadData: null,
        userOnlineData: null,
        userOfflineData: null,
        userLocationUpdateData: null,
        groupMemberJoinedData: null,
        groupMemberLeftData: null,
        systemNotificationData: null,
        newMomentNearbyData: null
      };
    }

    const result: WebSocketParseResult = {
      message: message,
      newMessageData: null,
      typingIndicatorData: null,
      messageReadData: null,
      userOnlineData: null,
      userOfflineData: null,
      userLocationUpdateData: null,
      groupMemberJoinedData: null,
      groupMemberLeftData: null,
      systemNotificationData: null,
      newMomentNearbyData: null
    };

    // 根据消息类型转换对应的数据
    switch (message.type) {
      case WebSocketMessageType.NEW_MESSAGE:
        result.newMessageData = WebSocketMessageConverter.toNewMessageData(message.data);
        break;
      case WebSocketMessageType.TYPING_INDICATOR:
        result.typingIndicatorData = WebSocketMessageConverter.toTypingIndicatorData(message.data);
        break;
      case WebSocketMessageType.MESSAGE_READ:
        result.messageReadData = WebSocketMessageConverter.toMessageReadData(message.data);
        break;
      case WebSocketMessageType.USER_ONLINE:
        result.userOnlineData = WebSocketMessageConverter.toUserOnlineData(message.data);
        break;
      case WebSocketMessageType.USER_OFFLINE:
        result.userOfflineData = WebSocketMessageConverter.toUserOfflineData(message.data);
        break;
      case WebSocketMessageType.USER_LOCATION_UPDATE:
        result.userLocationUpdateData = WebSocketMessageConverter.toUserLocationUpdateData(message.data);
        break;
      case WebSocketMessageType.GROUP_MEMBER_JOINED:
        result.groupMemberJoinedData = WebSocketMessageConverter.toGroupMemberJoinedData(message.data);
        break;
      case WebSocketMessageType.GROUP_MEMBER_LEFT:
        result.groupMemberLeftData = WebSocketMessageConverter.toGroupMemberLeftData(message.data);
        break;
      case WebSocketMessageType.SYSTEM_NOTIFICATION:
        result.systemNotificationData = WebSocketMessageConverter.toSystemNotificationData(message.data);
        break;
      case WebSocketMessageType.NEW_MOMENT_NEARBY:
        result.newMomentNearbyData = WebSocketMessageConverter.toNewMomentNearbyData(message.data);
        break;
    }

    return result;
  }
}
