import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	islandFlight,
	islandMorphIn,
	islandMorphOut,
	islandRectOf,
	sanitizeIslandSource,
	type IslandRect
} from '../island-morph';

// 灵动岛交接动效契约（docs/design-system.md §22）：
//   - 起跳帧与锚点严格同框（中心对齐 + 长宽比缩放，不再被 clamp 拉偏）；
//   - 只动 transform/opacity：内容层单独走 --cst-island-content，显字晚于显影；
//   - 入口/出口分开编排：出口文字先散、壳后飞回；reduced-motion 直接 0ms。

function mockReduced(reduced: boolean) {
	vi.stubGlobal('matchMedia', (query: string) => ({
		matches: reduced && query.includes('prefers-reduced-motion: reduce'),
		media: query,
		onchange: null,
		addListener: () => {},
		removeListener: () => {},
		addEventListener: () => {},
		removeEventListener: () => {},
		dispatchEvent: () => false
	}));
}

/** 给节点一个可控视口矩形（jsdom 默认全 0，无法走真实分支）。 */
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
			toJSON: () => ({})
		}) as DOMRect;
}

const TARGET: IslandRect = { x: 626, y: 8, width: 189, height: 36 };

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('islandRectOf / sanitizeIslandSource', () => {
	it('islandRectOf 忽略零尺寸元素', () => {
		const el = document.createElement('div');
		expect(islandRectOf(el)).toBeNull();
		expect(islandRectOf(null)).toBeNull();
	});

	it('锚点已滚出视口时被钳回视口内（胶囊不从屏幕外起飞）', () => {
		const offscreen: IslandRect = { x: 284, y: -46, width: 80, height: 28 };
		const safe = sanitizeIslandSource(offscreen, { width: 1440, height: 1000 });
		expect(safe).not.toBeNull();
		expect(safe!.y).toBeGreaterThanOrEqual(0);
		// 宽度/高度不被改写（只修正位置，不虚构尺寸）
		expect(safe!.width).toBe(80);
		expect(safe!.height).toBe(28);
	});

	it('视口内锚点原样返回', () => {
		const src: IslandRect = { x: 300, y: 120, width: 120, height: 30 };
		expect(sanitizeIslandSource(src, { width: 1440, height: 1000 })).toEqual(src);
	});
});

describe('islandFlight 纯几何', () => {
	it('中心位移 + 长宽比缩放：锚点与胶囊严格同框起跳', () => {
		const source: IslandRect = { x: 284, y: 46, width: 80, height: 28 };
		const flight = islandFlight(TARGET, source);
		// 中心：source (324, 60) → target (720.5, 26)
		expect(flight.dx).toBeCloseTo(324 - (626 + 189 / 2), 5);
		expect(flight.dy).toBeCloseTo(60 - (8 + 36 / 2), 5);
		expect(flight.scaleX).toBeCloseTo(80 / 189, 5);
		expect(flight.scaleY).toBeCloseTo(28 / 36, 5);
		expect(flight.distance).toBeGreaterThan(0);
	});

	it('极端长宽比做宽松钳制（不把胶囊压成一条线/拉成巨幕）', () => {
		const tiny = islandFlight(TARGET, { x: 0, y: 0, width: 4, height: 4 });
		expect(tiny.scaleX).toBeGreaterThanOrEqual(0.2);
		expect(tiny.scaleY).toBeGreaterThanOrEqual(0.45);
		const huge = islandFlight(TARGET, { x: 0, y: 0, width: 1200, height: 900 });
		expect(huge.scaleX).toBeLessThanOrEqual(2.2);
		expect(huge.scaleY).toBeLessThanOrEqual(2.0);
	});
});

describe('islandMorphIn（入口编排）', () => {
	it('reduced-motion：0ms 直接到终态', () => {
		mockReduced(true);
		const node = document.createElement('div');
		mockRect(node, TARGET);
		const cfg = islandMorphIn(node, {});
		expect(cfg.duration).toBe(0);
	});

	it('无锚点矩形：退化为短淡入（不硬切）', () => {
		mockReduced(false);
		const node = document.createElement('div');
		mockRect(node, TARGET);
		const cfg = islandMorphIn(node, {});
		expect(cfg.duration).toBeGreaterThan(0);
		expect(cfg.duration).toBeLessThanOrEqual(200);
	});

	it('起跳帧（t=0）贴在锚点：位移/缩放等于几何差，透明度为 0、内容未显', () => {
		mockReduced(false);
		const node = document.createElement('div');
		mockRect(node, TARGET);
		const source: IslandRect = { x: 284, y: 46, width: 80, height: 28 };
		const cfg = islandMorphIn(node, { getSource: () => source });
		const css0 = String(cfg.css?.(0, 1) ?? '');
		expect(css0).toContain('translate3d(' + islandFlight(TARGET, source).dx + 'px');
		expect(css0).toContain('opacity:0');
		expect(css0).toContain('--cst-island-content:0');
	});

	it('末帧（t=1）回到原位：identity transform、内容可见；只出现合成器属性', () => {
		mockReduced(false);
		const node = document.createElement('div');
		mockRect(node, TARGET);
		const cfg = islandMorphIn(node, { getSource: () => ({ x: 284, y: 46, width: 80, height: 28 }) });
		const css1 = String(cfg.css?.(1, 0) ?? '');
		expect(css1).toContain('translate3d(0px, 0px, 0)');
		expect(css1).toContain('scale(1, 1)');
		expect(css1).toContain('opacity:1');
		expect(css1).toContain('--cst-island-content:1');
		// 合成器纪律：不出现 left/top/width/height 等布局属性
		expect(css1).not.toMatch(/(^|;)(left|top|width|height):/);
	});
});

describe('islandMorphOut（出口编排；Svelte out 的 t 从 1 → 0）', () => {
	it('起跳帧（t=1，当前可见态）：identity + 全不透明 + 内容可见', () => {
		mockReduced(false);
		const node = document.createElement('div');
		mockRect(node, TARGET);
		const cfg = islandMorphOut(node, { getSource: () => ({ x: 284, y: 46, width: 80, height: 28 }) });
		const css1 = String(cfg.css?.(1, 0) ?? '');
		expect(css1).toContain('translate3d(0px, 0px, 0)');
		expect(css1).toContain('opacity:1');
		expect(css1).toContain('--cst-island-content:1');
	});

	it('收尾帧（t=0，已移除态）落回锚点：位移/缩放等于几何差，容器与内容都已淡出', () => {
		mockReduced(false);
		const node = document.createElement('div');
		mockRect(node, TARGET);
		const source: IslandRect = { x: 284, y: 46, width: 80, height: 28 };
		const cfg = islandMorphOut(node, { getSource: () => source });
		const flight = islandFlight(TARGET, source);
		const css0 = String(cfg.css?.(0, 1) ?? '');
		expect(css0).toContain('translate3d(' + flight.dx + 'px, ' + flight.dy + 'px, 0)');
		expect(css0).toContain('opacity:0');
		expect(css0).toContain('--cst-island-content:0');
	});

	it('出口文字先散：走完 20% 离场时内容已基本消失，壳仍可见', () => {
		mockReduced(false);
		const node = document.createElement('div');
		mockRect(node, TARGET);
		const cfg = islandMorphOut(node, { getSource: () => ({ x: 284, y: 46, width: 80, height: 28 }) });
		// t=0.8 对应离场进度 p=0.2
		const css = String(cfg.css?.(0.8, 0.2) ?? '');
		expect(css).toMatch(/--cst-island-content:0\.2/);
		expect(css).toMatch(/opacity:0\.79/);
	});
});