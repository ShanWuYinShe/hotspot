// MapAnchorMarker 契约测试：浏览锚点必须按地理坐标投影，不能钉在视口中心。
import { describe, it, expect, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import MapAnchorMarker from '../MapAnchorMarker.svelte';
import { setMapSource } from '@/stores/mapSource.svelte';

const bounds = { minLat: 30, maxLat: 31, minLng: 117, maxLng: 118 };

beforeEach(() => {
	setMapSource('osm');
});

function renderMarker(anchor: { lat: number; lng: number }) {
	return render(MapAnchorMarker, {
		props: { anchor: anchor as never, bounds, onSelect: () => {} }
	});
}

describe('MapAnchorMarker', () => {
	it('渲染可点击的锚点按钮与投影定位', () => {
		const { container } = renderMarker({ lat: 30.5, lng: 117.5 });
		const btn = container.querySelector('button') as HTMLElement;
		expect(btn).not.toBeNull();
		expect(btn.style.left).toBe('50%');
		// 30.5° 在墨卡托投影中点不是线性 50%
		expect(btn.style.top).toBe('50.12851269994047%');
		expect(btn.style.transform).toContain('translate(-50%,-100%)');
	});

	it('偏离中心的地理坐标投影到偏离中心的位置', () => {
		const { container } = renderMarker({ lat: 30.25, lng: 117.25 });
		const btn = container.querySelector('button') as HTMLElement;
		const left = parseFloat(btn.style.left);
		const top = parseFloat(btn.style.top);
		expect(left).toBeCloseTo(25, 5);
		// 纬度越小越靠南，投影到视口下方（不是中心）
		expect(top).toBeGreaterThan(74);
		expect(top).toBeLessThan(76);
		// 关键回归：不得把锚点钉在视口中心
		expect(left).not.toBe(50);
		expect(top).not.toBe(50);
	});

	it('高德源走 GCJ-02 纠偏，OSM 源保持 WGS-84', () => {
		const offAnchor = { lat: 30.25, lng: 117.25 };
		setMapSource('amap');
		const amap = renderMarker(offAnchor);
		const amapBtn = amap.container.querySelector('button') as HTMLElement;
		const amapLeft = amapBtn.style.left;
		const amapTop = amapBtn.style.top;
		amap.unmount();

		setMapSource('osm');
		const osm = renderMarker(offAnchor);
		const osmBtn = osm.container.querySelector('button') as HTMLElement;
		expect(amapLeft).not.toBe(osmBtn.style.left);
		expect(amapTop).not.toBe(osmBtn.style.top);
	});
});
