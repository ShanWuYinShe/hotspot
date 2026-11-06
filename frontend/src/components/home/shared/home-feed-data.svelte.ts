import { onMount } from 'svelte';
import { page } from '$app/state';
import { auth } from '@/stores/auth.svelte';
import { homeFeed, type HomeFeedTab } from '@/stores/home-feed.svelte';
import { createCursorPagination } from '@/lib/composables/cursor-pager.svelte';
import { searchMoments, fetchFollowingMoments } from '@/api';
import type { MomentItem } from '@/api/types';
import { formatDistance } from '@/lib/utils';

export const PAGE_SIZE = 20;

export interface ActiveNeighbor {
	userId: string;
	nickname: string;
	avatarUrl: string | null;
	location?: string;
}

export interface RailNearbyMoment {
	id: string;
	image: string;
	title: string;
	meta: string;
}

export interface TrendingTopic {
	tag: string;
	count: number;
	desc: string;
}

export const TRENDING_TOPICS: TrendingTopic[] = [
	{ tag: '周末去哪儿', count: 32, desc: '周末好去处灵感分享' },
	{ tag: '老街咖啡香', count: 24, desc: '寻访城市角落的特调' },
	{ tag: '云龙湖夜跑', count: 18, desc: '一起出发，用脚步丈量夜色' },
	{ tag: '徐州历史漫步', count: 15, desc: '古城砖瓦间的岁月温度' },
	{ tag: '宝藏小店', count: 11, desc: '街头巷尾的惊喜发现' }
];

const paged = (res: { code: number; message?: string; data?: { items?: MomentItem[]; next_cursor?: string | null } | null }) => {
	if (res.code !== 0) throw new Error(res.message || '加载失败');
	return { items: res.data?.items ?? [], nextCursor: res.data?.next_cursor ?? null };
};

export function useHomeFeedData() {
	let booting = $state(true);

	onMount(() => {
		homeFeed.reset();
	});

	const discover = createCursorPagination<MomentItem>({
		fetchPage: async (cursor) =>
			paged(await searchMoments({ sort_by: 'popular', pagination: { limit: PAGE_SIZE, ...(cursor ? { cursor } : {}) } })),
		getKey: (m) => m.moment_id
	});

	const following = createCursorPagination<MomentItem>({
		fetchPage: async (cursor) =>
			paged(await fetchFollowingMoments({ pagination: { limit: PAGE_SIZE, ...(cursor ? { cursor } : {}) } })),
		getKey: (m) => m.moment_id
	});

	// 发现流：全局预加载（带 !error 守卫，失败不自动重试，由 LoadError 重试接管）
	$effect(() => {
		if (!discover.loaded && !discover.loading && !discover.error) {
			void discover.load().finally(() => {
				booting = false;
			});
		}
	});

	// 关注流：登录用户加载
	$effect(() => {
		if (auth.status === 'authenticated') {
			if (!following.loaded && !following.loading && !following.error) {
				void following.load();
			}
		}
	});

	const activeTab = $derived(homeFeed.activeTab);
	const currentPager = $derived(activeTab === 'discover' ? discover : following);
	const currentItems = $derived(currentPager.items);
	const currentLoading = $derived(booting || (currentItems.length === 0 && currentPager.loading));
	const currentError = $derived(currentPager.error);
	const hasMore = $derived(currentPager.hasMore);
	const loadingMore = $derived(currentPager.loadingMore);
	const demoMode = $derived(page.url.searchParams.get('demo'));

	// 活跃邻居（至多 3 位）
	const activeNeighbors = $derived.by(() => {
		const seen = new Set<string>();
		const list: ActiveNeighbor[] = [];
		for (const m of discover.items) {
			if (m.user?.user_id && !seen.has(m.user.user_id) && m.user.user_id !== auth.user?.user_id) {
				seen.add(m.user.user_id);
				list.push({
					userId: m.user.user_id,
					nickname: m.user.nickname,
					avatarUrl: m.user.avatar_url,
					location: m.location?.location_name ?? undefined
				});
				if (list.length >= 3) break;
			}
		}
		return list;
	});

	// 右侧 Rail 附近活动
	const railNearbyMoments = $derived.by(() =>
		discover.items
			.filter((m) => m.moment_id && m.location?.location_name)
			.slice(0, 3)
			.map((m) => ({
				id: m.moment_id,
				image: m.images?.[0]?.url ?? '',
				title: m.description?.trim() || '无标题瞬间',
				meta: [m.location?.location_name, formatDistance(m.location?.distance)].filter(Boolean).join(' · ')
			}))
	);

	return {
		get activeTab() { return activeTab; },
		get currentPager() { return currentPager; },
		get currentItems() { return currentItems; },
		get currentLoading() { return currentLoading; },
		get currentError() { return currentError; },
		get hasMore() { return hasMore; },
		get loadingMore() { return loadingMore; },
		get demoMode() { return demoMode; },
		get activeNeighbors() { return activeNeighbors; },
		get railNearbyMoments() { return railNearbyMoments; },
		trendingTopics: TRENDING_TOPICS,
		setTab(tab: HomeFeedTab) {
			homeFeed.setTab(tab);
		},
		loadMore() {
			return currentPager.loadMore();
		},
		reload() {
			return currentPager.load();
		}
	};
}

export type HomeFeedData = ReturnType<typeof useHomeFeedData>;
