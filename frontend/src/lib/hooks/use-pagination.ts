/**
 * 通用分页 Hook (React)
 *
 * 管理 cursor 分页状态，带竞态保护和去重
 */

import { useState, useRef, useCallback } from "react";
import { isSuccess } from "@/lib/api/client";
import type { ApiResponse, PaginatedData } from "@/lib/types/api";
import { withMinimumLoadingTime } from "@/lib/utils/loading";

interface UsePaginationOptions<T> {
  initialLoading?: boolean;
  uniqueKey?: keyof T | ((item: T) => string | number);
}

export interface PaginationState<T> {
  items: T[];
  loading: boolean;
  loadingMore: boolean;
  hasMore: boolean;
  error: string | null;
  load: (
    fetcher: (cursor?: string) => Promise<ApiResponse<PaginatedData<T>>>,
    refresh?: boolean,
  ) => Promise<void>;
  reset: () => void;
  setError: (error: string | null) => void;
  addItem: (item: T) => void;
  updateItem: (updater: (item: T) => T | null) => void;
  removeItem: (predicate: (item: T) => boolean) => void;
  setItems: (newItems: T[]) => void;
}

export function usePagination<T>(
  options?: UsePaginationOptions<T>,
): PaginationState<T> {
  const [items, setItemsState] = useState<T[]>([]);
  const [loading, setLoading] = useState(options?.initialLoading ?? true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [error, setErrorState] = useState<string | null>(null);

  const cursorRef = useRef<string | undefined>(undefined);
  const loadVersionRef = useRef(0);
  const isLoadingMoreRef = useRef(false);

  const load = useCallback(
    async (
      fetcher: (cursor?: string) => Promise<ApiResponse<PaginatedData<T>>>,
      refresh = false,
    ) => {
      if (!refresh) {
        if (isLoadingMoreRef.current) return;
        isLoadingMoreRef.current = true;
      }

      const version = ++loadVersionRef.current;

      if (refresh) {
        setItemsState([]);
        cursorRef.current = undefined;
        setHasMore(true);
        setLoading(true);
        setErrorState(null);
      } else {
        setLoadingMore(true);
      }

      try {
        const response = refresh
          ? await withMinimumLoadingTime(() => fetcher(undefined))
          : await fetcher(cursorRef.current);

        if (version !== loadVersionRef.current) return;

        if (isSuccess(response) && response.data) {
          const newItems = response.data.items ?? [];

          if (refresh) {
            setItemsState(newItems);
          } else {
            setItemsState((prev) => {
              if (options?.uniqueKey) {
                const uniqueKeyOpt = options.uniqueKey;
                const getKey = (item: T): unknown =>
                  typeof uniqueKeyOpt === "function"
                    ? uniqueKeyOpt(item)
                    : item[uniqueKeyOpt as keyof T];

                const existingKeys = new Set(prev.map(getKey));
                const uniqueNewItems = newItems.filter(
                  (item) => !existingKeys.has(getKey(item)),
                );
                return [...prev, ...uniqueNewItems];
              }
              return [...prev, ...newItems];
            });
          }

          setHasMore(!!response.data.next_cursor);
          cursorRef.current = response.data.next_cursor;
          setErrorState(null);
        } else {
          setErrorState(response.message || "加载失败");
        }
      } catch (e) {
        if (version !== loadVersionRef.current) return;
        setErrorState(e instanceof Error ? e.message : "加载失败");
      } finally {
        if (version === loadVersionRef.current) {
          setLoading(false);
          setLoadingMore(false);
        }
        if (!refresh) isLoadingMoreRef.current = false;
      }
    },
    [options?.uniqueKey],
  );

  const reset = useCallback(() => {
    setItemsState([]);
    cursorRef.current = undefined;
    setHasMore(true);
    setLoading(options?.initialLoading ?? true);
    setErrorState(null);
  }, [options?.initialLoading]);

  const setError = useCallback((newError: string | null) => {
    setErrorState(newError);
  }, []);

  const addItem = useCallback((item: T) => {
    setItemsState((prev) => [item, ...prev]);
  }, []);

  const updateItem = useCallback((updater: (item: T) => T | null) => {
    setItemsState((prev) => prev.map((item) => updater(item) ?? item));
  }, []);

  const removeItem = useCallback((predicate: (item: T) => boolean) => {
    setItemsState((prev) => prev.filter((item) => !predicate(item)));
  }, []);

  const setItems = useCallback((newItems: T[]) => {
    setItemsState(newItems);
  }, []);

  return {
    items,
    loading,
    loadingMore,
    hasMore,
    error,
    load,
    reset,
    setError,
    addItem,
    updateItem,
    removeItem,
    setItems,
  };
}
