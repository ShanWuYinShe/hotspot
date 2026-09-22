// WS 客户端核心行为测试：Auth 时序、心跳 Ping/Pong、重连退避、close code 策略、
// 订阅本地权威 + SubscribeAck 回滚、auth-failed 事件映射。
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { WsClient } from './client';

/** 最小 Fake WebSocket（jsdom 无原生实现）。 */
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

  // --- 测试辅助 ---
  open() {
    this.readyState = FakeWebSocket.OPEN;
    this.onopen?.();
  }

  message(obj: unknown) {
    this.onmessage?.({ data: JSON.stringify(obj) });
  }

  serverClose(code: number) {
    this.readyState = FakeWebSocket.CLOSED;
    this.onclose?.({ code });
  }
}

const lastSent = (socket: FakeWebSocket): { type: string; payload: unknown } | null =>
  socket.sent.length > 0 ? (JSON.parse(socket.sent[socket.sent.length - 1]) as never) : null;

const sentOfType = (socket: FakeWebSocket, type: string) =>
  socket.sent.map((s) => JSON.parse(s) as { type: string; payload: any }).filter((m) => m.type === type);

const ESTABLISHED = (over: Record<string, unknown> = {}) => ({
  type: 'ConnectionEstablished',
  payload: {
    connection_id: 'c1',
    user_id: 'u1',
    server_time: '2025-01-01T00:00:00.000Z',
    resume_token: 'tok-1',
    server_seq: '42',
    config: { ping_interval: 30, ping_timeout: 10, max_subscriptions: { user: 500, group: 100 } },
    ...over
  }
});

beforeEach(() => {
  vi.useFakeTimers();
  FakeWebSocket.instances = [];
  vi.stubGlobal('WebSocket', FakeWebSocket);
  // 固定 jitter=0，退避断言可精确到 1s/2s/4s
  vi.spyOn(Math, 'random').mockReturnValue(0);
});

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe('连接与鉴权', () => {
  it('连接打开后立即发送 Auth{client_type:web}（不携带 token）', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    client.connect();
    const socket = FakeWebSocket.instances[0];
    expect(socket).toBeDefined();
    expect(client.currentStatus).toBe('connecting');

    socket.open();
    expect(lastSent(socket)).toEqual({ type: 'Auth', payload: { client_type: 'web' } });
  });

  it('ConnectionEstablished 后进入 connected；Ping 收到 Pong 则重置超时不重连', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    client.connect();
    const socket = FakeWebSocket.instances[0];
    socket.open();
    socket.message(ESTABLISHED());
    expect(client.currentStatus).toBe('connected');
    expect(client.isConnected).toBe(true);

    // ping_interval=30s 时发送 Ping
    vi.advanceTimersByTime(30_000);
    expect(lastSent(socket)).toEqual({ type: 'Ping', payload: {} });

    // Pong 重置超时：再等 9s 无重连
    socket.message({ type: 'Pong', payload: {} });
    vi.advanceTimersByTime(9_000);
    expect(FakeWebSocket.instances).toHaveLength(1);
  });

  it('Ping 后 ping_timeout 内未收到 Pong 视为断线并重连，携带 resume_token', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    client.connect();
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].message(ESTABLISHED());

    vi.advanceTimersByTime(30_000); // Ping 发出，启动 10s Pong 超时
    vi.advanceTimersByTime(10_000); // 超时 → 断线
    vi.advanceTimersByTime(1_000); // 退避 1s → 重连
    expect(FakeWebSocket.instances).toHaveLength(2);

    const socket2 = FakeWebSocket.instances[1];
    socket2.open();
    expect(lastSent(socket2)).toEqual({
      type: 'Auth',
      payload: { client_type: 'web', resume_token: 'tok-1' }
    });
  });
});

