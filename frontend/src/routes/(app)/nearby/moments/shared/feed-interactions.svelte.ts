// Feed 交互状态工厂。
// 持有每条 FeedItem 的 liked/saved 交互状态，乐观更新 + 失败回滚。

import { setReaction, favoriteMoment } from '@/api';
import { requireAuth } from '@/stores/auth.svelte';
import { showToast } from '@/stores/toast.svelte';

export interface FeedItem {
	id: string;
	user: string;
	avatar: string;
	location: string;
	tag: string;
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
	// 注意：同步是整体覆盖，会重置 toggleLike/toggleSave 的乐观状态。
	$effect(() => {
		items = getInitial();
	});

	// in-flight 去重：同一目标的写请求未完成时忽略重复点击，
	// 避免连点导致乐观状态双翻转 + 并发请求乱序（UI 与后端状态漂移）。
	const likePending = new Set<string>();
	const savePending = new Set<string>();

	// 写操作先过登录闸（requireAuth 弹登录提示并在登录后重放动作）；
	// 业务失败（HTTP 200 + code≠0，api client 不抛错）必须显式判 code 回滚，
	// 否则 UI 假成功、刷新后打回原形（对齐 nearby/groups 与 harmonyapp 范式）。
	const toggleLike = async (id: FeedItem['id']) => {
		if (likePending.has(id)) return;
		if (!requireAuth({ action: '点赞', targetId: id, run: () => void toggleLike(id) })) return;
		items = items.map((it) => (it.id === id ? { ...it, liked: !it.liked } : it));
		const target = items.find((it) => it.id === id);
		if (target) {
			const prevLiked = target.liked;
			likePending.add(id);
			try {
				const res = await setReaction({ target_type: 'moment', target_id: id, emoji: target.likeEmoji, active: !prevLiked });
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
