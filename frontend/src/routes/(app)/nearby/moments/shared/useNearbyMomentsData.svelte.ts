import { goto } from '$app/navigation';
import { platform } from '@/stores/platform.svelte';
import { startDefaultLocation } from '@/lib/start-default-location';
import { readLastLocation } from '@/lib/map-auto-locate';
import { createFeedInteractions, type FeedItem } from '@/components/moments/moment-feed.svelte';
import { fetchNearbyMoments, netError } from '@/api';
import { formatCardTime, formatDistance } from '@/lib/utils';
import { computeBspLayoutSequence } from '@/lib/semantic-bsp';
import { ROUTES } from '@/lib/routes';
import {
	getMomentsProgress,
	setMomentsProgress,
	feedProgressKey
} from '@/lib/moments-feed-progress.svelte';

export interface NearbyMomentsData {
	readonly apiFeedItems: FeedItem[];
	readonly tags: string[];
	readonly activeTag: string | null;
	readonly loading: boolean;
	readonly loadingMore: boolean;
	readonly loadError: string | null;
	readonly locationName: string;
	readonly visibleCount: number;
	readonly filtered: FeedItem[];
	readonly showListEnd: boolean;
	readonly feedHasMore: boolean;
	readonly bspLayouts: ReturnType<typeof computeBspLayoutSequence>;
	readonly interactions: ReturnType<typeof createFeedInteractions>;
	readonly pageSubtitle: string;
	readonly islandMetaText: string;
	setTag: (tag: string | null) => void;
	reload: () => void;
	handleLoadMore: () => Promise<void>;
	openMoment: (id: string, focusComments?: boolean) => void;
}

const PAGE_LIMIT = 20;
const INITIAL_VISIBLE = 12;

function mapMoment(m: import('@/api/types').MomentItem, i: number): FeedItem {
	const mine = (m.reactions || []).find((r) => r.reacted_by_me);
	const firstImg = m.images?.[0];
	const w = firstImg?.width ?? null;
	const h = firstImg?.height ?? null;
	return {
		id: m.moment_id ?? `local-${i + 1}`,
		user: m.user?.nickname ?? '',
		userId: m.user?.user_id,
		avatar: m.user?.avatar_url ?? '',
		location: [m.location?.location_name, formatDistance(m.location?.distance)].filter(Boolean).join(' · '),
		tag: m.tags?.[0] ?? '',
		tags: m.tags ?? [],
		content: m.description ?? '',
		image: firstImg?.url ?? '',
		imageWidth: w,
		imageHeight: h,
		aspectRatio: w && h && h > 0 ? w / h : null,
		likes: (m.reactions ?? []).reduce((sum, r) => sum + r.count, 0),
		comments: m.comments_count ?? 0,
		time: formatCardTime(m.created_at),
		authorRelation: m.author_relation?.is_following ?? false,
		liked: Boolean(mine),
		saved: Boolean(m.is_favorited),
		likeEmoji: mine?.emoji ?? '👍'
	};
}

