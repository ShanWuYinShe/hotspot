// 通用动态（Moment）数据契约与交互状态工厂。
// 支持 nearby/moments 与 me/moments 跨页面复用。

import { setReaction, favoriteMoment } from '@/api';
import { requireAuth } from '@/stores/auth.svelte';
import { showToast } from '@/stores/toast.svelte';

export interface FeedItem {
	id: string;
	/** 软删除墓碑：为 true 时调用方渲染占位而非内容卡（可选，其余列表默认正常卡）。 */
	is_deleted?: boolean;
	user: string;
	/** 作者 user_id（可选；提供时卡片作者区可点击进入主页，与首页信息流卡一致）。 */
	userId?: string;
	avatar: string;
	location: string;
	/** 我是否关注了作者（来自 spec MomentItem.author_relation；匿名/缺省为 false）。 */
	authorRelation?: boolean;
	/** 展示用首标签；筛选走全量 tags（首标签之外同样可命中） */
	tag: string;
	tags: string[];
	content: string;
	image: string;
	likes: number;
	comments: number;
	time: string;
	liked: boolean;
	saved: boolean;
	likeEmoji: string;
}

export interface FeedInteractions {
	items: FeedItem[];
	toggleLike: (id: FeedItem['id']) => Promise<void>;
	toggleSave: (id: FeedItem['id']) => Promise<void>;
}

export function createFeedInteractions(getInitial: () => FeedItem[]): FeedInteractions {
	let items = $state<FeedItem[]>(getInitial());
	let lastExternal = getInitial();

	// 外部数据更新时同步——通过 getter 保持对数据源的响应。
	// 必须用「数组身份」守卫：本 effect 在 items 被本地乐观写入后也会重跑
	// （Svelte 5 对「effect 内写过的 state」的外部写入会重新调度 effect），
	// 若无条件回写 getInitial()，乐观更新会被数据源旧值立即打回（点赞图标
	// 弹回的根因）。仅当数据源产出新数组（页面 store 刷新，替换式更新）时
	// 才重新同步。
	$effect(() => {
		const next = getInitial();
		if (next !== lastExternal) {
			lastExternal = next;
			items = next;
		}
	});

	// in-flight 去重：同一目标的写请求未完成时忽略重复点击
	const likePending = new Set<string>();
	const savePending = new Set<string>();

	const toggleLike = async (id: FeedItem['id']) => {
		if (likePending.has(id)) return;
		if (!requireAuth({ action: '点赞', targetId: id, run: () => void toggleLike(id) })) return;
		// 在乐观翻转前捕获原值：请求 active 取「点击前的相反值」，回滚也回到
		// 原值。此前从翻转后的数组读 target 再取反，语义整体反转——点赞实际
		// 下发 active:false（服务端 no-op，点赞永不落库），回滚也是空操作。
		const wasLiked = items.find((it) => it.id === id)?.liked ?? false;
		items = items.map((it) => (it.id === id ? { ...it, liked: !it.liked } : it));
		const target = items.find((it) => it.id === id);
		if (target) {
			likePending.add(id);
			try {
				const res = await setReaction({ target_type: 'moment', target_id: id, emoji: target.likeEmoji || '👍', active: !wasLiked });
				if (res.code !== 0) throw new Error(res.message || '操作失败');
			} catch (err) {
				items = items.map((it) => (it.id === id ? { ...it, liked: wasLiked } : it));
				showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
			} finally {
				likePending.delete(id);
			}
		}
	};

	const toggleSave = async (id: FeedItem['id']) => {
		if (savePending.has(id)) return;
		if (!requireAuth({ action: '收藏', targetId: id, run: () => void toggleSave(id) })) return;
		// 同 toggleLike：翻转前捕获原值，请求与回滚都基于它。
		const existing = items.find((it) => it.id === id);
		if (!existing) return;
		const wasSaved = existing.saved;
		items = items.map((it) => (it.id === id ? { ...it, saved: !it.saved } : it));
		{
			savePending.add(id);
			try {
				const res = await favoriteMoment({ moment_id: id, active: !wasSaved });
				if (res.code !== 0) throw new Error(res.message || '操作失败');
			} catch (err) {
				items = items.map((it) => (it.id === id ? { ...it, saved: wasSaved } : it));
				showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
			} finally {
				savePending.delete(id);
			}
		}
	};

	return {
		get items() {
			return items;
		},
		toggleLike,
		toggleSave
	};
}