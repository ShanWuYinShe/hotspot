import { describe, it, expect } from 'vitest';
import { computeAdaptiveGrid, adaptiveGridStyle } from '../adaptive-grid';

const gap = [0.75, 0.875, 1] as const;

describe('computeAdaptiveGrid', () => {
	it('桌面 auto：按最小轨反推列数与轨道宽度（与 auto-fill 一致）', () => {
		const m = computeAdaptiveGrid({
			width: 1680,
			viewportWidth: 1920,
			count: 2,
			rootPx: 16,
			gapRem: gap,
			minTrackRem: 18.75
		});
		expect(m.columns).toBe(5);
		expect(m.trackPx).toBeCloseTo((1680 - 4 * 16) / 5, 3);
		expect(m.centered).toBe(true);
	});

	it('数量达到列数后不再居中（最后一行保持列对齐）', () => {
		const m = computeAdaptiveGrid({
			width: 1680,
			viewportWidth: 1920,
			count: 5,
			rootPx: 16,
			gapRem: gap,
			minTrackRem: 18.75
		});
		expect(m.columns).toBe(5);
		expect(m.centered).toBe(false);
	});

	it('fixedCols 用视口断点、不用容器宽度（受限内容列也保持原列数）', () => {
		const m = computeAdaptiveGrid({
			width: 904,
			viewportWidth: 1920,
			count: 2,
			rootPx: 16,
			gapRem: gap,
			fixedCols: { base: 2, lg: 3, xl: 4 }
		});
		expect(m.columns).toBe(4);
		expect(m.trackPx).toBeCloseTo((904 - 3 * 16) / 4, 3);
		expect(m.centered).toBe(true);
	});

	it('手机 <640：2 列；单卡居中且宽度 = 半列（不放大）', () => {
		const m = computeAdaptiveGrid({
			width: 358,
			viewportWidth: 390,
			count: 1,
			rootPx: 16,
			gapRem: gap,
			minTrackRem: 18.75
		});
		expect(m.columns).toBe(2);
		expect(m.trackPx).toBeCloseTo((358 - 12) / 2, 3);
		expect(m.centered).toBe(true);
	});

	it('BSP 网格 allowCenter=false 时不居中', () => {
		const m = computeAdaptiveGrid({
			width: 1680,
			viewportWidth: 1920,
			count: 2,
			rootPx: 16,
			gapRem: gap,
			minTrackRem: 18.75,
			allowCenter: false
		});
		expect(m.centered).toBe(false);
	});
});

describe('adaptiveGridStyle', () => {
	it('稀疏：固定 count 个轨道并居中；密集：显式列数', () => {
		const sparse = adaptiveGridStyle({ width: 1680, viewportWidth: 1920, count: 2, rootPx: 16, gapRem: gap, minTrackRem: 18.75 });
		expect(sparse).toContain('repeat(2,');
		expect(sparse).toContain('justify-content:center');
		const dense = adaptiveGridStyle({ width: 1680, viewportWidth: 1920, count: 8, rootPx: 16, gapRem: gap, minTrackRem: 18.75 });
		expect(dense).toContain('repeat(5,');
		expect(dense).not.toContain('justify-content:center');
	});

	it('大屏等比放大：rootPx 翻倍时轨道宽度按比例放大', () => {
		const normal = computeAdaptiveGrid({ width: 1680, viewportWidth: 1920, count: 8, rootPx: 16, gapRem: gap, minTrackRem: 18.75 });
		const scaled = computeAdaptiveGrid({ width: 3360, viewportWidth: 3840, count: 8, rootPx: 32, gapRem: gap, minTrackRem: 18.75 });
		expect(scaled.trackPx / normal.trackPx).toBeCloseTo(2, 3);
	});
});

