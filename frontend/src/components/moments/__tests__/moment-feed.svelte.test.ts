// toggleLike/toggleSave 契约测试：请求 active 必须是「点击前状态取反」，
// 失败回滚必须回到点击前状态（曾因从乐观翻转后的数组读旧值导致语义整体
// 反转——点赞实际下发 active:false 永不落库，回滚空操作）。
import { describe, it, expect, vi, beforeEach } from 'vitest';

const setReaction = vi.hoisted(() => vi.fn());
const favoriteMoment = vi.hoisted(() => vi.fn());
const requireAuthMock = vi.hoisted(() => vi.fn(() => true));
const showToastMock = vi.hoisted(() => vi.fn());

vi.mock('@/api', () => ({ setReaction, favoriteMoment }));
vi.mock('@/stores/auth.svelte', () => ({ requireAuth: requireAuthMock }));
vi.mock('@/stores/toast.svelte', () => ({ showToast: showToastMock }));

import { createFeedInteractions, type FeedItem } from '../moment-feed.svelte';

// createFeedInteractions 内部使用 $effect 同步外部数据源，必须在 effect root
// 内创建（runes 的模块级约束）；数据源用 $state 承载，模拟真实页面「getInitial
// 读取响应式 store」的用法——effect 仅在数据源变化时重同步，不会覆盖本地乐观更新。
function makeFeed(initial: FeedItem[]) {
	let source = $state<FeedItem[]>(initial);
	let feed: ReturnType<typeof createFeedInteractions>;
	$effect.root(() => {
		feed = createFeedInteractions(() => source);
	});
	return feed!;
}

function item(overrides: Partial<FeedItem> = {}): FeedItem {
	return {
		id: 'm1',
		user: 'u1',
		avatar: '',
		location: '',
		tag: '',
		tags: [],
		content: 'hello',
		image: '',
		likes: 0,
		comments: 0,
		time: '',
		liked: false,
		saved: false,
		likeEmoji: '👍',
		...overrides
	};
}

describe('createFeedInteractions 点赞/收藏契约', () => {
	beforeEach(() => {
		vi.clearAllMocks();
	});

	it('给未点赞的动态点赞：下发 active:true 且本地翻转为已赞', async () => {
		setReaction.mockResolvedValue({ code: 0 });
		const feed = makeFeed([item({ liked: false })]);
		await feed.toggleLike('m1');
		expect(setReaction).toHaveBeenCalledWith(expect.objectContaining({ target_id: 'm1', active: true }));
		expect(feed.items[0].liked).toBe(true);
	});

	it('取消点赞：下发 active:false 且本地翻转为未赞', async () => {
		setReaction.mockResolvedValue({ code: 0 });
		const feed = makeFeed([item({ liked: true })]);
		await feed.toggleLike('m1');
		expect(setReaction).toHaveBeenCalledWith(expect.objectContaining({ target_id: 'm1', active: false }));
		expect(feed.items[0].liked).toBe(false);
	});

	it('点赞失败：回滚到点击前的未赞状态', async () => {
		setReaction.mockResolvedValue({ code: 1, message: '失败' });
		const feed = makeFeed([item({ liked: false })]);
		await feed.toggleLike('m1');
		expect(feed.items[0].liked).toBe(false);
		expect(showToastMock).toHaveBeenCalled();
	});

	it('收藏失败：回滚到点击前的未收藏状态', async () => {
		favoriteMoment.mockResolvedValue({ code: 1, message: '失败' });
		const feed = makeFeed([item({ saved: false })]);
		await feed.toggleSave('m1');
		expect(favoriteMoment).toHaveBeenCalledWith(expect.objectContaining({ moment_id: 'm1', active: true }));
		expect(feed.items[0].saved).toBe(false);
	});

	it('数据源产出新数组时重新同步（外部刷新仍生效）', async () => {
		setReaction.mockResolvedValue({ code: 0 });
		let source = $state<FeedItem[]>([item({ liked: false })]);
		let feed!: ReturnType<typeof createFeedInteractions>;
		$effect.root(() => {
			feed = createFeedInteractions(() => source);
		});
		await feed.toggleLike('m1');
		expect(feed.items[0].liked).toBe(true); // 乐观更新不被打回
		source = [item({ liked: false, likes: 7 })]; // 外部刷新：新数组身份
		await new Promise((r) => setTimeout(r, 0));
		expect(feed.items[0].likes).toBe(7); // 外部数据生效
	});
});
