/**
 * 监听地图锚点位置变化的复用 Hook (React)
 *
 * 首次立即加载，后续变化防抖刷新。自动清理定时器。
 */

import { useEffect, useRef } from 'react';
import { useMapAnchorStore } from '@/lib/stores/map-anchor';

export function useLocationAwareLoad(
  loadFn: (refresh: boolean) => Promise<void>,
  options?: { debounceMs?: number },
) {
  const debounceMs = options?.debounceMs ?? 500;
  const prevUpdatedAtRef = useRef<number | undefined>(undefined);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const loadFnRef = useRef(loadFn);
  loadFnRef.current = loadFn;

  const updatedAt = useMapAnchorStore((state) => state.anchor.updatedAt);

  useEffect(() => {
    if (prevUpdatedAtRef.current === undefined) {
      prevUpdatedAtRef.current = updatedAt;
      void loadFnRef.current(false);
      return;
    }

    if (prevUpdatedAtRef.current === updatedAt) return;

    prevUpdatedAtRef.current = updatedAt;
    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => {
      void loadFnRef.current(true);
    }, debounceMs);

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
      prevUpdatedAtRef.current = undefined;
    };
  }, [updatedAt, debounceMs]);
}
