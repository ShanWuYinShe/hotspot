<script lang="ts">
	import { onMount } from 'svelte';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import type { UserSearchItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Users from 'lucide-svelte/icons/users';

	let users = $state([] as UserSearchItem[]);
	let loading = $state(true);
	let loadingMore = $state(false);
	let cursor = $state(undefined as string | undefined);
	let hasMore = $state(true);

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	function formatDistance(meters: number): string {
		if (meters < 1000) return `${Math.round(meters)}m`;
		return `${(meters / 1000).toFixed(1)}km`;
	}

	async function loadUsers(refresh = false) {
		if (refresh) {
			users = [];
			cursor = undefined;
			hasMore = true;
			loading = true;
		} else {
			loadingMore = true;
		}

		try {
			const response = await usersService.searchByLocation({
				reference_point: getAnchor(),
				radius: mapAnchorStore.radius * 1000,
				limit: 20,
				cursor: refresh ? undefined : cursor
			});

			if (isSuccess(response) && response.data) {
				users = refresh ? response.data.items : [...users, ...response.data.items];
				hasMore = response.data.pagination.has_more;
				cursor = response.data.pagination.next_cursor;
			} else {
				toast.error('加载失败');
			}
		} catch {
			toast.error('加载失败');
		} finally {
			loading = false;
			loadingMore = false;
		}
	}

	$effect(() => {
		mapAnchorStore.anchor.updatedAt;
		loadUsers(true);
	});

	onMount(() => {
		loadUsers(true);
	});
</script>

<svelte:head>
	<title>附近的人 - HotSpot</title>
</svelte:head>

<div class="container mx-auto max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8">
	<div class="flex items-center gap-4">
		<Button variant="ghost" size="icon" onclick={() => history.back()}>
			<ArrowLeft class="h-5 w-5" />
		</Button>
		<h1 class="text-2xl font-bold">附近的人</h1>
	</div>

	{#if loading}
		<div class="columns-1 gap-4 sm:columns-2 lg:columns-3">
			{#each Array(6) as _}
				<div class="mb-4 break-inside-avoid">
					<Card>
						<CardContent class="flex items-center gap-4 p-4">
							<Skeleton class="h-12 w-12 rounded-full" />
							<div class="flex-1 space-y-2">
								<Skeleton class="h-4 w-24" />
								<Skeleton class="h-3 w-full" />
							</div>
						</CardContent>
					</Card>
				</div>
			{/each}
		</div>
	{:else if users.length === 0}
		<div class="flex flex-col items-center justify-center py-16 text-center text-muted-foreground">
			<Users class="mb-4 h-12 w-12 opacity-50" />
			<p>附近暂无其他用户</p>
		</div>
	{:else}
		<div class="columns-1 gap-4 sm:columns-2 lg:columns-3">
			{#each users as user (user.user_id)}
				<div class="mb-4 break-inside-avoid">
					<a href="/users/{user.user_id}">
						<Card class="group/card cursor-pointer overflow-hidden border border-border/60 bg-card/90 dark:bg-card/30 backdrop-blur-md transition-all duration-300 hover:border-primary/50 hover:bg-card hover:shadow-lg hover:shadow-primary/10 hover:-translate-y-1 dark:hover:bg-card/60 dark:hover:border-primary/40">
							<CardContent class="p-4">
								<div class="flex items-start gap-4">
									<div class="relative shrink-0">
										<Avatar class="h-12 w-12 ring-2 ring-transparent transition-all duration-300 group-hover/card:ring-primary/20">
											<AvatarImage src={user.avatar_url} alt={user.nickname} />
											<AvatarFallback class="bg-primary/5 text-primary font-medium">{user.nickname.slice(0, 2)}</AvatarFallback>
										</Avatar>
										{#if user.is_online}
											<span class="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-background bg-green-500 shadow-sm"></span>
										{/if}
									</div>
									<div class="min-w-0 flex-1 space-y-1">
										<div class="flex items-center justify-between gap-2">
											<p class="truncate font-semibold text-foreground transition-colors group-hover/card:text-primary">{user.nickname}</p>
											{#if user.location}
												<span class="flex shrink-0 items-center gap-0.5 rounded-full bg-muted/50 px-1.5 py-0.5 text-[10px] text-muted-foreground">
													<MapPin class="h-3 w-3" />
													{formatDistance(user.location.distance)}
												</span>
											{/if}
										</div>
										
										{#if user.bio}
											<p class="line-clamp-2 text-sm text-muted-foreground/80">{user.bio}</p>
										{/if}
										
										{#if user.tags && user.tags.length > 0}
											<div class="mt-2 flex flex-wrap gap-1.5 pt-1">
												{#each user.tags.slice(0, 3) as tag}
													<Badge variant="outline" class="rounded-md border-primary/10 bg-primary/5 px-1.5 py-0 text-[10px] font-normal text-primary/80">#{tag}</Badge>
												{/each}
											</div>
										{/if}
									</div>
								</div>
							</CardContent>
						</Card>
					</a>
				</div>
			{/each}
		</div>

		{#if hasMore}
			<div class="flex justify-center pt-4">
				<Button variant="outline" onclick={() => loadUsers(false)} disabled={loadingMore}>
					{#if loadingMore}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					加载更多
				</Button>
			</div>
		{/if}
	{/if}
</div>
