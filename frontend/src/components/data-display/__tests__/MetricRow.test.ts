// MetricRow 契约测试：标签/值渲染、tone 类、列数与点击交互。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import MetricRow from '../MetricRow.svelte';

describe('MetricRow', () => {
	it('渲染全部 label 与数值 value', () => {
		render(MetricRow, {
			props: {
				items: [
					{ label: '附近', value: 5 },
					{ label: '在线', value: '12' }
				]
			}
		});
		expect(screen.getByText('附近')).toBeTruthy();
		expect(screen.getByText('在线')).toBeTruthy();
		expect(screen.getByText('5')).toBeTruthy();
		expect(screen.getByText('12')).toBeTruthy();
	});

	it('tone=accent 应用主题色类', () => {
		const { container } = render(MetricRow, {
			props: { items: [{ label: '活跃', value: 3, tone: 'accent' }] }
		});
		expect(container.querySelector('.text-accent-default')).not.toBeNull();
	});

	it('grid 列数等于 items 数量', () => {
		const { container } = render(MetricRow, {
			props: {
				items: [
					{ label: 'A', value: 1 },
					{ label: 'B', value: 2 },
					{ label: 'C', value: 3 }
				]
			}
		});
		const grid = container.querySelector('[role="group"]') as HTMLElement;
		expect(grid.style.gridTemplateColumns).toContain('repeat(3');
	});

	it('item 带有 onClick 时赋予可点击交互并响应点击', async () => {
		const handleClick = vi.fn();
		render(MetricRow, {
			props: {
				items: [
					{ label: '动态', value: 10, onClick: handleClick },
					{ label: '粉丝', value: 20 }
				]
			}
		});
		const clickableItem = screen.getByRole('button');
		expect(clickableItem).toBeTruthy();
		await fireEvent.click(clickableItem);
		expect(handleClick).toHaveBeenCalledTimes(1);
	});

	it('variant=capsule 渲染微岛胶囊形态', () => {
		const { container } = render(MetricRow, {
			props: {
				variant: 'capsule',
				items: [
					{ label: '瞬间', value: 4 },
					{ label: '关注者', value: 5 }
				]
			}
		});
		const grid = container.querySelector('[role="group"]') as HTMLElement;
		expect(grid.className).toContain('gap-2.5');
		expect(container.querySelector('.rounded-xl')).not.toBeNull();
	});

	it('variant=strip 使用发丝分隔与 24px 数字档（R13 资料页/枢纽 hero）', () => {
		const { container } = render(MetricRow, {
			props: {
				variant: 'strip',
				items: [
					{ label: '人附近', value: 6 },
					{ label: '瞬间', value: 12 },
					{ label: '在线', value: 3, tone: 'positive' }
				]
			}
		});
		const group = container.querySelector('[role="group"]') as HTMLElement;
		expect(group.className).toContain('divide-x');
		expect(group.className).not.toContain('grid');
		// 数字档：24px（text-2xl）+ tabular-nums；标签次级色
		const value = screen.getByText('6');
		expect(value.className).toContain('text-2xl');
		expect(value.className).toContain('tabular-nums');
		expect(value.className).toContain('text-text-primary');
		// 未点选态不渲染按钮（避免无意义可聚焦元素）
		expect(screen.queryByRole('button')).toBeNull();
	});

	it('variant=strip 的可点项渲染为按钮并响应点击', async () => {
		const handleClick = vi.fn();
		render(MetricRow, {
			props: {
				variant: 'strip',
				items: [
					{ label: '瞬间', value: 4, onClick: handleClick },
					{ label: '关注者', value: 5 }
				]
			}
		});
		const btn = screen.getByRole('button');
		await fireEvent.click(btn);
		expect(handleClick).toHaveBeenCalledTimes(1);
	});
});
