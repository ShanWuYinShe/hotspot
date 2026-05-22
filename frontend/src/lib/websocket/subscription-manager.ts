/**
 * WebSocket 订阅管理器
 * 严格遵循 @spec/api_spec_websocket.md
 *
 * 管理客户端"期望订阅"状态与服务端"已生效订阅"快照，
 * 支持漂移检测、自动对账、重连恢复。
 */

import { useWebSocketStore } from './store';
import type {
  SubscriptionOperation,
  SubscribeAckPayload,
  SubscriptionsSnapshotPayload,
} from './types';

export type SubscriptionType = 'user' | 'group';

export class SubscriptionManager {
  private desiredSubscriptions = new Map<SubscriptionType, Set<string>>();
  private effectiveSubscriptions = new Map<SubscriptionType, Set<string>>();
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
  private cleanupFns: (() => void)[] = [];

  constructor() {
    this.initMap(this.desiredSubscriptions);
    this.initMap(this.effectiveSubscriptions);

    this.cleanupFns.push(
      useWebSocketStore.getState().on('SubscribeAck', (p) => this.handleSubscribeAck(p)),
      useWebSocketStore.getState().on('SubscriptionsSnapshot', (p) =>
        this.handleSubscriptionsSnapshot(p),
      ),
      useWebSocketStore.getState().on('ConnectionEstablished', () =>
        this.restoreAndReconcileSubscriptions(),
      ),
    );
  }

  destroy() {
    this.cleanupFns.forEach((fn) => fn());
    this.cleanupFns = [];
  }

  subscribe(type: SubscriptionType, ids: string[]): Promise<boolean> {
    const normalized = this.normalize(ids);
    if (normalized.length === 0) return Promise.resolve(true);
    this.applyDesired('subscribe', type, normalized);
    return this.sendOrQueue('subscribe', type, normalized);
  }

  unsubscribe(type: SubscriptionType, ids: string[]): Promise<boolean> {
    const normalized = this.normalize(ids);
    if (normalized.length === 0) return Promise.resolve(true);
    this.applyDesired('unsubscribe', type, normalized);
    return this.sendOrQueue('unsubscribe', type, normalized);
  }

  setSubscriptions(type: SubscriptionType, ids: string[]): Promise<boolean> {
    const normalized = this.normalize(ids);
    this.applyDesired('set', type, normalized);
    return this.sendOrQueue('set', type, normalized);
  }

  getSubscriptions(type: SubscriptionType): string[] {
    return Array.from(this.desiredSubscriptions.get(type) ?? []);
  }

  getEffectiveSubscriptions(type: SubscriptionType): string[] {
    return Array.from(this.effectiveSubscriptions.get(type) ?? []);
  }

