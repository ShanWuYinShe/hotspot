// API client facade —— 按域拆分后的统一出口：
// - 传输核心（request / 主动续期 / 幂等键 / 双版本 pick / netError）见 ./core.ts
// - 各业务域端点包装见 ./domains/*.ts
// 本文件只做 re-export，不承载实现；新增端点请放入对应域文件。
export * from './core';
import type { ApiResponse } from './core';
export { Endpoints } from '@/api/endpoints';

// netError —— 网络层失败（fetch 抛错/超时）时的统一兜底响应：
// code=-1 与业务错误码空间区分，data 传调用方的空态回退（null / { items: [] }）。
// 收敛散落各页的 `.catch(() => netError(…))` 样板。
export function netError<T>(fallback: T): ApiResponse<T> {
	return { code: -1, message: '网络异常', data: fallback };
}

export * from './domains/auth';
export * from './domains/users';
export * from './domains/media';
export * from './domains/moments';
export * from './domains/groups';
export * from './domains/search';
export * from './domains/notify';
export * from './domains/system';
export * from './domains/reports';
export * from './domains/chat';
export * from './domains/comments';
export * from './domains/map';
export * from './domains/share';
export * from './domains/admin';
