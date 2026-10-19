import { describe, it, expect } from 'vitest';
import { resolveHomeSegment, resolveFollowingEmpty } from '../home-feed-mode';

// 锁定首页双段的冷启动矩阵：默认段与关注段空态文案。
// 这两条都依赖 following_count —— 接口在「零关系」与「有关注但没人更新」两种情况下
// 都返回空数组，文案错了会直接误导用户（很多产品在这里说错话）。

describe('resolveHomeSegment（默认段）', () => {
	it('未登录默认「发现」（关注流无匿名版）', () => {
		expect(resolveHomeSegment(false, 0)).toBe('discover');
		expect(resolveHomeSegment(false, 12)).toBe('discover');
	});

	it('已登录且有关注默认「关注」', () => {
		expect(resolveHomeSegment(true, 1)).toBe('following');
		expect(resolveHomeSegment(true, 37)).toBe('following');
	});

	it('已登录但零社交关系默认「发现」（保证首屏有内容）', () => {
		expect(resolveHomeSegment(true, 0)).toBe('discover');
	});

	it('关注计数缺失时不误判为「有关注」', () => {
		expect(resolveHomeSegment(true, null)).toBe('discover');
		expect(resolveHomeSegment(true, undefined)).toBe('discover');
	});
});

describe('resolveFollowingEmpty（关注段空态）', () => {
	it('零关系 → 引导去关注，而不是说「关注的人没发动态」', () => {
		const s = resolveFollowingEmpty(0);
		expect(s.kind).toBe('no-relations');
		expect(s.title).toContain('还没有关注');
		expect(s.actionLabel).toBe('去附近看看');
	});

	it('有关注但无新内容 → 空态文案换一套，动作指向发现', () => {
		const s = resolveFollowingEmpty(5);
		expect(s.kind).toBe('no-activity');
		expect(s.title).toContain('还没发新动态');
		expect(s.actionLabel).toBe('去发现看看');
	});

	it('计数缺失按零关系处理（保守：宁可引导关注）', () => {
		expect(resolveFollowingEmpty(null).kind).toBe('no-relations');
		expect(resolveFollowingEmpty(undefined).kind).toBe('no-relations');
	});
});
