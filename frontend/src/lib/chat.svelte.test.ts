// useChat 状态机契约测试：历史加载、输入、发送成功/失败、发送错误回滚。
// useChat 内含 $effect，须在组件初始化上下文中实例化（见 __tests__/ChatHost.svelte）。
import { describe, it, expect, vi } from 'vitest';
import { render } from '@testing-library/svelte';
import ChatHost from './__tests__/ChatHost.svelte';
import type { ChatMessage, ChatMessageUpdate, UseChatOptions } from './chat.svelte';

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
			expect(sendMessage).toHaveBeenCalledWith('在吗');
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
});
