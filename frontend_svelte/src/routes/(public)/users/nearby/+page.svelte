<script lang="ts">
	import { usersService } from '$lib/api/services';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { useLocationAwareLoad } from '$lib/hooks/use-location-aware-load.svelte';
	import { Button } from '$lib/components/ui/button';
	import UserCard from '$lib/components/common/user-card.svelte';
	import MasonryLayout from '$lib/components/common/masonry-layout.svelte';
	import MasonryLayoutItem from '$lib/components/common/masonry-layout-item.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { UserItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Users from 'lucide-svelte/icons/users';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

	// ✨ 使用新的 usePagination Hook
	const pagination = usePagination<UserItem>({
		uniqueKey: (item) => item.user.user_id
	});

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	async function loadUsers(refresh = false) {
		await pagination.load(
			(cursor) => usersService.search({
				radius: mapAnchorStore.radius * 1000,
				reference_point: getAnchor(),
				pagination: { limit: 20, cursor }
			}),
			refresh
		);
	}

	// 监听位置变化：首次立即加载，后续变化 500ms 防抖
	useLocationAwareLoad((refresh) => loadUsers(refresh));

</script>

<svelte:head>
	<title>附近的人 - HotSpot</title>
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
					<h1 class="truncate text-2xl font-bold">附近的人</h1>
					<p class="mt-1 text-sm text-muted-foreground">基于当前浏览位置，按距离排序展示可互动用户</p>
				</div>
			</div>

			<div class="hidden shrink-0 items-center gap-1 rounded-2xl border border-border/50 bg-muted/70 px-3 py-1.5 text-xs text-muted-foreground sm:flex">
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
						style="height: {180 + (i % 3) * 30}px"
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
			onAction={() => loadUsers(true)}
		/>
	{:else if pagination.items.length === 0}
		<EmptyState icon={Users} title="附近暂无其他用户" description="等待附近的人加入" />
	{:else}
		<MasonryLayout 
			columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} 
			gap={20}
			animated={true}
			animationDelay={50}
		>
			{#each pagination.items as user (user.user.user_id)}
				<MasonryLayoutItem>
					<UserCard
						{user}
						showDistance
						showBio
						showTags
						showOnlineStatus
						variant="square"
						class="overflow-hidden border-border/60 bg-card/95 shadow-md shadow-black/5 dark:shadow-black/30"
					/>
				</MasonryLayoutItem>
			{/each}
		</MasonryLayout>

		<LoadMoreButton
			loading={pagination.loadingMore}
			hasMore={pagination.hasMore}
			onLoadMore={() => loadUsers(false)}
		/>
	{/if}
</div>
