import { create } from 'zustand';

import { GroupMessageItem, PrivateMessageItem } from '@/types/api';

type TypingState = {
  privateTyping: Record<number, boolean>;
  groupTyping: Record<string, Record<number, boolean>>;
};

interface ChatState extends TypingState {
  privateMessages: Record<number, PrivateMessageItem[]>;
  groupMessages: Record<string, GroupMessageItem[]>;
  setPrivateMessages: (userId: number, messages: PrivateMessageItem[]) => void;
  setGroupMessages: (groupId: string, messages: GroupMessageItem[]) => void;
  appendPrivateMessage: (userId: number, message: PrivateMessageItem) => void;
  appendGroupMessage: (groupId: string, message: GroupMessageItem) => void;
  updatePrivateMessage: (
    userId: number,
    messageId: string,
    updater: (message: PrivateMessageItem) => PrivateMessageItem,
  ) => void;
  updateGroupMessage: (
    groupId: string,
    messageId: string,
    updater: (message: GroupMessageItem) => GroupMessageItem,
  ) => void;
  setPrivateTyping: (userId: number, isTyping: boolean) => void;
  setGroupTyping: (groupId: string, userId: number, isTyping: boolean) => void;
}

export const useChatStore = create<ChatState>((set) => ({
  privateMessages: {},
  groupMessages: {},
  privateTyping: {},
  groupTyping: {},
  setPrivateMessages: (userId, messages) =>
    set((state) => ({ privateMessages: { ...state.privateMessages, [userId]: messages } })),
  setGroupMessages: (groupId, messages) =>
    set((state) => ({ groupMessages: { ...state.groupMessages, [groupId]: messages } })),
  appendPrivateMessage: (userId, message) =>
    set((state) => ({
      privateMessages: {
        ...state.privateMessages,
        [userId]: [...(state.privateMessages[userId] ?? []), message],
      },
    })),
  appendGroupMessage: (groupId, message) =>
    set((state) => ({
      groupMessages: {
        ...state.groupMessages,
        [groupId]: [...(state.groupMessages[groupId] ?? []), message],
      },
    })),
  updatePrivateMessage: (userId, messageId, updater) =>
    set((state) => ({
      privateMessages: {
        ...state.privateMessages,
        [userId]: (state.privateMessages[userId] ?? []).map((msg) =>
          msg.message_id === messageId ? updater(msg) : msg,
        ),
      },
    })),
  updateGroupMessage: (groupId, messageId, updater) =>
    set((state) => ({
      groupMessages: {
        ...state.groupMessages,
        [groupId]: (state.groupMessages[groupId] ?? []).map((msg) =>
          msg.message_id === messageId ? updater(msg) : msg,
        ),
      },
    })),
  setPrivateTyping: (userId, isTyping) =>
    set((state) => ({
      privateTyping: { ...state.privateTyping, [userId]: isTyping },
    })),
  setGroupTyping: (groupId, userId, isTyping) =>
    set((state) => ({
      groupTyping: {
        ...state.groupTyping,
        [groupId]: {
          ...(state.groupTyping[groupId] ?? {}),
          [userId]: isTyping,
        },
      },
    })),
}));
