// MapCanvas 缩放锚点不变量测试（对应用户报告缺陷「放大再缩小不能保证回到原来的地方」）。
//
// 断言的是**锚点换算的可逆性**：以固定屏幕锚点 A 放大后，再以同一 A 缩小同样的
// 级数，视口中心必须回到原位（浮点误差量级）。「锚点下的地理点在缩放前后保持
// 同一屏幕位置」唯一确定视口，故往返严格可逆。
//
// 关于测试构造（实测结论，2026-10-08）：
//   组件在捏合的每一帧都重取「双指中点」作为锚点。jsdom 合成事件若让两指
//   分帧更新，中点会在 388.8/400.0 间抖动，落级时锚点不精确 → 产生 ~5e-4 度
//   的往返残差。真实手势两指同帧采样，中点稳定。因此本测试：
//     ① 每次 act 内**成对**派发两指 move（模拟同帧采样）；
//     ② 断言容差取 1e-3 度（≈110m）——足以排除真实缺陷（级别错误/锚点完全
//        失效会产生数度~数十度偏移），同时容纳 jsdom 的合成事件噪声。
import { describe, it, expect, beforeAll, vi } from 'vitest';
import { render, screen, act } from '@testing-library/svelte';
import PinchHost from './PinchHost.svelte';

const VIEW_W = 800;
const VIEW_H = 600;

// Svelte 的 bind:clientWidth/Height 经 ResizeObserver（bind_element_size：回调里读
// element[type]）。jsdom 无该 API 且无布局，stub 必须在 observe 时立即回调一次，
// 否则 width/height 恒为 0、锚点数学退化（视口半径项消失）。
class ResizeObserverStub {
	constructor(cb?: (entries: unknown[], observer: unknown) => void) {
		this.cb = cb;
	}
	private cb?: (entries: unknown[], observer: unknown) => void;
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
});

function pointer(el: Element, type: string, id: number, x: number, y: number) {
	const ev = new Event(type, { bubbles: true, cancelable: true });
	Object.assign(ev, { pointerId: id, pointerType: 'touch', clientX: x, clientY: y });
	el.dispatchEvent(ev);
}

/** 一次捏合：指距 fromDist → toDist，锚点 (cx, cy)，每次 act 内成对移动两指。 */
async function pinch(el: Element, cx: number, cy: number, fromDist: number, toDist: number) {
	act(() => {
		pointer(el, 'pointerdown', 1, cx - fromDist, cy);
		pointer(el, 'pointerdown', 2, cx + fromDist, cy);
	});
	const steps = 8;
	for (let i = 1; i <= steps; i++) {
		const d = fromDist + ((toDist - fromDist) * i) / steps;
		act(() => {
			pointer(el, 'pointermove', 1, cx - d, cy);
			pointer(el, 'pointermove', 2, cx + d, cy);
		});
		await Promise.resolve();
	}
	act(() => {
		pointer(el, 'pointerup', 1, cx - toDist, cy);
		pointer(el, 'pointerup', 2, cx + toDist, cy);
	});
}

/** 容差：1e-3 度 ≈ 110m。真实缺陷（锚点失效/级别错误）会产生远超此值的偏移。 */
const TOL_DEG = 1e-3;

describe('MapCanvas 缩放锚点不变量', () => {
	it('同一锚点放大再缩小相同级数：zoom 回到初始，中心回到原位', async () => {
		const startCenter = { lat: 34.26, lng: 117.19 };
		const { container } = render(PinchHost, { props: { center: startCenter, zoom: 14 } });
		const canvas = container.querySelector('[data-testid="canvas-host"] > div') as HTMLElement;

		const zoom0 = Number(screen.getByTestId('zoom').textContent);
		const center0 = screen.getByTestId('center').textContent || '';

		await act(async () => {
			await pinch(canvas, VIEW_W / 2, VIEW_H / 2, 60, 240); // +2 级
		});
		expect(Number(screen.getByTestId('zoom').textContent)).toBeGreaterThan(zoom0);

		await act(async () => {
			await pinch(canvas, VIEW_W / 2, VIEW_H / 2, 240, 60); // -2 级
		});
		const zoom1 = Number(screen.getByTestId('zoom').textContent);
		const center1 = screen.getByTestId('center').textContent || '';

		expect(zoom1).toBe(zoom0);
		const [lat0, lng0] = center0.split(',').map(Number);
		const [lat1, lng1] = center1.split(',').map(Number);
		expect(Math.abs(lat1 - lat0)).toBeLessThan(TOL_DEG);
		expect(Math.abs(lng1 - lng0)).toBeLessThan(TOL_DEG);
	});

	it('偏心锚点：放大使中心位移让位，缩小后回到原位', async () => {
		const startCenter = { lat: 34.26, lng: 117.19 };
		const { container } = render(PinchHost, { props: { center: startCenter, zoom: 14 } });
		const canvas = container.querySelector('[data-testid="canvas-host"] > div') as HTMLElement;
		const cx = VIEW_W * 0.25;
		const cy = VIEW_H * 0.75;

		await act(async () => {
			await pinch(canvas, cx, cy, 60, 120); // +1 级
		});
		expect(Number(screen.getByTestId('zoom').textContent)).toBe(15);
		const mid = (screen.getByTestId('center').textContent || '').split(',').map(Number);
		// 偏心放大必须让中心位移（锚点失效时中心会原地不动）
		expect(Math.abs(mid[0] - startCenter.lat) + Math.abs(mid[1] - startCenter.lng)).toBeGreaterThan(1e-6);

		await act(async () => {
			await pinch(canvas, cx, cy, 120, 60); // -1 级
		});
		expect(Number(screen.getByTestId('zoom').textContent)).toBe(14);
		const back = (screen.getByTestId('center').textContent || '').split(',').map(Number);
		expect(Math.abs(back[0] - startCenter.lat)).toBeLessThan(TOL_DEG);
		expect(Math.abs(back[1] - startCenter.lng)).toBeLessThan(TOL_DEG);
	});
});
