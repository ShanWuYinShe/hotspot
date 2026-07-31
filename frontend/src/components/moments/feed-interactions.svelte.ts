// Feed 交互状态工厂（迁移自旧版 MomentsFeedPage 内 useFeedInteractions）。
// 持有每条 FeedItem 的 liked/saved 交互状态，乐观更新 + API 失败回滚。

import { setReaction, favoriteMoment } from '@/api';

export interface FeedItem {
	id: number | string;
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

	// 外部数据更新时同步（原 useEffect setItems(initial)）——通过 getter 保持对数据源的响应。
	// 注意：同步是整体覆盖，会重置 toggleLike/toggleSave 的乐观状态（与旧版语义一致）。
	$effect(() => {
		items = getInitial();
	});

	const toggleLike = async (id: FeedItem['id']) => {
		items = items.map((it) => (it.id === id ? { ...it, liked: !it.liked } : it));
		const target = items.find((it) => it.id === id);
		if (target) {
			try {
				await setReaction({ target_type: 'moment', target_id: String(id), emoji: target.likeEmoji, active: !target.liked });
			} catch {
				items = items.map((it) => (it.id === id ? { ...it, liked: target.liked } : it));
			}
		}
	};

	const toggleSave = async (id: FeedItem['id']) => {
		items = items.map((it) => (it.id === id ? { ...it, saved: !it.saved } : it));
		const target = items.find((it) => it.id === id);
		if (target) {
			try {
				await favoriteMoment({ moment_id: String(id), active: !target.saved });
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
