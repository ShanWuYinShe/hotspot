// MockBadge：mock 接管时必须肉眼可见（此前零提示，多次把假数据当真排查）。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';

const toggleState = vi.hoisted(() => ({ on: true }));

vi.mock('@/mocks/toggle', () => ({
	isMockEnabled: () => toggleState.on,
	subscribe: (fn: () => void) => {
		fn();
		return () => {};
	}
}));

import MockBadge from '../MockBadge.svelte';

describe('MockBadge', () => {
	beforeEach(() => {
		toggleState.on = true;
	});

	it('mock 开启时显示 Mock 数据徽标', () => {
		render(MockBadge);
		expect(screen.getByText('Mock 数据')).toBeTruthy();
	});

	it('mock 关闭时不渲染任何节点', () => {
		toggleState.on = false;
		render(MockBadge);
		expect(screen.queryByText('Mock 数据')).toBeNull();
	});
});
