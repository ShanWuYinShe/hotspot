<script lang="ts">
	import { onMount } from 'svelte';
	import { momentsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import CreateMomentModal from '$lib/components/common/create-moment-modal.svelte';
	import type { UserMomentListItem } from '$lib/types/api';
	import Plus from 'lucide-svelte/icons/plus';
	import Image from 'lucide-svelte/icons/image';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';

	let moments = $state<UserMomentListItem[]>([]);
	let isLoading = $state(true);
	let isLoadingMore = $state(false);
	let hasMore = $state(true);
	let cursor = $state<string | undefined>(undefined);
	let showCreateMoment = $state(false);

	async function loadMoments(refresh = false) {
		if (refresh) {
			cursor = undefined;
			hasMore = true;
			isLoading = true;
		} else {
			isLoadingMore = true;
		}

		try {
			const response = await momentsService.getMyMoments({
				limit: 20,
				cursor: refresh ? undefined : cursor
			});

			if (isSuccess(response) && response.data) {
				if (refresh) {
					moments = response.data.items;
				} else {
					moments = [...moments, ...response.data.items];
				}
				hasMore = response.data.pagination.has_more;
				cursor = response.data.pagination.next_cursor;
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
			isLoadingMore = false;
		}
	}

	onMount(() => {
		loadMoments(true);
	});
</script>

<svelte:head>
	<title>我的瞬间 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
	<div class="flex items-center justify-between">
		<h1 class="text-2xl font-bold">我的瞬间</h1>
		<div class="flex gap-2">
			<Button variant="outline" size="sm" onclick={() => loadMoments(true)} disabled={isLoading}>
				<RefreshCw class="mr-2 h-4 w-4 {isLoading ? 'animate-spin' : ''}" />
				刷新
			</Button>
			<Button size="sm" onclick={() => (showCreateMoment = true)}>
				<Plus class="mr-2 h-4 w-4" />
				发布
			</Button>
		</div>
	</div>

	{#if isLoading}
		<div class="space-y-4">
			{#each Array(3) as _}
				<Card>
					<CardContent class="p-4">
						<div class="space-y-3">
							<Skeleton class="h-4 w-full" />
							<Skeleton class="h-4 w-3/4" />
							<Skeleton class="h-32 w-full" />
						</div>
					</CardContent>
				</Card>
			{/each}
		</div>
	{:else if moments.length === 0}
		<Card>
			<CardContent class="flex flex-col items-center justify-center py-12">
				<Image class="mb-4 h-12 w-12 text-muted-foreground" />
				<p class="mb-2 text-lg font-medium">还没有瞬间</p>
				<p class="mb-4 text-muted-foreground">发布你的第一个瞬间吧</p>
				<Button onclick={() => (showCreateMoment = true)}>
					<Plus class="mr-2 h-4 w-4" />
					发布瞬间
				</Button>
			</CardContent>
		</Card>
	{:else}
		<div class="space-y-4">
			<div class="columns-1 gap-4 sm:columns-2 xl:columns-3">
				{#each moments as moment (moment.moment_id)}
					<div class="mb-4 break-inside-avoid">
						<Card class="overflow-hidden">
							<a href="/moments/{moment.moment_id}" class="block p-4 hover:bg-accent/50">
								<p class="line-clamp-3 whitespace-pre-wrap">{moment.description}</p>
								{#if moment.images && moment.images.length > 0}
									<div class="mt-3 flex gap-2 overflow-x-auto">
										{#each moment.images.slice(0, 4) as image, i}
											<img
												src={image.url}
												alt="图片 {i + 1}"
												class="h-20 w-20 rounded object-cover"
											/>
										{/each}
										{#if moment.images.length > 4}
											<div class="flex h-20 w-20 items-center justify-center rounded bg-muted text-sm text-muted-foreground">
												+{moment.images.length - 4}
											</div>
										{/if}
									</div>
								{/if}
								<div class="mt-3 flex items-center gap-4 text-sm text-muted-foreground">
									<span>{moment.likes_count} 点赞</span>
									<span>{moment.comments_count} 评论</span>
									<span>{new Date(moment.created_at).toLocaleDateString('zh-CN')}</span>
								</div>
							</a>
						</Card>
					</div>
				{/each}
			</div>

			{#if hasMore}
				<div class="flex justify-center">
					<Button
						variant="outline"
						onclick={() => loadMoments(false)}
						disabled={isLoadingMore}
					>
						{#if isLoadingMore}
							<RefreshCw class="mr-2 h-4 w-4 animate-spin" />
						{/if}
						加载更多
					</Button>
				</div>
			{/if}
		</div>
	{/if}
</div>

<CreateMomentModal bind:open={showCreateMoment} onCreated={() => loadMoments(true)} />
