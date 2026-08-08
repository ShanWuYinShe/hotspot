// MapPinMarker 契约测试：图钉渲染、计数徽标与选中态。
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/svelte';
import MapPinMarker from '../MapPinMarker.svelte';

const bounds = { minLat: 30, maxLat: 31, minLng: 117, maxLng: 118 };
const pin = { id: 'p1', type: 'user', lat: 30.5, lng: 117.5, count: 1 } as never;

describe('MapPinMarker', () => {
	it('渲染图钉 SVG 按钮', () => {
		const { container } = render(MapPinMarker, {
			props: { pin, bounds, isSelected: false, onSelect: () => {} }
		});
		expect(container.querySelector('button')).not.toBeNull();
		expect(container.querySelector('svg')).not.toBeNull();
	});

	it('count>1 时渲染计数徽标', () => {
		const { container } = render(MapPinMarker, {
			props: {
				pin: { id: 'p1', type: 'user', lat: 30.5, lng: 117.5, count: 3 } as never,
				bounds,
				isSelected: false,
				onSelect: () => {}
			}
		});
		expect(container.textContent).toContain('3');
	});

	it('isSelected 时应用选中缩放与高 z-index', () => {
		const { container } = render(MapPinMarker, {
			props: { pin, bounds, isSelected: true, onSelect: () => {} }
		});
		const btn = container.querySelector('button') as HTMLElement;
		expect(btn.style.transform).toContain('scale(1.32)');
		expect(btn.style.zIndex).toBe('30');
	});
});
