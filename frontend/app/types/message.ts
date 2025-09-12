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
  ConversationSummary,

  // 基础类型
  ApiResponse,
  PaginatedResponse,
  Message, // 使用统一的Message接口
} from "~/types";

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
