/**
 * 消息与会话类型定义
 * 基于最新 API 设计，无兼容性包袱
 */

// 重新导出主类型文件中的相关类型
export type {
  // 消息相关
  SendGroupMessageRequest,
  SendGroupMessageResponse,
  GroupMessageHistoryRequest,
  GroupMessageHistoryResponse,
  SendDirectMessageRequest,
  SendDirectMessageResponse,
  DirectMessageHistoryRequest,
  DirectMessageHistoryResponse,
  DeleteGroupMessageRequest,
  DeleteDirectMessageRequest,
  DeleteMessageResponse,
  MarkReadRequest,
  MarkReadResponse,
  GetGroupUnreadCountRequest,
  UnreadCountResponse,
  ConversationsRequest,
  ConversationsResponse,

  // 消息项类型
  DirectMessageHistoryItem,
  GroupMessageHistoryItem,
  ConversationItem,

  // 基础类型
  ApiResponse,
  PaginatedResponse,
} from "~/types";

// 消息类型（基于最新 API 设计）
export interface Message {
  id: string;
  message_id?: string;
  sender_id: string;
  sender_name: string;
  sender_avatar?: string;
  content: string;
  message_type: string;
  sent_at: string;
  created_at: string;
  read_at?: string;
  is_read?: boolean;
  latitude: number;
  longitude: number;
  // 群组消息字段
  group_id?: string;
  // 私聊消息字段
  conversation_id?: string;
  recipient_id?: string;
}

// 会话类型（基于最新 API 设计）
export interface Conversation {
  id: string;
  name: string;
  avatar?: string;
  last_message: string;
  last_message_time: string;
  unread_count: number;
  type: "user" | "group";
  is_pinned?: boolean;
  is_muted?: boolean;
  // 私聊会话字段
  participant_id?: string;
  // 群组会话字段
  group_id?: string;
  member_count?: number;
}
