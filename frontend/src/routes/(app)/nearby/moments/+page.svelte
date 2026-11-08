<script lang="ts">
	import { ArrowUp, Map as MapIcon, MapPin, Search, Sparkles } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import AdaptiveCardGrid from '@/components/layout/AdaptiveCardGrid.svelte';
	import SparseContentHint from '@/components/data-display/SparseContentHint.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import InfiniteSentinel from '@/components/list/InfiniteSentinel.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import MomentCard from '@/components/moments/MomentCard.svelte';
	import { ROUTES } from '@/lib/routes';
	import { contentFade } from '@/lib/motion';
	import { useNearbyMomentsData } from './shared/useNearbyMomentsData.svelte';

	// 附近瞬间（/nearby/moments）：纯地理轴页面（按距离排序的 nearby feed）。
	// 架构重构：数据加载、分页、进度缓存与交互状态收敛至 useNearbyMomentsData()。
	const data = useNearbyMomentsData();
	const device = $derived(platform.platform);
</script>

<svelte:head>
	<title>附近瞬间 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentClassName="pt-4 pb-6">
	{#snippet header()}
		<PageHeader
			variant="integrated"
			title="附近瞬间"
			subtitle={data.pageSubtitle}
			islandMeta={data.islandMetaText}
		>
			{#snippet islandIcon()}
				<Sparkles class="size-3.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
			{#snippet subtitleIcon()}
				<MapPin class="size-2.5 text-accent-default" strokeWidth={1.75} />
			{/snippet}
			{#snippet trailing()}
				<div class="flex items-center gap-2">
					{#if device !== 'desktop'}
						<ActionIconButton variant="ghost" icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} />
						<ActionIconButton variant="ghost" icon={MapIcon} label="地图浏览" onClick={() => void goto(ROUTES.map)} />
					{/if}
				</div>
			{/snippet}
			{#snippet islandExpanded()}
				<div class="flex flex-col gap-2.5">
					<div class="flex items-center justify-between text-xs pb-0.5">
						<span class="font-medium text-text-primary">话题筛选</span>
						{#if data.activeTag}
							<button
								type="button"
								class="text-xxs text-accent-default hover:underline"
								onclick={() => data.setTag(null)}
							>
								重置筛选
							</button>
						{/if}
					</div>
					{#if data.tags.length > 0}
						<div class="w-full overflow-x-auto no-scrollbar py-0.5">
							<TabBar
								items={[{ id: '', label: '全部' }, ...data.tags.map((t) => ({ id: t, label: t }))]}
								value={data.activeTag ?? ''}
								onChange={(id) => data.setTag(id === '' || data.activeTag === id ? null : id)}
								class="pr-1 whitespace-nowrap"
							/>
						</div>
					{/if}
					<div class="flex items-center justify-between border-t border-border-default/15 pt-2 text-xxs text-text-secondary">
						<div class="flex items-center gap-1.5 text-text-secondary/70">
							<kbd class="rounded border border-border-default/30 bg-surface-island px-1 py-0.2 font-mono text-[10px]">Esc</kbd>
							<span>收起</span>
						</div>
						<button
							type="button"
							class="group flex items-center gap-1 text-accent-default hover:underline transition-colors font-medium text-xxs"
							onclick={() => {
								const scroller = document.getElementById('main-content');
								scroller?.scrollTo({ top: 0, behavior: 'smooth' });
							}}
						>
							<span>平滑回顶</span>
							<ArrowUp class="size-3 transition-transform group-hover:-translate-y-0.5" />
						</button>
					</div>
				</div>
			{/snippet}
			{#snippet below()}
				{#if data.tags.length > 0}
					<div>
						<TabBar
							items={[{ id: '', label: '全部' }, ...data.tags.map((t) => ({ id: t, label: t }))]}
							value={data.activeTag ?? ''}
							onChange={(id) => data.setTag(id === '' || data.activeTag === id ? null : id)}
						/>
					</div>
				{/if}
			{/snippet}
		</PageHeader>
	{/snippet}

	<div class="min-w-0" use:contentFade={{ showSkeleton: data.loading, trigger: data.activeTag ?? '' }}>
		{#if data.loading}
			<SkeletonFeed
				variant="card"
				count={8}
				cardClass="h-full"
				class="grid grid-flow-dense grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4 lg:grid-cols-[repeat(auto-fill,minmax(18.75rem,1fr))] [grid-auto-rows:clamp(12.1875rem,20vw,17.1875rem)]"
				label="加载附近瞬间…"
			/>
		{:else if data.loadError}
			<LoadError message={data.loadError} onRetry={data.reload} />
		{:else if data.filtered.length > 0}
			<div class="w-full">
				<AdaptiveCardGrid
					count={data.visibleCount}
					minTrackRem={18.75}
					gapRem={[0.75, 1, 1]}
					centerSparse={data.bspLayouts.slice(0, data.visibleCount).every((l) => l.tier === '1x1')}
					class="grid-flow-dense"
					style="grid-auto-rows: clamp(12.1875rem, 20vw, 17.1875rem);"
				>
					{#each data.filtered.slice(0, data.visibleCount) as item, index (item.id)}
						{@const layout = data.bspLayouts[index] ?? { tier: '1x1', cls: 'col-span-1 row-span-1', horizontal: false, featured: false }}
						<div class={layout.cls}>
							<MomentCard
								class="h-full w-full rounded-lg"
								{item}
								priority={index < 4}
								compact={device === 'mobile'}
								tier={layout.tier}
								liked={item.liked}
								onLike={() => void data.interactions.toggleLike(item.id)}
								onClick={() => data.openMoment(item.id)}
								onCommentClick={() => data.openMoment(item.id, true)}
							/>
						</div>
					{/each}
				</AdaptiveCardGrid>

				{#if data.filtered.length > 0 && data.filtered.length < 3}
					<SparseContentHint
						title={`只找到 ${data.filtered.length} 条附近瞬间`}
						description="换个标签看看，或去全景地图探索身边的足迹"
						actionLabel="去地图探索"
						onAction={() => void goto(ROUTES.map)}
					/>
				{/if}
			</div>

			{#if data.feedHasMore}
				<div class="flex justify-center px-4 py-6">
					<InfiniteSentinel hasMore={data.feedHasMore} loading={data.loadingMore} onLoadMore={data.handleLoadMore} />
					<InlineLoading label="加载更多…" />
				</div>
			{/if}

			{#if data.showListEnd}
				<ListEnd />
			{/if}
		{:else}
			<EmptyState
				spotlight
				iconTone="accent"
				title={data.activeTag ? `没有找到带有「${data.activeTag}」的瞬间` : '附近还没有瞬间'}
				description={data.activeTag ? '换个标签看看，或去附近发现更多生活记录。' : '去全景地图看看身边正在发生什么，或者发布属于你的第一条瞬间。'}
			>
				{#snippet icon()}
					<Sparkles class="size-7" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>
		{/if}
	</div>
</StandardPageLayout>
