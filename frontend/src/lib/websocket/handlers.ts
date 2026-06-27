/**
 * WebSocket 事件处理器
 * 统一处理 WebSocket 推送的事件，更新应用状态
 * 严格遵循 spec/api_spec_websocket.md
 */

import { websocketStore } from './store.svelte';
import { toast } from '$lib/stores/toast.svelte';
import { browser, dev } from '$app/environment';
import { goto } from '$app/navigation';
import { getLocale, t, translateWithLocale } from '$lib/i18n';
import { formatDateTime } from '$lib/utils/format';

// Dev-only 日志工具
const log = dev ? console.log.bind(console) : () => {};
const warn = dev ? console.warn.bind(console) : () => {};
const logError = dev ? console.error.bind(console) : () => {};
import { authStore } from '$lib/stores/auth.svelte';
import { presenceStore } from '$lib/stores/presence.svelte';
import type {
  PrivateMessageNewPayload,
  GroupMessageNewPayload,
  SubscriptionOperation,
  SubscribeAckPayload,
  SubscriptionsSnapshotPayload,
  UserOnlinePayload,
  UserOfflinePayload,
  GroupMemberLeftPayload,
  GroupMemberRemovedPayload,
  GroupMemberMutedPayload,
  GroupMemberUnmutedPayload,
  GroupUpdatedPayload,
  GroupDismissedPayload,
  AccountBannedPayload,
  GroupMemberJoinedPayload
} from './types';

// ============ 事件处理器类型 ============

export type EventHandler<T = unknown> = (payload: T) => void | Promise<void>;

// ============ 订阅类型 ============

export type SubscriptionType = 'user' | 'group';

// ============ 订阅管理器 ============

class SubscriptionManager {
  // 客户端“期望订阅”状态（离线也会更新，用于重连恢复）
  private desiredSubscriptions = new Map<SubscriptionType, Set<string>>();

  // 服务端“已生效订阅”快照（由 SubscribeAck 回传）
  private effectiveSubscriptions = new Map<SubscriptionType, Set<string>>();

  // 待确认的订阅操作（包含定时器引用）
  private pendingAcks = new Map<
    string,
    {
      operation: SubscriptionOperation;
      type: SubscriptionType;
      ids: string[];
      resolve: (success: boolean) => void;
      timer: ReturnType<typeof setTimeout>;
    }
  >();

  private isDriftReconciling = false;
  private lastDriftReconcileAt = 0;
  private static readonly DRIFT_RECONCILE_COOLDOWN_MS = 5000;
  private isRestoreInFlight = false;
  private lastReconcileTrigger: 'subscribe_ack' | 'subscriptions_snapshot' | null = null;
  private lastReconcileAt = 0;
  private cleanupFns: (() => void)[] = [];

  constructor() {
    this.initializeSubscriptionMap(this.desiredSubscriptions);
    this.initializeSubscriptionMap(this.effectiveSubscriptions);

    this.cleanupFns.push(
      websocketStore.on('SubscribeAck', (payload) => {
        this.handleSubscribeAck(payload);
      }),
      websocketStore.on('SubscriptionsSnapshot', (payload) => {
        this.handleSubscriptionsSnapshot(payload);
      }),
      websocketStore.on('ConnectionEstablished', () => {
        void this.restoreAndReconcileSubscriptions();
      })
    );
  }

  destroy() {
    this.cleanupFns.forEach((fn) => fn());
    this.cleanupFns = [];
  }

  /** 增量添加订阅 */
  subscribe(type: SubscriptionType, ids: string[]): Promise<boolean> {
    const normalizedIds = this.normalizeIds(ids);
    if (normalizedIds.length === 0) return Promise.resolve(true);
    this.applyDesiredOperation('subscribe', type, normalizedIds);
    return this.sendOrQueueRequest('subscribe', type, normalizedIds);
  }

  /** 增量移除订阅 */
  unsubscribe(type: SubscriptionType, ids: string[]): Promise<boolean> {
    const normalizedIds = this.normalizeIds(ids);
    if (normalizedIds.length === 0) return Promise.resolve(true);
    this.applyDesiredOperation('unsubscribe', type, normalizedIds);
    return this.sendOrQueueRequest('unsubscribe', type, normalizedIds);
  }

  /** 全量覆盖订阅（重连时使用） */
  setSubscriptions(type: SubscriptionType, ids: string[]): Promise<boolean> {
    const normalizedIds = this.normalizeIds(ids);
    this.applyDesiredOperation('set', type, normalizedIds);
    return this.sendOrQueueRequest('set', type, normalizedIds);
  }

