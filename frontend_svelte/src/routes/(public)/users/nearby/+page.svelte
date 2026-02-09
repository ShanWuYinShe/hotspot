<script lang="ts">
	import { usersService } from '$lib/api/services';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import UserCard from '$lib/components/common/user-card.svelte';
	import MasonryLayout from '$lib/components/common/masonry-layout.svelte';
	import MasonryLayoutItem from '$lib/components/common/masonry-layout-item.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { UserItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
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
				search: {
					reference_point: getAnchor(),
					pagination: { limit: 20, cursor }
				},
				radius: mapAnchorStore.radius * 1000
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
			void loadUsers();
			return () => {};
		}
		prevUpdatedAt = now;
		if (locationDebounceTimer) clearTimeout(locationDebounceTimer);
		locationDebounceTimer = setTimeout(() => {
			void loadUsers(true);
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
	<title>附近的人 - HotSpot</title>
</svelte:head>

<div class="container mx-auto max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8">
	<div class="flex items-center gap-4">
		<Button variant="ghost" size="icon" onclick={() => history.back()} aria-label="返回">
			<ArrowLeft class="h-5 w-5" />
		</Button>
		<h1 class="text-2xl font-bold">附近的人</h1>
	</div>

	{#if pagination.loading}
		<MasonryLayout columns={{ mobile: 1, tablet: 2, desktop: 3, wide: 4 }} gap={16}>
			{#each Array(6) as _}
				<MasonryLayoutItem>
					<div class="h-48 animate-pulse rounded-2xl bg-muted"></div>
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
			gap={16}
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
