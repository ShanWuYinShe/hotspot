import { describe, expect, it } from 'vitest';
import {
	computeFlightCurve,
	evalBezier,
	islandFlight,
	islandRectOf,
	islandTextRectOf,
	type IslandRect
} from '../island-morph';

// 灵动岛几何契约（docs/design-system.md §22）：
//   - 锚点取「文字矩形」而不是块级元素矩形（起跳点与肉眼一致）；
//   - 几何差按中心对齐 + 精确缩放，极端长宽比才走缩放保护；
//   - 纯值快照：不持有 DOM 引用，可跨帧插值/比较。

function mockRect(node: HTMLElement, rect: IslandRect) {
	node.getBoundingClientRect = () =>
		({
			x: rect.x,
			y: rect.y,
			left: rect.x,
			top: rect.y,
			width: rect.width,
			height: rect.height,
			right: rect.x + rect.width,
			bottom: rect.y + rect.height,
			toJSON: () => rect
		}) as DOMRect;
}

describe('islandRectOf / islandTextRectOf', () => {
	it('空节点与零尺寸返回 null（不做无效动画）', () => {
		expect(islandRectOf(null)).toBeNull();
		const el = document.createElement('div');
		mockRect(el, { x: 0, y: 0, width: 0, height: 0 });
		expect(islandRectOf(el)).toBeNull();
	});

	it('把元素矩形收敛成纯值快照', () => {
		const el = document.createElement('div');
		mockRect(el, { x: 100, y: 80, width: 120, height: 28 });
		expect(islandRectOf(el)).toEqual({ x: 100, y: 80, width: 120, height: 28 });
	});

	it('文字矩形取不到时回退元素矩形（jsdom / 空节点）', () => {
		const el = document.createElement('h1');
		el.textContent = '附近的人';
		mockRect(el, { x: 120, y: 76, width: 96, height: 28 });
		// jsdom 的 Range 没有布局：Range 矩形不可用 → 回退元素矩形
		expect(islandTextRectOf(el)).toEqual({ x: 120, y: 76, width: 96, height: 28 });
	});
});

describe('islandFlight', () => {
	it('中心对齐：位移 = 锚点中心 - 泊位中心', () => {
		const target = { x: 600, y: 8, width: 200, height: 36 };
		const source = { x: 100, y: 90, width: 80, height: 28 };
		const flight = islandFlight(target, source);
		expect(flight.dx).toBeCloseTo(100 + 40 - (600 + 100), 5);
		expect(flight.dy).toBeCloseTo(90 + 14 - (8 + 18), 5);
		expect(flight.distance).toBeCloseTo(Math.hypot(flight.dx, flight.dy), 5);
	});

	it('缩放 = 锚点尺寸 / 泊位尺寸（长宽比独立，不强行等比）', () => {
		const target = { x: 0, y: 0, width: 200, height: 40 };
		const source = { x: 0, y: 0, width: 100, height: 30 };
		const flight = islandFlight(target, source);
		expect(flight.scaleX).toBeCloseTo(0.5, 5);
		expect(flight.scaleY).toBeCloseTo(0.75, 5);
	});

	it('极端长宽比走缩放保护（不把胶囊压成一条线/拉成巨幕）', () => {
		const target = { x: 0, y: 0, width: 200, height: 40 };
		const tiny = islandFlight(target, { x: 0, y: 0, width: 2, height: 2 });
		expect(tiny.scaleX).toBeGreaterThanOrEqual(0.2);
		expect(tiny.scaleY).toBeGreaterThanOrEqual(0.45);
		const huge = islandFlight(target, { x: 0, y: 0, width: 1200, height: 400 });
		expect(huge.scaleX).toBeLessThanOrEqual(2.2);
		expect(huge.scaleY).toBeLessThanOrEqual(2.0);
	});
});

describe('computeFlightCurve & evalBezier', () => {
	it('生成抛物线飞行轨道：起点、终点为中心对齐，控制点向上拱起', () => {
		const source = { x: 200, y: 150, width: 100, height: 40 };
		const target = { x: 600, y: 10, width: 200, height: 36 };
		const curve = computeFlightCurve(source, target, 48);

		expect(curve.p0).toEqual({ x: 250, y: 170 });
		expect(curve.p2).toEqual({ x: 700, y: 28 });
		// 拱起高度高过起点与终点的最高者
		expect(curve.p1.y).toBe(28 - 48);
	});

	it('贝塞尔曲线插值：t=0 为起点，t=1 为终点，t=0.5 落在拱起高处', () => {
		const source = { x: 200, y: 150, width: 100, height: 40 };
		const target = { x: 600, y: 10, width: 200, height: 36 };
		const curve = computeFlightCurve(source, target, 48);

		const at0 = evalBezier(curve, 0);
		expect(at0.x).toBeCloseTo(curve.p0.x, 5);
		expect(at0.y).toBeCloseTo(curve.p0.y, 5);

		const at1 = evalBezier(curve, 1);
		expect(at1.x).toBeCloseTo(curve.p2.x, 5);
		expect(at1.y).toBeCloseTo(curve.p2.y, 5);

		const mid = evalBezier(curve, 0.5);
		// 曲线中间位置高度显著高于线性插值
		const linearMidY = (curve.p0.y + curve.p2.y) / 2;
		expect(mid.y).toBeLessThan(linearMidY);
	});

	it('快速滚动导致负视口坐标时安全保护：起点纵坐标回退至顶栏下沿', () => {
		const fastScrollSource = { x: 200, y: -120, width: 100, height: 40 };
		const target = { x: 600, y: 10, width: 200, height: 36 };
		const curve = computeFlightCurve(fastScrollSource, target, 48);
		expect(curve.p0.y).toBe(56 + 20);
	});
});