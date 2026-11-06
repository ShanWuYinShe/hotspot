import { platform } from '@/stores/platform.svelte';
import { startDefaultLocation } from '@/lib/start-default-location';
import { LOCATION_ANCHOR_UPDATED_EVENT, clearLastLocation, readLastLocation } from '@/lib/map-auto-locate';
import type { NearbyUser } from '@/components/data-display/UserCard.svelte';
import { searchUsers, follow, netError } from '@/api';
import { formatDistance } from '@/lib/utils';
import { computeBspLayoutSequence } from '@/lib/semantic-bsp';
import { showToast } from '@/stores/toast.svelte';
import { auth, requireAuth } from '@/stores/auth.svelte';
import { NEARBY_CATEGORIES } from '../../shared/categories';
import {
	getPeopleProgress,
	setPeopleProgress,
	clearPeopleProgress,
	peopleProgressKey
} from '../people-progress.svelte';

export interface NearbyPeopleData {
	readonly sortOptions: string[];
	readonly sort: string;
	readonly effectiveSort: string;
	readonly tagFilter: string;
	readonly allUsers: NearbyUser[];
	readonly loading: boolean;
	readonly loadError: string | null;
	readonly locationName: string;
	readonly defaultLocationName: string;
	readonly anchor: ReturnType<typeof readLastLocation>;
	readonly list: NearbyUser[];
	readonly visibleUsers: NearbyUser[];
	readonly visibleCount: number;
	readonly hasMoreVisible: boolean;
	readonly bspLayouts: ReturnType<typeof computeBspLayoutSequence>;
	readonly followedIds: Set<string>;
	readonly followPending: Set<string>;
	readonly pageSubtitle: string;
	readonly filterTrigger: string;
	readonly islandMetaText: string;
	setSort: (s: string) => void;
	setTagFilter: (tag: string) => void;
	reload: () => void;
	revealMore: () => void;
	toggleFollow: (id: string) => Promise<void>;
	clearAnchor: () => void;
}

