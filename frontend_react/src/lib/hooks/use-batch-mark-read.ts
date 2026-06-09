/**
 * 批量已读标记 Hook (React)
 *
 * 1. 使用 IntersectionObserver 检测消息可见性
 * 2. 批量标记：收集可见的未读消息 ID，定时批量发送
 * 3. 防抖优化：避免频繁 API 调用
 * 4. 离开页面时：批量标记所有可见过的消息
 */

import { useEffect, useRef } from "react";

export interface UseBatchMarkReadOptions {
  markAsRead: (messageIds?: string[]) => Promise<void>;
  debounceMs?: number;
  visibilityThreshold?: number;
  enabled?: boolean;
}

export interface BatchMarkReadResult {
  observe: (element: HTMLElement, messageId: string) => void;
  unobserve: (element: HTMLElement) => void;
  markAllVisibleAsRead: () => Promise<void>;
}

export function useBatchMarkRead(
  options: UseBatchMarkReadOptions,
): BatchMarkReadResult {
  const {
    markAsRead,
    debounceMs = 2000,
    visibilityThreshold = 1000,
    enabled = true,
  } = options;

  const visibleMessageIdsRef = useRef<string[]>([]);
  const pendingMessageIdsRef = useRef<string[]>([]);
  const elementToIdMapRef = useRef(new Map<HTMLElement, string>());
  const visibilityTimestampsRef = useRef(new Map<string, number>());
  const observerRef = useRef<IntersectionObserver | null>(null);
  const debounceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const visibilityTimersRef = useRef(
    new Map<string, ReturnType<typeof setTimeout>>(),
  );

  function appendUniqueIds(target: string[], ids: string[]): string[] {
    if (ids.length === 0) return target;
    const existing = new Set(target);
    const next = [...target];
    for (const id of ids) {
      if (existing.has(id)) continue;
      existing.add(id);
      next.push(id);
    }
    return next;
  }

  async function batchMarkAsRead() {
    if (!enabled || pendingMessageIdsRef.current.length === 0) return;

    const idsToMark = [...pendingMessageIdsRef.current];
    pendingMessageIdsRef.current = [];

    try {
      await markAsRead(idsToMark);
    } catch {
      pendingMessageIdsRef.current = appendUniqueIds(
        pendingMessageIdsRef.current,
        idsToMark,
      );
    }
  }

  function scheduleBatchMarkAsRead() {
    if (debounceTimerRef.current) clearTimeout(debounceTimerRef.current);
    debounceTimerRef.current = setTimeout(() => {
      batchMarkAsRead();
    }, debounceMs);
  }

  // handleIntersection 需要通过 ref 间接引用，以避免闭包陷阱
  const handleIntersectionRef = useRef<
    (entries: IntersectionObserverEntry[]) => void
  >(() => {});

  handleIntersectionRef.current = (entries: IntersectionObserverEntry[]) => {
    if (!enabled) return;
    const now = Date.now();

    for (const entry of entries) {
      const messageId = elementToIdMapRef.current.get(
        entry.target as HTMLElement,
      );
      if (!messageId) continue;

      if (entry.isIntersecting) {
        if (!visibilityTimestampsRef.current.has(messageId)) {
          visibilityTimestampsRef.current.set(messageId, now);
        }

        const visibleSince = visibilityTimestampsRef.current.get(messageId)!;
        const visibleDuration = now - visibleSince;

        if (visibleDuration >= visibilityThreshold) {
          if (!visibleMessageIdsRef.current.includes(messageId)) {
            visibleMessageIdsRef.current = [
              ...visibleMessageIdsRef.current,
              messageId,
            ];
            pendingMessageIdsRef.current = appendUniqueIds(
              pendingMessageIdsRef.current,
              [messageId],
            );
            scheduleBatchMarkAsRead();
          }
        } else {
          const existingTimer = visibilityTimersRef.current.get(messageId);
          if (existingTimer) clearTimeout(existingTimer);

          const timer = setTimeout(() => {
            visibilityTimersRef.current.delete(messageId);
            if (elementToIdMapRef.current.has(entry.target as HTMLElement)) {
              handleIntersectionRef.current([entry]);
            }
          }, visibilityThreshold - visibleDuration);
          visibilityTimersRef.current.set(messageId, timer);
        }
      } else {
        visibilityTimestampsRef.current.delete(messageId);
        const existingTimer = visibilityTimersRef.current.get(messageId);
        if (existingTimer) {
          clearTimeout(existingTimer);
          visibilityTimersRef.current.delete(messageId);
        }
      }
    }
  };

  function ensureObserver(): boolean {
    if (observerRef.current) return true;
    if (!enabled) return false;

    observerRef.current = new IntersectionObserver(
      (entries) => handleIntersectionRef.current(entries),
      {
        root: null,
        rootMargin: "0px",
        threshold: 0.5,
      },
    );

    for (const [element] of elementToIdMapRef.current) {
      observerRef.current.observe(element);
    }

    return true;
  }

  function observe(element: HTMLElement, messageId: string) {
    if (!enabled) return;
    elementToIdMapRef.current.set(element, messageId);
    if (ensureObserver()) {
      observerRef.current!.observe(element);
    }
  }

  function unobserve(element: HTMLElement) {
    if (!observerRef.current) return;
    observerRef.current.unobserve(element);
    elementToIdMapRef.current.delete(element);
  }

  async function markAllVisibleAsRead() {
    if (debounceTimerRef.current) {
      clearTimeout(debounceTimerRef.current);
      debounceTimerRef.current = null;
    }
    await batchMarkAsRead();
  }

  useEffect(() => {
    ensureObserver();

    return () => {
      markAllVisibleAsRead().finally(() => {
        if (observerRef.current) {
          observerRef.current.disconnect();
          observerRef.current = null;
        }
        if (debounceTimerRef.current) {
          clearTimeout(debounceTimerRef.current);
          debounceTimerRef.current = null;
        }
        for (const timer of visibilityTimersRef.current.values()) {
          clearTimeout(timer);
        }
        visibilityTimersRef.current.clear();
        elementToIdMapRef.current.clear();
        visibilityTimestampsRef.current.clear();
        visibleMessageIdsRef.current = [];
        pendingMessageIdsRef.current = [];
      });
    };
  }, [enabled]);

  return { observe, unobserve, markAllVisibleAsRead };
}
