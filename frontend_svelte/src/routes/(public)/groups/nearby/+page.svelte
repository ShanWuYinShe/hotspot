<script lang="ts">
	import { groupsService } from '$lib/api/services';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import GroupCard from '$lib/components/common/group-card.svelte';
	import MasonryLayout from '$lib/components/common/masonry-layout.svelte';
	import MasonryLayoutItem from '$lib/components/common/masonry-layout-item.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { GroupItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
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
	let prevUpdatedAt: number | undefined = $state(undefined);
	let locationDebounceTimer: ReturnType<typeof setTimeout> | null = null;

	$effect(() => {
		const now = mapAnchorStore.anchor.updatedAt;
		if (prevUpdatedAt === undefined) {
			prevUpdatedAt = now;
			void loadGroups();
			return () => {};
		}
		prevUpdatedAt = now;
		if (locationDebounceTimer) clearTimeout(locationDebounceTimer);
		locationDebounceTimer = setTimeout(() => {
			void loadGroups(true);
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
	<title>附近群组 - HotSpot</title>
</svelte:head>

<div class="container mx-auto max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8">
	<div class="flex items-center gap-4">
		<Button variant="ghost" size="icon" onclick={() => history.back()} aria-label="返回">
			<ArrowLeft class="h-5 w-5" />
		</Button>
		<h1 class="text-2xl font-bold">附近群组</h1>
	</div>

	{#if pagination.loading}
		<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
			{#each Array(6) as _}
				<MasonryLayoutItem>
					<div class="h-52 animate-pulse rounded-2xl bg-muted"></div>
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
