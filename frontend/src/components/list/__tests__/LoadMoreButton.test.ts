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

	it('视觉形态守恒：胶囊圆角 + 次级文字色 + 无投影 + 标准描边加深 hover', () => {
		// 迁移到 ui/button secondary + flat 的形态契约：
		// - 无投影（flat 裁掉 shadow-raised-sm）
		// - 保留标准 secondary 描边加深 hover（hover:border-border-strong），
		//   与全站描边按钮的 hover 语言对齐
		render(LoadMoreButton, { props: { onLoad: () => {} } });
		const cls = ' ' + (screen.getByRole('button', { name: '加载更多' }) as HTMLElement).className + ' ';
		for (const token of ['rounded-full', 'px-5', 'text-text-secondary', 'hover:text-text-primary', 'hover:border-border-strong']) {
			expect(cls).toContain(' ' + token + ' ');
		}
		for (const token of ['rounded-md', 'h-10', 'shadow-raised-sm', 'shadow-none']) {
			expect(cls).not.toContain(' ' + token + ' ');
		}
	});
});