  /** 根据连接状态发送请求；离线时仅缓存意图 */
  private sendOrQueueRequest(
    operation: SubscriptionOperation,
    type: SubscriptionType,
    ids: string[]
  ): Promise<boolean> {
    if (!websocketStore.isConnected) {
      log(
        `[WebSocket] 未连接，缓存订阅意图: operation=${operation}, type=${type}, ids=${ids.length}`
      );
      return Promise.resolve(true);
    }
    return this.sendRequest(operation, type, ids);
  }

  /** 统一的订阅请求发送 + 超时等待 */
  private sendRequest(
    operation: SubscriptionOperation,
    type: SubscriptionType,
    ids: string[]
  ): Promise<boolean> {
    if (!websocketStore.isConnected) {
      log(
        `[WebSocket] 发送订阅请求时连接已断开，转为缓存意图: operation=${operation}, type=${type}`
      );
      return Promise.resolve(true);
    }

    const clientMsgId = this.generateClientMsgId();

    websocketStore.send({
      type: 'SetSubscriptions',
      payload: { client_request_id: clientMsgId, operation, type, ids }
    });

    return new Promise((resolve) => {
      const timer = setTimeout(() => {
        if (this.pendingAcks.has(clientMsgId)) {
          warn(`[WebSocket] ${operation}超时:`, type, ids);
          this.pendingAcks.delete(clientMsgId);
          resolve(false);
        }
      }, 10000);

      this.pendingAcks.set(clientMsgId, { operation, type, ids, resolve, timer });
    });
  }

  /**
   * 获取当前订阅
   */
  getSubscriptions(type: SubscriptionType): string[] {
    const subscriptions = this.desiredSubscriptions.get(type);
    return subscriptions ? Array.from(subscriptions) : [];
  }

  getEffectiveSubscriptions(type: SubscriptionType): string[] {
    const subscriptions = this.effectiveSubscriptions.get(type);
    return subscriptions ? Array.from(subscriptions) : [];
  }

  /**
   * 检查是否已订阅
   */
  isSubscribed(type: SubscriptionType, id: string): boolean {
    return this.desiredSubscriptions.get(type)?.has(id) ?? false;
  }

  /**
   * 处理订阅确认
   */
  private handleSubscribeAck(payload: SubscribeAckPayload) {
    const { client_request_id, operation, type, ids, code, failed_ids } = payload;
    const subscriptionType = type as SubscriptionType;
    const failedIds = this.normalizeIds(failed_ids?.map((f) => f.subscription_id) ?? []);
    const successfulIds = this.excludeFailedIds(ids, failedIds);
    this.updateEffectiveSnapshot(subscriptionType, successfulIds);

    if (failedIds.length > 0) {
      this.rollbackDesiredByFailedIds(operation, subscriptionType, failedIds, successfulIds);
    }
    if (!this.isRestoreInFlight) {
      void this.reconcileDriftIfNeeded('subscribe_ack');
    }

    const pending = this.pendingAcks.get(client_request_id);
    if (!pending) {
      warn('[WebSocket] 收到未知的 SubscribeAck:', client_request_id);
      return;
    }

    // 清理定时器并删除待确认记录
    clearTimeout(pending.timer);
    this.pendingAcks.delete(client_request_id);

    if (code === 0) {
      log(`[WebSocket] 订阅${operation}成功:`, type, ids.length, '个');
    } else {
      logError('[WebSocket] 订阅操作部分失败:', payload.message, failed_ids);
      if (failedIds.length > 0) {
        log(`[WebSocket] 已回滚失败订阅项: ${failedIds.length}/${ids.length} 个`);
      }
    }

    // 全部成功时 resolve(true)，部分失败时 resolve(false)
    pending.resolve(code === 0);
  }

  private handleSubscriptionsSnapshot(payload: SubscriptionsSnapshotPayload) {
    const normalizedUserIds = this.normalizeIds(payload.user_ids ?? []);
    const normalizedGroupIds = this.normalizeIds(payload.group_ids ?? []);
    this.updateEffectiveSnapshot('user', normalizedUserIds);
    this.updateEffectiveSnapshot('group', normalizedGroupIds);
    void this.reconcileDriftIfNeeded('subscriptions_snapshot');
  }

  private async requestSubscriptionsSnapshot() {
    if (!websocketStore.isConnected) return;
    websocketStore.send({
      type: 'GetSubscriptions',
      payload: { client_request_id: this.generateClientMsgId() }
    });
  }

