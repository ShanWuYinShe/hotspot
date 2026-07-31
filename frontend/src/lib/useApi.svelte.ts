// createApiState — 通用数据请求工厂（迁移自 React useApi hook）。
// 每个调用创建独立响应式状态（data/loading/error + refetch），
// id 自增实现竞态覆盖：旧请求结果不会污染新请求。

import type { ApiResponse } from '@/api/client';

export interface ApiState<T> {
  data: T | null;
  loading: boolean;
  error: string | null;
  execute: () => Promise<void>;
}

export function createApiState<T>(fetcher: () => Promise<ApiResponse<T>>): ApiState<T> {
  let data = $state<T | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let id = 0;

  const execute = async () => {
    const current = ++id;
    loading = true;
    error = null;
    try {
      const res = await fetcher();
      if (current !== id) return;
      if (res.code === 0) {
        data = res.data;
        loading = false;
        error = null;
      } else {
        data = null;
        loading = false;
        error = res.message ?? `Error ${res.code}`;
      }
    } catch (err) {
      if (current !== id) return;
      data = null;
      loading = false;
      error = err instanceof Error ? err.message : '网络错误';
    }
  };

  return {
    get data() { return data; },
    get loading() { return loading; },
    get error() { return error; },
    execute,
  };
}
