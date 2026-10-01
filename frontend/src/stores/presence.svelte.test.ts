// presence store 测试：UserOnline/UserOffline → onlineUsers；watchPresence 订阅/取消。
// 沿用 client.test.ts 的 Fake WebSocket 模式（jsdom 无原生实现）。
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { wsClient } from '@/lib/ws/client';
import { initPresence, isOnline, onlineUsers, seedOnline, watchPresence } from '@/stores/presence.svelte';

class FakeWebSocket {
  static OPEN = 1;
  static CONNECTING = 0;
  static CLOSED = 3;
  static instances: FakeWebSocket[] = [];

  url: string;
  readyState = FakeWebSocket.CONNECTING;
  sent: string[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((ev: { data: string }) => void) | null = null;
  onclose: ((ev: { code: number }) => void) | null = null;
  onerror: (() => void) | null = null;

  constructor(url: string) {
    this.url = url;
    FakeWebSocket.instances.push(this);
  }

  send(data: string) {
    this.sent.push(data);
  }

  close(code?: number) {
    this.readyState = FakeWebSocket.CLOSED;
    if (this.onclose) this.onclose({ code: code ?? 1000 });
  }

  open() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.();
  }

  message(obj: unknown) {
    this.onmessage?.({ data: JSON.stringify(obj) });
  }
}

const sentOfType = (socket: FakeWebSocket, type: string) =>
  socket.sent.map((s) => JSON.parse(s) as { type: string; payload: any }).filter((m) => m.type === type);

const ESTABLISHED = {
  type: 'ConnectionEstablished',
  payload: {
    connection_id: 'c1',
    user_id: 'u1',
    server_time: '2025-01-01T00:00:00.000Z',
    resume_token: 'tok-1',
    server_seq: '42',
    config: { ping_interval: 30, ping_timeout: 10, max_subscriptions: { user: 500, group: 100 } }
  }
};

beforeEach(() => {
  FakeWebSocket.instances = [];
  vi.stubGlobal('WebSocket', FakeWebSocket);
  onlineUsers.clear();
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  wsClient.disconnect();
});

describe('presence', () => {
  it('initPresence 后 UserOnline/UserOffline 维护 onlineUsers', () => {
    const off = initPresence();
    wsClient.connect();
    const socket = FakeWebSocket.instances[0];
    socket.open();

    socket.message({ type: 'UserOnline', payload: { user_id: 'u2', online_at: '2025-01-01T00:00:00.000Z' } });
    expect(isOnline('u2')).toBe(true);
    expect(onlineUsers.has('u2')).toBe(true);

    socket.message({ type: 'UserOffline', payload: { user_id: 'u2', offline_at: '2025-01-01T00:01:00.000Z' } });
    expect(isOnline('u2')).toBe(false);

    off();
  });

  it('watchPresence 去重订阅；引用计数归零才取消订阅', () => {
    wsClient.connect();
    const socket = FakeWebSocket.instances[0];
    socket.open();
    socket.message(ESTABLISHED);

    // 未连接/已连接均可：去重后 subscribe
    const unwatch = watchPresence(['u1', 'u2', 'u1']);
    const subMsg = sentOfType(socket, 'SetSubscriptions').at(-1) as any;
    expect(subMsg.payload).toMatchObject({
      operation: 'subscribe',
      items: [{ type: 'user', ids: ['u1', 'u2'] }]
    });

    // 引用计数：第二个 watcher 重叠 id 不重复订阅
    const unwatch2 = watchPresence(['u2']);
    expect(sentOfType(socket, 'SetSubscriptions')).toHaveLength(1);

    // 释放第一个 watcher：u1 归零 → unsubscribe u1；u2 仍有引用 → 不取消
    unwatch();
    const unsub1 = sentOfType(socket, 'SetSubscriptions').at(-1) as any;
    expect(unsub1.payload).toMatchObject({
      operation: 'unsubscribe',
      items: [{ type: 'user', ids: ['u1'] }]
    });

    // 释放第二个 watcher：u2 归零 → unsubscribe u2
    unwatch2();
    const unsub2 = sentOfType(socket, 'SetSubscriptions').at(-1) as any;
    expect(unsub2.payload).toMatchObject({
      operation: 'unsubscribe',
      items: [{ type: 'user', ids: ['u2'] }]
    });
  });

  it('watchPresence 未连接时仅更新本地订阅，连接后 set 全量重放', () => {
    const unwatch = watchPresence(['u9']);
    wsClient.connect();
    const socket = FakeWebSocket.instances[0];
    socket.open();
    // 未连接期间不发送
    expect(sentOfType(socket, 'SetSubscriptions')).toHaveLength(0);

    socket.message(ESTABLISHED);
    const replay = sentOfType(socket, 'SetSubscriptions')[0] as any;
    expect(replay.payload.operation).toBe('set');
    expect(replay.payload.items).toEqual([{ type: 'user', ids: ['u9'] }]);

    unwatch();
  });

  it('seedOnline 以 HTTP 快照补初始在线态；UserOffline 仍可修正', () => {
    const off = initPresence();
    wsClient.connect();
    const socket = FakeWebSocket.instances[0];
    socket.open();

    // 订阅前已在线的对端收不到 UserOnline 事件（spec 无订阅时初始快照），
    // 由调用方以 HTTP 快照（presence_status）补种子。
    seedOnline(['u7', 'u7', '']);
    expect(isOnline('u7')).toBe(true);

    // 种子只是初始态：后续 UserOffline 事件照常修正。
    socket.message({ type: 'UserOffline', payload: { user_id: 'u7', offline_at: '2025-01-01T00:02:00.000Z' } });
    expect(isOnline('u7')).toBe(false);

    off();
  });
});