  /**
   * 恢复订阅（重连后，使用 set 操作全量覆盖）
   */
  private async restoreSubscriptions() {
    log('[WebSocket] 恢复订阅（基于离线订阅意图）...');

    for (const type of ['user', 'group'] as SubscriptionType[]) {
      const ids = Array.from(this.desiredSubscriptions.get(type) ?? []);
      await this.sendOrQueueRequest('set', type, ids);
    }
  }

  private async restoreAndReconcileSubscriptions() {
    this.isRestoreInFlight = true;
    try {
      await this.restoreSubscriptions();
      await this.waitForPendingAcks();
      await this.requestSubscriptionsSnapshot();
    } finally {
      this.isRestoreInFlight = false;
    }
  }

  /**
   * 清理所有待确认的订阅操作
   */
  cleanup() {
    log('[SubscriptionManager] 清理所有待确认的订阅操作');

    // 清理所有待确认的定时器
    for (const [_clientMsgId, pending] of this.pendingAcks) {
      clearTimeout(pending.timer);
      // 拒绝所有待确认的 Promise
      pending.resolve(false);
    }

    this.pendingAcks.clear();
    log('[SubscriptionManager] 清理完成');
  }

  /**
   * 生成客户端消息 ID（spec 建议使用 UUID）
   */
  private generateClientMsgId(): string {
    return crypto.randomUUID();
  }

  private initializeSubscriptionMap(map: Map<SubscriptionType, Set<string>>) {
    map.set('user', new Set());
    map.set('group', new Set());
  }

  private excludeFailedIds(ids: string[], failedIds: string[]): string[] {
    if (failedIds.length === 0) {
      return this.normalizeIds(ids);
    }
    const failedSet = new Set(this.normalizeIds(failedIds));
    return this.normalizeIds(ids).filter((id) => !failedSet.has(id));
  }

  private normalizeIds(ids: string[]): string[] {
    const unique = new Set<string>();
    for (const id of ids) {
      const normalized = id.trim();
      if (!normalized) continue;
      unique.add(normalized);
    }
    return Array.from(unique);
  }

  private applyDesiredOperation(
    operation: SubscriptionOperation,
    type: SubscriptionType,
    ids: string[]
  ) {
    const desiredSet = this.desiredSubscriptions.get(type);
    if (!desiredSet) return;
    switch (operation) {
      case 'subscribe':
        ids.forEach((id) => desiredSet.add(id));
        break;
      case 'unsubscribe':
        ids.forEach((id) => desiredSet.delete(id));
        break;
      case 'set':
        desiredSet.clear();
        ids.forEach((id) => desiredSet.add(id));
        break;
    }
  }

  private updateEffectiveSnapshot(type: SubscriptionType, ids: string[]) {
    const effectiveSet = this.effectiveSubscriptions.get(type);
    if (!effectiveSet) return;
    effectiveSet.clear();
    this.normalizeIds(ids).forEach((id) => effectiveSet.add(id));
  }

  private rollbackDesiredByFailedIds(
    operation: SubscriptionOperation,
    type: SubscriptionType,
    failedIds: string[],
    serverSnapshotIds: string[]
  ) {
    const desiredSet = this.desiredSubscriptions.get(type);
    if (!desiredSet) return;
    switch (operation) {
      case 'subscribe':
        // spec: failed_ids 需要回滚本地状态
        failedIds.forEach((id) => desiredSet.delete(id));
        break;
      case 'unsubscribe':
        // 回滚取消失败项，保持继续订阅
        failedIds.forEach((id) => desiredSet.add(id));
        break;
      case 'set': {
        // set 失败场景直接对齐服务端快照，避免本地意图漂移
        desiredSet.clear();
        this.normalizeIds(serverSnapshotIds).forEach((id) => desiredSet.add(id));
        break;
      }
    }
  }

  private toSortedIds(set: Set<string> | undefined): string[] {
    return set ? Array.from(set).sort() : [];
  }

  private async waitForPendingAcks(timeoutMs = 1500) {
    const startedAt = Date.now();
    while (this.pendingAcks.size > 0 && Date.now() - startedAt < timeoutMs) {
      await new Promise((resolve) => setTimeout(resolve, 50));
    }
  }

  private computeDrift(type: SubscriptionType): {
    missing_on_server: string[];
    extra_on_server: string[];
  } {
    const desired = this.desiredSubscriptions.get(type) ?? new Set<string>();
    const effective = this.effectiveSubscriptions.get(type) ?? new Set<string>();
    const missing_on_server = this.toSortedIds(
      new Set(Array.from(desired).filter((id) => !effective.has(id)))
    );
    const extra_on_server = this.toSortedIds(
      new Set(Array.from(effective).filter((id) => !desired.has(id)))
    );
    return { missing_on_server, extra_on_server };
  }

