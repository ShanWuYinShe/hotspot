<script lang="ts">
	import { onMount } from 'svelte';
	import { usersService } from '$lib/api/services';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import UserCard from '$lib/components/common/user-card.svelte';
	import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { FollowerItem } from '$lib/types/api';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

	const pagination = usePagination<FollowerItem>({
		uniqueKey: (item) => item.user.user_id
	});

	async function loadFollowers(refresh = false) {
		await pagination.load(
			(cursor) => usersService.getFollowers({ pagination: { limit: 24, cursor } }),
			refresh
		);
	}

	onMount(() => {
		loadFollowers(true);
	});
</script>

<svelte:head>
	<title>粉丝 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">粉丝</h1>
			<p class="text-sm text-muted-foreground">关注你的用户</p>
		</div>
		<Button variant="outline" size="sm" onclick={() => loadFollowers(true)} disabled={pagination.loading}>
			<RefreshCw class="mr-2 h-4 w-4 {pagination.loading ? 'animate-spin' : ''}" />
			刷新
		</Button>
	</div>

	{#if pagination.loading}
		<ListSkeleton type="user" count={6} />
	{:else if pagination.error}
		<EmptyState 
			icon={AlertTriangle} 
			title="加载失败" 
			description={pagination.error} 
			actionText="重试"
			onAction={() => loadFollowers(true)}
		/>
	{:else if pagination.items.length === 0}
		<EmptyState icon={UserPlus} title="暂无粉丝" />
	{:else}
		<div class="columns-1 gap-3 sm:columns-2 lg:columns-3 xl:columns-4">
			{#each pagination.items as item (item.user.user_id)}
				<div class="mb-3 break-inside-avoid">
					<UserCard user={item.user} showBio showTags variant="square" />
				</div>
			{/each}
		</div>
		<LoadMoreButton
			loading={pagination.loadingMore}
			hasMore={pagination.hasMore}
			onLoadMore={() => loadFollowers(false)}
		/>
	{/if}
</div>
