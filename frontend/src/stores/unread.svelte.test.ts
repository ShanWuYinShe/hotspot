// unread store 测试：WS 事件增量、自己发送过滤、bootstrap 校准与清零。
// auth store 用 vi.mock 隔离（status 可控）；WS 沿用 client.test.ts 的 Fake 模式。
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// vi.mock 工厂会被提升到文件顶部：共享的可变状态必须经 vi.hoisted 创建
const mockAuth = vi.hoisted(() => ({
	status: 'authenticated' as 'authenticated' | 'anonymous' | 'loading',
	user: { user_id: 'u1', nickname: '我', avatar_url: '' },
}));

vi.mock('@/stores/auth.svelte', () => ({
	auth: mockAuth,
}));

import { wsClient, WS_NOTIFY_EVENT } from '@/lib/ws/client';
import {
	applyBootstrapUnread,
	initUnread,
	resetUnread,
	setChatUnread,
	setNotifyUnread,
	unread,
} from '@/stores/unread.svelte';

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

beforeEach(() => {
	FakeWebSocket.instances = [];
	vi.stubGlobal('WebSocket', FakeWebSocket);
	mockAuth.status = 'authenticated';
	resetUnread();
});

afterEach(() => {
	vi.unstubAllGlobals();
	vi.restoreAllMocks();
	wsClient.disconnect();
});

describe('unread', () => {
	it('ws:notify 事件使通知未读 +1；applyBootstrapUnread 可校准', () => {
		const off = initUnread();
		expect(unread.notify).toBe(0);

		window.dispatchEvent(new CustomEvent(WS_NOTIFY_EVENT, { detail: { type: 'NewFollower', payload: { notify_id: 'n1' } } }));
		expect(unread.notify).toBe(1);

		// bootstrap 校准覆盖为服务端真相（notify + 私聊/群聊未读）
		applyBootstrapUnread({ notify_total: 3, notify_by_type: {} as never, private_chat_unread_count: 2, group_chat_unread_count: 5, total_badge: 10 });
		expect(unread.notify).toBe(3);
		expect(unread.chat).toBe(7);
		expect(unread.total).toBe(10);

		off();
	});

	it('PrivateMessageNew 使聊天未读 +1；自己发送的回显不计入', () => {
		const off = initUnread();
		wsClient.connect();
		const socket = FakeWebSocket.instances[0];
		socket.open();
		socket.message({
			type: 'ConnectionEstablished',
			payload: { connection_id: 'c1', user_id: 'u1', server_time: '2025-01-01T00:00:00.000Z', resume_token: 't', server_seq: '1', config: { ping_interval: 30, ping_timeout: 10, max_subscriptions: { user: 500, group: 100 } } }
		});

		socket.message({ type: 'PrivateMessageNew', payload: { message_id: 'm1', target_user_id: 'u1', sender: { user_id: 'u2', nickname: '对方' } } });
		expect(unread.chat).toBe(1);

		// 自己在另一端发送的回显：不涨徽标
		socket.message({ type: 'PrivateMessageNew', payload: { message_id: 'm2', target_user_id: 'u9', sender: { user_id: 'u1', nickname: '我' } } });
		expect(unread.chat).toBe(1);

		socket.message({ type: 'GroupMessageNew', payload: { message_id: 'm3', group_id: 'g1', sender: { user_id: 'u3', nickname: '群友' } } });
		expect(unread.chat).toBe(2);

		off();
	});

	it('匿名态不计数；resetUnread / set 负值钳位', () => {
		const off = initUnread();
		mockAuth.status = 'anonymous';
		window.dispatchEvent(new CustomEvent(WS_NOTIFY_EVENT, { detail: { type: 'NewFollower', payload: {} } }));
		expect(unread.notify).toBe(0);

		mockAuth.status = 'authenticated';
		setNotifyUnread(-5);
		expect(unread.notify).toBe(0);
		setChatUnread(4);
		resetUnread();
		expect(unread.total).toBe(0);

		off();
	});
});
