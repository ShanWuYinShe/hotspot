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

	it('点击触发 onClick', async () => {
		const onClick = vi.fn();
		render(ActivityRow, { props: { moment, index: 1, onClick } });
		await fireEvent.click(screen.getByRole('button'));
		expect(onClick).toHaveBeenCalledTimes(1);
	});
});
