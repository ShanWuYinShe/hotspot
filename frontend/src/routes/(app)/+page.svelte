<script lang="ts">
	import { goto } from '$app/navigation';
	import { ChevronRight, Search, Radar, Map } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import SectionHead from '@/components/home/SectionHead.svelte';
	import HeroStats from '@/components/home/HeroStats.svelte';
	import { type NearbyUser } from '@/components/home/nearby-users';
	import type { MomentItem } from '@/api/types';
	import { fetchHomeFeed } from '@/components/home/home-feed';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import MomentRow from '@/components/home/ActivityRow.svelte';
	import NearbyUserRows from '@/components/home/NearbyUserRows.svelte';
	import GroupCarousel, { type NearbyGroup } from '@/components/home/GroupCarousel.svelte';
	import { ROUTES } from '@/lib/routes';
	import { startDefaultLocation, type DefaultLocation } from '@/lib/start-default-location';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import { follow } from '@/api';
	import { showToast } from '@/stores/toast.svelte';

	// 首页「附近」：Hero 统计 + 附近动态 + 群组轮播 + 推荐列表。
	// 三端统一 PageHeader（title + 额外信息 + trailing 工具区）+ 内容区：
	// 桌面/平板：左侧 340px 摘要栏 + 右侧完整用户列表（2 栏 grid）。
	// 移动：单栏堆叠。

	// ── 数据获取（Promise.all + cancelled 竞态）────────
	let nearbyUsers = $state<NearbyUser[]>([]);
	let moments = $state<MomentItem[]>([]);
	let nearbyGroups = $state<NearbyGroup[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let reloadKey = $state(0);
	let defaultLocation = $state<DefaultLocation | null>(null);
	const device = $derived(platform.platform);

	$effect(() => startDefaultLocation((loc) => {
			defaultLocation = loc ?? null;
}));

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchData() {
			try {
				loading = true;
				error = null;
				const feed = await fetchHomeFeed(defaultLocation?.point);
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
	let followPending = $state<Set<string>>(new Set());
	const toggleFollow = (id: string) => {
		if (followPending.has(id)) return;
		const n = new Set(followingIds);
		const next = !n.has(id);
		next ? n.add(id) : n.delete(id);
		followingIds = n;
		followPending = new Set(followPending).add(id);
		// spec: users/set_follow 两态统一；乐观更新 + 失败回滚
		follow({ user_id: id, active: next })
			.catch(() => {
				const r = new Set(followingIds);
				next ? r.delete(id) : r.add(id);
				followingIds = r;
				showToast('操作失败，请重试', 'danger');
			})
			.finally(() => {
				const rest = new Set(followPending);
				rest.delete(id);
				followPending = rest;
			});
	};
	const onlineCount = $derived(nearbyUsers.filter((u) => u.isOnline).length);
	const heroStats = $derived({
		nearby: String(nearbyUsers.length),
		active: String(moments.length),
		online: String(onlineCount)
	});

</script>
<svelte:head>
	<title>首页 · HotSpot</title>
</svelte:head>


{#snippet momentRows(items: MomentItem[], onClick: (momentId: string) => void)}
	<ListContainer bordered>
		{#each items as moment, i (moment.moment_id)}
			<MomentRow {moment} index={i} onClick={() => onClick(moment.moment_id)} />
		{/each}
	</ListContainer>
{/snippet}

{#snippet homeContent()}
	<!-- 移动端成功态内容（loading/error/empty 三态由外层 156-166 统一承载，只写一次）。
	     首页为大纲页：列表限长 + 「更多」链接导航到专区 -->
	<HeroStats onExplore={() => void goto(ROUTES.nearbyPeople)} stats={heroStats} locationName={defaultLocation?.location_name ?? ''} />
	<div>
		<SectionHead title="附近活动" action="更多" onAction={() => void goto(ROUTES.nearbyMoments)} />
		{@render momentRows(moments.slice(0, 3), (id) => void goto(ROUTES.moment(id)))}
	</div>
	<GroupCarousel groups={nearbyGroups} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
	<div>
		<SectionHead title="人物推荐" action="查看全部" onAction={() => void goto(ROUTES.nearbyPeople)} />
		<!-- 对齐 Figma HomePage：人物推荐仅展示前 4 位，其余引导到附近的人专区 -->
		<NearbyUserRows users={nearbyUsers.slice(0, 4)} {followingIds} {followPending} onToggle={toggleFollow} />
	</div>
{/snippet}

	<StandardPageLayout
		contentClassName={platform.platform === 'desktop' ? 'animate-fade-in pt-4 pb-8' : 'animate-fade-in flex flex-col gap-5 pt-4 pb-16'}
	>
	{#snippet header()}
		<!-- ═══ 首页 — 统一 PageHeader（三端同标题；附近内容拆分到 /nearby 聚合页）═══ -->
		<PageHeader title="首页" titleSize={device === 'desktop' ? 'xl' : '3xl'} subtitle="发现周边有趣的人和事" subtitleSize={device === 'desktop' ? 'xs' : 'sm'} sticky>
			{#snippet trailing()}
				<div class="flex items-center gap-2">
					{#if onlineCount > 0}
						<div class="flex items-center gap-1.5 px-3 py-1.5 rounded-full border border-border-default bg-surface-primary shadow-raised-xs">
							<span class="rounded-full size-1.5 bg-status-online"></span>
							<span class="text-xs text-text-secondary">{onlineCount} 在线</span>
						</div>
					{/if}
					{#if platform.platform !== 'desktop'}
						<ActionIconButton icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} variant="ghost" size={36} />
					{/if}
					<ActionIconButton icon={Map} label="地图" onClick={() => void goto(ROUTES.map)} variant="ghost" size={36} />
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	{#if loading}
		<!-- 加载中：与移动端一致的统一 spinner（桌面/平板不再静默空渲染） -->
		<InlineLoading label="加载首页数据…" />
	{:else if error}
		<LoadError message={error} onRetry={() => reloadKey++} />
	{:else if moments.length === 0 && nearbyUsers.length === 0 && nearbyGroups.length === 0}
		<EmptyState title="附近还没有内容" description="去发布第一条动态，或到附近的人认识新朋友。">
			{#snippet icon()}
				<Radar class="size-6 opacity-subtle text-text-secondary" strokeWidth={1.5} />
			{/snippet}
		</EmptyState>
	{:else if platform.platform === 'desktop'}
		<!-- Desktop：左摘要栏（340px fixed）+ 右用户列表（两栏 grid）
		     平板（600-1279）不进入此分支：固定的 340px 在 ≤880vp 时会把右列压到
		     ≤500px、600-700vp 时压到 ≤260px，使 NearbyUserRows 不可读。平板直接走
		     homeContent（单栏堆叠，4 条人物推荐 + 查看全部），与移动端保持同款
		     内容密度。 -->
		<div class="grid grid-cols-[21.25rem_minmax(0,1fr)] gap-6">
			<!-- Left: hero + activity + groups (340px) -->
			<div class="flex flex-col gap-6 border-r border-border-default pr-6">
				<HeroStats onExplore={() => void goto(ROUTES.nearbyPeople)} stats={heroStats} locationName={defaultLocation?.location_name ?? ''} />
				<div>
					<SectionHead title="附近活动" />
					{@render momentRows(moments, (id) => void goto(ROUTES.moment(id)))}
				</div>
				<GroupCarousel groups={nearbyGroups} onSeeAll={() => void goto(ROUTES.nearbyGroups)} />
			</div>

			<!-- Right: full nearby-users list -->
			<div>
				<div class="flex items-end justify-between mb-5">
					<div>
						<span class="eyebrow">附近的人</span>
						<p class="text-xs mt-1 text-text-secondary"  >
							5km 范围内 · 实时更新
						</p>
					</div>
					<button
						type="button"
						class="hover-btn tap-92 flex items-center gap-0.5 focus-ring surface-ghost rounded-full px-2 py-0.5 text-accent-default"
						onclick={() => void goto(ROUTES.nearbyPeople)}
					>
						<span class="text-xs">查看全部</span>
						<ChevronRight class="size-3.5"  strokeWidth={1.5} />
					</button>
				</div>

				<div class="mt-6">
					<NearbyUserRows users={nearbyUsers} {followingIds} {followPending} onToggle={toggleFollow} />
				</div>
			</div>
		</div>
	{:else}
		{@render homeContent()}
	{/if}
</StandardPageLayout>
