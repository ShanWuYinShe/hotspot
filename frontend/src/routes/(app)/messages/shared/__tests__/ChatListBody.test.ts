// ChatListBody 契约测试：列表渲染、会话操作入口（桌面 hover ⋯ / 移动长按、右键）。
import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ChatListBody from '../ChatListBody.svelte';
import type { ChatItem } from '@/lib/chat-list';

function chat(id: string, name: string, over: Partial<ChatItem> = {}): ChatItem {
	return {
		id,
		peerUserId: id,
		name,
		avatar: '',
		lastMessage: `${name} 的消息`,
		time: '10:24',
		timestamp: '2026-08-10T10:24:00Z',
		unread: 0,
		isPinned: false,
		isMuted: false,
		online: false,
		...over
	};
}

const CHATS = [chat('a', '阿禾'), chat('b', 'Mika')];

function renderBody(onSelect = vi.fn(), onMore = vi.fn()) {
	return render(ChatListBody, {
		props: {
			chats: CHATS,
			loadError: null,
			needsAuth: false,
			loading: false,
			onRetry: () => {},
			onSelect,
			onMore
		}
	});
}

afterEach(() => {
	vi.useRealTimers();
});

describe('ChatListBody', () => {
	it('role=list 的直接子节点仍是 listitem（长按包装不破坏列表语义）', () => {
		const { container } = renderBody();
		const list = container.querySelector('[role="list"]') as HTMLElement;
		expect(list).toBeTruthy();
		const rows = Array.from(list.children);
		expect(rows.length).toBe(2);
		for (const row of rows) expect(row.getAttribute('role')).toBe('listitem');
	});

	it('点击行触发 onSelect', async () => {
		const onSelect = vi.fn();
		renderBody(onSelect);
		await fireEvent.click(screen.getByRole('button', { name: '打开与 阿禾 的会话' }));
		expect(onSelect).toHaveBeenCalledTimes(1);
	});

	it('移动端长按 500ms 打开会话操作菜单，并吞掉随后的点击（不误进会话）', async () => {
		vi.useFakeTimers();
		const onSelect = vi.fn();
		const onMore = vi.fn();
		renderBody(onSelect, onMore);
		const row = screen.getByRole('button', { name: '打开与 Mika 的会话' });
		await fireEvent.pointerDown(row, { pointerType: 'touch', clientX: 20, clientY: 20 });
		await vi.advanceTimersByTimeAsync(520);
		expect(onMore).toHaveBeenCalledTimes(1);
		expect(onMore.mock.calls[0][0].id).toBe('b');
		// 长按抬手后的 click 被捕获阶段拦下，不会打开会话
		await fireEvent.click(row);
		expect(onSelect).not.toHaveBeenCalled();
	});

	it('短按抬手不触发长按，正常进入会话', async () => {
		vi.useFakeTimers();
		const onSelect = vi.fn();
		const onMore = vi.fn();
		renderBody(onSelect, onMore);
		const row = screen.getByRole('button', { name: '打开与 阿禾 的会话' });
		await fireEvent.pointerDown(row, { pointerType: 'touch', clientX: 20, clientY: 20 });
		await vi.advanceTimersByTimeAsync(120);
		await fireEvent.pointerUp(row);
		await fireEvent.click(row);
		expect(onMore).not.toHaveBeenCalled();
		expect(onSelect).toHaveBeenCalledTimes(1);
	});

	it('右键（contextmenu）打开对应行的会话操作菜单', async () => {
		const onMore = vi.fn();
		renderBody(vi.fn(), onMore);
		await fireEvent.contextMenu(screen.getByRole('button', { name: '打开与 阿禾 的会话' }));
		expect(onMore).toHaveBeenCalledTimes(1);
		expect(onMore.mock.calls[0][0].id).toBe('a');
	});
});