  getDriftState() {
    const user = this.computeDrift('user');
    const group = this.computeDrift('group');
    return {
      user,
      group,
      has_drift:
        user.missing_on_server.length > 0 ||
        user.extra_on_server.length > 0 ||
        group.missing_on_server.length > 0 ||
        group.extra_on_server.length > 0
    };
  }

  private async reconcileDriftIfNeeded(trigger: 'subscribe_ack' | 'subscriptions_snapshot') {
    const drift = this.getDriftState();
    if (!drift.has_drift) return;

    const now = Date.now();
    if (
      this.isDriftReconciling ||
      now - this.lastDriftReconcileAt < SubscriptionManager.DRIFT_RECONCILE_COOLDOWN_MS
    ) {
      return;
    }

    this.isDriftReconciling = true;
    this.lastDriftReconcileAt = now;
    try {
      warn('[WebSocket] 检测到订阅漂移，准备对账恢复', { trigger, drift });
      this.lastReconcileTrigger = trigger;
      this.lastReconcileAt = now;
      for (const type of ['user', 'group'] as SubscriptionType[]) {
        const desiredIds = this.toSortedIds(this.desiredSubscriptions.get(type));
        await this.sendOrQueueRequest('set', type, desiredIds);
      }
    } finally {
      this.isDriftReconciling = false;
    }
  }

  debugState() {
    const drift = this.getDriftState();
    const snapshot = {
      desired: {
        user: this.toSortedIds(this.desiredSubscriptions.get('user')),
        group: this.toSortedIds(this.desiredSubscriptions.get('group'))
      },
      effective: {
        user: this.toSortedIds(this.effectiveSubscriptions.get('user')),
        group: this.toSortedIds(this.effectiveSubscriptions.get('group'))
      },
      drift,
      drift_summary: {
        user_missing_count: drift.user.missing_on_server.length,
        user_extra_count: drift.user.extra_on_server.length,
        group_missing_count: drift.group.missing_on_server.length,
        group_extra_count: drift.group.extra_on_server.length
      },
      reconcile: {
        in_progress: this.isDriftReconciling,
        last_trigger: this.lastReconcileTrigger,
        last_at: this.lastReconcileAt > 0 ? new Date(this.lastReconcileAt).toISOString() : null
      },
      restore_in_flight: this.isRestoreInFlight
    };
    if (dev) {
      console.group('📡 WebSocket Subscription State');
      console.log(snapshot);
      console.groupEnd();
    }
    return snapshot;
  }
}

// ============ 全局事件处理器 ============

class WebSocketEventHandlers {
  // 私聊消息处理器
  private privateMessageHandlers = new Set<EventHandler<PrivateMessageNewPayload>>();
  private privateMessageEditedHandlers = new Set<EventHandler>();
  private privateMessageRecalledHandlers = new Set<EventHandler>();

  // 群聊消息处理器
  private groupMessageHandlers = new Set<EventHandler<GroupMessageNewPayload>>();
  private groupMessageEditedHandlers = new Set<EventHandler>();
  private groupMessageRecalledHandlers = new Set<EventHandler>();

  // 通知处理器
  private notificationHandlers = new Set<EventHandler>();

  // 在线状态处理器
  private userOnlineHandlers = new Set<EventHandler>();
  private userOfflineHandlers = new Set<EventHandler>();
  private cleanupFns: (() => void)[] = [];

  private isCurrentUser(userId: string): boolean {
    return userId === authStore.currentUserId;
  }

  private getGroupFieldLabels(locale = getLocale()): Record<string, string> {
    return {
      name: translateWithLocale(locale, 'websocket.fieldName'),
      avatar_url: translateWithLocale(locale, 'websocket.fieldAvatar'),
      description: translateWithLocale(locale, 'websocket.fieldDescription'),
      tags: translateWithLocale(locale, 'websocket.fieldTags'),
      join_method: translateWithLocale(locale, 'websocket.fieldJoinMethod'),
      is_searchable: translateWithLocale(locale, 'websocket.fieldSearchable'),
      max_members: translateWithLocale(locale, 'websocket.fieldMaxMembers'),
      has_password: translateWithLocale(locale, 'websocket.fieldPassword'),
      location: translateWithLocale(locale, 'websocket.fieldLocation')
    };
  }

