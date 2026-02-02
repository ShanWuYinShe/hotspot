<script lang="ts">
	import { momentsService } from '$lib/api/services';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import MasonryLayout from '$lib/components/common/masonry-layout.svelte';
	import MasonryLayoutItem from '$lib/components/common/masonry-layout-item.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { MomentSearchItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import Image from 'lucide-svelte/icons/image';

	// ✨ 使用新的 usePagination Hook
	const pagination = usePagination<MomentSearchItem>({
		uniqueKey: 'moment_id'
	});

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	async function loadMoments(refresh = false) {
		await pagination.load(
			(cursor) => momentsService.searchByLocation({
				reference_point: getAnchor(),
				radius: mapAnchorStore.radius * 1000,
				limit: 20,
				cursor
			}),
			refresh
		);
	}

	// 监听位置变化：首次立即加载，后续变化 500ms 防抖
	let prevUpdatedAt: number | undefined = $state(undefined);
	let locationDebounceTimer: ReturnType<typeof setTimeout> | null = null;

	$effect(() => {
		const now = mapAnchorStore.anchor.updatedAt;
		if (prevUpdatedAt === undefined) {
			prevUpdatedAt = now;
			void loadMoments();
			return () => {};
		}
		prevUpdatedAt = now;
		if (locationDebounceTimer) clearTimeout(locationDebounceTimer);
		locationDebounceTimer = setTimeout(() => {
			void loadMoments(true);
		}, 500);
		return () => {
			if (locationDebounceTimer) {
				clearTimeout(locationDebounceTimer);
				locationDebounceTimer = null;
			}
		};
	});
</script>

<svelte:head>
	<title>附近动态 - HotSpot</title>
</svelte:head>

<div class="container mx-auto max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8">
	<div class="flex items-center gap-4">
		<Button variant="ghost" size="icon" onclick={() => history.back()}>
			<ArrowLeft class="h-5 w-5" />
		</Button>
		<h1 class="text-2xl font-bold">附近动态</h1>
	</div>

	{#if pagination.loading}
		<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={20}>
			{#each Array(6) as _}
				<MasonryLayoutItem>
					<div class="h-64 animate-pulse rounded-2xl bg-muted"></div>
				</MasonryLayoutItem>
			{/each}
		</MasonryLayout>
	{:else if pagination.items.length === 0}
		<EmptyState icon={Image} title="附近暂无动态" description="等待附近的人发布动态" />
	{:else}
		<MasonryLayout 
			columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} 
			gap={20}
			animated={true}
			animationDelay={50}
		>
			{#each pagination.items as moment (moment.moment_id)}
				<MasonryLayoutItem>
					<MomentCard {moment} />
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
