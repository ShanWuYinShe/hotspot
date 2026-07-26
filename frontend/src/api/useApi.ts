import { useState, useEffect, useCallback, useRef } from 'react';
import type { ApiResponse } from './client';

interface UseApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
}

export function useApi<T>(
  fetcher: () => Promise<ApiResponse<T>>,
  deps: unknown[]
) {
  const [state, setState] = useState<UseApiState<T>>({ data: null, loading: true, error: null });
  const idRef = useRef(0);

  const execute = useCallback(async () => {
    const id = ++idRef.current;
    setState(prev => ({ ...prev, loading: true, error: null }));
    try {
      const res = await fetcher();
      if (id !== idRef.current) return;
      if (res.code === 0) {
        setState({ data: res.data, loading: false, error: null });
      } else {
        setState({ data: null, loading: false, error: res.message ?? `Error ${res.code}` });
      }
    } catch (err) {
      if (id !== idRef.current) return;
      setState({ data: null, loading: false, error: err instanceof Error ? err.message : '网络错误' });
    }
  }, deps);

  useEffect(() => {
    execute();
  }, [execute]);

  return { ...state, refetch: execute };
}
