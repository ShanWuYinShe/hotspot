// MapHeatDot 契约测试：数目标签、点击下钻、定位、直径分档、token 样式与 GCJ-02 纠偏。
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/svelte';
import MapHeatDot from '../MapHeatDot.svelte';
import { setMapSource } from '@/stores/mapSource.svelte';
import type { MapHeatBucket } from '@/api/types';

// 旧测试硬编码期望值按「无纠偏」计算，统一在 OSM 源下跑；
// 纠偏语义由新增用例显式切到 amap 覆盖。
beforeEach(() => {
	setMapSource('osm');
});

const bounds = { minLat: 30, maxLat: 31, minLng: 117, maxLng: 118 };

function bucketWith(total: number, breakdown: Record<string, number | null> | null = null) {
	return {
		cell_token: 't',
		level: 12,
		location: { latitude: 30.5, longitude: 117.5 },
		total_count: total,
		breakdown
	};
}

function renderDot(
	total: number,
	breakdown: Record<string, number | null> | null = null,
	onSelect?: (bucket: MapHeatBucket) => void,
	location: { latitude: number; longitude: number } = { latitude: 30.5, longitude: 117.5 }
) {
	const { container } = render(MapHeatDot, {
		props: { bucket: { ...bucketWith(total, breakdown), location } as never, bounds, onSelect }
	});
	return container.firstElementChild as HTMLElement;
}

