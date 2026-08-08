// SectionHead 契约测试：标题渲染与操作按钮显隐/回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import SectionHead from '../SectionHead.svelte';

describe('SectionHead', () => {
	it('渲染标题', () => {
		render(SectionHead, { props: { title: '附近活动' } });
		expect(screen.getByText('附近活动')).toBeTruthy();
	});

	it('action 与 onAction 齐备时渲染操作按钮', () => {
		render(SectionHead, { props: { title: '附近的人', action: '查看全部', onAction: () => {} } });
		expect(screen.getByRole('button', { name: /查看全部/ })).toBeTruthy();
	});

	it('点击操作按钮触发 onAction', async () => {
		const onAction = vi.fn();
		render(SectionHead, { props: { title: '附近的人', action: '查看全部', onAction } });
		await fireEvent.click(screen.getByRole('button'));
		expect(onAction).toHaveBeenCalledTimes(1);
	});

	it('无 action 时不渲染按钮', () => {
		render(SectionHead, { props: { title: '附近活动' } });
		expect(screen.queryByRole('button')).toBeNull();
	});
});
