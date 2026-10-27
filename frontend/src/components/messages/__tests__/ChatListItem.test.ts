// ChatListItem 契约测试：会话信息渲染、选中态、未读层级与操作入口（R13 P2-24 后）。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ChatListItem from '../ChatListItem.svelte';

const chat = {
	id: 'c1',
	peerUserId: 'u1',
	name: '鹿鸣',
	avatar: 'https://example.com/a.jpg',
	lastMessage: '周末见',
	time: '10:24',
	timestamp: '2026-08-10T10:24:00Z',
	unread: 3,
	isPinned: false,
	isMuted: false,
	online: true
};

describe('ChatListItem', () => {
	it('渲染会话名称/最后消息/时间/未读数', () => {
		render(ChatListItem, { props: { chat, onClick: () => {} } });
		expect(screen.getByText('鹿鸣')).toBeTruthy();
		expect(screen.getByText('周末见')).toBeTruthy();
		expect(screen.getByText('10:24')).toBeTruthy();
		expect(screen.getByText('3')).toBeTruthy();
	});

	it('点击行触发 onClick', async () => {
		const onClick = vi.fn();
		render(ChatListItem, { props: { chat, onClick } });
		await fireEvent.click(screen.getByRole('button', { name: '打开与 鹿鸣 的会话' }));
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('isSelected 时行呈现选中态', () => {
		render(ChatListItem, { props: { chat, onClick: () => {}, isSelected: true } });
		expect(screen.getByRole('button', { name: '打开与 鹿鸣 的会话' }).getAttribute('aria-pressed')).toBe('true');
	});

	// R13 会话行层级：未读 = 数字徽标 + 行底淡 tint（同一实现），昵称加一档字重；
	// 已读行不得带 tint/加粗，避免整列都「很响」。
	it('未读会话行带淡 tint 与 semibold 昵称', () => {
		const { container } = render(ChatListItem, { props: { chat, onClick: () => {} } });
		const row = container.querySelector('div.group') as HTMLElement;
		expect(row.className).toContain('bg-accent-soft');
		expect(screen.getByText('鹿鸣').className).toContain('font-semibold');
	});

	it('已读会话行不带 tint，昵称回落到 medium', () => {
		const { container } = render(ChatListItem, {
			props: { chat: { ...chat, unread: 0 }, onClick: () => {} }
		});
		const row = container.querySelector('div.group') as HTMLElement;
		expect(row.className).not.toContain('bg-accent-soft');
		expect(screen.getByText('鹿鸣').className).toContain('font-medium');
		expect(screen.queryByText('3')).toBeNull();
	});

	it('会话行未读徽标遵循紧凑型徽标契约（h-5 px-2 text-xxs）', () => {
		render(ChatListItem, { props: { chat, onClick: () => {} } });
		const badge = screen.getByText('3');
		expect(badge.className).toContain('h-5');
		expect(badge.className).toContain('px-2');
		expect(badge.className).toContain('text-xxs');
	});

	// P2-24：会话操作收成一个入口——桌面 hover/focus 显现（desktop-reveal），
	// 移动端不常驻按钮（由列表级长按/右键打开，见 ChatListBody 测试）。
	it('传 onMore 时渲染单一「更多操作」入口（桌面 hover 显现，非移动常驻）', async () => {
		const onClick = vi.fn();
		const onMore = vi.fn();
		render(ChatListItem, { props: { chat, onClick, onMore } });
		const btn = screen.getByRole('button', { name: `${chat.name} 的更多操作` });
		expect(btn.className).toContain('desktop-reveal');
		expect(btn.className).toContain('md:flex');
		expect(btn.className).not.toContain('md:hidden');
		await fireEvent.click(btn);
		expect(onMore).toHaveBeenCalledTimes(1);
		expect(onClick).not.toHaveBeenCalled();
	});

	it('不传 onMore 时不渲染「更多操作」按钮', () => {
		render(ChatListItem, { props: { chat, onClick: () => {} } });
		expect(screen.queryByRole('button', { name: `${chat.name} 的更多操作` })).toBeNull();
	});

	// 置顶/免打扰改为昵称旁静态标识，不再产生行内按钮（右缘不再堆控件）
	it('置顶/免打扰仅作静态标识，不渲染行内切换按钮', () => {
		render(ChatListItem, {
			props: { chat: { ...chat, isPinned: true, isMuted: true }, onClick: () => {} }
		});
		expect(screen.queryByRole('button', { name: '取消置顶' })).toBeNull();
		expect(screen.queryByRole('button', { name: '开启免打扰' })).toBeNull();
		expect(screen.queryByRole('button', { name: '关闭免打扰' })).toBeNull();
	});
});