describe('重连策略', () => {
  it('指数退避 1s/2s/4s；close 4001（会话失效）不自动重连', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    client.connect();
    FakeWebSocket.instances[0].open();

    FakeWebSocket.instances[0].serverClose(1011);
    vi.advanceTimersByTime(1_000);
    expect(FakeWebSocket.instances).toHaveLength(2);

    FakeWebSocket.instances[1].serverClose(1011);
    vi.advanceTimersByTime(2_000);
    expect(FakeWebSocket.instances).toHaveLength(3);

    FakeWebSocket.instances[2].serverClose(1011);
    vi.advanceTimersByTime(4_000);
    expect(FakeWebSocket.instances).toHaveLength(4);

    // 4001：认证失败/会话被驱逐，契约要求不自动重连
    FakeWebSocket.instances[3].serverClose(4001);
    vi.advanceTimersByTime(60_000);
    expect(FakeWebSocket.instances).toHaveLength(4);
    expect(client.currentStatus).toBe('disconnected');
  });

  it('disconnect（登出）后不自动重连', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    client.connect();
    FakeWebSocket.instances[0].open();

    client.disconnect('logout');
    expect(client.currentStatus).toBe('disconnected');
    vi.advanceTimersByTime(60_000);
    expect(FakeWebSocket.instances).toHaveLength(1);
  });
});

describe('订阅管理', () => {
  it('本地订阅为权威：连接后 set 全量重放；SubscribeAck failed_ids 回滚本地', () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });

    // 未连接时订阅：仅更新本地，不发送
    client.setSubscriptions('subscribe', [{ type: 'user', ids: ['u1', 'u2'] }]);
    client.connect();
    const socket = FakeWebSocket.instances[0];
    socket.open();
    expect(sentOfType(socket, 'SetSubscriptions')).toHaveLength(0);

    // 连接建立后：set 全量重放本地订阅
    socket.message(ESTABLISHED());
    const replay = sentOfType(socket, 'SetSubscriptions')[0] as any;
    expect(replay.payload.operation).toBe('set');
    expect(replay.payload.items).toEqual([{ type: 'user', ids: ['u1', 'u2'] }]);

    // 增量 subscribe：发送并记录 client_request_id
    const reqId = client.setSubscriptions('subscribe', [{ type: 'user', ids: ['u3'] }]) as string;
    const subMsg = sentOfType(socket, 'SetSubscriptions').at(-1) as any;
    expect(subMsg.payload).toMatchObject({
      client_request_id: reqId,
      operation: 'subscribe',
      items: [{ type: 'user', ids: ['u3'] }]
    });

    // Ack：u2 被拒绝 → console.warn + 本地回滚
    socket.message({
      type: 'SubscribeAck',
      payload: {
        client_request_id: reqId,
        operation: 'subscribe',
        accepted: [{ type: 'user', ids: ['u1', 'u3'] }],
        code: 20100,
        message: '订阅权限不足',
        failed_ids: [{ type: 'user', subscription_id: 'u2', reason: 'denied' }]
      }
    });
    expect(warnSpy).toHaveBeenCalled();

    // 再次重连重放：本地已回滚 u2，只重放 u1/u3
    socket.serverClose(1011);
    vi.advanceTimersByTime(1_000);
    const socket2 = FakeWebSocket.instances[1];
    socket2.open();
    socket2.message(ESTABLISHED());
    const replay2 = sentOfType(socket2, 'SetSubscriptions').at(-1) as any;
    expect(replay2.payload.items).toEqual([{ type: 'user', ids: ['u1', 'u3'] }]);
  });
});

