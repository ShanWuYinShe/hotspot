/**
 * 消息未读计数 Store (Zustand)
 * 监听 WebSocket 推送的消息事件来更新未读计数
 */

import { create } from "zustand";
import { useAuthStore } from "./auth";
import { useWebSocketStore } from "@/lib/websocket/store";
import type {
  PrivateMessageNewPayload,
  GroupMessageNewPayload,
} from "@/lib/websocket/types";

export interface UnreadConversation {
  id: string;
  type: "private" | "group";
  count: number;
  lastMessage?: {
    content: string;
    timestamp: string;
    senderName: string;
  };
}

interface MessageState {
  unreadConversations: Map<string, UnreadConversation>;
  totalUnreadCount: number;
  unreadConversationCount: number;

  markConversationAsRead: (conversationId: string) => void;
  markAllAsRead: () => void;
  initialize: () => void;
  destroy: () => void;
}

let cleanupFns: (() => void)[] = [];
let unsubscribeAuth: (() => void) | null = null;

function computeTotals(conversations: Map<string, UnreadConversation>) {
  let totalUnreadCount = 0;
  for (const conv of conversations.values()) {
    totalUnreadCount += conv.count;
  }
  return { totalUnreadCount, unreadConversationCount: conversations.size };
}

export const useMessageStore = create<MessageState>((set) => ({
  unreadConversations: new Map(),
  totalUnreadCount: 0,
  unreadConversationCount: 0,

  markConversationAsRead(conversationId) {
    set((state) => {
      if (!state.unreadConversations.has(conversationId)) return state;
      const next = new Map(state.unreadConversations);
      next.delete(conversationId);
      return { unreadConversations: next, ...computeTotals(next) };
    });
  },

  markAllAsRead() {
    set({
      unreadConversations: new Map(),
      totalUnreadCount: 0,
      unreadConversationCount: 0,
    });
  },

  initialize() {
    if (typeof window === "undefined") return;

    const incrementUnread = (
      conversationId: string,
      type: "private" | "group",
      payload: {
        content?: string;
        sent_at: string;
        sender: { nickname: string };
      },
    ) => {
      const lastMessage = {
        content: payload.content ?? "",
        timestamp: payload.sent_at,
        senderName: payload.sender.nickname,
      };

      set((state) => {
        const next = new Map(state.unreadConversations);
        const existing = next.get(conversationId);
        next.set(conversationId, {
          id: conversationId,
          type,
          count: (existing?.count ?? 0) + 1,
          lastMessage,
        });
        return { unreadConversations: next, ...computeTotals(next) };
      });
    };

    cleanupFns.push(
      useWebSocketStore
        .getState()
        .on("PrivateMessageNew", (payload: PrivateMessageNewPayload) => {
          if (payload.sender.user_id === useAuthStore.getState().currentUserId)
            return;
          incrementUnread(
            `private_${payload.sender.user_id}`,
            "private",
            payload,
          );
        }),
    );

    cleanupFns.push(
      useWebSocketStore
        .getState()
        .on("GroupMessageNew", (payload: GroupMessageNewPayload) => {
          if (payload.sender.user_id === useAuthStore.getState().currentUserId)
            return;
          incrementUnread(`group_${payload.group_id}`, "group", payload);
        }),
    );

    unsubscribeAuth = useAuthStore.subscribe((state, prevState) => {
      if (state.isAuthenticated === prevState.isAuthenticated) return;
      if (!state.isAuthenticated) {
        set({
          unreadConversations: new Map(),
          totalUnreadCount: 0,
          unreadConversationCount: 0,
        });
      }
    });
  },

  destroy() {
    if (unsubscribeAuth) {
      unsubscribeAuth();
      unsubscribeAuth = null;
    }
    cleanupFns.forEach((fn) => fn());
    cleanupFns = [];
  },
}));
