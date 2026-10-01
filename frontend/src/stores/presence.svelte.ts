// 在线状态 store（runes 模块单例）。
// 职责：监听 UserOnline/UserOffline 事件维护 onlineUsers 集合；通过
// watchPresence() 以订阅计数（引用计数）方式管理 user 订阅，供列表/详情页
// 把 HTTP 快照的在线状态升级为 WS 实时推送。
//
// initPresence() 必须在组件初始化期间同步调用一次（+layout.svelte script 顶层，
// 与 initWs() 一致），内部才完成事件绑定。

import { wsClient } from '@/lib/ws/client';

/** 当前在线用户集合（$state 深层响应，add/delete 即触发 UI 更新）。 */
export const onlineUsers = $state(new Set<string>());

/** 内部订阅引用计数：id → 活跃 watch 数（归零才 unsubscribe，避免多组件重叠订阅互相误伤）。 */
const watching = new Map<string, number>();

/**
 * 初始化在线状态监听。返回清理函数（解绑事件，不清空已收集状态）。
 */
export function initPresence(): () => void {
  const offOnline = wsClient.on('UserOnline', (p) => {
    const id = (p as { user_id?: string })?.user_id;
    if (id) onlineUsers.add(id);
  });
  const offOffline = wsClient.on('UserOffline', (p) => {
    const id = (p as { user_id?: string })?.user_id;
    if (id) onlineUsers.delete(id);
  });
  return () => {
    offOnline();
    offOffline();
  };
}

/**
 * 清空在线状态集合。登出/会话失效/切换账号时调用：
 * 在线集合是上一个账号会话期间收集的，跨账号残留会让新账号首帧看到错误在线点。
 */
export function resetPresence(): void {
  onlineUsers.clear();
}

/**
 * 用 HTTP 快照（presence_status）种子化在线集合：WS 订阅只覆盖「订阅之后的状态
 * 变化」，订阅前已在线的对端不会再收到 UserOnline 事件（spec 无订阅时初始快照），
 * 需调用方以列表/资料/会话接口携带的 presence 快照补初始态。只补「在线」；
 * 离线修正仍由 UserOffline 事件驱动。
 */
export function seedOnline(ids: string[]): void {
  for (const id of new Set(ids.filter(Boolean))) onlineUsers.add(id);
}

/**
 * 订阅一批用户的在线状态（去重 + 引用计数）。
 * 未连接时 setSubscriptions 仅更新本地订阅状态，重连后自动重放，安全。
 * @returns 清理函数：释放引用计数并取消订阅（调用多次幂等安全）。
 */
export function watchPresence(ids: string[]): () => void {
  const unique = [...new Set(ids.filter(Boolean))];
  const added: string[] = [];
  for (const id of unique) {
    const count = (watching.get(id) ?? 0) + 1;
    watching.set(id, count);
    if (count === 1) added.push(id);
  }
  if (added.length > 0) {
    wsClient.setSubscriptions('subscribe', [{ type: 'user', ids: added }]);
  }

  return () => {
    const released: string[] = [];
    for (const id of unique) {
      const count = (watching.get(id) ?? 0) - 1;
      if (count <= 0) {
        watching.delete(id);
        released.push(id);
      } else {
        watching.set(id, count);
      }
    }
    if (released.length > 0) {
      wsClient.setSubscriptions('unsubscribe', [{ type: 'user', ids: released }]);
    }
  };
}

/** 某用户当前是否在线（组件内读取即订阅，可安全用于 $derived）。 */
export function isOnline(userId: string): boolean {
  return onlineUsers.has(userId);
}

/** 响应式在线状态入口（组件内读取即订阅）。 */
export const presence = {
  get isOnline() {
    return isOnline;
  },
  get onlineUsers() {
    return onlineUsers;
  },
};
