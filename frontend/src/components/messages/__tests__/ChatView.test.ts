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
	wsClient: {
		isConnected: false,
		sendPrivateTyping: vi.fn(),
		onStatusChange: vi.fn<(cb: (s: string) => void) => () => void>(() => () => {})
	},
	statusCb: null as ((s: string) => void) | null
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

	it('发送被拒时直显后端方向文案（拉黑方视角）', async () => {
		mocks.sendPrivateMessage.mockResolvedValue({
			code: 10200,
			message: '你已拉黑对方，无法发送',
			data: { forbidden_reason: 'blocked_by_user' }
		});
		renderChat();
		const input = screen.getByLabelText('消息输入框');
		await fireEvent.input(input, { target: { value: '还能说话吗' } });
		await fireEvent.keyDown(input, { key: 'Enter' });
		await waitFor(() => {
			expect(screen.getByText('你已拉黑对方，无法发送')).toBeTruthy();
		});
		// 乐观消息已回滚：列表中没有该文本的消息气泡
		expect(screen.queryByText('还能说话吗')).toBeNull();
	});

	it('返回按钮触发 onBack', async () => {
		const onBack = vi.fn();
		renderChat(onBack);
		await fireEvent.click(screen.getByRole('button', { name: '返回' }));
		expect(onBack).toHaveBeenCalledTimes(1);
	});

	it('10100 未登录时显示登录引导而非错误重试', async () => {
		mocks.getPrivateMessages.mockResolvedValue({ code: 10100, message: 'Access Token 无效或过期' });
		renderChat();
		await waitFor(() => {
			expect(screen.getByText('去登录')).toBeTruthy();
		});
		// 死胡同重试与输入栏一并隐藏
		expect(screen.queryByText('重试')).toBeNull();
		expect(screen.queryByLabelText('消息输入框')).toBeNull();
	});

	it('断线超阈值重连后自动补拉历史并去重合并（spec L163-165）', async () => {
		mocks.wsClient.onStatusChange.mockImplementation((cb: (s: string) => void) => {
			mocks.statusCb = cb;
			return () => {};
		});
		renderChat();
		await waitFor(() => expect(screen.getAllByText('你好，最近怎么样？').length).toBeGreaterThan(0));

		// 断线期间服务端新增消息 m9（sent_at 比本地历史 m1 更新）
		mocks.getPrivateMessages.mockResolvedValue({
			code: 0,
			data: {
				items: [
					{
						message_id: 'm9',
						type: 'text',
						content: '断线期间的新消息',
						sender: { user_id: 'user_sophia', nickname: '阿禾', avatar_url: '' },
						sent_at: '2026-08-09T11:00:00Z'
					},
					...HISTORY.data.items
				],
				next_cursor: null
			}
		});

		// 断开 → 11 秒后重连（超过 10s 阈值）
		const nowSpy = vi.spyOn(Date, 'now').mockReturnValue(1_000_000);
		mocks.statusCb?.('disconnected');
		nowSpy.mockReturnValue(1_011_000);
		mocks.statusCb?.('connected');
		nowSpy.mockRestore();

		await waitFor(() => {
			// 补拉调用（第二次 get_history）且新消息与旧消息并存（去重合并）
			expect(mocks.getPrivateMessages).toHaveBeenCalledTimes(2);
			expect(screen.getAllByText('断线期间的新消息').length).toBeGreaterThan(0);
			expect(screen.getAllByText('你好，最近怎么样？').length).toBe(1);
		});
		// 新消息按时间正序排在旧消息之后
		const newer = screen.getByText('断线期间的新消息');
		const older = screen.getByText('你好，最近怎么样？');
		expect(
			older.compareDocumentPosition(newer) & Node.DOCUMENT_POSITION_FOLLOWING
		).toBeTruthy();
	});

	it('回放旧消息按 sent_at 插入正序位置（resume 续传/实时交错不乱序）', async () => {
		renderChat();
		await waitFor(() => expect(screen.getAllByText('你好，最近怎么样？').length).toBeGreaterThan(0));

		// 模拟 resume 回放：比本地历史更早的消息晚于实时消息到达
		window.dispatchEvent(
			new CustomEvent('test:private-new', {
				detail: {
					message_id: 'mOld',
					target_user_id: 'user_sophia',
					sender: { user_id: 'user_sophia', nickname: '阿禾' },
					type: 'text',
					content: '回放的更早消息',
					sent_at: '2026-08-09T09:00:00Z'
				}
			})
		);
		await waitFor(() => expect(screen.getAllByText('回放的更早消息').length).toBeGreaterThan(0));
		const older = screen.getByText('回放的更早消息');
		const newer = screen.getByText('你好，最近怎么样？');
		expect(older.compareDocumentPosition(newer) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});

	it('接收其它设备同步发送的自己消息', async () => {
		renderChat();
		await waitFor(() => expect(screen.getAllByText('你好，最近怎么样？').length).toBeGreaterThan(0));

		// 模拟用户在移动端向 user_sophia 发送一条私聊
		window.dispatchEvent(
			new CustomEvent('test:private-new', {
				detail: {
					message_id: 'mOtherDevice',
					target_user_id: 'user_sophia',
					sender: { user_id: 'me', nickname: '我' },
					type: 'text',
					content: '来自手机端的消息',
					sent_at: '2026-08-09T10:05:00Z'
				}
			})
		);
		await waitFor(() => expect(screen.getAllByText('来自手机端的消息').length).toBeGreaterThan(0));
	});
});
