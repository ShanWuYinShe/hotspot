/**
 * WebSocket 事件处理器
 * 统一处理 WebSocket 推送的事件，更新应用状态
 * 严格遵循 @spec/api_spec_websocket.md
 */

import { toast } from 'sonner';
import { useWebSocketStore } from './store';
import { useAuthStore } from '@/lib/stores/auth';
import type {
  PrivateMessageNewPayload,
  GroupMessageNewPayload,
  UserOnlinePayload,
  UserOfflinePayload,
  GroupMemberLeftPayload,
  GroupMemberRemovedPayload,
  GroupMemberMutedPayload,
  GroupMemberUnmutedPayload,
  GroupUpdatedPayload,
  GroupDismissedPayload,
  AccountBannedPayload,
  GroupMemberJoinedPayload,
} from './types';

const isDev = import.meta.env.DEV;
const log = isDev ? console.log.bind(console) : () => {};

export type EventHandler<T = unknown> = (payload: T) => void;

interface PresenceActions {
  setOnline: (userId: string, status: string, statusText?: string, onlineAt?: string) => void;
  setOffline: (userId: string, offlineAt: string) => void;
}

let presenceStore: PresenceActions | null = null;

export function setPresenceStore(store: PresenceActions) {
  presenceStore = store;
}

export class WebSocketEventHandlers {
  private privateMessageHandlers = new Set<EventHandler>();
  private privateMessageEditedHandlers = new Set<EventHandler>();
  private privateMessageRecalledHandlers = new Set<EventHandler>();
  private groupMessageHandlers = new Set<EventHandler>();
  private groupMessageEditedHandlers = new Set<EventHandler>();
  private groupMessageRecalledHandlers = new Set<EventHandler>();
  private notificationHandlers = new Set<EventHandler>();
  private userOnlineHandlers = new Set<EventHandler>();
  private userOfflineHandlers = new Set<EventHandler>();
  private cleanupFns: (() => void)[] = [];

  constructor() {
    this.registerDefaultHandlers();
  }

  destroy() {
    this.cancelAllTyping();
    this.cleanupFns.forEach((fn) => fn());
    this.cleanupFns = [];
  }

  // ─── 自定义处理器注册 ───

  onPrivateMessage(handler: EventHandler<PrivateMessageNewPayload>) {
    this.privateMessageHandlers.add(handler as EventHandler);
    return () => this.privateMessageHandlers.delete(handler as EventHandler);
  }

  onGroupMessage(handler: EventHandler<GroupMessageNewPayload>) {
    this.groupMessageHandlers.add(handler as EventHandler);
    return () => this.groupMessageHandlers.delete(handler as EventHandler);
  }

  onNotification(handler: EventHandler) {
    this.notificationHandlers.add(handler);
    return () => this.notificationHandlers.delete(handler);
  }

  onUserOnline(handler: EventHandler) {
    this.userOnlineHandlers.add(handler);
    return () => this.userOnlineHandlers.delete(handler);
  }

  onUserOffline(handler: EventHandler) {
    this.userOfflineHandlers.add(handler);
    return () => this.userOfflineHandlers.delete(handler);
  }

  // ─── Typing ───

  private static readonly TYPING_THROTTLE_MS = 3000;
  private typingThrottleTimers = new Map<string, ReturnType<typeof setTimeout>>();
  private typingLastSent = new Map<string, number>();
  private typingDebounceTimers = new Map<string, ReturnType<typeof setTimeout>>();
  private activeTypingTargets = new Set<string>();

  sendPrivateTyping(targetUserId: string, isTyping: boolean) {
    this.dispatchTyping(`private:${targetUserId}`, isTyping, (typing) => ({
      type: 'PrivateTyping',
      payload: { target_user_id: targetUserId, is_typing: typing },
    }));
  }

  sendGroupTyping(groupId: string, isTyping: boolean) {
    this.dispatchTyping(`group:${groupId}`, isTyping, (typing) => ({
      type: 'GroupTyping',
      payload: { group_id: groupId, is_typing: typing },
    }));
  }

  cancelAllTyping() {
    for (const key of this.activeTypingTargets) {
      const colonIndex = key.indexOf(':');
      if (colonIndex === -1) continue;
      const type = key.substring(0, colonIndex);
      const id = key.substring(colonIndex + 1);
      useWebSocketStore.getState().send(
        type === 'private'
          ? { type: 'PrivateTyping', payload: { target_user_id: id, is_typing: false } }
          : { type: 'GroupTyping', payload: { group_id: id, is_typing: false } },
      );
    }
    for (const timer of this.typingThrottleTimers.values()) clearTimeout(timer);
    for (const timer of this.typingDebounceTimers.values()) clearTimeout(timer);
    this.typingThrottleTimers.clear();
    this.typingDebounceTimers.clear();
    this.typingLastSent.clear();
    this.activeTypingTargets.clear();
  }

  // ─── 私有方法 ───

  private isCurrentUser(userId: string): boolean {
    return userId === useAuthStore.getState().currentUserId;
  }

