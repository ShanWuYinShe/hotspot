/**
 * 聊天 WebSocket 订阅生命周期管理 Hook
 */

import { useRef, useCallback, useState } from "react";
import { SubscriptionManager } from "@/lib/websocket/subscription-manager";

export type ChatSubscriptionType = "user" | "group";

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

function normalizeIds(ids: string[]): string[] {
  const unique = new Set<string>();
  for (const id of ids) {
    const normalized = id.trim();
    if (!normalized) continue;
    unique.add(normalized);
  }
  return Array.from(unique);
}

function isSameTarget(
  a: ChatSubscriptionTarget | null,
  b: ChatSubscriptionTarget,
): boolean {
  if (!a) return false;
  if (a.type !== b.type) return false;
  if (a.ids.length !== b.ids.length) return false;
  const aSorted = [...a.ids].sort();
  const bSorted = [...b.ids].sort();
  return aSorted.every((id, index) => id === bSorted[index]);
}

export function useChatWsSubscription(
  subscriptionManager: SubscriptionManager,
) {
  const [isSubscribed, setIsSubscribed] = useState(false);
  const [isAttemptingSubscribe, setIsAttemptingSubscribe] = useState(false);

  const syncTokenRef = useRef(0);
  const previousTargetRef = useRef<ChatSubscriptionTarget | null>(null);

  const reset = useCallback(() => {
    syncTokenRef.current += 1;
    setIsSubscribed(false);
    setIsAttemptingSubscribe(false);
  }, []);

  const sync = useCallback(
    (options: SyncChatSubscriptionOptions) => {
      const ids = normalizeIds(options.ids);
      const nextTarget: ChatSubscriptionTarget = { type: options.type, ids };
      const previousTarget = previousTargetRef.current;
      const targetChanged = !isSameTarget(previousTarget, nextTarget);

      if (targetChanged && previousTarget && previousTarget.ids.length > 0) {
        void subscriptionManager.unsubscribe(
          previousTarget.type,
          previousTarget.ids,
        );
      }
      if (targetChanged) {
        previousTargetRef.current = nextTarget;
        syncTokenRef.current += 1;
        setIsSubscribed(false);
        setIsAttemptingSubscribe(false);
      }

      if (!options.connected) {
        if (targetChanged && ids.length > 0) {
          void subscriptionManager.subscribe(options.type, ids);
        }
        setIsSubscribed(false);
        setIsAttemptingSubscribe(false);
        return;
      }

      if (ids.length === 0 || isSubscribed || isAttemptingSubscribe) {
        return;
      }

      setIsAttemptingSubscribe(true);
      const attemptToken = ++syncTokenRef.current;

      subscriptionManager.subscribe(options.type, ids).then((success) => {
        if (attemptToken !== syncTokenRef.current) return;
        setIsSubscribed(success);
        setIsAttemptingSubscribe(false);
        if (!success) {
          options.onSubscribeFailed?.();
        }
      });
    },
    [subscriptionManager, isSubscribed, isAttemptingSubscribe],
  );

  const cleanup = useCallback(
    (type: ChatSubscriptionType, ids: string[]) => {
      const normalizedIds = normalizeIds(ids);
      syncTokenRef.current += 1;
      setIsSubscribed(false);
      setIsAttemptingSubscribe(false);
      if (normalizedIds.length > 0) {
        void subscriptionManager.unsubscribe(type, normalizedIds);
      }
      const previousTarget = previousTargetRef.current;
      if (previousTarget && previousTarget.type === type) {
        if (isSameTarget(previousTarget, { type, ids: normalizedIds })) {
          previousTargetRef.current = null;
        }
      }
    },
    [subscriptionManager],
  );

  return { isSubscribed, isAttemptingSubscribe, reset, sync, cleanup };
}
