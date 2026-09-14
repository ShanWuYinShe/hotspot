// LoadMoreButton 契约测试：文案、loading 态与点击回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import LoadMoreButton from '../LoadMoreButton.svelte';

describe('LoadMoreButton', () => {
	it('默认文案「加载更多」', () => {
		render(LoadMoreButton, { props: { onLoad: () => {} } });
		expect(screen.getByRole('button', { name: '加载更多' })).toBeTruthy();
	});

	it('点击触发 onLoad', async () => {
		const onLoad = vi.fn();
		render(LoadMoreButton, { props: { label: '加载更多 (3)', onLoad } });
		await fireEvent.click(screen.getByRole('button', { name: '加载更多 (3)' }));
		expect(onLoad).toHaveBeenCalledTimes(1);
	});

	it('loading 时显示「加载中…」且禁用', () => {
		render(LoadMoreButton, { props: { loading: true, onLoad: () => {} } });
		const btn = screen.getByRole('button', { name: '加载中…' });
		expect((btn as HTMLButtonElement).disabled).toBe(true);
	});
});