  private redirectIfViewingGroup(groupId: string): void {
    if (!browser) return;
    const currentPath = window.location.pathname;
    if (
      currentPath.includes(`/groups/${groupId}`) ||
      currentPath.includes(`/chat/groups/${groupId}`)
    ) {
      void goto('/groups');
    }
  }

  constructor() {
    this.registerDefaultHandlers();
  }

  // ============ 辅助方法 ============

  /** 注册通知类事件：log + toast + 转发到 notificationHandlers */
  private onNotify<K extends keyof import('./types').WebSocketEventMap>(
    event: K,
    getMessage: (payload: import('./types').WebSocketEventMap[K]) => string,
    toastType: 'info' | 'success' | 'warning' | 'error' = 'info'
  ) {
    this.cleanupFns.push(
      websocketStore.on(event, (payload) => {
        log(`[WS] ${event}:`, payload);
        toast[toastType](getMessage(payload));
        this.notificationHandlers.forEach((handler) => handler(payload));
      })
    );
  }

  /** 注册消息类事件：log + 转发到自定义 handlers */
  private onMessage<K extends keyof import('./types').WebSocketEventMap>(
    event: K,
    handlers: Set<EventHandler<import('./types').WebSocketEventMap[K]>>
  ) {
    this.cleanupFns.push(
      websocketStore.on(event, (payload) => {
        log(`[WS] ${event}:`, payload);
        handlers.forEach((handler) => handler(payload));
      })
    );
  }

  destroy() {
    this.cancelAllTyping();
    this.cleanupFns.forEach((fn) => fn());
    this.cleanupFns = [];
  }

  /**
   * 注册默认处理器
   */
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
    this.onNotify('NewFollower', (p) =>
      t('notificationDropdown.content.newFollower', { nickname: p.actor.nickname })
    );
    this.onNotify(
      'MutualFollow',
      (p) => t('notificationDropdown.content.mutualFollow', { nickname: p.actor.nickname }),
      'success'
    );

    // 生活瞬间互动
    this.onNotify('MomentLiked', (p) =>
      t('notificationDropdown.content.momentLiked', { nickname: p.actor.nickname })
    );
    this.onNotify(
      'MomentCommented',
      (p) =>
        `${p.actor.nickname} ${getLocale() === 'zh-CN' ? '评论了你的动态' : 'commented on your moment'}`
    );
    this.onNotify(
      'CommentLiked',
      (p) =>
        `${p.actor.nickname} ${getLocale() === 'zh-CN' ? '赞了你的评论' : 'liked your comment'}`
    );
    this.onNotify(
      'CommentReply',
      (p) =>
        `${p.actor.nickname} ${getLocale() === 'zh-CN' ? '回复了你的评论' : 'replied to your comment'}`
    );

    // 群组邀请/申请
    this.onNotify('GroupInviteReceived', (p) =>
      t('notificationDropdown.content.groupInviteReceived', {
        nickname: p.inviter.nickname,
        groupName: p.group.name
      })
    );
    this.onNotify(
      'GroupApplicationReceived',
      (p) =>
        `${p.applicant.nickname} ${getLocale() === 'zh-CN' ? '申请加入群组' : 'requested to join a group'}`
    );
    this.onNotify(
      'GroupAnnouncementCreated',
      (p) =>
        `${translateWithLocale(getLocale(), 'notificationDropdown.titles.systemAnnouncement')}: ${p.title}`
    );
    this.onNotify('MessageMentioned', (p) => {
      const targetLabel =
        p.target_type === 'moment' ? t('websocket.momentLabel') : t('websocket.commentLabel');
      return getLocale() === 'zh-CN'
        ? `${p.actor.nickname} 在${targetLabel}中提到了你`
        : `${p.actor.nickname} mentioned you in the ${targetLabel}`;
    });

    // 群组邀请响应（条件 toast）
    this.cleanupFns.push(
      websocketStore.on('GroupInviteResponse', (payload) => {
        log('[WS] GroupInviteResponse:', payload);
        if (payload.status === 'accepted') {
          toast.success(t('websocket.groupInviteAccepted', { nickname: payload.user.nickname }));
        }
        this.notificationHandlers.forEach((handler) => handler(payload));
      })
    );

    // 群组申请响应（条件 toast）
    this.cleanupFns.push(
      websocketStore.on('GroupApplicationResponse', (payload) => {
        log('[WS] GroupApplicationResponse:', payload);
        if (payload.status === 'accepted') {
          toast.success(t('websocket.groupApplicationAccepted', { groupName: payload.group.name }));
        }
        this.notificationHandlers.forEach((handler) => handler(payload));
      })
    );

