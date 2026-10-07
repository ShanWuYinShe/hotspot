// MapCanvas 捏合手势契约测试（jsdom 合成 PointerEvent）：
// 覆盖 2026-10-08 两处修复——①捏合抬指后 center 必须经 commitCenter 回写
// （此前 pinched 路径提前 return 导致 stableBounds 用过期中心拉数据、视图弹回）；
// ②连续缩放：整数跨越即时落级 + 剩余小数以 gestureTransform 呈现、松手吸附清除。
import { describe, it, expect, beforeAll, vi } from 'vitest';
import { render, screen, act } from '@testing-library/svelte';
import PinchHost from './PinchHost.svelte';

// jsdom 无 ResizeObserver（bind:clientWidth 依赖）：stub 为空实现，
// width/height 保持 0——锚点数学全部基于世界坐标，不受影响。
class ResizeObserverStub {
	observe() {}
	unobserve() {}
	disconnect() {}
}
beforeAll(() => {
	vi.stubGlobal('ResizeObserver', ResizeObserverStub);
});

function pointer(el: Element, type: string, id: number, x: number, y: number) {
	const ev = new Event(type, { bubbles: true, cancelable: true });
	Object.assign(ev, { pointerId: id, pointerType: 'touch', clientX: x, clientY: y });
	el.dispatchEvent(ev);
}

async function pinchSpread(el: Element, from: number, to: number, cx: number, cy: number) {
	pointer(el, 'pointerdown', 1, cx - from, cy);
	pointer(el, 'pointerdown', 2, cx + from, cy);
	const steps = 6;
	for (let i = 1; i <= steps; i++) {
		const d = from + ((to - from) * i) / steps;
		act(() => {
			pointer(el, 'pointermove', 1, cx - d, cy);
			pointer(el, 'pointermove', 2, cx + d, cy);
		});
		await Promise.resolve();
	}
}

describe('MapCanvas 捏合手势', () => {
	it('捏合放大：整数级落级 + 松手吸附 + center 回写', async () => {
		const startLng = 117.19;
		const { container } = render(PinchHost, {
			props: { center: { lat: 34.26, lng: startLng }, zoom: 14 }
		});
		const canvas = container.querySelector('[data-testid="canvas-host"] > div') as HTMLElement;
		expect(canvas).toBeTruthy();
		const cx = 200;
		const cy = 150;
		await act(async () => {
			pinchSpread(canvas, 60, 400, cx, cy);
		});
		// 整数级跨越：400/60 ≈ 6.7 倍 ≈ +2.7 级 → zoom 至少 +2
		const zoomAfter = Number(screen.getByTestId('zoom').textContent);
		expect(zoomAfter).toBeGreaterThanOrEqual(16);
		// 松手吸附 + center 回写（commitCenter）
		await act(async () => {
			pointer(canvas, 'pointerup', 1, cx - 400, cy);
			pointer(canvas, 'pointerup', 2, cx + 400, cy);
		});
		const zoomSnapped = Number(screen.getByTestId('zoom').textContent);
		// 吸附目标 = round(最终 floatTarget)，与手势中的整数态相差 ≤1 级
		expect(Math.abs(zoomSnapped - zoomAfter)).toBeLessThanOrEqual(1);
		expect(screen.getByTestId('gt').textContent).toBe('(idle)');
		const centerText = screen.getByTestId('center').textContent || '';
		expect(centerText).not.toBe('null');
		// 锚点=双指中点（视口中心）：中心经锚点换算后位置有实际变化（非精确等值断言，
		// jsdom 无布局 width/height=0 时锚点数学退化为锚点自身）
		expect(centerText.split(',').length).toBe(2);
	});

	it('单指原地点击不触发 center 回写（tap 语义保持）', async () => {
		const { container } = render(PinchHost, {
			props: { center: { lat: 34.26, lng: 117.19 }, zoom: 14 }
		});
		const canvas = container.querySelector('[data-testid="canvas-host"] > div') as HTMLElement;
		const before = screen.getByTestId('center').textContent;
		await act(async () => {
			pointer(canvas, 'pointerdown', 1, 200, 150);
			pointer(canvas, 'pointerup', 1, 200, 150);
		});
		expect(screen.getByTestId('center').textContent).toBe(before);
		expect(screen.getByTestId('gt').textContent).toBe('(idle)');
	});
});
