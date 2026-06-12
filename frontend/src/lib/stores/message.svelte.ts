/**
 * 消息未读计数 Store
 * 监听 WebSocket 推送的消息事件来更新未读计数
 */

import { websocketStore } from '$lib/websocket';
import { browser } from '$app/environment';
import { authStore } from './auth.svelte';
import type { PrivateMessageNewPayload, GroupMessageNewPayload } from '$lib/websocket/types';

// ============ 消息类型 ============

export interface UnreadConversation {
  id: string; // 对话ID（私聊为user_id，群聊为group_id）
  type: 'private' | 'group';
  count: number;
  lastMessage?: {
    content: string;
    timestamp: string;
    senderName: string;
  };
}

// ============ 消息未读计数 Store ============

class MessageStore {
  // 未读对话列表（使用 $state.raw 避免 Map 自赋值反模式）
  unreadConversations = $state.raw<Map<string, UnreadConversation>>(new Map());

  // 总未读消息数
  totalUnreadCount = $derived(
    Array.from(this.unreadConversations.values()).reduce((sum, conv) => sum + conv.count, 0)
  );

  // 未读对话数
  unreadConversationCount = $derived(this.unreadConversations.size);

  // 清理函数数组（用于取消 WebSocket 监听）
  private cleanupFunctions: (() => void)[] = [];
  // $effect.root() 清理函数
  private destroyEffectRoot: (() => void) | null = null;

  constructor() {
    if (!browser) return;

    this.registerMessageHandlers();

    // 响应式监听认证状态：登出时清空消息未读，防止跨用户数据泄漏
    this.destroyEffectRoot = $effect.root(() => {
      $effect(() => {
        if (!authStore.isAuthenticated) {
          this.unreadConversations = new Map();
        }
      });
    });
  }

  // ============ 内部工具方法 ============

  /** 不可变更新 Map：clone → mutate → reassign */
  private updateMap(mutator: (map: Map<string, UnreadConversation>) => void) {
    const next = new Map(this.unreadConversations);
    mutator(next);
    this.unreadConversations = next;
  }

  /** 统一的消息未读递增逻辑（私聊/群聊共用） */
  private incrementUnread(
    conversationId: string,
    type: 'private' | 'group',
    payload: { content?: string; sent_at: string; sender: { nickname: string } }
  ) {
    const lastMessage = {
      content: payload.content ?? '',
      timestamp: payload.sent_at,
      senderName: payload.sender.nickname
    };

    this.updateMap((map) => {
      const existing = map.get(conversationId);
      map.set(conversationId, {
        id: conversationId,
        type,
        count: (existing?.count ?? 0) + 1,
        lastMessage
      });
    });
  }

  // ============ WebSocket 事件注册 ============

  private registerMessageHandlers() {
    this.cleanupFunctions.push(
      websocketStore.on('PrivateMessageNew', (payload: PrivateMessageNewPayload) => {
        if (payload.sender.user_id === authStore.currentUserId) return;
        this.incrementUnread(`private_${payload.sender.user_id}`, 'private', payload);
      })
    );

    this.cleanupFunctions.push(
      websocketStore.on('GroupMessageNew', (payload: GroupMessageNewPayload) => {
        if (payload.sender.user_id === authStore.currentUserId) return;
        this.incrementUnread(`group_${payload.group_id}`, 'group', payload);
      })
    );
  }

  // ============ 公共方法 ============

  /**
   * 用会话列表接口返回的未读数对齐本地未读 Map。
   * WebSocket 只能在运行时累加，登录后已有的未读只能从列表接口获取，
   * 因此每次加载会话列表后调用，保证底栏/侧栏角标与「X 条未读」与服务端一致。
   * 只更新传入列表里出现的会话，避免覆盖其它来源（如 WS）维护的未读。
   */
  syncFromChatList(
    privateChats: { other_user: { user_id: string }; unread_count: number }[],
    groupChats: { group: { group_id: string }; my_state?: { unread_count?: number } }[]
  ) {
    this.updateMap((map) => {
      for (const chat of privateChats) {
        const id = `private_${chat.other_user.user_id}`;
        if (chat.unread_count > 0) {
          map.set(id, { id, type: 'private', count: chat.unread_count });
        } else {
          map.delete(id);
        }
      }
      for (const group of groupChats) {
        const id = `group_${group.group.group_id}`;
        const count = group.my_state?.unread_count ?? 0;
        if (count > 0) {
          map.set(id, { id, type: 'group', count });
        } else {
          map.delete(id);
        }
      }
    });
  }

  markConversationAsRead(conversationId: string) {
    this.updateMap((map) => map.delete(conversationId));
  }

  markAllAsRead() {
    this.unreadConversations = new Map();
  }

  destroy() {
    if (this.destroyEffectRoot) {
      this.destroyEffectRoot();
      this.destroyEffectRoot = null;
    }
    this.cleanupFunctions.forEach((cleanup) => cleanup());
    this.cleanupFunctions = [];
  }
}

// ============ 导出单例 ============

export const messageStore = new MessageStore();
