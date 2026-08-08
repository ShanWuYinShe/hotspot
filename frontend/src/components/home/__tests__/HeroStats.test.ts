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
});
