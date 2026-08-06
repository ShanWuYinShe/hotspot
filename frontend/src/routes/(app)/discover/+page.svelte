<script lang="ts">
	import { goto } from '$app/navigation';
	import { Search as SearchIcon } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { fetchHomeFeed } from '@/components/home/home-feed';
	import { type Activity } from '@/components/home/activity-data';
	import NearbyUserRows from '@/components/home/NearbyUserRows.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import ActivityRow from '@/components/home/ActivityRow.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import GroupCarousel, { type NearbyGroup } from '@/components/home/GroupCarousel.svelte';
	import { type NearbyUser } from '@/components/home/nearby-users';
	import { ROUTES } from '@/lib/routes';

	// 发现聚合页（平板/移动端顶级导航）：聚合「附近的人 / 附近群组 / 附近动态」
	// 与搜索入口——平板/移动端没有桌面侧栏的空间放多个 nearby 菜单，统一收口到
	// 本页聚合，各区块「查看全部」再跳对应专区（/nearby/people|groups|moments）。
	// 数据复用首页同一组接口（getTrending / getNearbyUsers / getNearbyGroups）。

	let nearbyUsers = $state<NearbyUser[]>([]);
	let activities = $state<Activity[]>([]);
	let nearbyGroups = $state<NearbyGroup[]>([]);
	let referencePoint = $state<{ latitude: number; longitude: number } | null>(null);

	$effect(() => startDefaultLocation((loc) => {
referencePoint = loc?.point ?? null;
}));

	$effect(() => {
		let cancelled = false;
		async function fetchData() {
			const feed = await fetchHomeFeed(referencePoint);
			if (cancelled) return;
			activities = feed.activities;
			nearbyUsers = feed.users;
			nearbyGroups = feed.groups;
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
				<span class="text-text-secondary text-xs">{nearbyUsers.length} 人 · {nearbyGroups.length} 群</span>
			{/snippet}
		</PageHeader>
	{/snippet}

	<!-- 搜索入口（对齐 Figma 搜索卡片：rounded-panel border bg-card） -->
	<button
		type="button"
		class="hover-btn flex w-full items-center gap-3 rounded-panel border bg-surface-primary p-3 text-left outline-none"
		style="border-color:var(--border-default);"
		onclick={() => void goto(ROUTES.search)}
	>
		<SearchIcon class="size-5 shrink-0 text-accent-default" strokeWidth={1.8} />
		<span class="min-w-0 flex-1 py-2 text-base text-text-secondary">搜索用户、群组、动态、地点…</span>
	</button>

	<!-- 附近的人 -->
	<div class="mt-6">
		<SectionHead title="附近的人" action="查看全部" onAction={() => void goto(ROUTES.nearbyPeople)} />
<NearbyUserRows users={nearbyUsers.slice(0, 4)} {followingIds} onToggle={toggleFollow} />
	</div>

	<!-- 附近群组 -->
	<div class="mt-6">
		<SectionHead title="附近群组" action="查看全部" onAction={() => void goto(ROUTES.nearbyGroups)} />
		<GroupCarousel groups={nearbyGroups.slice(0, 4)} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
	</div>

	<!-- 附近动态 -->
		<div class="mt-6">
			<SectionHead title="附近动态" action="查看全部" onAction={() => void goto(ROUTES.nearbyMoments)} />
			<ListContainer bordered>
				{#each activities.slice(0, 3) as item (item.id)}
					<ActivityRow item={item} onClick={() => void goto(ROUTES.event(item.id))} />
				{/each}
			</ListContainer>
		</div>
	</StandardPageLayout>