    // 群组成员加入（仅 toast，不转发 notification）
    this.cleanupFns.push(
      websocketStore.on('GroupMemberJoined', (payload: GroupMemberJoinedPayload) => {
        log('[WS] GroupMemberJoined:', payload);
        toast.info(t('websocket.groupMemberJoined', { nickname: payload.user.nickname }));
      })
    );

    // 私聊已读回执（spec: PrivateMessageRead）
    this.cleanupFns.push(
      websocketStore.on('PrivateMessageRead', (payload) => {
        log('[WS] 私聊消息已读:', payload);
      })
    );

    // 订阅状态快照（spec: SubscriptionsSnapshot）
    this.cleanupFns.push(
      websocketStore.on('SubscriptionsSnapshot', (payload) => {
        log('[WS] 订阅状态快照:', payload);
      })
    );

    // 群组成员离开（GroupMemberLeft）
    this.cleanupFns.push(
      websocketStore.on('GroupMemberLeft', (payload: GroupMemberLeftPayload) => {
        log('[WS] GroupMemberLeft:', payload);
        toast.info(t('websocket.groupMemberLeft', { nickname: payload.user.nickname }));
      })
    );

    // 群组成员被移除（GroupMemberRemoved）
    this.cleanupFns.push(
      websocketStore.on('GroupMemberRemoved', (payload: GroupMemberRemovedPayload) => {
        log('[WS] GroupMemberRemoved:', payload);

        if (this.isCurrentUser(payload.user.user_id)) {
          toast.warning(t('websocket.youWereRemovedFromGroup'));
          this.redirectIfViewingGroup(payload.group_id);
        } else {
          const reasonText =
            payload.reason === 'banned'
              ? t('websocket.reasonBanned')
              : t('websocket.reasonRemoved');
          toast.info(
            t('websocket.groupMemberRemoved', {
              nickname: payload.user.nickname,
              reason: reasonText
            })
          );
        }
      })
    );

    // 群组成员被禁言（GroupMemberMuted）
    this.cleanupFns.push(
      websocketStore.on('GroupMemberMuted', (payload: GroupMemberMutedPayload) => {
        log('[WS] GroupMemberMuted:', payload);

        const mutedUntil = formatDateTime(payload.muted_until, getLocale());
        if (this.isCurrentUser(payload.user.user_id)) {
          toast.warning(t('websocket.youWereMutedUntil', { time: mutedUntil }));
        } else {
          toast.info(
            t('websocket.groupMemberMutedUntil', {
              nickname: payload.user.nickname,
              time: mutedUntil
            })
          );
        }
      })
    );

    // 群组成员解除禁言（GroupMemberUnmuted）
    this.cleanupFns.push(
      websocketStore.on('GroupMemberUnmuted', (payload: GroupMemberUnmutedPayload) => {
        log('[WS] GroupMemberUnmuted:', payload);

        if (this.isCurrentUser(payload.user.user_id)) {
          toast.success(t('websocket.youWereUnmuted'));
        } else {
          toast.info(t('websocket.groupMemberUnmuted', { nickname: payload.user.nickname }));
        }
      })
    );

    // 群组信息更新（GroupUpdated）
    this.cleanupFns.push(
      websocketStore.on('GroupUpdated', (payload: GroupUpdatedPayload) => {
        log('[WS] GroupUpdated:', payload);

        const locale = getLocale();
        const fieldLabels = this.getGroupFieldLabels(locale);
        const updatedFields = Object.keys(payload.updated_fields);
        if (updatedFields.length > 0) {
          const separator = locale === 'zh-CN' ? '、' : ', ';
          const fieldsText = updatedFields.map((f) => fieldLabels[f] || f).join(separator);
          toast.success(t('websocket.groupUpdated', { fields: fieldsText }));
        }
      })
    );

    // 群组解散（GroupDismissed，spec 命名）
    this.cleanupFns.push(
      websocketStore.on('GroupDismissed', (payload: GroupDismissedPayload) => {
        log('[WS] GroupDismissed:', payload);
        toast.warning(t('websocket.groupDismissed'));
        this.redirectIfViewingGroup(payload.group_id);
      })
    );

    // 系统公告
    this.cleanupFns.push(
      websocketStore.on('SystemAnnouncement', (payload) => {
        log('[WS] 系统公告:', payload);
        switch (payload.severity) {
          case 'error':
            toast.error(payload.title, payload.content);
            break;
          case 'warning':
            toast.warning(payload.title);
            break;
          default:
            toast.info(payload.title);
        }
        this.notificationHandlers.forEach((handler) => handler(payload));
      })
    );

