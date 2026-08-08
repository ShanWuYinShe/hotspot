// ChatListItem 契约测试：会话信息渲染、选中态与点击回调。
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
	unread: 3,
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

	it('点击触发 onClick', async () => {
		const onClick = vi.fn();
		render(ChatListItem, { props: { chat, onClick } });
		await fireEvent.click(screen.getByRole('button', { name: '打开与 鹿鸣 的会话' }));
		expect(onClick).toHaveBeenCalledTimes(1);
	});

	it('isSelected 时行呈现选中态', () => {
		render(ChatListItem, { props: { chat, onClick: () => {}, isSelected: true } });
		expect(screen.getByRole('button').getAttribute('aria-pressed')).toBe('true');
	});
});
