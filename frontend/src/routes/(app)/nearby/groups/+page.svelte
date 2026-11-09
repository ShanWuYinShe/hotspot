<script lang="ts">
	import { formatCount } from '@/lib/utils';
	import { ArrowUp, Map as MapIcon, MapPin, Search, Users, LogOut } from '@lucide/svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import InfiniteSentinel from '@/components/list/InfiniteSentinel.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import AdaptiveCardGrid from '@/components/layout/AdaptiveCardGrid.svelte';
	import SparseContentHint from '@/components/data-display/SparseContentHint.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import GroupCard from '@/components/data-display/GroupCard.svelte';
	import { NEARBY_CATEGORIES } from '../shared/categories';
	import { ROUTES } from '@/lib/routes';
	import { contentFade } from '@/lib/motion';
	import { useNearbyGroupsData } from './shared/useNearbyGroupsData.svelte';

	// 附近群组（/nearby/groups）：独立路由页。分类过滤 + 网格 / 桌面 BSP 卡片墙。
	// 架构重构：状态、数据加载与加入/退出业务逻辑收敛至 useNearbyGroupsData()。
	const data = useNearbyGroupsData();
	const device = $derived(platform.platform);
</script>

<svelte:head>
	<title>附近群组 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentClassName="pt-3 lg:pt-2.5 pb-6">
	{#snippet header()}
		<PageHeader
			variant="integrated"
			title="附近群组"
			subtitle={data.pageSubtitle}
			islandMeta={data.islandMetaText}
		>
			{#snippet islandIcon()}
				<Users class="size-3.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
			{#snippet subtitleIcon()}
				<MapPin class="size-2.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
			{#snippet trailing()}
				<div class="flex items-center gap-2">
					{#if !data.loading && !data.loadError && data.filtered.length}
						<span class="flex items-center gap-1.5 text-xs tabular-nums text-text-secondary">
							<span class="size-1.5 rounded-full bg-status-online"></span>
							<span class="font-medium text-text-status-online-on-soft">{formatCount(data.totalOnline)} 人在线</span>
						</span>
					{/if}
					{#if device !== 'desktop'}
						<ActionIconButton variant="ghost" icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} />
						<ActionIconButton variant="ghost" icon={MapIcon} label="地图浏览" onClick={() => void goto(ROUTES.map)} />
					{/if}
				</div>
			{/snippet}
			{#snippet islandExpanded()}
				<div class="flex flex-col gap-3">
					<div class="flex items-center justify-between text-xs pb-0.5">
						<span class="font-medium text-text-primary">圈子分类</span>
						{#if data.totalOnline > 0}
							<span class="inline-flex items-center gap-1.5 text-xs text-text-status-online-on-soft">
								<span class="size-1.5 rounded-full bg-status-online"></span>
								<span>{formatCount(data.totalOnline)} 人在线</span>
							</span>
						{/if}
					</div>
					<div class="w-full">
						<TabBar
							items={NEARBY_CATEGORIES.map((c) => ({ id: c, label: c }))}
							value={data.activeCategory}
							onChange={(id) => data.setCategory(id)}
							class="flex-wrap gap-1.5"
						/>
					</div>
				</div>
			{/snippet}
			{#snippet below()}
				<div class="flex items-center justify-between gap-3">
					<div class="min-w-0 flex-1">
						<TabBar
							items={NEARBY_CATEGORIES.map((c) => ({ id: c, label: c }))}
							value={data.activeCategory}
							onChange={(id) => data.setCategory(id)}
							class={device === 'desktop' ? 'flex-wrap' : 'pr-1'}
						/>
					</div>
					{#if !data.loading && !data.loadError && data.filtered.length}
						<span class="hidden sm:inline-flex shrink-0 items-center gap-1.5 rounded-full border border-border-default/30 bg-surface-island/60 px-2.5 py-1 text-xs tabular-nums text-text-secondary backdrop-blur-md shadow-xs">
							<span class="size-1.5 rounded-full bg-status-online"></span>
							<span class="font-medium text-text-status-online-on-soft">{formatCount(data.totalOnline)} 人在线</span>
						</span>
					{/if}
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	<div class="min-w-0" use:contentFade={{ showSkeleton: data.loading, trigger: data.activeCategory }}>
		{#if data.loading}
			<SkeletonFeed
				variant="card"
				count={8}
				cardClass="h-full"
				class="grid grid-flow-dense grid-cols-2 gap-3 px-1 sm:grid-cols-3 sm:gap-3.5 lg:grid-cols-[repeat(auto-fill,minmax(18.75rem,1fr))] [grid-auto-rows:clamp(12.1875rem,20vw,17.1875rem)]"
				label="加载附近群组…"
			/>
		{:else if data.loadError}
			<LoadError message={data.loadError} onRetry={data.reload} />
		{:else if data.filtered.length}
			<div class="w-full">
				<AdaptiveCardGrid
					count={data.visibleGroups.length}
					minTrackRem={18.75}
					gapRem={[0.75, 0.875, 0.875]}
					centerSparse={data.bspLayouts.slice(0, data.visibleGroups.length).every((l) => l.tier === '1x1')}
					class="grid-flow-dense px-1"
					style="grid-auto-rows: clamp(12.1875rem, 20vw, 17.1875rem);"
				>
					{#each data.visibleGroups as group, index (group.id)}
						{@const layout = data.bspLayouts[index] ?? { tier: '1x1', cls: 'col-span-1 row-span-1', horizontal: false, featured: false }}
						<div class={layout.cls}>
							<GroupCard
								{group}
								priority={index < 4}
								featured={layout.featured}
								horizontal={layout.horizontal}
								tier={layout.tier}
								pending={data.joinPending.has(group.id)}
								pendingApproval={data.pendingApprovals.has(group.id)}
								onToggle={() => data.toggleJoin(group.id)}
								onOpen={() => void goto(ROUTES.group(group.id))}
								class="h-full w-full"
							/>
						</div>
					{/each}
				</AdaptiveCardGrid>
				{#if data.filtered.length > 0 && data.filtered.length < 3}
					<SparseContentHint
						title={`只找到 ${data.filtered.length} 个附近群组`}
						description="换个分类看看，或去附近发现更多圈子"
						actionLabel="去附近看看"
						onAction={() => void goto(ROUTES.nearby)}
					/>
				{/if}
			</div>
			{#if data.hasMoreVisible}
				<div class="flex justify-center px-4 py-6">
					<InfiniteSentinel hasMore={data.hasMoreVisible} loading={false} onLoadMore={data.revealMore} />
					<InlineLoading label="加载更多…" />
				</div>
			{:else}
				<ListEnd />
			{/if}
		{:else}
			<EmptyState
				spotlight
				iconTone="accent"
				title="没有找到相关群组"
				description="换个分类看看，或去全景地图发现身边的圈子。"
			>
				{#snippet icon()}
					<Users class="size-7" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>
		{/if}
	</div>
</StandardPageLayout>

<ConfirmDialog
	open={data.leaveTarget !== null}
	title="退出群组「{data.leaveTarget?.name ?? ''}」？"
	description="退出后将不再接收该群的聊天与公告。"
	icon={LogOut}
	confirmLabel="确认退出"
	onCancel={data.dismissLeave}
	onConfirm={data.confirmLeave}
/>
