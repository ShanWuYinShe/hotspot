// NearbyUserRows 契约测试：用户行渲染与关注切换回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import NearbyUserRows from '../NearbyUserRows.svelte';

const users = [
	{ id: 'u1', name: '鹿鸣', avatar: '', tag: '摄影', distance: '0.3km', isOnline: true, isFollowing: false },
	{ id: 'u2', name: '清野', avatar: '', tag: '徒步', distance: '0.8km', isOnline: false, isFollowing: true }
];

const baseProps = {
	users,
	isFollowed: (id: string) => id === 'u1',
	onToggle: () => {}
};

describe('NearbyUserRows', () => {
	it('渲染用户名称/标签/距离', () => {
		render(NearbyUserRows, { props: { ...baseProps } });
		expect(screen.getByText('鹿鸣')).toBeTruthy();
		expect(screen.getByText('#摄影')).toBeTruthy();
		expect(screen.getByText('0.3km')).toBeTruthy();
	});

	it('点击关注按钮触发 onToggle 并传 id', async () => {
		const onToggle = vi.fn();
		render(NearbyUserRows, { props: { ...baseProps, onToggle } });
		const buttons = screen.getAllByRole('button', { name: '已关注' });
		await fireEvent.click(buttons[0]);
		expect(onToggle).toHaveBeenCalledWith('u1');
	});

	it('已关注用户显示「已关注」', () => {
		render(NearbyUserRows, { props: { ...baseProps } });
		expect(screen.getByRole('button', { name: '已关注' })).toBeTruthy();
	});

	it('请求进行中的用户按钮禁点', () => {
		render(NearbyUserRows, {
			props: { ...baseProps, isPending: (id: string) => id === 'u2' }
		});
		expect((screen.getByRole('button', { name: '关注' }) as HTMLButtonElement).disabled).toBe(true);
	});
});