export function useNearbyPeopleData(): NearbyPeopleData {
	const sortOptions = $derived(auth.user ? ['距离', '在线', '共同好友'] : ['距离', '在线']);
	let sort = $state('距离');
	const effectiveSort = $derived(sortOptions.includes(sort) ? sort : '距离');
	let tagFilter = $state('全部');
	let followedIds = $state<Set<string>>(new Set());
	let followPending = $state(new Set<string>());
	let visibleCount = $state(12);

	let allUsers = $state<NearbyUser[]>([]);
	let loading = $state(true);
	const initialAnchor = readLastLocation();
	let anchor = $state(initialAnchor);
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(
		initialAnchor ? { latitude: initialAnchor.lat, longitude: initialAnchor.lng } : null
	);
	let locationName = $state(initialAnchor?.name ?? (initialAnchor ? '自定义位置' : ''));
	let defaultLocationName = $state('');
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	$effect(() => startDefaultLocation((loc) => {
		defaultLocationName = loc?.location_name ?? '';
		if (anchor) return;
		referencePoint = loc?.point ?? null;
		locationName = loc?.location_name ?? '';
	}));

	function applyAnchor() {
		anchor = readLastLocation();
		clearPeopleProgress();
		if (anchor) {
			referencePoint = { latitude: anchor.lat, longitude: anchor.lng };
			locationName = anchor.name ?? '自定义位置';
		} else {
			referencePoint = null;
			locationName = '';
		}
	}

	$effect(() => {
		window.addEventListener(LOCATION_ANCHOR_UPDATED_EVENT, applyAnchor);
		return () => window.removeEventListener(LOCATION_ANCHOR_UPDATED_EVENT, applyAnchor);
	});

	$effect(() => {
		void reloadKey;
		const progressKey = peopleProgressKey(referencePoint);
		const saved = getPeopleProgress();
		if (saved && progressKey !== '' && saved.key === progressKey) {
			allUsers = saved.users;
			visibleCount = saved.visibleCount;
			followedIds = new Set(saved.users.filter((u) => u.isFollowing).map((u) => u.id));
			loadError = null;
			loading = false;
			return;
		}
		let cancelled = false;
		async function fetchUsers() {
			try {
				loading = true;
				loadError = null;
				const req = {
					radius: 5000,
					sort_by: 'distance' as const,
					...(referencePoint ? { reference_point: referencePoint } : {})
				};
				const res = await searchUsers(req).catch(() => netError({ items: [] }));
				if (cancelled) return;
				const data = res.code === 0 ? res.data : null;
				if (res.code !== 0) {
					if (!cancelled) loadError = res.message ?? '加载失败';
					return;
				}
				if (data) {
					const items: NearbyUser[] = (data.items || []).map((u) => ({
						id: u.user?.user_id ?? '',
						name: u.user?.nickname ?? '',
						distance: formatDistance(u.location?.distance),
						avatar: u.user?.avatar_url ?? '',
						isOnline: u.presence_status === 'online',
						tag: u.tags?.[0] ?? '',
						tags: u.tags ?? [],
						mutual: u.social_context?.mutual_friends_count ?? 0,
						note: u.bio ?? '',
						isFollowing: u.relation?.is_following ?? false
					}));
					allUsers = items;
					followedIds = new Set(items.filter((u) => u.isFollowing).map((u) => u.id));
					setPeopleProgress({ key: progressKey, users: items, visibleCount });
				}
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchUsers();
		return () => {
			cancelled = true;
		};
	});

	const list = $derived.by(() => {
		const selfId = auth.user?.user_id;
		const next = allUsers.filter(
			(user) =>
				(!selfId || user.id !== selfId) &&
				(tagFilter === '全部' || (user.tags?.includes(tagFilter) ?? false))
		);
		if (sort === '在线') return [...next].sort((a, b) => Number(b.isOnline) - Number(a.isOnline));
		if (sort === '共同好友') return [...next].sort((a, b) => (b.mutual ?? 0) - (a.mutual ?? 0));
		return next;
	});

	const device = $derived(platform.platform);
	const visibleUsers = $derived(list.slice(0, visibleCount));
	const bspLayouts = $derived(
		computeBspLayoutSequence(
			visibleUsers.map((u) => ({
				isOnline: u.isOnline,
				isFeatured: Boolean(u.isOnline || (u.mutual && u.mutual > 0)),
				likes: u.mutual,
				hasImage: Boolean(u.avatar)
			})),
			{ device, kind: 'user' }
		)
	);

	const hasMoreVisible = $derived(visibleCount < list.length);
	function revealMore() {
		if (visibleCount >= list.length) return;
		visibleCount = Math.min(visibleCount + 8, list.length);
		const saved = getPeopleProgress();
		if (saved) setPeopleProgress({ ...saved, visibleCount });
	}

	const toggleFollow = async (id: string) => {
		if (followPending.has(id)) return;
		if (!requireAuth({ action: '关注', targetId: id, run: () => void toggleFollow(id) })) return;
		const next = new Set(followedIds);
		const willFollow = !next.has(id);
		willFollow ? next.add(id) : next.delete(id);
		followedIds = next;
		followPending = new Set(followPending).add(id);
		try {
			const res = await follow({ user_id: id, active: willFollow });
			if (res.code !== 0) throw new Error(res.message || '操作失败');
		} catch (err) {
			const r = new Set(followedIds);
			willFollow ? r.delete(id) : r.add(id);
			followedIds = r;
			showToast(err instanceof Error && err.message ? err.message : '操作失败，请重试', 'danger');
		} finally {
			const rest = new Set(followPending);
			rest.delete(id);
			followPending = rest;
		}
	};

	const pageSubtitle = $derived(
		locationName
			? `${locationName} · 5km 内${!loading && !loadError && list.length ? ` · ${list.length} 人` : ''}`
			: `5km 内 · 实时更新${!loading && !loadError && list.length ? ` · ${list.length} 人` : ''}`
	);
	const filterTrigger = $derived(`${tagFilter}:${effectiveSort}`);
	const islandMetaText = $derived(tagFilter === '全部' ? pageSubtitle : `${tagFilter} · ${pageSubtitle}`);

	return {
		get sortOptions() { return sortOptions; },
		get sort() { return sort; },
		get effectiveSort() { return effectiveSort; },
		get tagFilter() { return tagFilter; },
		get allUsers() { return allUsers; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get locationName() { return locationName; },
		get defaultLocationName() { return defaultLocationName; },
		get anchor() { return anchor; },
		get list() { return list; },
		get visibleUsers() { return visibleUsers; },
		get visibleCount() { return visibleCount; },
		get hasMoreVisible() { return hasMoreVisible; },
		get bspLayouts() { return bspLayouts; },
		get followedIds() { return followedIds; },
		get followPending() { return followPending; },
		get pageSubtitle() { return pageSubtitle; },
		get filterTrigger() { return filterTrigger; },
		get islandMetaText() { return islandMetaText; },
		setSort: (s: string) => { sort = s; },
		setTagFilter: (tag: string) => { tagFilter = tag; },
		reload: () => { reloadKey++; },
		revealMore,
		toggleFollow,
		clearAnchor: () => {
			clearLastLocation();
			applyAnchor();
			reloadKey++;
		}
	};
}
