// API client facade —— 按域拆分后的统一出口：
// - 传输核心（request / 主动续期 / 幂等键 / 双版本 pick / netError）见 ./core.ts
// - 各业务域端点包装见 ./domains/*.ts
// 本文件只做 re-export，不承载实现；新增端点请放入对应域文件。
export * from './core';
export { Endpoints } from '@/api/endpoints';

// 注：netError 只保留 ./core.ts 一份实现（此前本文件另有一份同名同实现的副本，
// 显式导出会遮蔽星号导出，两份可能静默漂移）。facade 只做 re-export。

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
