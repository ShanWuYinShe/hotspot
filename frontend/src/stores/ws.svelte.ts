// WebSocket 连接生命周期 store（runes 模块单例）。
// 职责：把 auth store 的登录态映射为 WS 连接/断开（authenticated → connect，
// anonymous → disconnect）；对外暴露连接状态供 UI 调试使用。
//
// initWs() 必须在组件初始化期间同步调用一次（+layout.svelte script 顶层，
// 与 initAuth() 一致），内部 $effect 才会正确注册到组件生命周期。
//
// 真实 WS 连接由真实会话驱动：服务端以 4001 关闭（Token 失效）时由
// lib/ws/client.ts 的 handleAuthCloseWithRefresh 刷新 token 后重连（最多 2 轮，
// 超过即派发 auth-failed 交给上层登出）。

import { wsClient, type WsStatus } from '@/lib/ws/client';
import { auth, type AuthStatus } from '@/stores/auth.svelte';

export type { WsStatus };

let status = $state<WsStatus>('idle');
let isConnected = $state(false);

/**
 * 登录态 → WS 动作（纯函数，供 effect 与单测复用）。
 * authenticated：connect；anonymous：disconnect；loading：等待。
 */
export function resolveWsAction(authStatus: AuthStatus): 'connect' | 'disconnect' | 'wait' {
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
    // auth.status 为响应式依赖：登录态变化时 effect 重跑，驱动连接/断开。
    const action = resolveWsAction(auth.status);
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
