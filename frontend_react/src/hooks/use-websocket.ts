'use client';

import { useEffect } from 'react';

import { websocketClient } from '@/lib/websocket/client';
import { useAuthStore } from '@/stores/auth';
import { useChatStore } from '@/stores/chat';
import { useNotificationsStore } from '@/stores/notifications';
import { MessageLocation, MessageMedia, MessageReplyTo, MessageType, UserInfo } from '@/types/api';
import { AnyWebSocketEvent } from '@/types/websocket';

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const parseMessageType = (value: unknown): MessageType =>
  typeof value === 'string' ? (value as MessageType) : 'text';

const parseUserInfo = (value: unknown): UserInfo | null => {
  if (!isRecord(value)) {
    return null;
  }
  if (typeof value.user_id !== 'number' || typeof value.nickname !== 'string' || typeof value.user_type !== 'string') {
    return null;
  }
  return value as unknown as UserInfo;
};

const parseMessageMedia = (value: unknown): MessageMedia | undefined => {
  if (!isRecord(value) || typeof value.url !== 'string') {
    return undefined;
  }
  return {
    url: value.url,
    thumbnail_url: typeof value.thumbnail_url === 'string' ? value.thumbnail_url : undefined,
    file_name: typeof value.file_name === 'string' ? value.file_name : undefined,
    file_size: typeof value.file_size === 'number' ? value.file_size : undefined,
    duration: typeof value.duration === 'number' ? value.duration : undefined,
    width: typeof value.width === 'number' ? value.width : undefined,
    height: typeof value.height === 'number' ? value.height : undefined,
    mime_type: typeof value.mime_type === 'string' ? value.mime_type : undefined,
  };
};

const parseMessageLocation = (value: unknown): MessageLocation | undefined => {
  if (!isRecord(value) || !isRecord(value.location)) {
    return undefined;
  }
  const location = value.location;
  if (
    typeof location.latitude !== 'number' ||
    typeof location.longitude !== 'number' ||
    typeof location.location_name !== 'string'
  ) {
    return undefined;
  }
  return {
    location: {
      latitude: location.latitude,
      longitude: location.longitude,
      location_name: location.location_name,
    },
    address: typeof value.address === 'string' ? value.address : undefined,
  };
};

const parseMessageReplyTo = (value: unknown): MessageReplyTo | undefined => {
  if (!isRecord(value)) {
    return undefined;
  }
  if (
    typeof value.message_id !== 'string' ||
    typeof value.user_id !== 'number' ||
    typeof value.nickname !== 'string' ||
    typeof value.content !== 'string'
  ) {
    return undefined;
  }
  return {
    message_id: value.message_id,
    user_id: value.user_id,
    nickname: value.nickname,
    content: value.content,
  };
};

