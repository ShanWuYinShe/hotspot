// ChatView 视图层契约测试：历史渲染、输入发送、返回按钮。
// 依赖（api / ws / stores）全部 mock，useChat 用真实实现，验证组件与状态机的接线。
import { describe, it, expect, vi, beforeAll, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import ChatView from '../ChatView.svelte';

const mocks = vi.hoisted(() => ({
	authUser: { user_id: 'me', nickname: '我' },
	isOnline: vi.fn(() => false),
	getPrivateMessages: vi.fn(),
	sendPrivateMessage: vi.fn(),
	markPrivateRead: vi.fn(),
	wsClient: { isConnected: false, sendPrivateTyping: vi.fn() }
}));

vi.mock('@/stores/auth.svelte', () => ({
	auth: {
		get user() {
			return mocks.authUser;
		}
	}
}));
vi.mock('@/stores/presence.svelte', () => ({
	presence: { isOnline: mocks.isOnline },
	watchPresence: () => () => {}
}));
vi.mock('@/api', () => ({
	getPrivateMessages: mocks.getPrivateMessages,
	sendPrivateMessage: mocks.sendPrivateMessage,
	markPrivateRead: mocks.markPrivateRead
}));
vi.mock('@/lib/ws/client', () => ({
	wsClient: mocks.wsClient,
	WS_PRIVATE_MESSAGE_EVENT: 'test:private-new',
	WS_PRIVATE_MESSAGE_UPDATE_EVENT: 'test:private-update',
	WS_PRIVATE_TYPING_EVENT: 'test:private-typing'
}));

const HISTORY = {
	code: 0,
	data: {
		items: [
			{
				message_id: 'm1',
				type: 'text',
				content: '你好，最近怎么样？',
				sender: { user_id: 'user_sophia', nickname: '阿禾', avatar_url: '' },
				sent_at: '2026-08-09T10:00:00Z'
			}
		],
		next_cursor: null
	}
};

beforeAll(() => {
	// jsdom 未实现 Element.scrollTo（useChat 滚动到底部使用）
	Object.defineProperty(Element.prototype, 'scrollTo', { value: vi.fn(), configurable: true });
});

function renderChat(onBack = vi.fn()) {
	return render(ChatView, {
		props: {
			peerUserId: 'user_sophia',
			contact: { name: '阿禾', avatar: '', online: false },
			showBackButton: true,
			onBack
		}
	});
}

describe('ChatView', () => {
	beforeEach(() => {
		mocks.getPrivateMessages.mockReset().mockResolvedValue(HISTORY);
		mocks.sendPrivateMessage.mockReset().mockResolvedValue({ code: 0, data: { message_id: 'm2' } });
		mocks.markPrivateRead.mockReset().mockResolvedValue({ code: 0 });
	});

	it('加载历史消息并标记已读', async () => {
		renderChat();
		await waitFor(() => {
			expect(screen.getAllByText('你好，最近怎么样？').length).toBeGreaterThan(0);
			expect(screen.getAllByText('阿禾').length).toBeGreaterThan(0);
			expect(mocks.markPrivateRead).toHaveBeenCalled();
		});
	});

	it('输入并按回车发送消息', async () => {
		renderChat();
		const input = screen.getByLabelText('消息输入框');
		await fireEvent.input(input, { target: { value: '在吗？' } });
		await fireEvent.keyDown(input, { key: 'Enter' });
		await waitFor(() => {
			expect(mocks.sendPrivateMessage).toHaveBeenCalledWith({
				target_user_id: 'user_sophia',
				type: 'text',
				content: '在吗？'
			});
			expect(screen.getAllByText('在吗？').length).toBeGreaterThan(0);
		});
	});

	it('返回按钮触发 onBack', async () => {
		const onBack = vi.fn();
		renderChat(onBack);
		await fireEvent.click(screen.getByRole('button', { name: '返回' }));
		expect(onBack).toHaveBeenCalledTimes(1);
	});
});
