// MetricRow 契约测试：标签/值渲染、tone 类、列数。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
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
});
