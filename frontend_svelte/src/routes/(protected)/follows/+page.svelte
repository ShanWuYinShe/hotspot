<script lang="ts">
	import { onMount } from 'svelte';
	import { usersService } from '$lib/api/services';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import UserCard from '$lib/components/common/user-card.svelte';
	import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import LoadMoreButton from '$lib/components/common/load-more-button.svelte';
	import type { FollowingItem } from '$lib/types/api';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import UserCheck from 'lucide-svelte/icons/user-check';
	import AlertTriangle from 'lucide-svelte/icons/triangle-alert';

	const pagination = usePagination<FollowingItem>({
		uniqueKey: (item) => item.user.user_id
	});

	async function loadFollowing(refresh = false) {
		await pagination.load(
			(cursor) => usersService.getFollowing({ pagination: { limit: 24, cursor } }),
			refresh
		);
	}

	onMount(() => {
		loadFollowing(true);
	});
</script>

<svelte:head>
	<title>关注 - HotSpot</title>
</svelte:head>

<div class="flex flex-col">
	<!-- Sticky header -->
	<div
		class="sticky top-0 z-10 px-5 pt-3 pb-3"
		style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
	>
		<div class="flex items-center justify-between">
			<div>
				<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">关注</h1>
				<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">你正在关注的用户</p>
			</div>
			<button
				class="flex items-center gap-1.5 rounded-full px-3 py-1.5 outline-none"
				style="background: var(--neu-bg); box-shadow: var(--neu-shadow-md); border: 1px solid var(--neu-border); font-size: 12px; color: var(--neu-text-muted);"
				onclick={() => loadFollowing(true)}
				disabled={pagination.loading}
			>
				<RefreshCw size={13} class={pagination.loading ? 'animate-spin' : ''} />
				刷新
			</button>
		</div>
	</div>

	<div class="flex flex-col px-4 pt-4 pb-6 gap-5">
		{#if pagination.loading}
			<ListSkeleton type="user" count={6} />
		{:else if pagination.error}
			<EmptyState
				icon={AlertTriangle}
				title="加载失败"
				description={pagination.error}
				actionText="重试"
				onAction={() => loadFollowing(true)}
			/>
		{:else if pagination.items.length === 0}
			<EmptyState icon={UserCheck} title="暂无关注" />
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
				onLoadMore={() => loadFollowing(false)}
			/>
		{/if}
	</div>
</div>
