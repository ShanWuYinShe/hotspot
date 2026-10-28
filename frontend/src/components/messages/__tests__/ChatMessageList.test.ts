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

});
