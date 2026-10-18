// scrollFade action 契约测试：只在真正溢出的一侧加渐隐类，到头后翻转。
import { describe, it, expect } from 'vitest';
import { scrollFade } from '../scroll-fade';

function makeEl(sw: number, cw: number, sl = 0) {
	const el = document.createElement('div');
	Object.defineProperty(el, 'scrollWidth', { value: sw, configurable: true });
	Object.defineProperty(el, 'clientWidth', { value: cw, configurable: true });
	Object.defineProperty(el, 'scrollLeft', { value: sl, writable: true, configurable: true });
	return el;
}

describe('scrollFade', () => {
	it('溢出时右侧加渐隐；滚到末尾后翻转为左侧', () => {
		const el = makeEl(682, 354);
		const action = scrollFade(el);
		expect(el.classList.contains('scroll-fade-right')).toBe(true);
		expect(el.classList.contains('scroll-fade-left')).toBe(false);

		el.scrollLeft = 328;
		el.dispatchEvent(new Event('scroll'));
		expect(el.classList.contains('scroll-fade-right')).toBe(false);
		expect(el.classList.contains('scroll-fade-left')).toBe(true);
		action.destroy();
	});

	it('不溢出时两侧都不加（避免「到头了还渐隐」的假提示）', () => {
		const el = makeEl(300, 300);
		const action = scrollFade(el);
		expect(el.classList.contains('scroll-fade-right')).toBe(false);
		expect(el.classList.contains('scroll-fade-left')).toBe(false);
		action.destroy();
	});

	it('destroy 后不再响应 scroll（避免组件销毁后写类）', () => {
		const el = makeEl(682, 354);
		const action = scrollFade(el);
		action.destroy();
		el.scrollLeft = 328;
		el.dispatchEvent(new Event('scroll'));
		expect(el.classList.contains('scroll-fade-left')).toBe(false);
	});
});
