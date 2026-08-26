// Feed 交互状态工厂。
// 持有每条 FeedItem 的 liked/saved 交互状态，乐观更新 + API 失败回滚。

import { setReaction, favoriteMoment } from '@/api';
import { requireAuth } from '@/stores/auth.svelte';

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

	// 写操作先过登录闸（requireAuth 弹登录提示并在登录后重放动作）；
	// 匿名直接发请求只会 10100 失败回滚，用户侧看不到任何反馈。
	const toggleLike = async (id: FeedItem['id']) => {
		if (!requireAuth({ action: '点赞', targetId: id, run: () => void toggleLike(id) })) return;
		items = items.map((it) => (it.id === id ? { ...it, liked: !it.liked } : it));
		const target = items.find((it) => it.id === id);
		if (target) {
			try {
				await setReaction({ target_type: 'moment', target_id: id, emoji: target.likeEmoji, active: !target.liked });
			} catch {
				items = items.map((it) => (it.id === id ? { ...it, liked: target.liked } : it));
			}
		}
	};

	const toggleSave = async (id: FeedItem['id']) => {
		if (!requireAuth({ action: '收藏', targetId: id, run: () => void toggleSave(id) })) return;
		items = items.map((it) => (it.id === id ? { ...it, saved: !it.saved } : it));
		const target = items.find((it) => it.id === id);
		if (target) {
			try {
				await favoriteMoment({ moment_id: id, active: !target.saved });
			} catch {
				items = items.map((it) => (it.id === id ? { ...it, saved: target.saved } : it));
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
