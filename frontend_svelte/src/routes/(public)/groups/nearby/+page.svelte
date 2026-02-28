<script lang="ts">
	import { groupsService } from '$lib/api/services';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { useLocationAwareLoad } from '$lib/hooks/use-location-aware-load.svelte';
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

	// ✨ 使用新的 usePagination Hook
	const pagination = usePagination<GroupItem>({
		uniqueKey: (item) => item.group.group_id
	});

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	async function loadGroups(refresh = false) {
		await pagination.load(
			(cursor) => groupsService.search({
				radius: mapAnchorStore.radius * 1000,
				reference_point: getAnchor(),
				pagination: { limit: 20, cursor }
			}),
			refresh
		);
	}

	// 监听位置变化：首次立即加载，后续变化 500ms 防抖
	useLocationAwareLoad((refresh) => loadGroups(refresh));

</script>

<svelte:head>
	<title>附近群组 - HotSpot</title>
</svelte:head>

<div class="container mx-auto max-w-7xl space-y-5 px-4 py-6 sm:px-6 lg:px-8">
	<div
		class="rounded-3xl border border-border/60 bg-linear-to-br from-background via-card/80 to-muted/40 p-4 shadow-sm sm:p-5"
	>
		<div class="flex items-start justify-between gap-4">
			<div class="flex min-w-0 items-start gap-3">
				<Button variant="ghost" size="icon" onclick={() => history.back()} aria-label="返回">
					<ArrowLeft class="h-5 w-5" />
				</Button>
				<div class="min-w-0">
					<h1 class="truncate text-2xl font-bold">附近群组</h1>
					<p class="mt-1 text-sm text-muted-foreground">同城兴趣群组推荐，按距离和活跃度排序</p>
				</div>
			</div>

			<div class="hidden shrink-0 items-center gap-1 rounded-2xl bg-muted/70 px-3 py-1.5 text-xs text-muted-foreground sm:flex">
				<MapPin class="h-3.5 w-3.5" />
				<span class="max-w-40 truncate">{mapAnchorStore.displayName}</span>
			</div>
		</div>
	</div>

	{#if pagination.loading}
		<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={20}>
			{#each Array(6) as _, i (i)}
				<MasonryLayoutItem>
					<div
						class="animate-pulse rounded-2xl bg-muted"
						style="height: {200 + (i % 3) * 28}px"
					></div>
				</MasonryLayoutItem>
			{/each}
		</MasonryLayout>
	{:else if pagination.error}
		<EmptyState 
			icon={AlertTriangle} 
			title="加载失败" 
			description={pagination.error} 
			actionText="重试"
			onAction={() => loadGroups(true)}
		/>
	{:else if pagination.items.length === 0}
		<EmptyState icon={UsersRound} title="附近暂无群组" description="等待附近建立新群组" />
	{:else}
		<MasonryLayout 
			columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} 
			gap={20}
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