    // 账号被封禁（spec 新增事件）
    this.cleanupFns.push(
      websocketStore.on('AccountBanned', (payload: AccountBannedPayload) => {
        log('[WS] AccountBanned:', payload);

        const durationText =
          payload.duration_days === 0
            ? t('websocket.permanent')
            : t('websocket.dayUnit', { count: payload.duration_days });
        toast.error(t('websocket.accountBanned', { duration: durationText }), payload.reason);

        // 先断开 WebSocket（设置 shouldReconnect=false，阻止自动重连），再登出
        websocketStore.disconnect();
        authStore.logout();
        // 与 HTTP 10201 路径一致：跳转封禁页并携带原因（而非 /auth）
        window.location.href = `/banned?reason=${encodeURIComponent(payload.reason)}`;
      })
    );

    // 在线状态（spec 简化：只有 user_id + 时间戳）
    this.cleanupFns.push(
      websocketStore.on('UserOnline', (payload: UserOnlinePayload) => {
        log('[WS] 用户上线:', payload);

        // 更新 presence store（简化为 online 状态）
        presenceStore.setOnline(payload.user_id, 'online', undefined, payload.online_at);

        // 调用自定义处理器
        this.userOnlineHandlers.forEach((handler) => handler(payload));
      })
    );

    this.cleanupFns.push(
      websocketStore.on('UserOffline', (payload: UserOfflinePayload) => {
        log('[WS] 用户离线:', payload);

        // 更新 presence store
        presenceStore.setOffline(payload.user_id, payload.offline_at);

        // 调用自定义处理器
        this.userOfflineHandlers.forEach((handler) => handler(payload));
      })
    );

    // 用户资料变更（spec 新增事件）
    this.cleanupFns.push(
      websocketStore.on('UserProfileUpdated', (payload) => {
        log('[WS] 用户资料变更:', payload);
        // 可用于更新缓存的用户信息
      })
    );

    // 私聊正在输入（spec: PrivateUserTyping）
    this.cleanupFns.push(
      websocketStore.on('PrivateUserTyping', (payload) => {
        log('[WS] 私聊正在输入:', payload);
      })
    );

    // 群聊正在输入（spec: GroupUserTyping）
    this.cleanupFns.push(
      websocketStore.on('GroupUserTyping', (payload) => {
        log('[WS] 群聊正在输入:', payload);
      })
    );

