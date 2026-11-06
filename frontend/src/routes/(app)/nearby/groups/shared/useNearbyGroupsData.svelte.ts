import { platform } from '@/stores/platform.svelte';
import { startDefaultLocation } from '@/lib/start-default-location';
import { readLastLocation } from '@/lib/map-auto-locate';
import { mapGroup, type Group } from './group-data';
import { NEARBY_CATEGORIES } from '../../shared/categories';
import { requireAuth } from '@/stores/auth.svelte';
import { computeBspLayoutSequence } from '@/lib/semantic-bsp';
import { searchGroups, joinGroup, leaveGroup, cancelGroupApplication, listMyApplications, netError } from '@/api';
import { showToast } from '@/stores/toast.svelte';

export interface NearbyGroupsData {
	readonly activeCategory: string;
	readonly groups: Group[];
	readonly visibleCount: number;
	readonly loading: boolean;
	readonly loadError: string | null;
	readonly locationName: string;
	readonly filtered: Group[];
	readonly visibleGroups: Group[];
	readonly hasMoreVisible: boolean;
	readonly bspLayouts: ReturnType<typeof computeBspLayoutSequence>;
	readonly totalOnline: number;
	readonly joinPending: Set<string>;
	readonly pendingApprovals: Map<string, string>;
	leaveTarget: Group | null;
	readonly pageSubtitle: string;
	readonly islandMetaText: string;
	setCategory: (cat: string) => void;
	reload: () => void;
	revealMore: () => void;
	toggleJoin: (id: string) => void;
	applyJoin: (id: string, next: boolean) => void;
	cancelApplication: (id: string) => Promise<void>;
	confirmLeave: () => void;
	dismissLeave: () => void;
}