export function useNearbyMomentsData(): NearbyMomentsData {
	let apiFeedItems = $state<FeedItem[]>([]);
	let tags = $state<string[]>([]);
	let loading = $state(true);
	let loadingMore = $state(false);
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);
	let nextCursor = $state('');
	let activeTag = $state<string | null>(null);
	let visibleCount = $state(INITIAL_VISIBLE);

	const savedAnchor = readLastLocation();
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(
		savedAnchor ? { latitude: savedAnchor.lat, longitude: savedAnchor.lng } : null
	);
	let locationName = $state(savedAnchor?.name ?? '');

	$effect(() => startDefaultLocation((loc) => {
		if (savedAnchor) return;
		referencePoint = loc?.point ?? null;
		locationName = loc?.location_name ?? '';
	}));

	$effect(() => {
		void reloadKey;
		const progressKey = feedProgressKey(referencePoint);
		const saved = getMomentsProgress();
		if (saved && progressKey !== '' && saved.key === progressKey) {
			apiFeedItems = saved.items;
			nextCursor = saved.nextCursor;
			visibleCount = saved.visibleCount;
			tags = saved.tags;
			loadError = null;
			loading = false;
			return;
		}
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				loadError = null;
				nextCursor = '';
				visibleCount = INITIAL_VISIBLE;
				const feedRes = await fetchNearbyMoments({
					...(referencePoint ? { reference_point: referencePoint } : {}),
					pagination: { limit: PAGE_LIMIT }
				}).catch(() => netError({ items: [] as import('@/api/types').MomentItem[], next_cursor: null }));
				if (cancelled) return;

				const feedData = feedRes.code === 0 ? feedRes.data : null;
				if (feedRes.code !== 0) {
					if (!cancelled) loadError = feedRes.message ?? '加载失败';
					return;
				}

				if (feedData) {
					apiFeedItems = (feedData.items || []).map(mapMoment);
					nextCursor = feedData.next_cursor ?? '';
					const seen = new Set<string>();
					const tagsList: string[] = [];
					for (const m of feedData.items || []) {
						for (const t of m.tags || []) {
							if (typeof t === 'string' && t && !seen.has(t)) {
								seen.add(t);
								tagsList.push(t);
							}
						}
					}
					tags = tagsList;
					setMomentsProgress({ savedAt: Date.now(), key: progressKey, items: apiFeedItems, nextCursor, visibleCount, tags });
				}
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	async function handleLoadMore() {
		if (loadingMore) return;
		if (nextCursor) {
			loadingMore = true;
			try {
				const res = await fetchNearbyMoments({
					...(referencePoint ? { reference_point: referencePoint } : {}),
					pagination: { limit: PAGE_LIMIT, cursor: nextCursor }
				});
				if (res.code === 0 && res.data) {
					const known = new Set(apiFeedItems.map((i) => i.id));
					const fresh = (res.data.items || []).map(mapMoment).filter((i) => !known.has(i.id));
					apiFeedItems = [...apiFeedItems, ...fresh];
					nextCursor = res.data.next_cursor ?? '';
					visibleCount = Math.min(visibleCount + 8, apiFeedItems.length);
					const saved = getMomentsProgress();
					if (saved) setMomentsProgress({ ...saved, items: apiFeedItems, nextCursor, visibleCount });
				}
			} catch {
				// 忽略重试
			} finally {
				loadingMore = false;
			}
			return;
		}
		visibleCount = Math.min(visibleCount + 8, apiFeedItems.length);
		const savedSlice = getMomentsProgress();
		if (savedSlice) setMomentsProgress({ ...savedSlice, visibleCount });
	}

	function openMoment(id: string, focusComments = false) {
		if (id.startsWith('local-')) return;
		void goto(focusComments ? `${ROUTES.moment(id)}?focus=comments` : ROUTES.moment(id));
	}

	const interactions = createFeedInteractions(() => apiFeedItems);

	const filtered = $derived.by(() => {
		return interactions.items.filter((i) => !activeTag || i.tags.includes(activeTag));
	});

	const showListEnd = $derived(!loadingMore && visibleCount >= filtered.length && !nextCursor);
	const feedHasMore = $derived(loadingMore || visibleCount < filtered.length || Boolean(nextCursor));
	const device = $derived(platform.platform);

	const bspLayouts = $derived(
		computeBspLayoutSequence(
			filtered.slice(0, visibleCount).map((item) => ({
				aspectRatio: item.aspectRatio,
				width: item.imageWidth,
				height: item.imageHeight,
				likes: item.likes,
				hasImage: Boolean(item.image)
			})),
			{ device }
		)
	);

	const pageSubtitle = $derived(locationName ? `${locationName} · 5km 内` : '5km 内 · 实时更新');
	const islandMetaText = $derived(activeTag ? `${activeTag} · ${pageSubtitle}` : pageSubtitle);

	return {
		get apiFeedItems() { return apiFeedItems; },
		get tags() { return tags; },
		get activeTag() { return activeTag; },
		get loading() { return loading; },
		get loadingMore() { return loadingMore; },
		get loadError() { return loadError; },
		get locationName() { return locationName; },
		get visibleCount() { return visibleCount; },
		get filtered() { return filtered; },
		get showListEnd() { return showListEnd; },
		get feedHasMore() { return feedHasMore; },
		get bspLayouts() { return bspLayouts; },
		get interactions() { return interactions; },
		get pageSubtitle() { return pageSubtitle; },
		get islandMetaText() { return islandMetaText; },
		setTag: (tag: string | null) => { activeTag = tag; },
		reload: () => { reloadKey++; },
		handleLoadMore,
		openMoment
	};
}
