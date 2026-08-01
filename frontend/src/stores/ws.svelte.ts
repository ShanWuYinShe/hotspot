// WebSocket 连接生命周期 store（runes 模块单例）。
// 职责：把 auth store 的登录态映射为 WS 连接/断开（authenticated → connect，
// anonymous → disconnect）；对外暴露连接状态供 UI 调试使用。
//
// initWs() 必须在组件初始化期间同步调用一次（+layout.svelte script 顶层，
// 与 initAuth() 一致），内部 $effect 才会正确注册到组件生命周期。

import { wsClient, type WsStatus } from '@/lib/ws/client';
import { auth } from '@/stores/auth.svelte';

export type { WsStatus };

let status = $state<WsStatus>('idle');
let isConnected = $state(false);

/**
 * 初始化 WS 生命周期绑定。返回清理函数。
 * auth status: 'authenticated' → connect；'anonymous' → disconnect（手动断开，
 * 登出后不再自动重连）；'loading' 等待。
 */
export function initWs(): () => void {
  const offStatus = wsClient.onStatusChange((s) => {
    status = s;
    isConnected = s === 'connected';
  });

  $effect(() => {
    if (auth.status === 'authenticated') {
      void wsClient.connect();
    } else if (auth.status === 'anonymous') {
      wsClient.disconnect();
    }
    // 'loading'：等待鉴权结果
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
  },
};
