// ChatMessageList 契约测试：R16 短会话降噪（「没有更多了」只在历史确实翻过页后出现）。
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import ChatMessageList from '../ChatMessageList.svelte';
import type { ChatController, ChatMessage } from '@/lib/chat.svelte';

const MESSAGES: ChatMessage[] = [
	{ id: 'm1', text: '你好', sent: false, time: '12:00', sentAt: '2026-08-10T12:00:00Z' }
];

/** 最小 ChatController 桩：只填 ChatMessageList 实际读取的字段。 */
function makeChat(over: Partial<Record<string, unknown>> = {}) {
	return {
		loading: false,
		loadError: false,
		messages: MESSAGES,
		dateGroups: [{ label: '今天', messages: MESSAGES }],
		loadingMore: false,
		loadingMoreFailed: false,
		hasMore: false,
		scrollEl: null as HTMLDivElement | null,
		lastSentId: '',
		showScrollDown: false,
		newMessageCount: 0,
		handleScroll: vi.fn(),
		reload: vi.fn(),
		loadMore: vi.fn(),
		scrollToBottom: vi.fn(),
		...over
	} as unknown as ChatController;
}

describe('ChatMessageList', () => {
	it('短会话（服务端一次返回全部，从未分页）不渲染「没有更多了」——顶部噪声消除', () => {
		render(ChatMessageList, { props: { chat: makeChat(), canRecall: () => false } });
		expect(screen.getByText('你好')).toBeTruthy();
		expect(screen.queryByText('没有更多了')).toBeNull();
	});

	// 渲染守卫：content 为空串的消息（群聊历史脏数据）不得生成「只剩时间戳的单个空气泡」；
	// 撤回消息仍要渲染「消息已撤回」，正常消息与日期分组不受影响。
	it('内容为空的消息不渲染气泡，撤回消息与日期分组保留', () => {
		const messages: ChatMessage[] = [
			{ id: 'e1', text: '   ', sent: false, time: '22:00', sentAt: '2026-08-10T22:00:00Z' },
			{ id: 'm1', text: '正常消息', sent: false, time: '22:01', sentAt: '2026-08-10T22:01:00Z' },
			{ id: 'r1', text: '', sent: false, time: '22:02', sentAt: '2026-08-10T22:02:00Z', recalled: true }
		];
		const chat = makeChat({ messages, dateGroups: [{ label: '今天', messages }] });
		const { container } = render(ChatMessageList, { props: { chat, canRecall: () => false } });
		expect(screen.getByText('今天')).toBeTruthy();
		expect(screen.getByText('正常消息')).toBeTruthy();
		expect(screen.getByText('消息已撤回')).toBeTruthy();
		expect(container.querySelectorAll('.max-w-chat-bubble')).toHaveLength(2);
	});

	// 全部消息内容为空时不应留下「空白消息区」：回退到统一空态。
	it('全部消息内容为空时回退空态，不渲染空气泡', () => {
		const messages: ChatMessage[] = [
			{ id: 'e1', text: '', sent: false, time: '22:00', sentAt: '2026-08-10T22:00:00Z' },
			{ id: 'e2', text: '  ', sent: true, time: '22:01', sentAt: '2026-08-10T22:01:00Z' }
		];
		const chat = makeChat({ messages, dateGroups: [{ label: '今天', messages }] });
		const { container } = render(ChatMessageList, { props: { chat, canRecall: () => false } });
		expect(screen.queryByText('今天')).toBeNull();
		expect(screen.getByText('还没有消息')).toBeTruthy();
		expect(container.querySelectorAll('.max-w-chat-bubble')).toHaveLength(0);
	});
});
