import { apiClient } from '@/lib/api/client';
import { ENDPOINTS } from '@/lib/api/endpoints';
import {
  Conversation,
  GroupMessageItem,
  MessageType,
  PaginatedData,
  PaginationParams,
  PrivateChatListItem,
  PrivateMessageItem,
} from '@/types/api';

export const PrivateChatService = {
  list: (params?: PaginationParams) =>
    apiClient.post<PaginatedData<PrivateChatListItem>>(ENDPOINTS.PRIVATE_CHATS.ME, params ?? {}),

  openTheDoor: (userId: number, limit?: number) =>
    apiClient.post<Conversation>(ENDPOINTS.PRIVATE_CHATS.OPEN_THE_DOOR, {
      other_user_id: userId,
      limit,
    }),
};

export const PrivateMessageService = {
  history: (params: { target_user_id: number; cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<PrivateMessageItem>>(ENDPOINTS.PRIVATE_MESSAGES.HISTORY, params),

  send: (data: {
    target_user_id: number;
    type: MessageType;
    content?: string;
    media_file_id?: string;
    reply_to_message_id?: string;
  }) => apiClient.post<PrivateMessageItem>(ENDPOINTS.PRIVATE_MESSAGES.SEND, data),
};

export const GroupMessageService = {
  history: (params: { group_chat_id: string; cursor?: string; limit?: number }) =>
    apiClient.post<PaginatedData<GroupMessageItem>>(ENDPOINTS.GROUP_MESSAGES.HISTORY, params),

  send: (data: {
    group_chat_id: string;
    type: MessageType;
    content?: string;
    media_file_id?: string;
    reply_to_message_id?: string;
    mentions?: number[];
  }) => apiClient.post<GroupMessageItem>(ENDPOINTS.GROUP_MESSAGES.SEND, data),
};
