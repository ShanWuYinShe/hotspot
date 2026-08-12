// GroupChatView 视图层契约测试：群信息/历史渲染、输入发送、返回按钮。
import { describe, it, expect, vi, beforeAll, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import GroupChatView from '../GroupChatView.svelte';

const mocks = vi.hoisted(() => ({
	authUser: { user_id: 'me', nickname: '我' },
	isOnline: vi.fn(() => false),
	getGroupDetail: vi.fn(),
	getGroupMessages: vi.fn(),
	sendGroupMessage: vi.fn(),
	markGroupRead: vi.fn(),
	wsClient: {
		isConnected: false,
		sendGroupTyping: vi.fn(),
		on: () => () => {},
		off: () => {},
		setSubscriptions: () => {},
		onStatusChange: vi.fn(() => () => {})
	}
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
	getGroupDetail: mocks.getGroupDetail,
	getGroupMessages: mocks.getGroupMessages,
	sendGroupMessage: mocks.sendGroupMessage,
	markGroupRead: mocks.markGroupRead
}));
vi.mock('@/lib/ws/client', () => ({
	wsClient: mocks.wsClient,
	WS_GROUP_MESSAGE_EVENT: 'test:group-new',
	WS_GROUP_MESSAGE_UPDATE_EVENT: 'test:group-update'
}));

const DETAIL = {
	code: 0,
	data: {
		group: { group_id: 'group_1', name: '测试群', avatar_url: '', members_count: 5 },
		online_members_count: 2,
		tags: ['摄影'],
		description: '测试群描述',
		location: { location_name: '', distance: 0 },
		my_state: { is_joined: true }
	}
};

const HISTORY = {
	code: 0,
	data: {
		items: [
			{
				message_id: 'gm1',
				type: 'text',
				content: '欢迎入群',
				sender: { user_id: 'user_lumos', nickname: '鹿鸣', avatar_url: '' },
				sent_at: '2026-08-09T09:00:00Z'
			}
		],
		next_cursor: null
	}
};

beforeAll(() => {
	// jsdom 未实现 Element.scrollTo（useChat 滚动到底部使用）
	Object.defineProperty(Element.prototype, 'scrollTo', { value: vi.fn(), configurable: true });
});

function renderGroup(onBack = vi.fn()) {
	return render(GroupChatView, {
		props: { groupId: 'group_1', showBackButton: true, onBack }
	});
}

describe('GroupChatView', () => {
	beforeEach(() => {
		mocks.getGroupDetail.mockReset().mockResolvedValue(DETAIL);
		mocks.getGroupMessages.mockReset().mockResolvedValue(HISTORY);
		mocks.sendGroupMessage.mockReset().mockResolvedValue({ code: 0, data: { message_id: 'gm2' } });
		mocks.markGroupRead.mockReset().mockResolvedValue({ code: 0 });
	});

	it('渲染群名与历史消息并标记已读', async () => {
		renderGroup();
		await waitFor(() => {
			expect(screen.getAllByText('测试群').length).toBeGreaterThan(0);
			expect(screen.getAllByText('欢迎入群').length).toBeGreaterThan(0);
			expect(mocks.markGroupRead).toHaveBeenCalled();
		});
	});

	it('输入并按回车发送群消息', async () => {
		renderGroup();
		const input = screen.getByLabelText('消息输入框');
		await fireEvent.input(input, { target: { value: '大家好' } });
		await fireEvent.keyDown(input, { key: 'Enter' });
		await waitFor(() => {
			expect(mocks.sendGroupMessage).toHaveBeenCalledWith({
				group_id: 'group_1',
				type: 'text',
				content: '大家好'
			});
			expect(screen.getAllByText('大家好').length).toBeGreaterThan(0);
		});
	});

	it('返回按钮触发 onBack', async () => {
		const onBack = vi.fn();
		renderGroup(onBack);
		await fireEvent.click(screen.getByRole('button', { name: '返回' }));
		expect(onBack).toHaveBeenCalledTimes(1);
	});
});
