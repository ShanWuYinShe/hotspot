// useChat 状态机契约测试：历史加载、输入、发送成功/失败、发送错误回滚。
// useChat 内含 $effect，须在组件初始化上下文中实例化（见 __tests__/ChatHost.svelte）。
import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import ChatHost from './__tests__/ChatHost.svelte';
import { formatDateLabel, groupMessagesByDay, type ChatMessage, type ChatMessageUpdate, type UseChatOptions } from './chat.svelte';

let chatRef: ReturnType<typeof import('./chat.svelte').useChat> | null = null;

function setup(opts: Partial<UseChatOptions> = {}) {
	const loadHistory = vi.fn().mockResolvedValue({
		messages: [{ id: 'm1', text: '你好', sent: false, time: '12:00' }] as ChatMessage[],
		next_cursor: null
	});
	const markRead = vi.fn();
	const sendMessage = vi.fn();
	const sendTypingSignal = vi.fn();
	const wsNewMessage = vi.fn((): ChatMessage | null => null);
	const wsMessageUpdate = vi.fn((): ChatMessageUpdate | null => null);
	const options: UseChatOptions = {
		targetId: () => 'p1',
		loadHistory,
		markRead,
		sendMessage,
		sendTypingSignal,
		wsNewMessageEvent: 'test:new',
		wsMessageUpdateEvent: 'test:update',
		wsNewMessage,
		wsMessageUpdate,
		...opts
	};
	chatRef = null;
	render(ChatHost, { props: { options, expose: (c) => (chatRef = c) } });
	return {
		get chat() {
			return chatRef;
		},
		loadHistory,
		markRead,
		sendMessage
	};
}

describe('useChat', () => {
	it('加载历史并标记已读', async () => {
		const h = setup();
		await vi.waitFor(() => expect(h.chat).not.toBeNull());
		const { loadHistory, markRead } = h;
		await vi.waitFor(() => {
			expect(loadHistory).toHaveBeenCalledTimes(1);
		});
		await vi.waitFor(() => {
			expect(h.chat!.messages).toHaveLength(1);
			expect(h.chat!.messages[0].text).toBe('你好');
			expect(markRead).toHaveBeenCalled();
		});
	});

	it('输入与发送：追加临时消息并在成功后替换为真实 id', async () => {
		const h = setup();
		await vi.waitFor(() => expect(h.chat).not.toBeNull());
		const { sendMessage } = h;
		sendMessage.mockResolvedValue({ code: 0, message_id: 'm2' });
		h.chat!.handleInputChange({ target: { value: '  在吗  ' } } as unknown as Event);
		expect(h.chat!.inputVal).toBe('  在吗  ');
		await h.chat!.handleSend();
		await vi.waitFor(() => {
			expect(sendMessage).toHaveBeenCalledWith('在吗', undefined);
			expect(h.chat!.messages.some((m) => m.id === 'm2' && m.text === '在吗' && m.sent)).toBe(true);
			expect(h.chat!.sendError).toBe('');
		});
	});

	it('发送失败：移除临时消息并暴露错误', async () => {
		const h = setup();
		await vi.waitFor(() => expect(h.chat).not.toBeNull());
		const { sendMessage } = h;
		sendMessage.mockResolvedValue({ code: 1, error: '对方已屏蔽你' });
		h.chat!.handleInputChange({ target: { value: 'hello' } } as unknown as Event);
		await h.chat!.handleSend();
		await vi.waitFor(() => {
			expect(h.chat!.messages.filter((m) => m.text === 'hello')).toHaveLength(0);
			expect(h.chat!.sendError).toBe('对方已屏蔽你');
		});
	});

	it('历史按服务端倒序返回时翻转为正序（越新的消息在下方）', async () => {
		const loadHistory = vi.fn().mockResolvedValue({
			messages: [
				{ id: 'm3', text: '最新', sent: false, time: '12:03' },
				{ id: 'm2', text: '中间', sent: false, time: '12:02' },
				{ id: 'm1', text: '最早', sent: false, time: '12:01' }
			] as ChatMessage[],
			next_cursor: null
		});
		const h = setup({ loadHistory });
		await vi.waitFor(() => expect(loadHistory).toHaveBeenCalledTimes(1));
		await vi.waitFor(() => {
			expect(h.chat!.messages.map((m) => m.id)).toEqual(['m1', 'm2', 'm3']);
		});
	});

	it('加载更早历史时按正序插到顶部（保持旧上新下）', async () => {
		const loadHistory = vi
			.fn()
			.mockResolvedValueOnce({
				messages: [{ id: 'm2', text: '较新', sent: false, time: '12:02' }] as ChatMessage[],
				next_cursor: 'older'
			})
			.mockResolvedValueOnce({
				messages: [{ id: 'm1', text: '较旧', sent: false, time: '12:01' }] as ChatMessage[],
				next_cursor: null
			});
		const h = setup({ loadHistory });
		await vi.waitFor(() => expect(loadHistory).toHaveBeenCalledTimes(1));
		const el = document.createElement('div');
		el.scrollTop = 0;
		// jsdom 未实现 scrollTo；测试仅关心消息顺序，桩掉滚动定位
		el.scrollTo = (() => {}) as typeof el.scrollTo;
		h.chat!.scrollEl = el;
		h.chat!.handleScroll();
		await vi.waitFor(() => expect(loadHistory).toHaveBeenCalledTimes(2));
		await vi.waitFor(() => {
			expect(h.chat!.messages.map((m) => m.id)).toEqual(['m1', 'm2']);
		});
	});

	it('groupMessagesByDay 按本地日期分组并保持组内顺序', () => {
		const today = new Date();
		const yesterday = new Date(today.getFullYear(), today.getMonth(), today.getDate() - 1, 12, 0);
		const msg = (id: string, sentAt: string): ChatMessage => ({
			id,
			text: id,
			sent: false,
			time: '12:00',
			sentAt
		});
		const groups = groupMessagesByDay([
			msg('a', yesterday.toISOString()),
			msg('b', today.toISOString()),
			msg('c', today.toISOString())
		]);
		expect(groups.map((g) => g.messages.map((m) => m.id))).toEqual([['a'], ['b', 'c']]);
		expect(groups[0].label).toBe('昨天');
		expect(groups[1].label).toBe('今天');
	});

	it('formatDateLabel 跨年日期带年份', () => {
		const now = new Date();
		const past = new Date(now.getFullYear() - 1, now.getMonth(), now.getDate(), 10, 0);
		expect(formatDateLabel(past.toISOString())).toBe(`${past.getFullYear()}年${past.getMonth() + 1}月${past.getDate()}日`);
	});
});