describe('MapHeatDot', () => {
	it('直接把该网格的对象总数显示出来（用户一眼可知聚合了多少内容）', () => {
		expect(renderDot(54).textContent?.trim()).toBe('54');
		expect(renderDot(7).textContent?.trim()).toBe('7');
	});

	it('超过 999 收敛为 999+，避免数字撑破圆点', () => {
		expect(renderDot(12345).textContent?.trim()).toBe('999+');
	});

	it('按桶坐标换算视口百分比并居中（Web Mercator 比例，国家尺度不发散）', () => {
		const dot = renderDot(5);
		expect(dot.style.left).toBe('50%');
		// 30.5° 在墨卡托投影中点不是线性 50%，此处为投影公式的硬编码期望值。
		expect(dot.style.top).toBe('50.12851269994047%');
		expect(dot.style.transform).toContain('translate(-50%,-50%)');
	});

	it('直径按数字位数分档，保证数字完整显示', () => {
		const cases: Array<[number, string]> = [
			[7, '28px'],
			[54, '28px'],
			[540, '34px'],
			[12345, '40px']
		];
		for (const [total, size] of cases) {
			const dot = renderDot(total);
			expect(dot.style.width, `total=${total}`).toBe(size);
			expect(dot.style.height, `total=${total}`).toBe(size);
		}
	});

	it('中性卡面 + accent 描边：既醒目又不与「实心蓝」的位置点/图钉同形同色', () => {
		const dot = renderDot(54);
		// jsdom 会把 style 属性规范化（冒号后补空格），比较前压掉空白
		const style = (dot.getAttribute('style') ?? '').replace(/\s+/g, '');
		// 卡面用 surface、文字用常规文字色：不占用任何「实心彩色」语义
		expect(style).toContain('background:var(--cst-surface-primary)');
		expect(style).toContain('color:var(--cst-text-primary)');
		// accent 只用于描边（保留品牌色醒目度），不得再作填充——
		// 填充成实心蓝圆点时，会与 --cst-marker-user 图钉和「当前位置点」混为一谈
		expect(style).toContain('border:var(--cst-border-width-strong)solidvar(--cst-accent-default)');
		expect(style).not.toContain('background:var(--cst-accent-default)');
		expect(dot.className).toContain('shadow-raised-md');
		// 旧实现靠 opacity 插值弱化，现为实色，不应再写入 opacity
		expect(style).not.toContain('opacity:');
	});

	it('用徽标排版契约（居中 + 不继承全局行高）', () => {
		const dot = renderDot(54);
		const cls = dot.className;
		for (const required of ['inline-flex', 'items-center', 'justify-center', 'leading-none', 'rounded-full']) {
			expect(cls, required).toContain(required);
		}
	});

	it('层级高于当前位置点（z-10 > z-index:5），数字不会被定位点盖住', () => {
		// 当前位置点固定在视口正中心且自带 z-index:5；地图中心常与数据密集区重合，
		// 热力徽标必须抬到它之上，否则该处数字整块不可见（实测 2026-10-03）。
		expect(renderDot(54).className).toContain('z-10');
	});

	it('提供含分类明细的无障碍名称', () => {
		const withBreakdown = renderDot(34, { user: 22, group: 4, moment: 8 });
		expect(withBreakdown.getAttribute('aria-label')).toBe('此处聚合 34 项：22 位用户 · 4 个群组 · 8 条动态');
	});

	it('breakdown 为 null 时无障碍名称退化为总数（不出现 undefined）', () => {
		const dot = renderDot(12, null);
		expect(dot.getAttribute('aria-label')).toBe('此处聚合 12 项');
	});

	it('渲染为 button：可聚焦、可键盘触达（style skill 焦点可见性契约）', () => {
		const dot = renderDot(54);
		expect(dot.tagName).toBe('BUTTON');
		expect(dot.getAttribute('type')).toBe('button');
		expect(dot.className).toContain('focus-ring');
		// 旧实现 pointer-events:none + role=img：点击穿透、无障碍读不出交互语义
		expect(dot.className).not.toContain('pointer-events-none');
	});

	it('点击徽标触发 onSelect 并带上该桶（下钻入口）', async () => {
		const onSelect = vi.fn();
		const { container } = render(MapHeatDot, {
			props: { bucket: bucketWith(34, { user: 22, group: 4, moment: 8 }) as never, bounds, onSelect }
		});
		const dot = container.firstElementChild as HTMLElement;
		await dot.click();
		expect(onSelect).toHaveBeenCalledTimes(1);
		expect(onSelect.mock.calls[0][0].cell_token).toBe('t');
	});

	it('未传 onSelect 时点击不抛错', () => {
		const dot = renderDot(54);
		expect(() => dot.click()).not.toThrow();
	});

	it('国内坐标：高德源走 GCJ-02 纠偏，OSM 源保持 WGS-84（两者结果不同）', () => {
		// 与 MapPinMarker 同根因：bounds 与桶心必须同空间投影，否则国内场景下
		// 底图瓦片（GCJ-02）与徽标（WGS-84）随 zoom 放大错位。
		// 桶心须偏离视口中心（中心两侧 GCJ 偏移近似对称）；amap 值在切源前
		// 快照并卸载实例——组件对 mapSource 响应式，不卸载会被同步改写。
		setMapSource('amap');
		const amap = render(MapHeatDot, {
			props: {
				bucket: { ...bucketWith(54), location: { latitude: 30.25, longitude: 117.25 } } as never,
				bounds
			}
		});
		const amapDot = amap.container.firstElementChild as HTMLElement;
		const amapLeft = amapDot.style.left;
		const amapTop = amapDot.style.top;
		amap.unmount();

		setMapSource('osm');
		const osmDot = renderDot(54, null, undefined, { latitude: 30.25, longitude: 117.25 });

		expect(amapLeft).not.toBe(osmDot.style.left);
		expect(amapTop).not.toBe(osmDot.style.top);
	});

	it('境外坐标：GCJ-02 与 WGS-84 一致，两种源下结果相同', () => {
		const abroadBounds = { minLat: 40, maxLat: 41, minLng: -74, maxLng: -73 };
		const abroadBucket = {
			cell_token: 'abroad',
			level: 12,
			location: { latitude: 40.5, longitude: -73.5 },
			total_count: 7,
			breakdown: null
		} as never;

		setMapSource('amap');
		const amap = render(MapHeatDot, { props: { bucket: abroadBucket, bounds: abroadBounds } });
		const amapDot = amap.container.firstElementChild as HTMLElement;

		setMapSource('osm');
		const osm = render(MapHeatDot, { props: { bucket: abroadBucket, bounds: abroadBounds } });
		const osmDot = osm.container.firstElementChild as HTMLElement;

		expect(amapDot.style.left).toBe(osmDot.style.left);
		expect(amapDot.style.top).toBe(osmDot.style.top);
	});
});
