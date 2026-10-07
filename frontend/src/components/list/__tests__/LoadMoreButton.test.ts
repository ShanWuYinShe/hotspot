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

	it('视觉形态守恒：胶囊圆角 + 次级文字色 + 无投影 + 标准 hover 文字', () => {
		// 迁移到 ui/button secondary 的前置约束（2026-10-07 实测结论）：
		// secondary 自带 shadow-raised-sm，而自定义 shadow token 不参与
		// tailwind-merge 冲突消解（class 名里同时出现 shadow-raised-sm 和 shadow-none），
		// 实测层叠顺序下 shadow-raised-sm 胜出 → 按钮会多一层投影。
		// 因此在原语支持「去投影」之前不得迁移；本用例用 token 边界（非子串）
		// 锁定当前形态，任何加投影/改圆角/改字色的迁移都会在这里失败。
		render(LoadMoreButton, { props: { onLoad: () => {} } });
		const cls = ` ${(screen.getByRole('button', { name: '加载更多' }) as HTMLElement).className} `;
		for (const token of ['rounded-full', 'px-5', 'py-2', 'text-text-secondary', 'hover:text-text-primary']) {
			expect(cls).toContain(` ${token} `);
		}
		for (const token of ['rounded-md', 'h-10', 'shadow-raised-sm', 'shadow-none', 'hover:border-border-strong']) {
			expect(cls).not.toContain(` ${token} `);
		}
	});
});