export function useWebSocket() {
  const accessToken = useAuthStore((state) => state.accessToken);
  const currentUserId = useAuthStore((state) => state.user?.user_id ?? null);
  const {
    appendPrivateMessage,
    appendGroupMessage,
    updatePrivateMessage,
    updateGroupMessage,
    setPrivateTyping,
    setGroupTyping,
  } = useChatStore();
  const { incrementUnread } = useNotificationsStore();

  useEffect(() => {
    if (accessToken) {
      websocketClient.connect(accessToken);
    } else {
      websocketClient.disconnect();
    }
  }, [accessToken]);

  useEffect(() => {
    const unsubscribe = websocketClient.on((event: AnyWebSocketEvent) => {
      if (!isRecord(event.payload)) {
        return;
      }

      if (event.type === 'PrivateMessageNew') {
        const payload = event.payload;
        const sender = parseUserInfo(payload.sender);
        const targetUserId = typeof payload.target_user_id === 'number' ? payload.target_user_id : null;
        const messageId = typeof payload.message_id === 'string' ? payload.message_id : null;
        const sentAt = typeof payload.sent_at === 'string' ? payload.sent_at : null;

        if (!sender || !targetUserId || !messageId || !sentAt) {
          return;
        }

        const partnerId = sender.user_id === currentUserId ? targetUserId : sender.user_id;

        appendPrivateMessage(partnerId, {
          message_id: messageId,
          type: parseMessageType(payload.type),
          content: typeof payload.content === 'string' ? payload.content : undefined,
          media: parseMessageMedia(payload.media),
          location: parseMessageLocation(payload.location),
          sender,
          reply_to: parseMessageReplyTo(payload.reply_to),
          sent_at: sentAt,
          is_recalled: false,
        });
        return;
      }

      if (event.type === 'GroupMessageNew') {
        const payload = event.payload;
        const sender = parseUserInfo(payload.sender);
        const groupId = typeof payload.group_chat_id === 'string' ? payload.group_chat_id : null;
        const messageId = typeof payload.message_id === 'string' ? payload.message_id : null;
        const sentAt = typeof payload.sent_at === 'string' ? payload.sent_at : null;

        if (!sender || !groupId || !messageId || !sentAt) {
          return;
        }

        appendGroupMessage(groupId, {
          message_id: messageId,
          type: parseMessageType(payload.type),
          content: typeof payload.content === 'string' ? payload.content : undefined,
          media: parseMessageMedia(payload.media),
          location: parseMessageLocation(payload.location),
          sender,
          reply_to: parseMessageReplyTo(payload.reply_to),
          mentions: Array.isArray(payload.mentions) ? (payload.mentions as number[]) : undefined,
          sent_at: sentAt,
          is_recalled: false,
        });
        return;
      }

      if (event.type === 'PrivateMessageEdited') {
        const payload = event.payload;
        const messageId = typeof payload.message_id === 'string' ? payload.message_id : null;
        const targetUserId = typeof payload.target_user_id === 'number' ? payload.target_user_id : null;
        const content = typeof payload.content === 'string' ? payload.content : null;
        const editedAt = typeof payload.edited_at === 'string' ? payload.edited_at : null;

        if (!messageId || !targetUserId || !content || !editedAt) {
          return;
        }

        updatePrivateMessage(targetUserId, messageId, (msg) => ({
          ...msg,
          content,
          edited_at: editedAt,
        }));
        return;
      }

      if (event.type === 'GroupMessageEdited') {
        const payload = event.payload;
        const messageId = typeof payload.message_id === 'string' ? payload.message_id : null;
        const groupId = typeof payload.group_chat_id === 'string' ? payload.group_chat_id : null;
        const content = typeof payload.content === 'string' ? payload.content : null;
        const editedAt = typeof payload.edited_at === 'string' ? payload.edited_at : null;

        if (!messageId || !groupId || !content || !editedAt) {
          return;
        }

        updateGroupMessage(groupId, messageId, (msg) => ({
          ...msg,
          content,
          edited_at: editedAt,
        }));
        return;
      }

      if (event.type === 'PrivateMessageRecalled') {
        const payload = event.payload;
        const messageId = typeof payload.message_id === 'string' ? payload.message_id : null;
        const targetUserId = typeof payload.target_user_id === 'number' ? payload.target_user_id : null;
        const recalledAt = typeof payload.recalled_at === 'string' ? payload.recalled_at : null;

        if (!messageId || !targetUserId || !recalledAt) {
          return;
        }

        updatePrivateMessage(targetUserId, messageId, (msg) => ({
          ...msg,
          is_recalled: true,
          recalled_at: recalledAt,
        }));
        return;
      }

      if (event.type === 'GroupMessageRecalled') {
        const payload = event.payload;
        const messageId = typeof payload.message_id === 'string' ? payload.message_id : null;
        const groupId = typeof payload.group_chat_id === 'string' ? payload.group_chat_id : null;
        const recalledAt = typeof payload.recalled_at === 'string' ? payload.recalled_at : null;

        if (!messageId || !groupId || !recalledAt) {
          return;
        }

        updateGroupMessage(groupId, messageId, (msg) => ({
          ...msg,
          is_recalled: true,
          recalled_at: recalledAt,
        }));
        return;
      }

      if (event.type === 'PrivateMessageDeleted') {
        const payload = event.payload;
        const messageId = typeof payload.message_id === 'string' ? payload.message_id : null;
        const targetUserId = typeof payload.target_user_id === 'number' ? payload.target_user_id : null;
        if (!messageId || !targetUserId) {
          return;
        }
        updatePrivateMessage(targetUserId, messageId, (msg) => ({
          ...msg,
          content: undefined,
          media: undefined,
          location: undefined,
          is_recalled: true,
          recalled_at: typeof payload.deleted_at === 'string' ? payload.deleted_at : msg.recalled_at,
        }));
        return;
      }

      if (event.type === 'GroupMessageDeleted') {
        const payload = event.payload;
        const messageId = typeof payload.message_id === 'string' ? payload.message_id : null;
        const groupId = typeof payload.group_chat_id === 'string' ? payload.group_chat_id : null;
        if (!messageId || !groupId) {
          return;
        }
        updateGroupMessage(groupId, messageId, (msg) => ({
          ...msg,
          content: undefined,
          media: undefined,
          location: undefined,
          is_recalled: true,
          recalled_at: typeof payload.deleted_at === 'string' ? payload.deleted_at : msg.recalled_at,
        }));
        return;
      }

      if (event.type === 'PrivateUserTyping') {
        const payload = event.payload;
        const userId = typeof payload.user_id === 'number' ? payload.user_id : null;
        const isTyping = typeof payload.is_typing === 'boolean' ? payload.is_typing : null;
        if (userId !== null && isTyping !== null) {
          setPrivateTyping(userId, isTyping);
        }
        return;
      }

      if (event.type === 'GroupUserTyping') {
        const payload = event.payload;
        const groupId = typeof payload.group_chat_id === 'string' ? payload.group_chat_id : null;
        const userId = typeof payload.user_id === 'number' ? payload.user_id : null;
        const isTyping = typeof payload.is_typing === 'boolean' ? payload.is_typing : null;
        if (groupId && userId !== null && isTyping !== null) {
          setGroupTyping(groupId, userId, isTyping);
        }
        return;
      }

      if (
        event.type === 'MomentLiked' ||
        event.type === 'MomentCommented' ||
        event.type === 'CommentReply' ||
        event.type === 'MessageMentioned' ||
        event.type === 'SystemAnnouncement' ||
        event.type === 'NewFollower' ||
        event.type === 'MutualFollow' ||
        event.type === 'FollowRequestReceived' ||
        event.type === 'FollowRequestHandled' ||
        event.type === 'GroupInviteReceived' ||
        event.type === 'GroupInviteResponse' ||
        event.type === 'GroupApplicationReceived' ||
        event.type === 'GroupApplicationResponse' ||
        event.type === 'GroupAnnouncementCreated'
      ) {
        incrementUnread();
      }
    });

    return () => {
      unsubscribe();
    };
  }, [
    appendGroupMessage,
    appendPrivateMessage,
    currentUserId,
    incrementUnread,
    setGroupTyping,
    setPrivateTyping,
    updateGroupMessage,
    updatePrivateMessage,
  ]);

  return {
    subscribeUsers: (userIds: number[]) => websocketClient.subscribeUsers(userIds),
    subscribeGroups: (groupIds: string[]) => websocketClient.subscribeGroups(groupIds),
    sendPrivateTyping: (userId: number, isTyping: boolean) =>
      websocketClient.sendPrivateTyping(userId, isTyping),
    sendGroupTyping: (groupId: string, isTyping: boolean) =>
      websocketClient.sendGroupTyping(groupId, isTyping),
  };
}
