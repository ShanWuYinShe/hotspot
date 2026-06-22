/**
 * Home 子组件数据加载复用 hook
 *
 * 内置:
 * - loadVersion 防竞态(后发请求作废先前响应)
 * - loading / data 状态管理
 * - location-aware(mapAnchorStore 变化防抖刷新)
 *
 * 调用方负责处理 refreshTrigger prop(在 $effect 中调 reload())。
 *
 * 使用示例:
 * ```typescript
 * const { data: users, loading, reload } = useWidget<UserItem[]>(loadUsers, {
 *   locationAware: true
 * });
 *
 * $effect(() => {
 *   refreshTrigger;
 *   reload();
 * });
 * ```
 */

import { dev } from '$app/environment';
import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';

interface UseWidgetOptions {
  /** 监听 mapAnchorStore 变化(首次记录时间戳,后续变化防抖刷新) */
  locationAware?: boolean;
  /** location-aware 防抖延迟(默认 500ms) */
  debounceMs?: number;
  /** 初始 loading 状态(默认 false) */
  initialLoading?: boolean;
}

export function useWidget<T>(
  loadFn: () => Promise<T>,
  options: UseWidgetOptions = {}
) {
  const { locationAware = false, debounceMs = 500, initialLoading = false } = options;

  let data = $state<T | null>(null);
  let loading = $state(initialLoading);
  let loadVersion = 0;
  let prevAnchorUpdatedAt: number | undefined;
  let timer: ReturnType<typeof setTimeout> | null = null;

  async function reload() {
    const version = ++loadVersion;
    loading = true;
    try {
      const result = await loadFn();
      if (version !== loadVersion) return;
      data = result;
    } catch (error) {
      if (version !== loadVersion) return;
      if (dev) console.error('Widget load failed:', error);
    } finally {
      if (version === loadVersion) loading = false;
    }
  }

  if (locationAware) {
    $effect(() => {
      const now = mapAnchorStore.anchor.updatedAt;
      // 首次:只记录时间戳,不触发加载(由调用方 refreshTrigger effect 触发)
      if (prevAnchorUpdatedAt === undefined) {
        prevAnchorUpdatedAt = now;
        return;
      }
      if (prevAnchorUpdatedAt === now) return;
      prevAnchorUpdatedAt = now;
      if (timer) clearTimeout(timer);
      timer = setTimeout(() => void reload(), debounceMs);
    });
  }

  return {
    get data() {
      return data;
    },
    get loading() {
      return loading;
    },
    reload,
    setData(next: T | null) {
      data = next;
    }
  };
}
