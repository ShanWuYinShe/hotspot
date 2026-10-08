// MapPinMarker 契约测试：图钉渲染、计数徽标、选中态与 GCJ-02 纠偏定位。
import { describe, it, expect, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import MapPinMarker from '../MapPinMarker.svelte';
import { setMapSource } from '@/stores/mapSource.svelte';

const bounds = { minLat: 30, maxLat: 31, minLng: 117, maxLng: 118 };
const pin = { id: 'p1', type: 'user', lat: 30.5, lng: 117.5, count: 1 } as never;

// 每个用例前重置为 OSM 源：旧测试的硬编码期望值按「无纠偏」计算，
// 国内坐标 + amap 源下会因 wgs84ToGcj02 偏移而改变。纠偏语义由新增用例显式覆盖。
beforeEach(() => {
	setMapSource('osm');
});

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

	it('按墨卡托比例定位（与 MapHeatDot 同口径）', () => {
		const { container } = render(MapPinMarker, {
			props: { pin, bounds, isSelected: false, onSelect: () => {} }
		});
		const btn = container.querySelector('button') as HTMLElement;
		expect(btn.style.left).toBe('50%');
		// 30.5° 在墨卡托投影中点不是线性 50%，此处为投影公式的硬编码期望值。
		expect(btn.style.top).toBe('50.12851269994047%');
	});

	it('isSelected 时应用选中缩放与高 z-index', () => {
		const { container } = render(MapPinMarker, {
			props: { pin, bounds, isSelected: true, onSelect: () => {} }
		});
		const btn = container.querySelector('button') as HTMLElement;
		expect(btn.style.transform).toContain('scale(1.32)');
		expect(btn.style.zIndex).toBe('30');
	});

	it('国内坐标：高德源走 GCJ-02 纠偏，OSM 源保持 WGS-84（两者结果不同）', () => {
		// 国内场景下 WGS-84 与 GCJ-02 偏差数百米；若漏纠偏，marker 与底图地标
		// 错位随 zoom 放大愈发明显（本次修复的缺陷）。
		// 图钉须偏离视口中心：中心点两侧 GCJ 偏移近似对称，相对位置不变。
		// amap 值在切源前快照并卸载实例：组件对 mapSource 是响应式的，
		// 不卸载的话 setMapSource('osm') 会把它同步改写，两者恒等。
		const offPin = { id: 'p3', type: 'user', lat: 30.25, lng: 117.25, count: 1 } as never;
		setMapSource('amap');
		const amap = render(MapPinMarker, {
			props: { pin: offPin, bounds, isSelected: false, onSelect: () => {} }
		});
		const amapBtn = amap.container.querySelector('button') as HTMLElement;
		const amapLeft = amapBtn.style.left;
		const amapTop = amapBtn.style.top;
		amap.unmount();

		setMapSource('osm');
		const osm = render(MapPinMarker, {
			props: { pin: offPin, bounds, isSelected: false, onSelect: () => {} }
		});
		const osmBtn = osm.container.querySelector('button') as HTMLElement;

		expect(amapLeft).not.toBe(osmBtn.style.left);
		expect(amapTop).not.toBe(osmBtn.style.top);
	});

	it('境外坐标：GCJ-02 与 WGS-84 一致，两种源下结果相同', () => {
		// outOfChina 时 wgs84ToGcj02 直通，两种源应给出同一位置。
		const abroadBounds = { minLat: 40, maxLat: 41, minLng: -74, maxLng: -73 };
		const abroadPin = { id: 'p2', type: 'user', lat: 40.5, lng: -73.5, count: 1 } as never;

		setMapSource('amap');
		const amap = render(MapPinMarker, {
			props: { pin: abroadPin, bounds: abroadBounds, isSelected: false, onSelect: () => {} }
		});
		const amapBtn = amap.container.querySelector('button') as HTMLElement;

		setMapSource('osm');
		const osm = render(MapPinMarker, {
			props: { pin: abroadPin, bounds: abroadBounds, isSelected: false, onSelect: () => {} }
		});
		const osmBtn = osm.container.querySelector('button') as HTMLElement;

		expect(amapBtn.style.left).toBe(osmBtn.style.left);
		expect(amapBtn.style.top).toBe(osmBtn.style.top);
	});
});