  isSubscribed(type: SubscriptionType, id: string): boolean {
    return this.desiredSubscriptions.get(type)?.has(id) ?? false;
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
        group.extra_on_server.length > 0,
    };
  }

  cleanupPending() {
    for (const [, pending] of this.pendingAcks) {
      clearTimeout(pending.timer);
      pending.resolve(false);
    }
    this.pendingAcks.clear();
  }

  // ─── 私有方法 ───

  private sendOrQueue(
    operation: SubscriptionOperation,
    type: SubscriptionType,
    ids: string[],
  ): Promise<boolean> {
    if (!useWebSocketStore.getState().connectionState.startsWith('connect')) {
      return Promise.resolve(true);
    }
    return this.sendRequest(operation, type, ids);
  }

  private sendRequest(
    operation: SubscriptionOperation,
    type: SubscriptionType,
    ids: string[],
  ): Promise<boolean> {
    const isConnected = useWebSocketStore.getState().connectionState === 'connected';
    if (!isConnected) return Promise.resolve(true);

    const clientMsgId = crypto.randomUUID();

    useWebSocketStore.getState().send({
      type: 'SetSubscriptions',
      payload: { client_request_id: clientMsgId, operation, type, ids },
    });

    return new Promise((resolve) => {
      const timer = setTimeout(() => {
        if (this.pendingAcks.has(clientMsgId)) {
          this.pendingAcks.delete(clientMsgId);
          resolve(false);
        }
      }, 10000);

      this.pendingAcks.set(clientMsgId, { operation, type, ids, resolve, timer });
    });
  }

  private handleSubscribeAck(payload: SubscribeAckPayload) {
    const { client_request_id, operation, type, ids, code, failed_ids } = payload;
    const subType = type as SubscriptionType;
    const failedIds = this.normalize(failed_ids?.map((f) => f.subscription_id) ?? []);
    const successfulIds = this.excludeFailed(ids, failedIds);
    this.updateEffective(subType, successfulIds);

    if (failedIds.length > 0) {
      this.rollbackDesired(operation, subType, failedIds, successfulIds);
    }
    if (!this.isRestoreInFlight) {
      this.reconcileDriftIfNeeded('subscribe_ack');
    }

    const pending = this.pendingAcks.get(client_request_id);
    if (!pending) return;

    clearTimeout(pending.timer);
    this.pendingAcks.delete(client_request_id);
    pending.resolve(code === 0);
  }

  private handleSubscriptionsSnapshot(payload: SubscriptionsSnapshotPayload) {
    this.updateEffective('user', this.normalize(payload.user_ids ?? []));
    this.updateEffective('group', this.normalize(payload.group_ids ?? []));
    this.reconcileDriftIfNeeded('subscriptions_snapshot');
  }

  private async restoreSubscriptions() {
    for (const type of ['user', 'group'] as SubscriptionType[]) {
      const ids = Array.from(this.desiredSubscriptions.get(type) ?? []);
      await this.sendOrQueue('set', type, ids);
    }
  }

  private async restoreAndReconcileSubscriptions() {
    this.isRestoreInFlight = true;
    try {
      await this.restoreSubscriptions();
      await this.waitForPendingAcks();
      const isConnected = useWebSocketStore.getState().connectionState === 'connected';
      if (isConnected) {
        useWebSocketStore.getState().send({
          type: 'GetSubscriptions',
          payload: { client_request_id: crypto.randomUUID() },
        });
      }
    } finally {
      this.isRestoreInFlight = false;
    }
  }

  private initMap(map: Map<SubscriptionType, Set<string>>) {
    map.set('user', new Set());
    map.set('group', new Set());
  }

  private normalize(ids: string[]): string[] {
    const unique = new Set<string>();
    for (const id of ids) {
      const n = id.trim();
      if (n) unique.add(n);
    }
    return Array.from(unique);
  }

  private excludeFailed(ids: string[], failedIds: string[]): string[] {
    if (failedIds.length === 0) return this.normalize(ids);
    const failedSet = new Set(this.normalize(failedIds));
    return this.normalize(ids).filter((id) => !failedSet.has(id));
  }

  private applyDesired(
    operation: SubscriptionOperation,
    type: SubscriptionType,
    ids: string[],
  ) {
    const set = this.desiredSubscriptions.get(type);
    if (!set) return;
    switch (operation) {
      case 'subscribe':
        ids.forEach((id) => set.add(id));
        break;
      case 'unsubscribe':
        ids.forEach((id) => set.delete(id));
        break;
      case 'set':
        set.clear();
        ids.forEach((id) => set.add(id));
        break;
    }
  }

  private updateEffective(type: SubscriptionType, ids: string[]) {
    const set = this.effectiveSubscriptions.get(type);
    if (!set) return;
    set.clear();
    this.normalize(ids).forEach((id) => set.add(id));
  }

  private rollbackDesired(
    operation: SubscriptionOperation,
    type: SubscriptionType,
    failedIds: string[],
    serverSnapshotIds: string[],
  ) {
    const set = this.desiredSubscriptions.get(type);
    if (!set) return;
    switch (operation) {
      case 'subscribe':
        failedIds.forEach((id) => set.delete(id));
        break;
      case 'unsubscribe':
        failedIds.forEach((id) => set.add(id));
        break;
      case 'set':
        set.clear();
        this.normalize(serverSnapshotIds).forEach((id) => set.add(id));
        break;
    }
  }

  private toSorted(set: Set<string> | undefined): string[] {
    return set ? Array.from(set).sort() : [];
  }

  private async waitForPendingAcks(timeoutMs = 1500) {
    const start = Date.now();
    while (this.pendingAcks.size > 0 && Date.now() - start < timeoutMs) {
      await new Promise((r) => setTimeout(r, 50));
    }
  }

  private computeDrift(type: SubscriptionType) {
    const desired = this.desiredSubscriptions.get(type) ?? new Set<string>();
    const effective = this.effectiveSubscriptions.get(type) ?? new Set<string>();
    return {
      missing_on_server: this.toSorted(new Set(Array.from(desired).filter((id) => !effective.has(id)))),
      extra_on_server: this.toSorted(new Set(Array.from(effective).filter((id) => !desired.has(id)))),
    };
  }

  private async reconcileDriftIfNeeded(_trigger: 'subscribe_ack' | 'subscriptions_snapshot') {
    const drift = this.getDriftState();
    if (!drift.has_drift) return;

    const now = Date.now();
    if (
      this.isDriftReconciling ||
      now - this.lastDriftReconcileAt < SubscriptionManager.DRIFT_RECONCILE_COOLDOWN_MS
    )
      return;

    this.isDriftReconciling = true;
    this.lastDriftReconcileAt = now;
    try {
      for (const type of ['user', 'group'] as SubscriptionType[]) {
        await this.sendOrQueue('set', type, this.toSorted(this.desiredSubscriptions.get(type)));
      }
    } finally {
      this.isDriftReconciling = false;
    }
  }
}
