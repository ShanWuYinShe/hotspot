// ChatEmptyPanel 契约测试：引导文案、最近会话建议、动作回调、窄面板不开 spotlight。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ChatEmptyPanel from '../ChatEmptyPanel.svelte';
import type { ChatItem } from '@/lib/chat-list';

function chat(id: string, name: string, over: Partial<ChatItem> = {}): ChatItem {
	return {
		id,
		peerUserId: id,
		name,
		avatar: '',
		lastMessage: `${name} 的最后一条消息`,
		time: '10:24',
		timestamp: '2026-08-10T10:24:00Z',
		unread: 0,
		isPinned: false,
		isMuted: false,
		online: false,
		...over
	};
}

describe('ChatEmptyPanel', () => {
	it('渲染引导标题与最近会话建议（最多 3 条）', () => {
		render(ChatEmptyPanel, {
			props: {
				recent: [chat('a', '阿禾'), chat('b', 'Mika'), chat('c', '野原'), chat('d', '第四位')],
				onSelect: () => {},
				onNewChat: () => {}
			}
		});
		expect(screen.getByText('选择一个会话')).toBeTruthy();
		expect(screen.getByText('阿禾')).toBeTruthy();
		expect(screen.getByText('野原')).toBeTruthy();
		// 最多 3 条建议：第 4 条不渲染
		expect(screen.queryByText('第四位')).toBeNull();
	});

	it('点击最近会话建议触发 onSelect', async () => {
		const onSelect = vi.fn();
		render(ChatEmptyPanel, {
			props: { recent: [chat('a', '阿禾')], onSelect, onNewChat: () => {} }
		});
		await fireEvent.click(screen.getByRole('button', { name: '打开与 阿禾 的会话' }));
		expect(onSelect).toHaveBeenCalledTimes(1);
		expect(onSelect.mock.calls[0][0].id).toBe('a');
	});

	it('点击「发起新会话」触发 onNewChat', async () => {
		const onNewChat = vi.fn();
		render(ChatEmptyPanel, {
			props: { recent: [], onSelect: () => {}, onNewChat }
		});
		await fireEvent.click(screen.getByRole('button', { name: '发起新会话' }));
		expect(onNewChat).toHaveBeenCalledTimes(1);
	});

	it('窄面板内不开 spotlight 装饰层（空态规范：小面板不开）', () => {
		const { container } = render(ChatEmptyPanel, {
			props: { recent: [], onSelect: () => {}, onNewChat: () => {} }
		});
		expect(container.querySelector('[data-empty-spotlight]')).toBeNull();
	});
});
