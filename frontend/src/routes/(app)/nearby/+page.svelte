<script lang="ts">
	import { goto } from '$app/navigation';
	import { Search as SearchIcon, Radar } from '@lucide/svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import MomentRow from '@/components/home/ActivityRow.svelte';
	import NearbyUserRows from '@/components/home/NearbyUserRows.svelte';
	import GroupCarousel, { type NearbyGroup } from '@/components/home/GroupCarousel.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import { type NearbyUser } from '@/components/home/nearby-users';
	import { fetchHomeFeed } from '@/components/home/home-feed';
	import { startDefaultLocation } from '@/lib/start-default-location';
	import { follow } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import type { MomentItem } from '@/api/types';
	import { ROUTES } from '@/lib/routes';

	// 附近聚合页（平板/移动端顶级导航「附近」tab）：
	// 搜索入口 + 附近的人 / 附近群组 / 附近动态，各区块「查看全部」跳对应专区。
	// 数据复用首页同一组接口（fetchHomeFeed），与首页信息架构一致。

	let nearbyUsers = $state<NearbyUser[]>([]);
	let moments = $state<MomentItem[]>([]);
	let nearbyGroups = $state<NearbyGroup[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let reloadKey = $state(0);
	let defaultLocation = $state<{ latitude: number; longitude: number } | null>(null);

	$effect(() =>
		startDefaultLocation((loc) => {
			defaultLocation = loc?.point ?? null;
		})
	);

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				error = null;
				const feed = await fetchHomeFeed(defaultLocation);
				if (cancelled) return;
				moments = feed.moments;
				nearbyUsers = feed.users;
				nearbyGroups = feed.groups;
			} catch (e) {
				if (!cancelled) error = e instanceof Error ? e.message : '加载失败';
			} finally {
				if (!cancelled) loading = false;
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
		const next = !n.has(id);
		next ? n.add(id) : n.delete(id);
		followingIds = n;
		// spec: users/set_follow 两态统一；乐观更新 + 失败回滚
		follow({ user_id: id, active: next }).catch(() => {
			const r = new Set(followingIds);
			next ? r.delete(id) : r.add(id);
			followingIds = r;
			showToast('操作失败，请重试', 'danger');
		});
	};
</script>

<svelte:head>
	<title>附近 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentClassName="animate-fade-in pt-4 pb-6">
	{#snippet header()}
		<PageHeader title="附近" subtitle="发现周边有趣的人和事" sticky>
			{#snippet trailing()}
				<span class="text-text-secondary text-xs">{nearbyUsers.length} 人 · {nearbyGroups.length} 群</span>
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if loading}
		<div class="flex flex-col items-center gap-3 py-20">
			<div class="spinner size-7"></div>
			<p class="text-text-secondary text-sm">加载附近内容…</p>
		</div>
	{:else if error}
		<LoadError message={error} onRetry={() => reloadKey++} />
	{:else if moments.length === 0 && nearbyUsers.length === 0 && nearbyGroups.length === 0}
		<EmptyState title="附近还没有内容" description="去发布第一条动态，或到附近的人认识新朋友。">
			{#snippet icon()}
				<Radar class="size-6 opacity-40 text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else}
		<!-- 搜索入口（对齐旧发现聚合页/首页样板） -->
		<button
			type="button"
			class="hover-btn flex w-full items-center gap-3 rounded-panel border border-border-default bg-surface-primary p-3 text-left focus-ring"
			onclick={() => void goto(ROUTES.search)}
		>
			<SearchIcon class="size-5 shrink-0 text-accent-default" strokeWidth={1.8} />
			<span class="min-w-0 flex-1 py-2 text-base text-text-secondary">搜索用户、群组、动态、地点…</span>
		</button>

		<div class="mt-6">
			<SectionHead title="附近的人" action="查看全部" onAction={() => void goto(ROUTES.nearbyPeople)} />
			<NearbyUserRows users={nearbyUsers.slice(0, 4)} {followingIds} onToggle={toggleFollow} />
		</div>

		<div class="mt-6">
			<SectionHead title="附近群组" action="查看全部" onAction={() => void goto(ROUTES.nearbyGroups)} />
			<GroupCarousel groups={nearbyGroups.slice(0, 4)} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
		</div>

		<div class="mt-6">
			<SectionHead title="附近动态" action="查看全部" onAction={() => void goto(ROUTES.nearbyMoments)} />
			<ListContainer bordered>
				{#each moments.slice(0, 3) as moment, i (moment.moment_id)}
					<MomentRow {moment} index={i} onClick={() => void goto(ROUTES.moment(moment.moment_id))} />
				{/each}
			</ListContainer>
		</div>
	{/if}
</StandardPageLayout>
