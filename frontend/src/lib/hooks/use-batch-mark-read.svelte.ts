/**
 * 批量已读标记 Hook
 *
 * 功能：
 * 1. 使用 IntersectionObserver 检测消息可见性
 * 2. 批量标记：收集可见的未读消息 ID，定时批量发送
 * 3. 防抖优化：避免频繁 API 调用
 * 4. 离开页面时：批量标记所有可见过的消息
 */

import { onMount } from 'svelte';
import { dev } from '$app/environment';

export interface UseBatchMarkReadOptions {
  /**
   * 标记已读的 API 调用函数
   */
  markAsRead: (messageIds?: string[]) => Promise<void>;

  /**
   * 防抖延迟（毫秒）
   * @default 2000
   */
  debounceMs?: number;

  /**
   * 消息在视口内停留多久才标记为已读（毫秒）
   * @default 1000
   */
  visibilityThreshold?: number;

  /**
   * 是否启用批量标记
   * @default true
   */
  enabled?: boolean;
}

export interface BatchMarkReadResult {
  /**
   * 注册消息元素用于可见性检测
   */
  observe: (element: HTMLElement, messageId: string) => void;

  /**
   * 取消观察消息元素
   */
  unobserve: (element: HTMLElement) => void;

  /**
   * 立即标记所有可见过的消息为已读
   */
  markAllVisibleAsRead: () => Promise<void>;

  /**
   * 清理资源
   */
  cleanup: () => void;
}

/**
 * 使用批量已读标记
 */
export function useBatchMarkRead(options: UseBatchMarkReadOptions): BatchMarkReadResult {
  const { markAsRead, debounceMs = 2000, visibilityThreshold = 1000, enabled = true } = options;

  // 已可见的消息 ID 数组（使用 $state.raw 因为只做整体替换，不需要深度响应式）
  let visibleMessageIds = $state.raw<string[]>([]);

  // 等待标记的消息 ID 数组
  let pendingMessageIds = $state.raw<string[]>([]);

  // 消息元素与 ID 的映射
  const elementToIdMap = new Map<HTMLElement, string>();

  // 消息可见时间戳
  const visibilityTimestamps = new Map<string, number>();

  // IntersectionObserver 实例
  let observer: IntersectionObserver | null = null;

  // 防抖定时器
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  // 可见性延迟检查定时器（消息 ID -> timer ID）
  const visibilityTimers = new Map<string, ReturnType<typeof setTimeout>>();

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

  /**
   * 批量标记已读
   */
  async function batchMarkAsRead() {
    if (!enabled || pendingMessageIds.length === 0) return;

    const idsToMark = [...pendingMessageIds];
    pendingMessageIds = [];

    try {
      await markAsRead(idsToMark);
      if (dev) console.log(`[BatchMarkRead] 已标记 ${idsToMark.length} 条消息为已读`);
    } catch (error) {
      if (dev) console.error('[BatchMarkRead] 标记已读失败:', error);
      // 失败时重新加入队列（去重，避免同一批消息无限堆积）
      pendingMessageIds = appendUniqueIds(pendingMessageIds, idsToMark);
    }
  }

  /**
   * 调度批量标记（防抖）
   */
  function scheduleBatchMarkAsRead() {
    if (debounceTimer) clearTimeout(debounceTimer);
    debounceTimer = setTimeout(() => {
      batchMarkAsRead();
    }, debounceMs);
  }

  /**
   * 处理元素可见性变化
   */
  function handleIntersection(entries: IntersectionObserverEntry[]) {
    if (!enabled) return;

    const now = Date.now();

    entries.forEach((entry) => {
      const messageId = elementToIdMap.get(entry.target as HTMLElement);
      if (!messageId) return;

      if (entry.isIntersecting) {
        // 元素进入视口
        if (!visibilityTimestamps.has(messageId)) {
          visibilityTimestamps.set(messageId, now);
        }

        // 检查是否停留足够时间
        const visibleSince = visibilityTimestamps.get(messageId)!;
        const visibleDuration = now - visibleSince;

        if (visibleDuration >= visibilityThreshold) {
          // 标记为可见（使用数组操作）
          if (!visibleMessageIds.includes(messageId)) {
            visibleMessageIds = [...visibleMessageIds, messageId];
            pendingMessageIds = appendUniqueIds(pendingMessageIds, [messageId]);
            scheduleBatchMarkAsRead();
          }
        } else {
          // 尚未停留足够时间，稍后再检查
          // 清除该消息之前的延迟检查定时器（如有）
          const existingTimer = visibilityTimers.get(messageId);
          if (existingTimer) clearTimeout(existingTimer);

          const timer = setTimeout(() => {
            visibilityTimers.delete(messageId);
            if (elementToIdMap.has(entry.target as HTMLElement)) {
              handleIntersection([entry]);
            }
          }, visibilityThreshold - visibleDuration);
          visibilityTimers.set(messageId, timer);
        }
      } else {
        // 元素离开视口
        visibilityTimestamps.delete(messageId);
        // 清除该消息的延迟检查定时器
        const existingTimer = visibilityTimers.get(messageId);
        if (existingTimer) {
          clearTimeout(existingTimer);
          visibilityTimers.delete(messageId);
        }
      }
    });
  }

  /**
   * 确保 IntersectionObserver 已创建（懒初始化）
   */
  function ensureObserver(): boolean {
    if (observer) return true;
    if (!enabled) return false;

    observer = new IntersectionObserver(handleIntersection, {
      root: null, // 视口
      rootMargin: '0px',
      threshold: 0.5 // 50% 可见时触发
    });

    // 观察之前挂起的元素
    for (const [element] of elementToIdMap) {
      observer.observe(element);
    }

    return true;
  }

  /**
   * 注册消息元素用于可见性检测
   * 如果 observer 尚未创建，会先懒初始化；如果初始化失败则暂存映射
   */
  function observe(element: HTMLElement, messageId: string) {
    if (!enabled) return;

    elementToIdMap.set(element, messageId);

    // 尝试懒初始化并观察
    if (ensureObserver()) {
      observer!.observe(element);
    }
    // 如果 observer 尚未准备好（不应发生），元素已在 map 中，ensureObserver 时会补观察
  }

  /**
   * 取消观察消息元素
   */
  function unobserve(element: HTMLElement) {
    if (!observer) return;

    observer.unobserve(element);
    elementToIdMap.delete(element);
  }

  /**
   * 立即标记所有可见过的消息为已读
   */
  async function markAllVisibleAsRead() {
    if (debounceTimer) {
      clearTimeout(debounceTimer);
      debounceTimer = null;
    }
    await batchMarkAsRead();
  }

  /**
   * 清理资源
   */
  function cleanup() {
    if (observer) {
      observer.disconnect();
      observer = null;
    }
    if (debounceTimer) {
      clearTimeout(debounceTimer);
      debounceTimer = null;
    }
    // 清除所有可见性延迟检查定时器
    for (const timer of visibilityTimers.values()) {
      clearTimeout(timer);
    }
    visibilityTimers.clear();
    elementToIdMap.clear();
    visibilityTimestamps.clear();
    // 数组使用重新赋值清空，不是 .clear()
    visibleMessageIds = [];
    pendingMessageIds = [];
  }

  // 初始化
  onMount(() => {
    ensureObserver();

    // 页面卸载时：先完成标记，再清理资源
    return () => {
      markAllVisibleAsRead().finally(() => cleanup());
    };
  });

  return {
    observe,
    unobserve,
    markAllVisibleAsRead,
    cleanup
  };
}
