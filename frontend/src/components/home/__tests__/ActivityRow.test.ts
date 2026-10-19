// ActivityRow 契约测试：活动文案/位置渲染与点击回调。
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ActivityRow from '../ActivityRow.svelte';

const moment = {
	description: '湖畔的周末市集开始了',
	location: { location_name: '云龙湖' }
} as never;

describe('ActivityRow', () => {
	it('渲染活动文案与位置', () => {
		render(ActivityRow, { props: { moment, index: 0, onClick: () => {} } });
		expect(screen.getByText('湖畔的周末市集开始了')).toBeTruthy();
		expect(screen.getByText('云龙湖')).toBeTruthy();
	});

	it('默认带距离（附近轴调用方行为不变）', () => {
		const withDistance = {
			description: '湖畔的周末市集',
			location: { location_name: '云龙湖', distance: 900 }
		} as never;
		render(ActivityRow, { props: { moment: withDistance, index: 0, onClick: () => {} } });
		expect(screen.getByText(/900m/)).toBeTruthy();
	});

	it('showDistance=false 时 meta 不带距离，但保留地点与时间', () => {
		const withDistance = {
			description: '湖畔的周末市集',
			created_at: '2026-10-19T00:00:00Z',
			location: { location_name: '云龙湖', distance: 900 }
		} as never;
		render(ActivityRow, { props: { moment: withDistance, index: 0, showDistance: false, onClick: () => {} } });
		// 非地理轴（关注流/热榜）不该出现距离
		expect(screen.queryByText(/900m/)).toBeNull();
		expect(screen.getByText(/云龙湖/)).toBeTruthy();
	});

	it('点击触发 onClick', async () => {
		const onClick = vi.fn();
		render(ActivityRow, { props: { moment, index: 1, onClick } });
		await fireEvent.click(screen.getByRole('button'));
		expect(onClick).toHaveBeenCalledTimes(1);
	});
});
