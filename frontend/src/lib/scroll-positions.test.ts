import { describe, expect, it } from 'vitest';
import { decideScroll, savePosition } from './scroll-positions';

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
