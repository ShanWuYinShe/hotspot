import { subscriptionManager } from '$lib/websocket';

export type ChatSubscriptionType = 'user' | 'group';

export interface SyncChatSubscriptionOptions {
  connected: boolean;
  type: ChatSubscriptionType;
  ids: string[];
  onSubscribeFailed?: () => void;
}

interface ChatSubscriptionTarget {
  type: ChatSubscriptionType;
  ids: string[];
}

export function useChatWsSubscription() {
  const state = $state({
    isSubscribed: false,
    isAttemptingSubscribe: false
  });

  let syncToken = 0;
  let previousTarget: ChatSubscriptionTarget | null = null;

  function normalizeIds(ids: string[]): string[] {
    const unique = new Set<string>();
    for (const id of ids) {
      const normalized = id.trim();
      if (!normalized) continue;
      unique.add(normalized);
    }
    return Array.from(unique);
  }

  function invalidatePendingAttempt() {
    syncToken += 1;
  }

  function isSameTarget(a: ChatSubscriptionTarget | null, b: ChatSubscriptionTarget): boolean {
    if (!a) return false;
    if (a.type !== b.type) return false;
    if (a.ids.length !== b.ids.length) return false;
    const aSorted = [...a.ids].sort();
    const bSorted = [...b.ids].sort();
    return aSorted.every((id, index) => id === bSorted[index]);
  }

  function reset() {
    invalidatePendingAttempt();
    state.isSubscribed = false;
    state.isAttemptingSubscribe = false;
  }

  function sync(options: SyncChatSubscriptionOptions) {
    const ids = normalizeIds(options.ids);
    const nextTarget: ChatSubscriptionTarget = {
      type: options.type,
      ids
    };
    const targetChanged = !isSameTarget(previousTarget, nextTarget);

    if (targetChanged && previousTarget && previousTarget.ids.length > 0) {
      void subscriptionManager.unsubscribe(previousTarget.type, previousTarget.ids);
    }
    if (targetChanged) {
      previousTarget = nextTarget;
      invalidatePendingAttempt();
      state.isSubscribed = false;
      state.isAttemptingSubscribe = false;
    }

    if (!options.connected) {
      if (targetChanged && ids.length > 0) {
        // 离线也要维护 desired，确保重连后恢复为最新目标
        void subscriptionManager.subscribe(options.type, ids);
      }
      state.isSubscribed = false;
      state.isAttemptingSubscribe = false;
      return;
    }

    if (ids.length === 0 || state.isSubscribed || state.isAttemptingSubscribe) {
      return;
    }

    state.isAttemptingSubscribe = true;
    const attemptToken = ++syncToken;

    subscriptionManager.subscribe(options.type, ids).then((success) => {
      if (attemptToken !== syncToken) return;
      state.isSubscribed = success;
      state.isAttemptingSubscribe = false;
      if (!success) {
        options.onSubscribeFailed?.();
      }
    });
  }

  function cleanup(type: ChatSubscriptionType, ids: string[]) {
    const normalizedIds = normalizeIds(ids);
    invalidatePendingAttempt();
    state.isSubscribed = false;
    state.isAttemptingSubscribe = false;
    if (normalizedIds.length > 0) {
      void subscriptionManager.unsubscribe(type, normalizedIds);
    }
    if (previousTarget && previousTarget.type === type) {
      const targetMatchesCleanup = isSameTarget(previousTarget, {
        type,
        ids: normalizedIds
      });
      if (targetMatchesCleanup) {
        previousTarget = null;
      }
    }
  }

  return {
    state,
    reset,
    sync,
    cleanup
  };
}