  private redirectIfViewingGroup(groupId: string) {
    const currentPath = window.location.pathname;
    if (currentPath.includes(`/groups/${groupId}`) || currentPath.includes(`/chat/groups/${groupId}`)) {
      window.location.href = '/messages';
    }
  }

  private onNotify<K extends keyof import('./types').WebSocketEventMap>(
    event: K,
    getMessage: (payload: import('./types').WebSocketEventMap[K]) => string,
    toastType: 'info' | 'success' | 'warning' | 'error' = 'info',
  ) {
    this.cleanupFns.push(
      useWebSocketStore.getState().on(event, (payload) => {
        log(`[WS] ${event}:`, payload);
        toast[toastType](getMessage(payload));
        this.notificationHandlers.forEach((h) => h(payload));
      }),
    );
  }

  private onMessage<K extends keyof import('./types').WebSocketEventMap>(
    event: K,
    handlers: Set<EventHandler<unknown>>,
  ) {
    this.cleanupFns.push(
      useWebSocketStore.getState().on(event, (payload) => {
        log(`[WS] ${event}:`, payload);
        handlers.forEach((h) => h(payload));
      }),
    );
  }

  private registerDefaultHandlers() {
    // 私聊消息
    this.onMessage('PrivateMessageNew', this.privateMessageHandlers);
    this.onMessage('PrivateMessageEdited', this.privateMessageEditedHandlers);
    this.onMessage('PrivateMessageRecalled', this.privateMessageRecalledHandlers);

    // 群聊消息
    this.onMessage('GroupMessageNew', this.groupMessageHandlers);
    this.onMessage('GroupMessageEdited', this.groupMessageEditedHandlers);
    this.onMessage('GroupMessageRecalled', this.groupMessageRecalledHandlers);

    // 关注关系
    this.onNotify('NewFollower', (p) => `${p.actor.nickname} 关注了你`);
    this.onNotify('MutualFollow', (p) => `你和 ${p.actor.nickname} 互相关注了`, 'success');

    // 动态互动
    this.onNotify('MomentLiked', (p) => `${p.actor.nickname} 赞了你的动态`);
    this.onNotify('MomentCommented', (p) => `${p.actor.nickname} 评论了你的动态`);
    this.onNotify('CommentLiked', (p) => `${p.actor.nickname} 赞了你的评论`);
    this.onNotify('CommentReply', (p) => `${p.actor.nickname} 回复了你的评论`);

    // 群组邀请/申请
    this.onNotify('GroupInviteReceived', (p) => `${p.inviter.nickname} 邀请你加入 ${p.group.name}`);
    this.onNotify('GroupApplicationReceived', (p) => `${p.applicant.nickname} 申请加入群组`);
    this.onNotify('GroupAnnouncementCreated', (p) => `系统公告: ${p.title}`);
    this.onNotify('MessageMentioned', (p) => {
      const target = p.target_type === 'moment' ? '动态' : '评论';
      return `${p.actor.nickname} 在${target}中提到了你`;
    });

    // 群组邀请响应
    this.cleanupFns.push(
      useWebSocketStore.getState().on('GroupInviteResponse', (payload) => {
        log('[WS] GroupInviteResponse:', payload);
        if (payload.status === 'accepted') toast.success(`${payload.user.nickname} 接受了群邀请`);
        this.notificationHandlers.forEach((h) => h(payload));
      }),
    );

    // 群组申请响应
    this.cleanupFns.push(
      useWebSocketStore.getState().on('GroupApplicationResponse', (payload) => {
        log('[WS] GroupApplicationResponse:', payload);
        if (payload.status === 'accepted') toast.success(`你的入群申请已通过`);
        this.notificationHandlers.forEach((h) => h(payload));
      }),
    );

    // 群组成员变动
    this.cleanupFns.push(
      useWebSocketStore.getState().on('GroupMemberJoined', (payload: GroupMemberJoinedPayload) => {
        log('[WS] GroupMemberJoined:', payload);
        toast.info(`${payload.user.nickname} 加入了群组`);
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('PrivateMessageRead', (payload) => {
        log('[WS] 私聊消息已读:', payload);
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('SubscriptionsSnapshot', (payload) => {
        log('[WS] 订阅状态快照:', payload);
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('GroupMemberLeft', (payload: GroupMemberLeftPayload) => {
        log('[WS] GroupMemberLeft:', payload);
        toast.info(`${payload.user.nickname} 离开了群组`);
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('GroupMemberRemoved', (payload: GroupMemberRemovedPayload) => {
        log('[WS] GroupMemberRemoved:', payload);
        if (this.isCurrentUser(payload.user.user_id)) {
          toast.warning('你已被移出群组');
          this.redirectIfViewingGroup(payload.group_id);
        } else {
          const reason = payload.reason === 'banned' ? '被封禁' : '被移除';
          toast.info(`${payload.user.nickname} ${reason}`);
        }
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('GroupMemberMuted', (payload: GroupMemberMutedPayload) => {
        log('[WS] GroupMemberMuted:', payload);
        if (this.isCurrentUser(payload.user.user_id)) {
          toast.warning(`你已被禁言至 ${payload.muted_until}`);
        } else {
          toast.info(`${payload.user.nickname} 被禁言`);
        }
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('GroupMemberUnmuted', (payload: GroupMemberUnmutedPayload) => {
        log('[WS] GroupMemberUnmuted:', payload);
        if (this.isCurrentUser(payload.user.user_id)) {
          toast.success('你已被解除禁言');
        } else {
          toast.info(`${payload.user.nickname} 被解除禁言`);
        }
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('GroupUpdated', (payload: GroupUpdatedPayload) => {
        log('[WS] GroupUpdated:', payload);
        const fieldLabels: Record<string, string> = {
          name: '名称',
          avatar_url: '头像',
          description: '简介',
          tags: '标签',
          join_method: '加入方式',
          is_searchable: '可搜索',
          max_members: '人数上限',
          has_password: '密码',
          location: '位置',
        };
        const fields = Object.keys(payload.updated_fields);
        if (fields.length > 0) {
          toast.success(`群组已更新: ${fields.map((f) => fieldLabels[f] || f).join('、')}`);
        }
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('GroupDismissed', (payload: GroupDismissedPayload) => {
        log('[WS] GroupDismissed:', payload);
        toast.warning('群组已解散');
        this.redirectIfViewingGroup(payload.group_id);
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('SystemAnnouncement', (payload) => {
        log('[WS] 系统公告:', payload);
        switch (payload.severity) {
          case 'error':
            toast.error(payload.title, { description: payload.content });
            break;
          case 'warning':
            toast.warning(payload.title);
            break;
          default:
            toast.info(payload.title);
        }
        this.notificationHandlers.forEach((h) => h(payload));
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('AccountBanned', (payload: AccountBannedPayload) => {
        log('[WS] AccountBanned:', payload);
        const duration = payload.duration_days === 0 ? '永久' : `${payload.duration_days} 天`;
        toast.error(`账号已被封禁 ${duration}`, { description: payload.reason });
        useWebSocketStore.getState().disconnect();
        useAuthStore.getState().logout();
        window.location.href = '/login';
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('UserOnline', (payload: UserOnlinePayload) => {
        log('[WS] 用户上线:', payload);
        presenceStore?.setOnline(payload.user_id, 'online', undefined, payload.online_at);
        this.userOnlineHandlers.forEach((h) => h(payload));
      }),
    );

    this.cleanupFns.push(
      useWebSocketStore.getState().on('UserOffline', (payload: UserOfflinePayload) => {
        log('[WS] 用户离线:', payload);
        presenceStore?.setOffline(payload.user_id, payload.offline_at);
        this.userOfflineHandlers.forEach((h) => h(payload));
      }),
    );

    // 仅日志的事件
    const logOnly = [
      'UserProfileUpdated',
      'PrivateUserTyping',
      'GroupUserTyping',
      'SelfReadSync',
      'SelfChatSettingsSync',
      'SelfSettingsUpdated',
      'SelfRelationSync',
    ] as const;
    for (const event of logOnly) {
      this.cleanupFns.push(
        useWebSocketStore.getState().on(event, (payload) => {
          log(`[WS] ${event}:`, payload);
        }),
      );
    }
  }

  private dispatchTyping(
    key: string,
    isTyping: boolean,
    createEvent: (is_typing: boolean) => import('./types').ClientEvent,
  ) {
    if (isTyping) {
      this.sendTypingThrottled(
        key,
        () => useWebSocketStore.getState().send(createEvent(true)),
        () => useWebSocketStore.getState().send(createEvent(false)),
      );
    } else {
      this.cancelTyping(key, () => useWebSocketStore.getState().send(createEvent(false)));
    }
  }

  private sendTypingThrottled(key: string, sendFn: () => void, cancelFn: () => void) {
    const now = Date.now();
    const lastSent = this.typingLastSent.get(key) ?? 0;

    if (now - lastSent >= WebSocketEventHandlers.TYPING_THROTTLE_MS) {
      sendFn();
      this.typingLastSent.set(key, now);
      this.activeTypingTargets.add(key);
    }

    const existing = this.typingDebounceTimers.get(key);
    if (existing) clearTimeout(existing);

    this.typingDebounceTimers.set(
      key,
      setTimeout(() => {
        cancelFn();
        this.typingDebounceTimers.delete(key);
        this.typingLastSent.delete(key);
        this.activeTypingTargets.delete(key);
      }, WebSocketEventHandlers.TYPING_THROTTLE_MS),
    );
  }

  private cancelTyping(key: string, sendFn: () => void) {
    const throttleTimer = this.typingThrottleTimers.get(key);
    if (throttleTimer) {
      clearTimeout(throttleTimer);
      this.typingThrottleTimers.delete(key);
    }
    const debounceTimer = this.typingDebounceTimers.get(key);
    if (debounceTimer) {
      clearTimeout(debounceTimer);
      this.typingDebounceTimers.delete(key);
    }
    if (this.activeTypingTargets.has(key)) {
      sendFn();
      this.activeTypingTargets.delete(key);
    }
    this.typingLastSent.delete(key);
  }
}
