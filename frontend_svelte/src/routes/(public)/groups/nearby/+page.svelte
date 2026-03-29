<script lang="ts">
	import { page } from '$app/state';
	import { groupsService } from '$lib/api/services';
	import { navigateToNearbyBackTarget } from '../../nearby-back';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { useLocationAwareLoad } from '$lib/hooks/use-location-aware-load.svelte';
	import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import { Button } from '$lib/components/ui/button';
	import GroupCard from '$lib/components/common/group-card.svelte';
	import MasonryLayout from '$lib/components/common/masonry-layout.svelte';
	import MasonryLayoutItem from '$lib/components/common/masonry-layout-item.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { GroupItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

	const pagination = usePagination<GroupItem>({
		uniqueKey: (item) => item.group.group_id
	});

	let hasLoadedOnce = $state(false);

	const returnTarget = $derived(page.url.searchParams.get('from') || '/search');

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	async function loadGroups(refresh = false) {
		await pagination.load(
			(cursor) =>
				groupsService.search({
					radius: mapAnchorStore.radius * 1000,
					reference_point: getAnchor(),
					pagination: { limit: 20, cursor }
				}),
			refresh
		);
		hasLoadedOnce = true;
	}

	function handleBack() {
		navigateToNearbyBackTarget(typeof window === 'undefined' ? undefined : window, returnTarget);
	}

	function handleRetry() {
		void loadGroups(true);
	}

	useLocationAwareLoad((refresh) => loadGroups(refresh));
</script>

<svelte:head>
	<title>附近群组 - HotSpot</title>
</svelte:head>

<div class="container mx-auto max-w-7xl space-y-4 px-3 py-4 sm:space-y-5 sm:px-6 sm:py-6 lg:px-8">
	<PageHeader title="附近群组" description="同城兴趣群组推荐，按距离和活跃度排序展示。">
		{#snippet actions()}
			<Button variant="ghost" size="icon" onclick={handleBack} aria-label="返回">
				<ArrowLeft class="h-5 w-5" />
			</Button>
			<div class="inline-flex shrink-0 items-center gap-1 rounded-2xl border border-border/50 bg-muted/70 px-3 py-1.5 text-xs text-muted-foreground">
				<MapPin class="h-3.5 w-3.5" />
				<span class="max-w-48 truncate">{mapAnchorStore.displayName}</span>
			</div>
		{/snippet}
	</PageHeader>

	<AnchorSummaryCard
		label="附近发现锚点"
		description={`当前按 ${mapAnchorStore.radius}km 范围加载群组结果`}
		subtle
	/>

	{#if pagination.loading}
		<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
			{#each Array(6) as _, i (i)}
				<MasonryLayoutItem>
					<div class="animate-pulse rounded-2xl bg-muted" style="height: {200 + (i % 3) * 28}px"></div>
				</MasonryLayoutItem>
			{/each}
		</MasonryLayout>
	{:else if pagination.error}
		<EmptyState
			icon={AlertTriangle}
			title="加载失败"
			description={pagination.error}
			actionText="重试"
			onAction={handleRetry}
		/>
	{:else if pagination.items.length === 0 && hasLoadedOnce}
		<EmptyState icon={UsersRound} title="附近暂无群组" description="换个浏览位置或稍后再来看看。" />
	{:else}
		<MasonryLayout
			columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }}
			gap={16}
			animated={true}
			animationDelay={50}
		>
			{#each pagination.items as group (group.group.group_id)}
				<MasonryLayoutItem>
					<GroupCard
						{group}
						showDistance
						showMembers
						variant="square"
						class="overflow-hidden border-border/60 bg-card/95 shadow-md shadow-black/5 dark:shadow-black/30"
					/>
				</MasonryLayoutItem>
			{/each}
		</MasonryLayout>

		<LoadMoreButton
			loading={pagination.loadingMore}
			hasMore={pagination.hasMore}
			onLoadMore={() => loadGroups(false)}
		/>
	{/if}
</div>
