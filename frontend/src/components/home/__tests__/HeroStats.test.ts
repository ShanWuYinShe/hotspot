// HeroStats 契约测试：统计数字渲染、纯展示（不承载跳转）、MetricRow strip 布局。
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import HeroStats from '../HeroStats.svelte';

const stats = { nearby: 5, active: 3, online: 12 };

describe('HeroStats', () => {
	it('渲染统计数字，且不再作为按钮/跳转入口', () => {
		render(HeroStats, { props: { stats } });
		expect(screen.queryByRole('button')).toBeNull();
		expect(screen.getByText('5')).toBeTruthy();
		expect(screen.getByText('3')).toBeTruthy();
		expect(screen.getByText('12')).toBeTruthy();
	});

	it('统计区走 MetricRow strip：发丝竖线分隔、无灰底药丸容器', () => {
		const { container } = render(HeroStats, { props: { stats } });
		const strip = container.querySelector('[role="group"]') as HTMLElement | null;
		expect(strip).not.toBeNull();
		expect(strip?.className).toContain('divide-x');
		expect(strip?.className).toContain('divide-border-subtle');
		expect(strip?.children.length).toBe(3);
	});

	it('标题行弹性契约：文本簇可收缩、副标题单行截断', () => {
		const { container } = render(HeroStats, {
			props: { stats, locationName: '一个很长很长的定位名称' }
		});
		const headerRow = container.firstElementChild?.firstElementChild as HTMLElement | undefined;
		const textCluster = headerRow?.children[0] as HTMLElement | undefined;
		expect(textCluster?.classList.contains('flex-1')).toBe(true);
		expect(textCluster?.classList.contains('min-w-0')).toBe(true);
		const subtitle = textCluster?.querySelector('p.text-xs');
		expect(subtitle?.classList.contains('truncate')).toBe(true);
		expect(subtitle?.textContent).toContain('一个很长很长的定位名称');
	});
});
