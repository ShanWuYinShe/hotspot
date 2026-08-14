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
});
