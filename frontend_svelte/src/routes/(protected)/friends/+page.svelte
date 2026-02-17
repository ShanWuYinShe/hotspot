<script lang="ts">
	import { onMount } from 'svelte';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { FriendItem } from '$lib/types/api';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Users from 'lucide-svelte/icons/users';

	let friends = $state.raw<FriendItem[]>([]);
	let isLoading = $state(true);
	let isLoadingMore = $state(false);
	let cursor = $state<string | undefined>(undefined);
	let hasMore = $state(true);

	async function loadFriends(refresh = false) {
		if (refresh) {
			isLoading = true;
			cursor = undefined;
			hasMore = true;
		} else {
			isLoadingMore = true;
		}

		try {
			const response = refresh
				? await withMinimumLoadingTime(() =>
						usersService.getFriends({
							pagination: { limit: 24, cursor: undefined }
						})
					)
				: await usersService.getFriends({
						pagination: { limit: 24, cursor }
					});
			if (isSuccess(response) && response.data) {
				friends = refresh ? response.data.items : [...friends, ...response.data.items];
				hasMore = !!response.data.next_cursor;
				cursor = response.data.next_cursor;
			}
		} catch {
			toast.error('加载好友失败');
		} finally {
			isLoading = false;
			isLoadingMore = false;
		}
	}

	onMount(() => {
		loadFriends(true);
	});
</script>

<svelte:head>
	<title>好友 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-2xl font-bold">好友</h1>
			<p class="text-sm text-muted-foreground">你们的互相关注列表</p>
		</div>
		<Button variant="outline" size="sm" onclick={() => loadFriends(true)} disabled={isLoading}>
			<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
			刷新
		</Button>
	</div>

	{#if isLoading}
		<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
			{#each Array(6) as _, i (i)}
				<Card>
					<CardContent class="flex items-center gap-3 p-4">
						<Skeleton class="h-12 w-12 rounded-full" />
						<div class="flex-1 space-y-2">
							<Skeleton class="h-4 w-24" />
							<Skeleton class="h-3 w-32" />
						</div>
					</CardContent>
				</Card>
			{/each}
		</div>
	{:else if friends.length === 0}
		<Card>
			<CardContent class="flex flex-col items-center justify-center py-12">
				<Users class="mb-4 h-10 w-10 text-muted-foreground" />
				<p class="text-muted-foreground">暂无好友</p>
			</CardContent>
		</Card>
	{:else}
		<div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
			{#each friends as friend (friend.user.user_id)}
				<a href="/users/{friend.user.user_id}">
					<Card class="cursor-pointer border-0 bg-card/70 shadow-sm shadow-black/5 transition-shadow hover:shadow-lg hover:shadow-black/10">
						<CardContent class="flex items-center gap-3 p-4">
							<Avatar class="h-12 w-12">
								<AvatarImage src={friend.user.avatar_url} alt={friend.user.nickname} />
								<AvatarFallback>{friend.user.nickname.slice(0, 2)}</AvatarFallback>
							</Avatar>
							<div class="min-w-0">
								<p class="truncate font-medium">{friend.user.nickname}</p>
								<p class="text-xs text-muted-foreground">
									{friend.remark ? `备注：${friend.remark}` : '互相关注'}
								</p>
							</div>
						</CardContent>
					</Card>
				</a>
			{/each}
		</div>
		{#if hasMore}
			<div class="flex justify-center pt-4">
				<Button variant="outline" onclick={() => loadFriends(false)} disabled={isLoadingMore}>
					{#if isLoadingMore}
						<RefreshCw class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					加载更多
				</Button>
			</div>
		{/if}
	{/if}
</div>