describe('事件映射', () => {
  it('SessionRevoked(device_evicted) 派发 auth-failed 中文提示', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    const onFailed = vi.fn();
    window.addEventListener('auth-failed', onFailed);

    client.connect();
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].message({
      type: 'SessionRevoked',
      payload: { reason: 'device_evicted', revoked_at: '2025-01-01T00:00:00.000Z' }
    });
    expect(onFailed).toHaveBeenCalledTimes(1);
    const detail = (onFailed.mock.calls[0][0] as CustomEvent).detail;
    expect(detail.message).toBe('账号在其他设备登录');

    window.removeEventListener('auth-failed', onFailed);
  });

  it('通知类事件派发 ws:notify（detail 含 type 与 payload.notify_id）', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    const onNotify = vi.fn();
    window.addEventListener('ws:notify', onNotify);

    client.connect();
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].message({
      type: 'NewFollower',
      payload: { notify_id: 'n-1', actor: { user_id: 'u2', nickname: '小美' }, followed_at: '2025-01-01T00:00:00.000Z' }
    });
    expect(onNotify).toHaveBeenCalledTimes(1);
    const detail = (onNotify.mock.calls[0][0] as CustomEvent).detail;
    expect(detail.type).toBe('NewFollower');
    expect(detail.payload.notify_id).toBe('n-1');

    window.removeEventListener('ws:notify', onNotify);
  });

  it('ContentMentioned（瞬间/评论 @，spec 事件名）派发 ws:notify', () => {
    // 回归锁：历史上前端误写 'MessageMentioned'（spec/后端均为 ContentMentioned），
    // 导致 Web 端永远收不到瞬间/评论 @ 通知。
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    const onNotify = vi.fn();
    window.addEventListener('ws:notify', onNotify);

    client.connect();
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].message({
      type: 'ContentMentioned',
      payload: {
        notify_id: 'n-2',
        target_type: 'moment',
        moment_id: 'm-1',
        comment_id: null,
        moment_summary: { moment_id: 'm-1', description_preview: '看看这个', cover_url: null },
        comment_preview: null,
        actor: { user_id: 'u2', nickname: '小美' },
        mentioned_at: '2025-01-01T00:00:00.000Z'
      }
    });
    expect(onNotify).toHaveBeenCalledTimes(1);
    const detail = (onNotify.mock.calls[0][0] as CustomEvent).detail;
    expect(detail.type).toBe('ContentMentioned');
    expect(detail.payload.notify_id).toBe('n-2');

    window.removeEventListener('ws:notify', onNotify);
  });

  it('4001 + 刷新成功 → 重连（spec close code 表：刷新后重连）', async () => {
    const refresh = vi.fn(async () => true);
    const client = new WsClient({ url: 'ws://test.local/ws/v1', refreshBeforeReconnect: refresh });
    client.connect();
    const first = FakeWebSocket.instances[0];
    first.open();
    first.message({
      type: 'ConnectionEstablished',
      payload: { connection_id: 'c1', user_id: 'u1', server_time: '2025-01-01T00:00:00.000Z', resume_token: 't', server_seq: '1', config: { ping_interval: 30, ping_timeout: 10, max_subscriptions: { user: 500, group: 100 } } }
    });

    first.onclose!({ code: 4001 });
    expect(refresh).toHaveBeenCalledTimes(1);
    await vi.waitFor(
      () => {
        expect(FakeWebSocket.instances.length).toBeGreaterThanOrEqual(2);
      },
      { timeout: 4000 }
    );
    // 刷新成功重连后，标记复位：再次 4001 仍会走刷新
    const second = FakeWebSocket.instances[1];
    second.open();
    second.onclose!({ code: 4001 });
    expect(refresh).toHaveBeenCalledTimes(2);
  });

  it('4001 + 无刷新通道 → 派发 auth-failed 且不重连', () => {
    const onFailed = vi.fn();
    window.addEventListener('auth-failed', onFailed);
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    client.connect();
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].onclose!({ code: 4001 });
    expect(onFailed).toHaveBeenCalledTimes(1);
    expect(FakeWebSocket.instances.length).toBe(1);
    window.removeEventListener('auth-failed', onFailed);
  });

  it('SessionRevoked 流程的 4001 不触发刷新（会话吊销刷新无意义）', () => {
    const refresh = vi.fn(async () => true);
    const client = new WsClient({ url: 'ws://test.local/ws/v1', refreshBeforeReconnect: refresh });
    client.connect();
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].message({
      type: 'SessionRevoked',
      payload: { reason: 'device_evicted', revoked_at: '2025-01-01T00:00:00.000Z' }
    });
    FakeWebSocket.instances[0].onclose!({ code: 4001 });
    expect(refresh).not.toHaveBeenCalled();
  });

  it('私聊消息事件派发 ws:private-message（detail 为 payload）', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    const onMsg = vi.fn();
    window.addEventListener('ws:private-message', onMsg);

    client.connect();
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].message({
      type: 'PrivateMessageNew',
      payload: {
        message_id: 'm-1',
        target_user_id: 'me',
        sender: { user_id: 'u2', nickname: '小美' },
        type: 'text',
        content: '你好',
        sent_at: '2025-01-01T00:00:00.000Z'
      }
    });
    expect(onMsg).toHaveBeenCalledTimes(1);
    const detail = (onMsg.mock.calls[0][0] as CustomEvent).detail;
    expect(detail.message_id).toBe('m-1');
    expect(detail.content).toBe('你好');

    window.removeEventListener('ws:private-message', onMsg);
  });

  it('PrivateTyping 服务端推送派发 ws:private-typing（detail 为 payload）', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    const onTyping = vi.fn();
    window.addEventListener('ws:private-typing', onTyping);

    client.connect();
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].message({
      type: 'PrivateTyping',
      payload: {
        target_user_id: 'me',
        actor: { user_id: 'u2', nickname: '小美' },
        is_typing: true
      }
    });
    expect(onTyping).toHaveBeenCalledTimes(1);
    const detail = (onTyping.mock.calls[0][0] as CustomEvent).detail;
    expect(detail.target_user_id).toBe('me');
    expect(detail.actor.user_id).toBe('u2');
    expect(detail.is_typing).toBe(true);

    window.removeEventListener('ws:private-typing', onTyping);
  });

  it('GroupMessageNew 派发 ws:group-message；GroupMessageEdited 派发 ws:group-message-update{type:edited}', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });
    const onNew = vi.fn();
    const onUpdate = vi.fn();
    window.addEventListener('ws:group-message', onNew);
    window.addEventListener('ws:group-message-update', onUpdate);

    client.connect();
    FakeWebSocket.instances[0].open();
    FakeWebSocket.instances[0].message({
      type: 'GroupMessageNew',
      payload: {
        message_id: 'gm-1',
        group_id: 'g1',
        sender: { user_id: 'u2', nickname: '小美' },
        type: 'text',
        content: '大家好',
        sent_at: '2025-01-01T00:00:00.000Z'
      }
    });
    expect(onNew).toHaveBeenCalledTimes(1);
    const newDetail = (onNew.mock.calls[0][0] as CustomEvent).detail;
    expect(newDetail.group_id).toBe('g1');
    expect(newDetail.content).toBe('大家好');

    FakeWebSocket.instances[0].message({
      type: 'GroupMessageEdited',
      payload: { message_id: 'gm-1', group_id: 'g1', content: '大家好呀', edited_at: '2025-01-01T00:00:01.000Z' }
    });
    expect(onUpdate).toHaveBeenCalledTimes(1);
    const updateDetail = (onUpdate.mock.calls[0][0] as CustomEvent).detail;
    expect(updateDetail.type).toBe('edited');
    expect(updateDetail.payload.group_id).toBe('g1');

    window.removeEventListener('ws:group-message', onNew);
    window.removeEventListener('ws:group-message-update', onUpdate);
  });

  it('sendPrivateTyping/sendGroupTyping 连接时发送契约 payload，未连接时静默返回 false', () => {
    const client = new WsClient({ url: 'ws://test.local/ws/v1' });

    // 未连接：不发送、返回 false
    expect(client.sendPrivateTyping('u2', true)).toBe(false);
    expect(client.sendGroupTyping('g1', true)).toBe(false);

    client.connect();
    const socket = FakeWebSocket.instances[0];
    socket.open();
    expect(client.sendPrivateTyping('u2', true)).toBe(true);
    expect(client.sendGroupTyping('g1', false)).toBe(true);

    expect(sentOfType(socket, 'SendPrivateTyping')).toEqual([
      { type: 'SendPrivateTyping', payload: { target_user_id: 'u2', is_typing: true } }
    ]);
    expect(sentOfType(socket, 'SendGroupTyping')).toEqual([
      { type: 'SendGroupTyping', payload: { group_id: 'g1', is_typing: false } }
    ]);
  });
});