    // 多端同步事件（⏳ 延期实现）
    this.cleanupFns.push(
      websocketStore.on('SelfReadSync', (_payload) => {
        // ⏳ 延期实现
      })
    );
    this.cleanupFns.push(
      websocketStore.on('SelfChatSettingsSync', (_payload) => {
        // ⏳ 延期实现
      })
    );
    this.cleanupFns.push(
      websocketStore.on('SelfSettingsUpdated', (_payload) => {
        // ⏳ 延期实现
      })
    );
    this.cleanupFns.push(
      websocketStore.on('SelfRelationSync', (_payload) => {
        // ⏳ 延期实现
      })
    );
  }

  // ============ 添加自定义处理器 ============

  /**
   * 添加私聊消息处理器
   */
  onPrivateMessage(handler: EventHandler<PrivateMessageNewPayload>) {
    this.privateMessageHandlers.add(handler);
    return () => this.privateMessageHandlers.delete(handler);
  }

  /**
   * 添加群聊消息处理器
   */
  onGroupMessage(handler: EventHandler<GroupMessageNewPayload>) {
    this.groupMessageHandlers.add(handler);
    return () => this.groupMessageHandlers.delete(handler);
  }

  /**
   * 添加通知处理器
   */
  onNotification(handler: EventHandler) {
    this.notificationHandlers.add(handler);
    return () => this.notificationHandlers.delete(handler);
  }

  /**
   * 添加用户在线状态处理器
   */
  onUserOnline(handler: EventHandler) {
    this.userOnlineHandlers.add(handler);
    return () => this.userOnlineHandlers.delete(handler);
  }

  onUserOffline(handler: EventHandler) {
    this.userOfflineHandlers.add(handler);
    return () => this.userOfflineHandlers.delete(handler);
  }

  // ============ 客户端事件发送 ============

  // Typing 节流状态（spec: 最多每 3 秒一次，停止 3 秒后自动发 false）
  private static readonly TYPING_THROTTLE_MS = 3000;
  private typingThrottleTimers = new Map<string, ReturnType<typeof setTimeout>>();
  private typingLastSent = new Map<string, number>();
  private typingDebounceTimers = new Map<string, ReturnType<typeof setTimeout>>();
  // 记录当前正在输入的目标，用于切换会话时取消
  private activeTypingTargets = new Set<string>();

  /** 发送私聊正在输入状态（带节流 + 自动取消） */
  sendPrivateTyping(targetUserId: string, isTyping: boolean) {
    this.dispatchTyping(`private:${targetUserId}`, isTyping, (typing) => ({
      type: 'PrivateTyping',
      payload: { target_user_id: targetUserId, is_typing: typing }
    }));
  }

  /** 发送群聊正在输入状态（带节流 + 自动取消） */
  sendGroupTyping(groupId: string, isTyping: boolean) {
    this.dispatchTyping(`group:${groupId}`, isTyping, (typing) => ({
      type: 'GroupTyping',
      payload: { group_id: groupId, is_typing: typing }
    }));
  }

  /** 统一的 typing 分发（节流 + 防抖逻辑复用） */
  private dispatchTyping(
    key: string,
    isTyping: boolean,
    createEvent: (is_typing: boolean) => import('./types').ClientEvent
  ) {
    if (isTyping) {
      this.sendTypingThrottled(
        key,
        () => websocketStore.send(createEvent(true)),
        () => websocketStore.send(createEvent(false))
      );
    } else {
      this.cancelTyping(key, () => websocketStore.send(createEvent(false)));
    }
  }

  /**
   * 切换会话时取消所有活跃的输入状态
   * spec: 切换会话时对上一个对象立即发 is_typing: false
   */
  cancelAllTyping() {
    for (const key of this.activeTypingTargets) {
      const colonIndex = key.indexOf(':');
      if (colonIndex === -1) continue;
      const type = key.substring(0, colonIndex);
      const id = key.substring(colonIndex + 1);
      if (type === 'private') {
        websocketStore.send({
          type: 'PrivateTyping',
          payload: { target_user_id: id, is_typing: false }
        });
      } else if (type === 'group') {
        websocketStore.send({
          type: 'GroupTyping',
          payload: { group_id: id, is_typing: false }
        });
      }
    }
    // 清理所有定时器和状态
    for (const timer of this.typingThrottleTimers.values()) clearTimeout(timer);
    for (const timer of this.typingDebounceTimers.values()) clearTimeout(timer);
    this.typingThrottleTimers.clear();
    this.typingDebounceTimers.clear();
    this.typingLastSent.clear();
    this.activeTypingTargets.clear();
  }

  /**
   * 节流发送 typing（内部方法）
   */
  private sendTypingThrottled(key: string, sendFn: () => void, cancelFn: () => void) {
    const now = Date.now();
    const lastSent = this.typingLastSent.get(key) ?? 0;
    const elapsed = now - lastSent;

    // 节流：距上次发送不足 3 秒则跳过
    if (elapsed >= WebSocketEventHandlers.TYPING_THROTTLE_MS) {
      sendFn();
      this.typingLastSent.set(key, now);
      this.activeTypingTargets.add(key);
    }

    // 重置防抖定时器：停止输入 3 秒后自动发 false
    const existingDebounce = this.typingDebounceTimers.get(key);
    if (existingDebounce) clearTimeout(existingDebounce);

    this.typingDebounceTimers.set(
      key,
      setTimeout(() => {
        cancelFn();
        this.typingDebounceTimers.delete(key);
        this.typingLastSent.delete(key);
        this.activeTypingTargets.delete(key);
      }, WebSocketEventHandlers.TYPING_THROTTLE_MS)
    );
  }

  /**
   * 取消 typing（内部方法）
   */
  private cancelTyping(key: string, sendFn: () => void) {
    // 清理节流和防抖定时器
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

    // 只有当我们之前确实发送过 typing:true 才发 false
    if (this.activeTypingTargets.has(key)) {
      sendFn();
      this.activeTypingTargets.delete(key);
    }
    this.typingLastSent.delete(key);
  }
}

// ============ 导出单例 ============

// 必须先创建实例，因为 WebSocketEventHandlers 依赖 websocketStore
export const subscriptionManager = new SubscriptionManager();
export const eventHandlers = new WebSocketEventHandlers();

if (dev && browser && typeof window !== 'undefined') {
  const debugWindow = window as Window & { __wsSubscriptions?: SubscriptionManager };
  debugWindow.__wsSubscriptions = subscriptionManager;
}
