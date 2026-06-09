import { post, paginate, generateIdempotencyKey } from "../client";
import { PRIVATE_CHATS, PRIVATE_MESSAGES } from "../endpoints";
import type {
  PrivateChatItem,
  PrivateChatOpenResult,
  PrivateMessageItem,
  MessageSent,
  SendMessageType,
  PaginationParams,
} from "@/lib/types/api";

export const privateChatsService = {
  getChats(params?: { pagination?: PaginationParams }) {
    return paginate<PrivateChatItem>(PRIVATE_CHATS.ME.LIST, params);
  },

  openChat(data: { target_user_id: string }) {
    return post<PrivateChatOpenResult>(PRIVATE_CHATS.OPEN_THE_DOOR, data);
  },

  updateSettings(data: {
    target_user_id: string;
    is_muted?: boolean;
    is_pinned?: boolean;
  }) {
    return post<void>(PRIVATE_CHATS.ME.UPDATE_SETTINGS, data);
  },

  deleteChat(data: { target_user_id: string }) {
    return post<void>(PRIVATE_CHATS.ME.DELETE, data);
  },
};

export const privateMessagesService = {
  send(data: {
    target_user_id: string;
    type: SendMessageType;
    content?: string;
    file_id?: string;
    share_code?: string;
    reply_to_message_id?: string;
  }) {
    return post<MessageSent>(PRIVATE_MESSAGES.SEND, data, {
      idempotencyKey: generateIdempotencyKey(),
    });
  },

  getHistory(params: {
    target_user_id: string;
    pagination?: PaginationParams;
  }) {
    return paginate<PrivateMessageItem>(PRIVATE_MESSAGES.GET_HISTORY, params);
  },

  update(data: { message_id: string; content: string }) {
    return post<void>(PRIVATE_MESSAGES.UPDATE, data);
  },

  recall(data: { message_id: string }) {
    return post<void>(PRIVATE_MESSAGES.RECALL, data);
  },

  markAsRead(data: { target_user_id: string; up_to_message_id?: string }) {
    return post<void>(PRIVATE_MESSAGES.ME.MARK_AS_READ, data);
  },

  markAllRead() {
    return post<void>(PRIVATE_MESSAGES.ME.MARK_ALL_READ, {});
  },
};
