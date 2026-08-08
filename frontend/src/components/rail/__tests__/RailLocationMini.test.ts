// RailLocationMini 契约测试：位置卡渲染与地图跳转。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import { goto } from '$app/navigation';
import RailLocationMini from '../RailLocationMini.svelte';

vi.mock('$app/navigation', () => ({ goto: vi.fn() }));

describe('RailLocationMini', () => {
	beforeEach(() => {
		vi.mocked(goto).mockClear();
	});

	it('渲染位置名称与描述', () => {
		render(RailLocationMini, { props: { name: '回龙窝', desc: '鼓楼区' } });
		expect(screen.getByText('回龙窝')).toBeTruthy();
		expect(screen.getByText('鼓楼区')).toBeTruthy();
	});

	it('点击跳转地图路由', async () => {
		render(RailLocationMini, { props: { name: '回龙窝' } });
		await fireEvent.click(screen.getByRole('button'));
		expect(goto).toHaveBeenCalledTimes(1);
	});
});
