// Feed 交互状态工厂（迁移自 React MomentsFeedPage.tsx 内 useFeedInteractions）。
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

export function createFeedInteractions(initial: FeedItem[]): FeedInteractions {
	let items = $state<FeedItem[]>(initial);

	// 外部数据更新时同步（原 useEffect setItems(initial)）
	$effect(() => {
		items = initial;
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
