// HeroStats 契约测试：主按钮 aria/点击与统计数字渲染。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import HeroStats from '../HeroStats.svelte';

const stats = { nearby: '5', active: '3', online: '12' };

describe('HeroStats', () => {
	it('渲染主按钮 aria-label 与统计数字', () => {
		render(HeroStats, { props: { stats, onExplore: () => {} } });
		expect(screen.getByRole('button', { name: '查看附近的人' })).toBeTruthy();
		expect(screen.getByText('5')).toBeTruthy();
		expect(screen.getByText('3')).toBeTruthy();
		expect(screen.getByText('12')).toBeTruthy();
	});

	it('点击触发 onExplore', async () => {
		const onExplore = vi.fn();
		render(HeroStats, { props: { stats, onExplore } });
		await fireEvent.click(screen.getByRole('button'));
		expect(onExplore).toHaveBeenCalledTimes(1);
	});

	it('标题行弹性契约：文本簇可收缩、副标题单行截断、箭头不压缩', () => {
		const { container } = render(HeroStats, {
			props: { stats, onExplore: () => {}, locationName: '一个很长很长的定位名称' }
		});
		const headerRow = container.querySelector('button > div');
		const textCluster = headerRow?.children[0] as HTMLElement | undefined;
		const chevron = headerRow?.children[1] as HTMLElement | undefined;
		// 左侧文本簇弹性（先收缩让位）
		expect(textCluster?.classList.contains('flex-1')).toBe(true);
		expect(textCluster?.classList.contains('min-w-0')).toBe(true);
		// 右箭头锁定不压缩
		expect(chevron?.classList.contains('shrink-0')).toBe(true);
		// 副标题单行截断（长定位名不顶出箭头、不贴边裁字）
		const subtitle = textCluster?.querySelector('p.text-xs');
		expect(subtitle?.classList.contains('truncate')).toBe(true);
		expect(subtitle?.textContent).toContain('一个很长很长的定位名称');
	});
});
