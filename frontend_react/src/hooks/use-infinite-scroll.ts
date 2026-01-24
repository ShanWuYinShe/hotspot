'use client';

import { useCallback, useRef, useEffect } from 'react';

interface UseInfiniteScrollOptions {
  threshold?: number;
  rootMargin?: string;
  enabled?: boolean;
}

export function useInfiniteScroll(
  onLoadMore: () => void,
  options: UseInfiniteScrollOptions = {}
) {
  const { threshold = 0.1, rootMargin = '100px', enabled = true } = options;
  
  const observerRef = useRef<IntersectionObserver | null>(null);
  const loadMoreRef = useRef<HTMLDivElement | null>(null);
  const isLoadingRef = useRef(false);

  const setLoadMoreRef = useCallback((node: HTMLDivElement | null) => {
    if (loadMoreRef.current) {
      observerRef.current?.unobserve(loadMoreRef.current);
    }

    loadMoreRef.current = node;

    if (node && observerRef.current) {
      observerRef.current.observe(node);
    }
  }, []);

  useEffect(() => {
    if (!enabled) {
      observerRef.current?.disconnect();
      return;
    }

    observerRef.current = new IntersectionObserver(
      (entries) => {
        const [entry] = entries;
        if (entry?.isIntersecting && !isLoadingRef.current) {
          isLoadingRef.current = true;
          onLoadMore();
          // 重置加载状态（假设 onLoadMore 会在一段时间后完成）
          setTimeout(() => {
            isLoadingRef.current = false;
          }, 500);
        }
      },
      {
        threshold,
        rootMargin,
      }
    );

    if (loadMoreRef.current) {
      observerRef.current.observe(loadMoreRef.current);
    }

    return () => {
      observerRef.current?.disconnect();
    };
  }, [enabled, onLoadMore, threshold, rootMargin]);

  return {
    loadMoreRef: setLoadMoreRef,
  };
}
