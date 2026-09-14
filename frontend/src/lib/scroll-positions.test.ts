import { describe, expect, it } from 'vitest';
import { decideScroll, restoreScrollTop, savePosition, type ScrollableElement } from './scroll-positions';

// 路由滚动决策契约：离开存位、返回恢复、进新页置顶、同路由保持。
describe('savePosition', () => {
	it('按 pathname 存位并刷新 recency', () => {
		const store = new Map<string, number>();
		savePosition(store, '/a', 100);
		savePosition(store, '/b', 200);
		savePosition(store, '/a', 150);
		expect(store.get('/a')).toBe(150);
		expect([...store.keys()]).toEqual(['/b', '/a']);
	});

	it('超 30 条淘汰最旧', () => {
		const store = new Map<string, number>();
		for (let i = 0; i < 35; i++) savePosition(store, `/p${i}`, i);
		expect(store.size).toBe(30);
		expect(store.has('/p0')).toBe(false);
		expect(store.has('/p34')).toBe(true);
	});
});

describe('decideScroll', () => {
	it('popstate 有存位则恢复', () => {
		const store = new Map([['/nearby/moments', 1234]]);
		expect(decideScroll('popstate', '/moments/x', '/nearby/moments', store)).toEqual({
			action: 'restore',
			top: 1234
		});
	});

	it('popstate 无存位则置顶', () => {
		expect(decideScroll('popstate', '/moments/x', '/nearby/moments', new Map())).toEqual({
			action: 'restore',
			top: 0
		});
	});

	it('跨 pathname 新进置顶', () => {
		expect(decideScroll('link', '/nearby/moments', '/moments/x', new Map())).toEqual({ action: 'top' });
	});

	it('同路由内导航（如 ?tab= 切换）保持不动', () => {
		expect(decideScroll('link', '/nearby/moments', '/nearby/moments', new Map())).toEqual({ action: 'keep' });
	});
});

// restoreScrollTop 契约：追到为止（布局异步长高时单次赋值会被钳位）。
describe('restoreScrollTop', () => {
	function makeEl(height: number, clientHeight = 700): ScrollableElement & { setHeight(h: number): void } {
		let top = 0;
		let currentHeight = height;
		return {
			get scrollHeight() {
				return currentHeight;
			},
			clientHeight,
			style: { scrollBehavior: 'smooth' },
			offsetHeight: 100,
			get scrollTop() {
				return top;
			},
			set scrollTop(v: number) {
				top = Math.min(Math.max(0, v), Math.max(0, currentHeight - clientHeight));
			},
			setHeight(h: number) {
				currentHeight = h;
			}
		};
	}

	it('高度足够时一次到位并恢复原 behavior', () => {
		const el = makeEl(3000);
		const queue: (() => void)[] = [];
		restoreScrollTop(el, 1500, (cb) => queue.push(cb));
		while (queue.length) queue.shift()!();
		expect(el.scrollTop).toBe(1500);
		expect(el.style.scrollBehavior).toBe('smooth');
	});

	it('布局后长高时追到目标（模拟瀑布流异步布局）', () => {
		const el = makeEl(800);
		const queue: (() => void)[] = [];
		restoreScrollTop(el, 1500, (cb) => queue.push(cb));
		queue.shift()!();
		expect(el.scrollTop).toBeLessThan(1500);
		el.setHeight(3000);
		while (queue.length) queue.shift()!();
		expect(el.scrollTop).toBe(1500);
		expect(el.style.scrollBehavior).toBe('smooth');
	});

	it('高度永远不够时收敛停下（不无限追）', () => {
		const el = makeEl(500);
		const queue: (() => void)[] = [];
		restoreScrollTop(el, 1500, (cb) => queue.push(cb));
		let steps = 0;
		while (queue.length && steps < 100) {
			queue.shift()!();
			steps++;
		}
		expect(queue.length).toBe(0);
		expect(el.style.scrollBehavior).toBe('smooth');
	});

	it('top<=0 直接置顶', () => {
		const el = makeEl(3000);
		el.scrollTop = 500;
		const queue: (() => void)[] = [];
		restoreScrollTop(el, 0, (cb) => queue.push(cb));
		expect(el.scrollTop).toBe(0);
		expect(queue.length).toBe(0);
	});

	it('空元素直接返回', () => {
		expect(() => restoreScrollTop(null, 100)).not.toThrow();
	});
});
