// MapCanvas 滚轮缩放契约测试：滚轮路径此前无覆盖（只有捏合测试）。
// 以国内坐标 + amap 源（GCJ 转换路径）验证：滚轮触发缩放且阻止默认滚动；
// 同锚点放大再缩小中心必须回到原位（与捏合的缩放锚点不变量同口径）。
// 注意：滚轮监听挂在画布父级（地图面板），事件须派发在画布上冒泡到父级。
import { describe, it, expect, beforeAll, vi } from 'vitest';
import { render, screen, act } from '@testing-library/svelte';
import PinchHost from './PinchHost.svelte';

const VIEW_W = 800;
const VIEW_H = 600;

class ResizeObserverStub {
	constructor(private cb?: (entries: unknown[], observer: unknown) => void) {}
	observe(target: Element) {
		this.cb?.([{ target, contentRect: { width: VIEW_W, height: VIEW_H } }], this);
	}
	unobserve() {}
	disconnect() {}
}

beforeAll(() => {
	vi.stubGlobal('ResizeObserver', ResizeObserverStub);
	Object.defineProperty(HTMLElement.prototype, 'clientWidth', { configurable: true, get: () => VIEW_W });
	Object.defineProperty(HTMLElement.prototype, 'clientHeight', { configurable: true, get: () => VIEW_H });
	HTMLElement.prototype.getBoundingClientRect = function () {
		return { x: 0, y: 0, left: 0, top: 0, right: VIEW_W, bottom: VIEW_H, width: VIEW_W, height: VIEW_H, toJSON: () => ({}) } as DOMRect;
	};
});

function wheel(el: Element, deltaY: number, x: number, y: number, opts: { ctrlKey?: boolean } = {}): Event {
	let ev: Event;
	try {
		ev = new WheelEvent('wheel', { bubbles: true, cancelable: true, deltaY, clientX: x, clientY: y, ctrlKey: opts.ctrlKey ?? false });
	} catch {
		ev = new Event('wheel', { bubbles: true, cancelable: true });
		Object.assign(ev, { deltaY, clientX: x, clientY: y, ctrlKey: opts.ctrlKey ?? false });
	}
	el.dispatchEvent(ev);
	return ev;
}

const TOL_DEG = 1e-3;

describe('MapCanvas 滚轮缩放', () => {
	it('滚轮放大触发 zoom+1 且阻止默认滚动', async () => {
		const { container } = render(PinchHost, { props: { center: { lat: 34.26, lng: 117.19 }, zoom: 14 } });
		const canvas = container.querySelector('[data-testid="canvas-host"] > div') as HTMLElement;
		expect(canvas).toBeTruthy();
		let ev!: Event;
		await act(async () => {
			ev = wheel(canvas, -120, 200, 450);
		});
		expect(ev.defaultPrevented).toBe(true);
		expect(Number(screen.getByTestId('zoom').textContent)).toBe(15);
	});

	it('同锚点放大再缩小：中心回到原位（锚点不变量）', async () => {
		const startCenter = { lat: 34.26, lng: 117.19 };
		const { container } = render(PinchHost, { props: { center: startCenter, zoom: 14 } });
		const canvas = container.querySelector('[data-testid="canvas-host"] > div') as HTMLElement;
		const center0 = screen.getByTestId('center').textContent || '';
		await act(async () => {
			wheel(canvas, -120, 200, 450);
		});
		expect(Number(screen.getByTestId('zoom').textContent)).toBe(15);
		const mid = (screen.getByTestId('center').textContent || '').split(',').map(Number);
		// 偏心锚点放大必须拉动中心（锚点失效时中心原地不动）
		expect(Math.abs(mid[0] - startCenter.lat) + Math.abs(mid[1] - startCenter.lng)).toBeGreaterThan(1e-6);
		await act(async () => {
			wheel(canvas, 120, 200, 450);
		});
		expect(Number(screen.getByTestId('zoom').textContent)).toBe(14);
		const [lat0, lng0] = center0.split(',').map(Number);
		const [lat1, lng1] = (screen.getByTestId('center').textContent || '').split(',').map(Number);
		expect(Math.abs(lat1 - lat0)).toBeLessThan(TOL_DEG);
		expect(Math.abs(lng1 - lng0)).toBeLessThan(TOL_DEG);
	});

	it('触控板双指捏合（ctrlKey=true）平滑缩放', async () => {
		const { container } = render(PinchHost, { props: { center: { lat: 34.26, lng: 117.19 }, zoom: 14 } });
		const canvas = container.querySelector('[data-testid="canvas-host"] > div') as HTMLElement;
		await act(async () => {
			// 触控板捏合：连续微小 delta 带 ctrlKey
			wheel(canvas, -25, 200, 450, { ctrlKey: true });
			wheel(canvas, -30, 200, 450, { ctrlKey: true });
		});
		expect(Number(screen.getByTestId('zoom').textContent)).toBe(15);
	});

	it('滚轮未达整级时产生即时连续 gestureTransform（零死区跟手）', async () => {
		const { container } = render(PinchHost, { props: { center: { lat: 34.26, lng: 117.19 }, zoom: 14 } });
		const canvas = container.querySelector('[data-testid="canvas-host"] > div') as HTMLElement;
		await act(async () => {
			wheel(canvas, -36, 200, 450);
		});
		// zoom 尚未跨整级，但 gestureTransform (gt) 已即时输出 scale > 1
		const transform = screen.getByTestId('gt').textContent || '';
		expect(transform).toContain('scale(');
		expect(transform).toContain('transform-origin:');
	});

	it('缩放跳级时保留上一级瓦片做垫底图层（防白屏闪烁）', async () => {
		const { container } = render(PinchHost, { props: { center: { lat: 34.26, lng: 117.19 }, zoom: 14 } });
		const canvas = container.querySelector('[data-testid="canvas-host"] > div') as HTMLElement;
		await act(async () => {
			wheel(canvas, -120, 200, 450);
		});
		// 应存在垫底图层与 scale 变换
		const bgLayer = container.querySelector('[data-testid="prev-layer"]') as HTMLElement;
		expect(bgLayer).toBeTruthy();
		expect(bgLayer.style.transform).toContain('scale(2)');
	});
});
