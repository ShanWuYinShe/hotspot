<script lang="ts">
	import { page } from '$app/state';
	import { momentsService } from '$lib/api/services';
	import { navigateToNearbyBackTarget } from '../../nearby-back';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { useLocationAwareLoad } from '$lib/hooks/use-location-aware-load.svelte';
	import AnchorSummaryCard from '$lib/components/common/anchor-summary-card.svelte';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import { Button } from '$lib/components/ui/button';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import MasonryLayout from '$lib/components/common/masonry-layout.svelte';
	import MasonryLayoutItem from '$lib/components/common/masonry-layout-item.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { MomentItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Image from 'lucide-svelte/icons/image';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

	const pagination = usePagination<MomentItem>({
		uniqueKey: 'moment_id'
	});

	let hasLoadedOnce = $state(false);

	const returnTarget = $derived(page.url.searchParams.get('from') || '/search');

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	async function loadMoments(refresh = false) {
		await pagination.load(
			(cursor) =>
				momentsService.search({
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
		void loadMoments(true);
	}

	useLocationAwareLoad((refresh) => loadMoments(refresh));
</script>

<svelte:head>
	<title>附近动态 - HotSpot</title>
</svelte:head>

<div class="container mx-auto max-w-7xl space-y-4 px-3 py-4 sm:space-y-5 sm:px-6 sm:py-6 lg:px-8">
	<PageHeader title="附近动态" description="附近用户刚刚发布的内容，按热度和时间综合推荐。">
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
		description={`当前按 ${mapAnchorStore.radius}km 范围加载动态结果`}
		subtle
	/>

	{#if pagination.loading}
		<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
			{#each Array(6) as _, i (i)}
				<MasonryLayoutItem>
					<div class="animate-pulse rounded-2xl bg-muted" style="height: {220 + (i % 3) * 34}px"></div>
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
		<EmptyState icon={Image} title="附近暂无动态" description="换个浏览位置或稍后再来看看。" />
	{:else}
		<MasonryLayout
			columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }}
			gap={16}
			animated={true}
			animationDelay={50}
		>
			{#each pagination.items as moment (moment.moment_id)}
				<MasonryLayoutItem>
					<MomentCard
						{moment}
						class="border-border/60 bg-card/95 shadow-md shadow-black/5 dark:shadow-black/30"
					/>
				</MasonryLayoutItem>
			{/each}
		</MasonryLayout>

		<LoadMoreButton
			loading={pagination.loadingMore}
			hasMore={pagination.hasMore}
			onLoadMore={() => loadMoments(false)}
		/>
	{/if}
</div>
