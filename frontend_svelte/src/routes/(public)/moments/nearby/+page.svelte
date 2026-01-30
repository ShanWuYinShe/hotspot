<script lang="ts">
	import { onMount } from 'svelte';
	import { momentsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import type { MomentSearchItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import Image from 'lucide-svelte/icons/image';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	let moments = $state([] as MomentSearchItem[]);
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

	async function loadMoments(refresh = false) {
		if (refresh) {
			moments = [];
			cursor = undefined;
			hasMore = true;
			loading = true;
		} else {
			loadingMore = true;
		}

		try {
			const response = await momentsService.searchByLocation({
				reference_point: getAnchor(),
				radius: mapAnchorStore.radius * 1000,
				limit: 20,
				cursor: refresh ? undefined : cursor
			});

			if (isSuccess(response) && response.data) {
				moments = refresh ? response.data.items : [...moments, ...response.data.items];
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
		loadMoments(true);
	});

	onMount(() => {
		loadMoments(true);
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

	{#if loading}
		<div class="columns-1 gap-4 sm:columns-2 lg:columns-3">
			{#each Array(6) as _}
				<div class="mb-4 break-inside-avoid">
					<Card>
						<CardContent class="p-4 space-y-3">
							<div class="flex items-center gap-3">
								<Skeleton class="h-10 w-10 rounded-full" />
								<div class="space-y-1.5">
									<Skeleton class="h-4 w-24" />
									<Skeleton class="h-3 w-16" />
								</div>
							</div>
							<Skeleton class="h-32 w-full rounded-lg" />
							<Skeleton class="h-4 w-full" />
							<Skeleton class="h-4 w-2/3" />
						</CardContent>
					</Card>
				</div>
			{/each}
		</div>
	{:else if moments.length === 0}
		<div class="flex flex-col items-center justify-center py-16 text-center text-muted-foreground">
			<Image class="mb-4 h-12 w-12 opacity-50" />
			<p>附近暂无动态</p>
		</div>
	{:else}
		<div class="columns-1 gap-4 sm:columns-2 lg:columns-3">
			{#each moments as moment (moment.moment_id)}
				<div class="mb-4 break-inside-avoid">
					<MomentCard {moment} />
				</div>
			{/each}
		</div>

		{#if hasMore}
			<div class="flex justify-center pt-4">
				<Button variant="outline" onclick={() => loadMoments(false)} disabled={loadingMore}>
					{#if loadingMore}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" />
					{/if}
					加载更多
				</Button>
			</div>
		{/if}
	{/if}
</div>
