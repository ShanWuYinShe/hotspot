// 搜索页 ↑↓ 导航决策单测（2026-10-31 R1：修复「↓ 落到页头返回钮 / ↑ 跳到页底」）。
import { describe, it, expect } from 'vitest';
import { isSearchNavKey, nextSearchNavIndex } from './search-nav';

describe('isSearchNavKey', () => {
	it('仅接管上下方向键', () => {
		expect(isSearchNavKey('ArrowDown')).toBe(true);
		expect(isSearchNavKey('ArrowUp')).toBe(true);
		expect(isSearchNavKey('Enter')).toBe(false);
		expect(isSearchNavKey('Escape')).toBe(false);
		expect(isSearchNavKey('arrowup')).toBe(false);
	});
});

describe('nextSearchNavIndex', () => {
	it('候选集为空 → 不接管', () => {
		expect(nextSearchNavIndex(-1, 0, 'ArrowDown')).toBeNull();
		expect(nextSearchNavIndex(0, 0, 'ArrowUp')).toBeNull();
	});

	it('焦点在候选集外（搜索框）：↓ 进入首个候选', () => {
		expect(nextSearchNavIndex(-1, 5, 'ArrowDown')).toBe(0);
	});

	it('焦点在候选集外（搜索框）：↑ 不跨区跳转（原实现会落到页底）', () => {
		expect(nextSearchNavIndex(-1, 5, 'ArrowUp')).toBeNull();
	});

	it('候选集内逐项移动', () => {
		expect(nextSearchNavIndex(0, 5, 'ArrowDown')).toBe(1);
		expect(nextSearchNavIndex(2, 5, 'ArrowDown')).toBe(3);
		expect(nextSearchNavIndex(3, 5, 'ArrowUp')).toBe(2);
	});

	it('两端收敛，不循环', () => {
		expect(nextSearchNavIndex(0, 5, 'ArrowUp')).toBe(0);
		expect(nextSearchNavIndex(4, 5, 'ArrowDown')).toBe(4);
	});

	it('单候选时上下都停在 0', () => {
		expect(nextSearchNavIndex(0, 1, 'ArrowUp')).toBe(0);
		expect(nextSearchNavIndex(0, 1, 'ArrowDown')).toBe(0);
		expect(nextSearchNavIndex(-1, 1, 'ArrowDown')).toBe(0);
	});
});
