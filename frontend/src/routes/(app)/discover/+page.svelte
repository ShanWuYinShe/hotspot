<script lang="ts">
	import { goto } from '$app/navigation';
	import { Coffee, Music, Users, Star, Search as SearchIcon } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import { loadDefaultLocation } from '@/lib/default-location';
	import { getTrending, getNearbyUsers, getNearbyGroups } from '@/api';
	import SectionHead from '@/routes/(app)/_home/SectionHead.svelte';
	import ActivityList, { type Activity } from '@/routes/(app)/_home/ActivityList.svelte';
	import GroupCarousel, { type NearbyGroup } from '@/routes/(app)/_home/GroupCarousel.svelte';
	import NearbyUsersFlatList from '@/routes/(app)/_home/NearbyUsersFlatList.svelte';
	import { type NearbyUser } from '@/routes/(app)/_home/nearby-users';
	import { ROUTES } from '@/lib/routes';
	import { formatDistance } from '@/lib/utils';

	// 发现聚合页（平板/移动端顶级导航）：聚合「附近的人 / 附近群组 / 附近动态」
	// 与搜索入口——平板/移动端没有桌面侧栏的空间放多个 nearby 菜单，统一收口到
	// 本页聚合，各区块「查看全部」再跳对应专区（/nearby/people|groups|moments）。
	// 数据复用首页同一组接口（getTrending / getNearbyUsers / getNearbyGroups）。

	let nearbyUsers = $state<NearbyUser[]>([]);
	let activities = $state<Activity[]>([]);
	let nearbyGroups = $state<NearbyGroup[]>([]);
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(null);

	$effect(() => {
		let cancelled = false;
		void loadDefaultLocation().then((loc) => {
			if (!cancelled) referencePoint = loc?.point ?? null;
		});
		return () => {
			cancelled = true;
		};
	});

	$effect(() => {
		let cancelled = false;
		async function fetchData() {
			const searchParams: Record<string, unknown> = { radius: 5000, pagination: { limit: 12 } };
			if (referencePoint) searchParams.reference_point = referencePoint;
			const [trendingRes, nearbyUsersRes, nearbyGroupsRes] = await Promise.all([
				getTrending().catch(() => ({ code: -1, data: null })),
				getNearbyUsers(searchParams).catch(() => ({ code: -1, data: { items: [] } })),
				getNearbyGroups(searchParams).catch(() => ({ code: -1, data: { items: [] } }))
			]);
			if (cancelled) return;

			const trendingData = trendingRes.code === 0 ? trendingRes.data : null;
			const nearbyUsersData = nearbyUsersRes.code === 0 ? nearbyUsersRes.data : null;
			const nearbyGroupsData = nearbyGroupsRes.code === 0 ? nearbyGroupsRes.data : null;

			if (trendingData) {
				activities = (trendingData.hot_moments || []).map((a, i) => ({
					id: i + 1,
					icon: [Coffee, Music, Users, Star][i % 4],
					title: a.description ?? '',
					sub: a.location?.location_name ?? '',
					color: ['var(--semantic-warning)', 'var(--neu-accent-primary)', 'var(--map-group)', 'var(--semantic-hot)'][i % 4],
					bg: [
						'color-mix(in srgb, var(--semantic-warning) 10%, transparent)',
						'var(--neu-accent-soft)',
						'color-mix(in srgb, var(--map-group) 10%, transparent)',
						'color-mix(in srgb, var(--semantic-hot) 10%, transparent)'
					][i % 4]
				}));
			}

			if (nearbyUsersData) {
				nearbyUsers = nearbyUsersData.items.map((u) => ({
					id: u.user?.user_id ?? '',
					name: u.user?.nickname ?? '',
					distance: formatDistance(u.location?.distance),
					avatar: u.user?.avatar_url ?? '',
					status: u.presence_status === 'online' ? '在线' : '离线',
					tag: u.tags?.[0] ?? ''
				}));
			}

			if (nearbyGroupsData) {
				nearbyGroups = nearbyGroupsData.items.map((g) => ({
					id: g.group?.group_id ?? '',
					name: g.group?.name ?? '',
					members: g.group?.members_count ?? 0,
					banner: g.group?.avatar_url ?? '',
					distance: formatDistance(g.location?.distance),
					privacy: g.join_method === 'invite_only' || g.join_method === 'password' ? 'private' : 'public'
				}));
			}
		}
		void fetchData();
		return () => {
			cancelled = true;
		};
	});

	let followingIds = $state<Set<string>>(new Set());
	const toggleFollow = (id: string) => {
		const n = new Set(followingIds);
		n.has(id) ? n.delete(id) : n.add(id);
		followingIds = n;
	};
</script>

<StandardPageLayout contentClassName="py-5 sm:py-6">
	{#snippet header()}
		<PageHeader title="发现" subtitle="聚合附近的人、群组与动态" sticky>
			{#snippet trailing()}
				<!-- 计数统一在标题右侧（全站样板） -->
				<span class="text-[var(--neu-text-muted)] text-xs">{nearbyUsers.length} 人 · {nearbyGroups.length} 群</span>
			{/snippet}
		</PageHeader>
	{/snippet}

	<!-- 搜索入口（对齐 Figma 搜索卡片：rounded-panel border bg-card） -->
	<button
		type="button"
		class="neu-hover-btn flex w-full items-center gap-3 rounded-panel border bg-[var(--neu-bg)] p-3 text-left outline-none"
		style="border-color:var(--neu-border);"
		onclick={() => void goto(ROUTES.search)}
	>
		<SearchIcon class="size-5 shrink-0 text-[var(--neu-accent-primary)]" strokeWidth={1.8} />
		<span class="min-w-0 flex-1 py-2 text-[15px] text-[var(--neu-text-muted)]">搜索用户、群组、动态、地点…</span>
	</button>

	<!-- 附近的人 -->
	<div class="mt-6">
		<SectionHead title="附近的人" action="查看全部" onAction={() => void goto(ROUTES.nearbyPeople)} />
		<NearbyUsersFlatList users={nearbyUsers.slice(0, 4)} followingIds={followingIds} onToggle={toggleFollow} />
	</div>

	<!-- 附近群组 -->
	<div class="mt-6">
		<SectionHead title="附近群组" action="查看全部" onAction={() => void goto(ROUTES.nearbyGroups)} />
		<GroupCarousel groups={nearbyGroups.slice(0, 4)} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
	</div>

	<!-- 附近动态 -->
	<div class="mt-6">
		<SectionHead title="附近动态" action="查看全部" onAction={() => void goto(ROUTES.nearbyMoments)} />
		<ActivityList items={activities.slice(0, 3)} onClick={(id) => void goto(ROUTES.event(id))} />
	</div>
</StandardPageLayout>
