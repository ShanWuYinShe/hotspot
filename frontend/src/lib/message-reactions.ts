import { setReaction } from '@/api';
import type { Reaction } from '@/api/types';
import { showToast } from '@/stores/toast.svelte';

// 消息表情反应（W05）：spec api_spec_reaction.md 支持 target_type='message'，
// 消息项本身携带 reactions: Reaction[]；WS 反应事件只带 target/emoji/actor，
// 不带计数，因此客户端按增量合并（与服务端最终值一致）。

/**
 * 反应白名单兜底：spec AppConfig.limits.reaction_emojis 可配置，默认值与
 * 后端 seed/默认配置一致；调用方可通过参数注入服务端下发的白名单。
 */
export const DEFAULT_REACTION_EMOJIS: string[] = ['👍', '❤️', '😂', '🎉', '👏', '🔥'];

/** 合并一条他人/实时反应事件到反应列表（active=true 新增/计数 +1，false 移除/计数 -1）。 */
export function applyReactionDelta(
	list: Reaction[] | undefined,
	emoji: string,
	active: boolean,
	mine = false
): Reaction[] {
	const current = list ?? [];
	if (!emoji) return current;
	if (active) {
		const existing = current.find((r) => r.emoji === emoji);
		if (existing) {
			return current.map((r) =>
				r.emoji === emoji ? { ...r, count: r.count + 1, reacted_by_me: r.reacted_by_me || mine } : r
			);
		}
		return [...current, { emoji, count: 1, reacted_by_me: mine }];
	}
	return current
		.map((r) =>
			r.emoji === emoji
				? { ...r, count: Math.max(0, r.count - 1), reacted_by_me: mine ? false : r.reacted_by_me }
				: r
		)
		.filter((r) => r.count > 0);
}

/**
 * 我自己切换反应（服务端语义：同一目标只会保留一个 emoji）。
 * active=false 移除我的反应；active=true 先清掉我此前的 emoji 再加新的。
 */
export function applyMyReaction(list: Reaction[] | undefined, emoji: string, active: boolean): Reaction[] {
	const current = list ?? [];
	const withoutMine = current
		.map((r) => (r.reacted_by_me ? { ...r, count: r.count - 1, reacted_by_me: false } : r))
		.filter((r) => r.count > 0);
	if (!active || !emoji) return withoutMine;
	const existing = withoutMine.find((r) => r.emoji === emoji);
	if (existing) {
		return withoutMine.map((r) => (r.emoji === emoji ? { ...r, count: r.count + 1, reacted_by_me: true } : r));
	}
	return [...withoutMine, { emoji, count: 1, reacted_by_me: true }];
}

/** 当前用户对该消息是否已用该 emoji 反应。 */
export function hasMyReaction(list: Reaction[] | undefined, emoji: string): boolean {
	return (list ?? []).some((r) => r.emoji === emoji && r.reacted_by_me);
}

/** 设置消息反应；返回是否成功（调用方据此回滚乐观状态）。 */
export async function setMessageReaction(messageId: string, emoji: string, active: boolean): Promise<boolean> {
	try {
		const res = await setReaction({ target_type: 'message', target_id: messageId, emoji, active });
		if (res.code !== 0) throw new Error(res.message || '操作失败');
		return true;
	} catch (err) {
		showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		return false;
	}
}
