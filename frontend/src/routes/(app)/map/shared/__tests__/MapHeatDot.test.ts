// MapHeatDot 契约测试：热力点定位/尺寸计算与 token 样式。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import MapHeatDot from '../MapHeatDot.svelte';

const bounds = { minLat: 30, maxLat: 31, minLng: 117, maxLng: 118 };
const bucket = {
	cell_token: 't',
	level: 12,
	location: { latitude: 30.5, longitude: 117.5 },
	total_count: 5
} as never;

describe('MapHeatDot', () => {
	it('渲染热力点并应用 token 背景/光晕', () => {
		const { container } = render(MapHeatDot, { props: { bucket, bounds } });
		const dot = container.firstElementChild as HTMLElement;
		expect(dot).not.toBeNull();
		expect(dot.style.background).toContain('var(--cst-accent-default)');
		expect(dot.style.boxShadow).toContain('var(--cst-marker-halo)');
	});

	it('按网格中心计算定位与尺寸', () => {
		const { container } = render(MapHeatDot, { props: { bucket, bounds } });
		const dot = container.firstElementChild as HTMLElement;
		expect(dot.style.left).toBe('50%');
		expect(dot.style.top).toBe('50%');
		expect(dot.style.width).toBe('16px');
		expect(dot.style.height).toBe('16px');
	});
});
