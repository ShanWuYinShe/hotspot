// RailSection 契约测试：标题、操作按钮与容器 token 类。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import RailSection from '../RailSection.svelte';

describe('RailSection', () => {
	it('渲染标题', () => {
		render(RailSection, { props: { title: '附近的人', children: (() => null) as never } });
		expect(screen.getByText('附近的人')).toBeTruthy();
	});

	it('action 与 onAction 齐备时渲染操作按钮并触发回调', async () => {
		const onAction = vi.fn();
		render(RailSection, {
			props: { title: '附近的人', action: '查看全部', onAction, children: (() => null) as never }
		});
		await fireEvent.click(screen.getByRole('button', { name: /查看全部/ }));
		expect(onAction).toHaveBeenCalledTimes(1);
	});

	it('容器应用 token 类', () => {
		const { container } = render(RailSection, {
			props: { title: '附近的人', children: (() => null) as never }
		});
		const box = container.firstElementChild as HTMLElement;
		expect(box.className).toContain('surface-flat');
		expect(box.className).toContain('rounded-md');
	});

	it('island 变体应用无边框 Bento 微岛类（Round 2 去框：投影承载边界）', () => {
		const { container } = render(RailSection, {
			props: { title: '活跃邻居', variant: 'island', children: (() => null) as never }
		});
		const box = container.firstElementChild as HTMLElement;
		// 2026-10-29 一致性收口：rounded-2xl 属未注册档（实际也是 16px），归一到 rounded-lg(20)
		expect(box.className).toContain('rounded-lg');
		expect(box.className).toContain('bg-surface-primary/70');
		expect(box.className).toContain('shadow-xs');
		// 去明显边界：island 不再画外框线（内容单元边界由投影与底色差表达）
		expect(box.className).not.toContain('border');
	});
});
