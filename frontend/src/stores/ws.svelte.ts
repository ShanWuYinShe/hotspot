// WebSocket 连接生命周期 store（runes 模块单例）。
// 职责：把 auth store 的登录态映射为 WS 连接/断开（authenticated → connect，
// anonymous → disconnect）；对外暴露连接状态供 UI 调试使用。
//
// initWs() 必须在组件初始化期间同步调用一次（+layout.svelte script 顶层，
// 与 initAuth() 一致），内部 $effect 才会正确注册到组件生命周期。
//
// mock 模式必须整体跳过真实 WS：MSW 只能拦截 HTTP，无法模拟 WebSocket；
// mock 登录态只由 localStorage 旗标驱动，真实后端没有对应会话，握手成功后
// 会立刻以 4001 auth failed 关闭 —— 客户端刷新 token 再重连，形成 1–2 秒一次
// 的「WS 握手 + refresh_token」无限循环（2026-10-28 实测：12 秒 9 轮，
// 并把后端 WS 握手限流打到 429）。

import { wsClient, type WsStatus } from '@/lib/ws/client';
import { auth, type AuthStatus } from '@/stores/auth.svelte';
import { isMockEnabled } from '@/mocks/toggle';

export type { WsStatus };

let status = $state<WsStatus>('idle');
let isConnected = $state(false);

/**
 * 登录态 + mock 开关 → WS 动作（纯函数，供 effect 与单测复用）。
 * - mock 启用：disconnect（真实后端只认真实会话，连接必然 4001 循环）；
 * - authenticated：connect；anonymous：disconnect；loading：等待。
 */
export function resolveWsAction(
  authStatus: AuthStatus,
  mockEnabled: boolean
): 'connect' | 'disconnect' | 'wait' {
  if (mockEnabled) return 'disconnect';
  if (authStatus === 'authenticated') return 'connect';
  if (authStatus === 'anonymous') return 'disconnect';
  return 'wait';
}

/**
 * 初始化 WS 生命周期绑定。返回清理函数。
 */
export function initWs(): () => void {
  const offStatus = wsClient.onStatusChange((s) => {
    status = s;
    isConnected = s === 'connected';
  });

  $effect(() => {
    // isMockEnabled() 非响应式，但 mock 开关切换会整页 reload（toggleMock），
    // 因此这里只在 effect 首次执行/登录态变化时读取即可。
    const action = resolveWsAction(auth.status, isMockEnabled());
    if (action === 'connect') {
      void wsClient.connect();
    } else if (action === 'disconnect') {
      wsClient.disconnect();
    }
    // 组件销毁时断开连接
    return () => wsClient.disconnect();
  });

  return () => {
    offStatus();
    wsClient.disconnect();
  };
}

/** 响应式连接状态（组件内读取即订阅）。 */
export const ws = {
  get status() {
    return status;
  },
  get isConnected() {
    return isConnected;
  }
};
