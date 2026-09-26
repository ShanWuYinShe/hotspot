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
	avatar: string;
	location: string;
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

	// 外部数据更新时同步——通过 getter 保持对数据源的响应。
	$effect(() => {
		items = getInitial();
	});

	// in-flight 去重：同一目标的写请求未完成时忽略重复点击
	const likePending = new Set<string>();
	const savePending = new Set<string>();

	const toggleLike = async (id: FeedItem['id']) => {
		if (likePending.has(id)) return;
		if (!requireAuth({ action: '点赞', targetId: id, run: () => void toggleLike(id) })) return;
		items = items.map((it) => (it.id === id ? { ...it, liked: !it.liked } : it));
		const target = items.find((it) => it.id === id);
		if (target) {
			const prevLiked = target.liked;
			likePending.add(id);
			try {
				const res = await setReaction({ target_type: 'moment', target_id: id, emoji: target.likeEmoji || '👍', active: !prevLiked });
				if (res.code !== 0) throw new Error(res.message || '操作失败');
			} catch (err) {
				items = items.map((it) => (it.id === id ? { ...it, liked: prevLiked } : it));
				showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
			} finally {
				likePending.delete(id);
			}
		}
	};

	const toggleSave = async (id: FeedItem['id']) => {
		if (savePending.has(id)) return;
		if (!requireAuth({ action: '收藏', targetId: id, run: () => void toggleSave(id) })) return;
		items = items.map((it) => (it.id === id ? { ...it, saved: !it.saved } : it));
		const target = items.find((it) => it.id === id);
		if (target) {
			const prevSaved = target.saved;
			savePending.add(id);
			try {
				const res = await favoriteMoment({ moment_id: id, active: !prevSaved });
				if (res.code !== 0) throw new Error(res.message || '操作失败');
			} catch (err) {
				items = items.map((it) => (it.id === id ? { ...it, saved: prevSaved } : it));
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
