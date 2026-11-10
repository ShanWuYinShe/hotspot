<script lang="ts">
	import { ArrowUp, Map as MapIcon, MapPin, Search, Users } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { platform } from '@/stores/platform.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import StandardPageLayout from '@/components/layout/page/StandardPageLayout.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import AdaptiveCardGrid from '@/components/layout/AdaptiveCardGrid.svelte';
	import SparseContentHint from '@/components/data-display/SparseContentHint.svelte';
	import Select from '@/components/forms/Select.svelte';
	import Button from '@/components/ui/button.svelte';
	import SkeletonFeed from '@/components/data-display/SkeletonFeed.svelte';
	import InlineLoading from '@/components/data-display/InlineLoading.svelte';
	import InfiniteSentinel from '@/components/list/InfiniteSentinel.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import ListEnd from '@/components/data-display/ListEnd.svelte';
	import UserCard from '@/components/data-display/UserCard.svelte';
	import { ROUTES } from '@/lib/routes';
	import { contentFade } from '@/lib/motion';
	import { NEARBY_CATEGORIES } from '../shared/categories';
	import { useNearbyPeopleData } from './shared/useNearbyPeopleData.svelte';

	// 附近的人：独立路由页（/nearby/people）。
	// 架构重构：数据加载、锚点侦测、会话进度与关注交互收敛至 useNearbyPeopleData()。
	const data = useNearbyPeopleData();
	const device = $derived(platform.platform);
</script>

<svelte:head>
	<title>附近的人 · HotSpot</title>
</svelte:head>

<StandardPageLayout contentClassName="pt-3 lg:pt-4 pb-6">
	{#snippet header()}
		<PageHeader
			variant="integrated"
			title="附近的人"
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
				<div class="flex items-center gap-3">
					{#if device !== 'desktop'}
						<ActionIconButton variant="ghost" icon={Search} label="搜索" onClick={() => void goto(ROUTES.search)} />
						<ActionIconButton variant="ghost" icon={MapIcon} label="地图浏览" onClick={() => void goto(ROUTES.map)} />
					{/if}
				</div>
			{/snippet}
			{#snippet islandExpanded()}
				<div class="flex flex-col gap-3">
					<div class="flex items-center justify-between text-xs pb-0.5">
						<span class="font-medium text-text-primary">分类筛选</span>
						<label class="inline-flex shrink-0 items-center gap-1.5 text-xs text-text-secondary">
							<span>排序</span>
							<Select value={data.effectiveSort} onChange={(v) => data.setSort(v)} ariaLabel="排序方式" variant="ghost">
								{#each data.sortOptions as option (option)}
									<option value={option}>{option}</option>
								{/each}
							</Select>
						</label>
					</div>

					<div class="w-full">
						<TabBar
							items={NEARBY_CATEGORIES.map((t) => ({ id: t, label: t }))}
							value={data.tagFilter}
							onChange={(id) => data.setTagFilter(id)}
							class="flex-wrap gap-1.5"
						/>
					</div>
				</div>
			{/snippet}
			{#snippet below()}
				<div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between sm:gap-3">
					<div class="min-w-0 flex-1">
						<TabBar
							items={NEARBY_CATEGORIES.map((t) => ({ id: t, label: t }))}
							value={data.tagFilter}
							onChange={(id) => data.setTagFilter(id)}
							class={device === 'desktop' ? 'flex-wrap' : 'pr-1'}
						/>
					</div>
					<label class="inline-flex shrink-0 items-center gap-1.5 self-end text-xs text-text-secondary sm:self-auto">
						<span>排序</span>
						<Select value={data.effectiveSort} onChange={(v) => data.setSort(v)} ariaLabel="排序方式" variant="ghost">
							{#each data.sortOptions as option (option)}
								<option value={option}>{option}</option>
							{/each}
						</Select>
					</label>
				</div>
			{/snippet}
		</PageHeader>
	{/snippet}

	<div>
		<div class="min-w-0" use:contentFade={{ showSkeleton: data.loading, trigger: data.filterTrigger }}>
			{#if data.loadError}
				<LoadError message={data.loadError} onRetry={data.reload} />
			{:else if data.loading}
				<SkeletonFeed
					variant="portrait"
					count={10}
					cardClass="h-full"
					class="grid grid-flow-dense grid-cols-2 gap-3 px-1 max-sm:px-0 sm:grid-cols-3 sm:gap-3.5 lg:grid-cols-[repeat(auto-fill,minmax(18.75rem,1fr))] [grid-auto-rows:clamp(12.1875rem,20vw,17.1875rem)]"
					label="加载附近用户…"
				/>
			{:else if !data.list.length}
				<EmptyState
					spotlight
					iconTone="accent"
					title={data.anchor ? (data.anchor.name ? `「${data.anchor.name}」附近暂时没有用户` : '当前查看的位置附近暂时没有用户') : '附近暂时没有匹配的用户'}
					description={data.anchor ? '正在按你设置的浏览位置筛选 5km 范围。' : undefined}
				>
					{#snippet icon()}
						<Users class="size-7" strokeWidth={1.5} />
					{/snippet}
					{#snippet action()}
						{#if data.anchor}
							<Button
								type="button"
								variant="secondary"
								size="sm"
								onclick={data.clearAnchor}
							>
								改用默认位置{data.defaultLocationName ? `（${data.defaultLocationName}）` : ''}
							</Button>
						{/if}
					{/snippet}
				</EmptyState>
			{:else}
				<div class="w-full">
					<AdaptiveCardGrid
						count={data.visibleUsers.length}
						minTrackRem={18.75}
						gapRem={[0.75, 0.875, 0.875]}
						centerSparse={data.bspLayouts.slice(0, data.visibleUsers.length).every((l) => l.tier === '1x1')}
						class="grid-flow-dense px-1 max-sm:px-0"
						style="grid-auto-rows: clamp(12.1875rem, 20vw, 17.1875rem);"
					>
						{#each data.visibleUsers as user, index (user.id)}
							{@const layout = data.bspLayouts[index] ?? { tier: '1x1', cls: 'col-span-1 row-span-1' }}
							<div class={layout.cls}>
								<UserCard
									{user}
									priority={index < 3}
									tier={layout.tier}
									followed={data.followedIds.has(user.id)}
									pending={data.followPending.has(user.id)}
									onFollow={() => data.toggleFollow(user.id)}
									onNavigate={() => void goto(ROUTES.user(user.id))}
									class="h-full w-full"
								/>
							</div>
						{/each}
					</AdaptiveCardGrid>
					{#if data.list.length > 0 && data.list.length < 3}
						<SparseContentHint
							title={`只找到 ${data.list.length} 位附近的人`}
							description="试试其他标签，或去附近看看更多人与群组"
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
				{:else if data.visibleCount >= data.list.length}
					<ListEnd />
				{/if}
			{/if}
		</div>
	</div>
</StandardPageLayout>
