// NotifyRow 契约测试：通知渲染与已读回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import NotifyRow from '../NotifyRow.svelte';

const notif = {
	id: 'n1',
	type: 'reaction',
	read: false,
	time: '2分钟前',
	user: { name: '鹿鸣', avatar: '' },
	text: '赞了你的动态'
} as never;

describe('NotifyRow', () => {
	it('渲染用户/文本/时间', () => {
		render(NotifyRow, {
			props: { notif, onRead: () => {}, onRemove: () => {} }
		});
		expect(screen.getByText('鹿鸣')).toBeTruthy();
		expect(screen.getByText('赞了你的动态')).toBeTruthy();
		expect(screen.getByText('2分钟前')).toBeTruthy();
	});

	it('点击行触发 onRead 并传 id', async () => {
		const onRead = vi.fn();
		render(NotifyRow, {
			props: { notif, onRead, onRemove: () => {} }
		});
		await fireEvent.click(screen.getByRole('button'));
		expect(onRead).toHaveBeenCalledWith('n1');
	});

	// R13 未读标识：左侧 3px 未读条（状态条 1-3px 为登记例外），
	// 未读语义保留在 aria-label + sr-only 文本里，不再用常驻脉冲光点。
	it('未读行渲染左侧未读条与未读语义', () => {
		const { container } = render(NotifyRow, {
			props: { notif, onRead: () => {}, onRemove: () => {} }
		});
		expect(container.querySelector('.bg-accent-default')).toBeTruthy();
		expect(screen.getByText('未读')).toBeTruthy();
		const row = screen.getByRole('button');
		expect(row.getAttribute('aria-label')).toContain('未读');
		// 旧的常驻脉冲光点已移除
		expect(container.querySelector('.animate-pulse')).toBeNull();
	});

	it('已读行不渲染未读条与未读语义', () => {
		const readNotif = {
			id: 'n1',
			type: 'reaction',
			read: true,
			time: '2分钟前',
			user: { name: '鹿鸣', avatar: '' },
			text: '赞了你的动态'
		} as never;
		const { container } = render(NotifyRow, {
			props: { notif: readNotif, onRead: () => {}, onRemove: () => {} }
		});
		expect(container.querySelector('.w-0\\.75')).toBeNull();
		expect(screen.queryByText('未读')).toBeNull();
		expect(screen.getByRole('button').getAttribute('aria-label')).not.toContain('未读');
	});
});
