// 发现态失败聚合决策单测（2026-10-31 R2：分区请求失败不得静默转空态）。
import { describe, it, expect } from 'vitest';
import { resolveDiscoveryErrorView } from './search-discovery-error';

const none = { moments: false, users: false, groups: false };
const one = { moments: true, users: false, groups: false };
const two = { moments: true, users: true, groups: false };
const all = { moments: true, users: true, groups: true };

describe('resolveDiscoveryErrorView', () => {
	it('全部成功 → 不渲染错误 UI', () => {
		expect(resolveDiscoveryErrorView(none, false)).toBe('none');
		expect(resolveDiscoveryErrorView(none, true)).toBe('none');
	});

	it('部分失败 → 内联错误条（保留已成功区块）', () => {
		expect(resolveDiscoveryErrorView(one, false)).toBe('inline');
		expect(resolveDiscoveryErrorView(two, true)).toBe('inline');
	});

	it('全部失败但有最近搜索 → 内联错误条，不整页报错', () => {
		expect(resolveDiscoveryErrorView(all, true)).toBe('inline');
	});

	it('全部失败且无任何本地内容 → 整块 LoadError', () => {
		expect(resolveDiscoveryErrorView(all, false)).toBe('full');
	});
});
