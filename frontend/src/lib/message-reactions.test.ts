import { describe, expect, it } from 'vitest';
import { DEFAULT_REACTION_EMOJIS, applyMyReaction, applyReactionDelta, hasMyReaction, resolveReactionEmojis } from './message-reactions';
import type { Reaction } from '@/api/types';

const base: Reaction[] = [
	{ emoji: '👍', count: 2, reacted_by_me: false },
	{ emoji: '❤️', count: 1, reacted_by_me: true }
];

describe('message-reactions', () => {
	it('他人新增已有 emoji：计数 +1，不改 reacted_by_me', () => {
		const next = applyReactionDelta(base, '👍', true, false);
		expect(next.find((r) => r.emoji === '👍')).toEqual({ emoji: '👍', count: 3, reacted_by_me: false });
	});

	it('新增新 emoji：追加计数 1；mine 时为已反应', () => {
		const next = applyReactionDelta(base, '🔥', true, true);
		expect(next.find((r) => r.emoji === '🔥')).toEqual({ emoji: '🔥', count: 1, reacted_by_me: true });
	});

	it('移除后计数归零的 emoji 被删除', () => {
		const next = applyReactionDelta(base, '❤️', false, true);
		expect(next.some((r) => r.emoji === '❤️')).toBe(false);
	});

	it('我的切换：旧 emoji 计数 -1，新 emoji 置为已反应', () => {
		const next = applyMyReaction(base, '🎉', true);
		// 移除我唯一的反应后，该 emoji 计数归零被过滤
		expect(next.find((r) => r.emoji === '❤️')).toBeUndefined();
		expect(next.find((r) => r.emoji === '🎉')).toEqual({ emoji: '🎉', count: 1, reacted_by_me: true });
	});

	it('取消我的反应：只减自身，不误删他人计数', () => {
		const next = applyMyReaction(base, '❤️', false);
		expect(next).toEqual([{ emoji: '👍', count: 2, reacted_by_me: false }]);
	});

	it('hasMyReaction 判定', () => {
		expect(hasMyReaction(base, '❤️')).toBe(true);
		expect(hasMyReaction(base, '👍')).toBe(false);
	});
});

describe('resolveReactionEmojis', () => {
	it('未注入且无 bootstrap 缓存时回退到服务端默认白名单', () => {
		expect(resolveReactionEmojis()).toEqual(DEFAULT_REACTION_EMOJIS);
	});

	it('优先使用服务端下发的白名单', () => {
		expect(resolveReactionEmojis(['👍'])).toEqual(['👍']);
	});

	it('下发了空数组时仍回退默认值（避免渲染空面板）', () => {
		expect(resolveReactionEmojis([])).toEqual(DEFAULT_REACTION_EMOJIS);
	});

	it('默认白名单与后端默认配置逐项一致', () => {
		expect(DEFAULT_REACTION_EMOJIS).toEqual(['👍', '❤️', '😂', '😮', '😢', '🙏']);
	});
});
