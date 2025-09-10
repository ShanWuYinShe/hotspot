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
  ConversationSummary,

  // 基础类型
  ApiResponse,
  PaginatedResponse,
} from "~/types";

// 消息类型（统一的消息接口）
export interface Message {
  id?: string; // 兼容字段，通常等于 message_id
  message_id: string; // UUID 转换为字符串
  sender_id: string; // 64位整数转换为字符串
  sender_name: string;
  content: string;
  message_type: 'text' | 'image' | 'voice' | 'video' | 'location' | 'file';
  sent_at: string; // ISO 8601 datetime string
  created_at?: string; // 兼容字段，通常等于 sent_at
  group_id?: string; // UUID 转换为字符串
  conversation_id?: string; // UUID 转换为字符串
  latitude?: number;
  longitude?: number;
  recipient_id?: string; // 私聊消息的接收者ID
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

// 转换函数：ConversationSummary 到 Conversation
export function convertConversationSummary(summary: import("~/types").ConversationSummary): Conversation {
  return {
    id: summary.conversation_id,
    name: summary.peer_nickname,
    last_message: summary.last_message_preview,
    last_message_time: summary.last_message_at,
    unread_count: summary.unread_count,
    type: "user", // ConversationSummary 通常是私聊会话
    participant_id: summary.peer_user_id,
  };
}