export function useNearbyGroupsData(): NearbyGroupsData {
	let activeCategory = $state('全部');
	let groups = $state<Group[]>([]);
	let visibleCount = $state(12);
	let loading = $state(true);

	const savedAnchor = readLastLocation();
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(
		savedAnchor ? { latitude: savedAnchor.lat, longitude: savedAnchor.lng } : null
	);
	let locationName = $state(savedAnchor?.name ?? '');
	let loadError = $state<string | null>(null);
	let reloadKey = $state(0);

	let joinPending = $state(new Set<string>());
	let pendingApprovals = $state(new Map<string, string>());
	let leaveTarget = $state<Group | null>(null);

	$effect(() => startDefaultLocation((loc) => {
		if (savedAnchor) return;
		referencePoint = loc?.point ?? null;
		locationName = loc?.location_name ?? '';
	}));

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchGroups() {
			loading = true;
			loadError = null;
			try {
				const res = await searchGroups({
					radius: 5000,
					sort_by: 'distance',
					...(referencePoint ? { reference_point: referencePoint } : {}),
					pagination: { limit: 24 }
				}).catch(() => netError({ items: [] }));
				if (cancelled) return;
				if (res.code !== 0) {
					loadError = res.message ?? '加载失败';
					return;
				}
				groups = (res.code === 0 && res.data ? res.data.items : []).map(mapGroup);
				void listMyApplications().then((mine) => {
					if (cancelled || mine.code !== 0) return;
					const approvals = new Map<string, string>();
					for (const a of mine.data?.items ?? []) {
						if (a.status === 'pending') approvals.set(a.group?.group_id ?? '', a.application_id);
					}
					pendingApprovals = approvals;
				});
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchGroups();
		return () => {
			cancelled = true;
		};
	});

	const device = $derived(platform.platform);

	const filtered = $derived(
		groups.filter((group) => activeCategory === '全部' || group.tags.includes(activeCategory))
	);
	const visibleGroups = $derived(filtered.slice(0, visibleCount));
	const hasMoreVisible = $derived(visibleCount < filtered.length);

	function revealMore() {
		if (visibleCount >= filtered.length) return;
		visibleCount = Math.min(visibleCount + 8, filtered.length);
	}

	const bspLayouts = $derived(
		computeBspLayoutSequence(
			visibleGroups.map((g) => ({
				members: g.members,
				isFeatured: g.members > 30 || g.joined,
				likes: g.online,
				hasImage: Boolean(g.banner)
			})),
			{ device, kind: 'group' }
		)
	);

	const totalOnline = $derived(filtered.reduce((sum, group) => sum + group.online, 0));

	const applyJoin = (id: string, next: boolean) => {
		joinPending = new Set(joinPending).add(id);
		groups = groups.map((g) => (g.id === id ? { ...g, joined: next } : g));
		const done = (res: { code: number; message?: string; data?: unknown }) => {
			const rest = new Set(joinPending);
			rest.delete(id);
			joinPending = rest;
			if (res.code !== 0) {
				groups = groups.map((g) => (g.id === id ? { ...g, joined: !next } : g));
				showToast(res.message || (next ? '加入失败，请重试' : '退出失败，请重试'), 'danger');
			} else if (next) {
				const joinData = res.data as { status?: string; application_id?: string | null } | null | undefined;
				if (joinData?.status === 'pending_approval') {
					groups = groups.map((g) => (g.id === id ? { ...g, joined: false } : g));
					pendingApprovals = new Map(pendingApprovals).set(id, joinData.application_id ?? '');
					showToast('申请已提交，待群主审核', 'success');
				} else {
					showToast('已加入群组', 'success');
				}
			} else {
				showToast('已退出群组', 'success');
			}
		};
		if (next) {
			void joinGroup(id).then(done).catch(() => done(netError(undefined)));
		} else {
			void leaveGroup(id).then(done).catch(() => done(netError(undefined)));
		}
	};

	const cancelApplication = async (id: string) => {
		const appId = pendingApprovals.get(id);
		if (!appId) return;
		joinPending = new Set(joinPending).add(id);
		const res = await cancelGroupApplication({ group_id: id, application_id: appId }).catch(() => netError(undefined));
		const rest = new Set(joinPending);
		rest.delete(id);
		joinPending = rest;
		if (res.code === 0) {
			const nextApprovals = new Map(pendingApprovals);
			nextApprovals.delete(id);
			pendingApprovals = nextApprovals;
			showToast('已取消申请', 'success');
		} else {
			showToast(res.message || '取消失败，请重试', 'danger');
		}
	};

	const toggleJoin = (id: string) => {
		const group = groups.find((g) => g.id === id);
		if (!group || joinPending.has(id)) return;
		if (pendingApprovals.has(id)) {
			void cancelApplication(id);
			return;
		}
		if (group.joined) {
			leaveTarget = group;
			return;
		}
		if (!requireAuth({ action: '加入群组', targetId: id, run: () => toggleJoin(id) })) return;
		applyJoin(id, true);
	};

	const pageSubtitle = $derived(
		locationName
			? `${locationName} · 5km 内${!loading && !loadError && filtered.length ? ` · ${filtered.length} 个群组` : ''}`
			: `5km 内 · 实时更新${!loading && !loadError && filtered.length ? ` · ${filtered.length} 个群组` : ''}`
	);
	const islandMetaText = $derived(
		activeCategory === '全部' ? pageSubtitle : `${activeCategory} · ${pageSubtitle}`
	);

	return {
		get activeCategory() { return activeCategory; },
		get groups() { return groups; },
		get visibleCount() { return visibleCount; },
		get loading() { return loading; },
		get loadError() { return loadError; },
		get locationName() { return locationName; },
		get filtered() { return filtered; },
		get visibleGroups() { return visibleGroups; },
		get hasMoreVisible() { return hasMoreVisible; },
		get bspLayouts() { return bspLayouts; },
		get totalOnline() { return totalOnline; },
		get joinPending() { return joinPending; },
		get pendingApprovals() { return pendingApprovals; },
		get leaveTarget() { return leaveTarget; },
		set leaveTarget(v: Group | null) { leaveTarget = v; },
		get pageSubtitle() { return pageSubtitle; },
		get islandMetaText() { return islandMetaText; },
		setCategory: (cat: string) => { activeCategory = cat; },
		reload: () => { reloadKey++; },
		revealMore,
		toggleJoin,
		applyJoin,
		cancelApplication,
		confirmLeave: () => {
			if (leaveTarget) applyJoin(leaveTarget.id, false);
			leaveTarget = null;
		},
		dismissLeave: () => {
			leaveTarget = null;
		}
	};
}
