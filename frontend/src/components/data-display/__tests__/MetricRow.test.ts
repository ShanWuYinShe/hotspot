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
		const dl = container.querySelector('dl') as HTMLElement;
		expect(dl.style.gridTemplateColumns).toContain('repeat(3');
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
});
