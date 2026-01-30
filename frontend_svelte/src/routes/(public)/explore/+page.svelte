<script lang="ts">
	import { onMount } from 'svelte';
	import { feedService, tagsService, momentsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Badge } from '$lib/components/ui/badge';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import type {
		FeedMomentItem,
		MomentSearchItem,
		TrendingTag
	} from '$lib/types/api';
	import Hash from 'lucide-svelte/icons/hash';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Compass from 'lucide-svelte/icons/compass';
	import Sparkles from 'lucide-svelte/icons/sparkles';

	// Data states
	let trendingTags = $state([] as TrendingTag[]);
	let feedMoments = $state([] as (FeedMomentItem | MomentSearchItem)[]);

	// Loading states
	let loadingTags = $state(true);
	let loadingFeed = $state(true);
	let loadingMoreFeed = $state(false);

	// Pagination states
	let feedCursor = $state(undefined as string | undefined);
	let feedHasMore = $state(true);

	// Error states
	let errorFeed = $state(false);

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	async function loadTrendingTags() {
		loadingTags = true;
		try {
			const response = await tagsService.getTrending({ limit: 10 });
			if (isSuccess(response) && response.data) {
				trendingTags = response.data.items;
			}
		} catch {
			// ignore
		} finally {
			loadingTags = false;
		}
	}

	async function loadFeed(refresh = false) {
		if (refresh) {
			feedMoments = [];
			feedCursor = undefined;
			feedHasMore = true;
			loadingFeed = true;
			errorFeed = false;
		} else {
			loadingMoreFeed = true;
		}

		try {
			// Always load recommended/all feed for Explore page
			// If not logged in, fallback to nearby or just generic recommendation logic
			let response;
			if (authStore.isAuthenticated) {
				response = await feedService.getMoments({
					source: 'all',
					reference_point: getAnchor(),
					limit: 20,
					cursor: refresh ? undefined : feedCursor
				});
			} else {
				// Not logged in, show nearby as recommendation
				response = await momentsService.searchByLocation({
					reference_point: getAnchor(),
					radius: mapAnchorStore.radius * 1000,
					limit: 20,
					cursor: refresh ? undefined : feedCursor
				});
			}

			if (isSuccess(response) && response.data) {
				feedMoments = refresh ? response.data.items : [...feedMoments, ...response.data.items];
				feedHasMore = response.data.pagination.has_more;
				feedCursor = response.data.pagination.next_cursor;
			} else {
				errorFeed = true;
				if (refresh) toast.error('加载动态失败');
			}
		} catch {
			errorFeed = true;
			if (refresh) toast.error('加载动态失败');
		} finally {
			loadingFeed = false;
			loadingMoreFeed = false;
		}
	}

	function refreshAll() {
		loadTrendingTags();
		loadFeed(true);
	}

	$effect(() => {
		mapAnchorStore.anchor.updatedAt; // Dependency
		refreshAll();
	});

	onMount(() => {
		refreshAll();
	});
</script>

<svelte:head>
	<title>探索 - HotSpot</title>
</svelte:head>

<div class="container mx-auto max-w-7xl space-y-8 px-4 pb-20 pt-6 sm:px-6 lg:px-8">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div class="space-y-1">
			<h1 class="text-3xl font-bold tracking-tight text-foreground">探索</h1>
			<p class="text-muted-foreground">发现附近的精彩人与事</p>
		</div>
		<Button variant="ghost" size="icon" class="rounded-full" onclick={refreshAll}>
			<RefreshCw class="h-4 w-4 {loadingFeed ? 'animate-spin' : ''}" />
		</Button>
	</div>

	<!-- Discovery Sections -->
	<div class="space-y-10">
		<!-- Trending Topics -->
		{#if !loadingTags && trendingTags.length > 0}
			<section class="space-y-4">
				<div class="flex items-center gap-2 px-1">
					<Hash class="h-5 w-5 text-primary" />
					<h2 class="text-lg font-semibold tracking-tight">热门话题</h2>
				</div>
				<div class="flex flex-wrap gap-2">
					{#each trendingTags as tag}
						<a href="/search?tag={encodeURIComponent(tag.name)}" class="group">
							<Badge variant="secondary" class="cursor-pointer rounded-full px-3 py-1.5 text-sm transition-all hover:bg-primary/10 hover:text-primary">
								#{tag.name}
								<span class="ml-1.5 text-xs opacity-50 group-hover:opacity-100">{tag.usage_count}</span>
							</Badge>
						</a>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Moments Feed (Masonry) -->
		<section class="space-y-6">
			<div class="flex items-center justify-between px-1">
				<div class="flex items-center gap-2">
					<Sparkles class="h-5 w-5 text-primary" />
					<h2 class="text-lg font-semibold tracking-tight">推荐动态</h2>
				</div>
			</div>

			{#if loadingFeed}
				<div class="columns-1 gap-4 sm:columns-2 lg:columns-3">
					{#each Array(6) as _}
						<div class="mb-4 break-inside-avoid">
							<Card class="border-0 bg-card/50">
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
			{:else if errorFeed}
				<div class="flex flex-col items-center justify-center py-12 text-center">
					<p class="mb-4 text-muted-foreground">加载动态失败</p>
					<Button variant="outline" onclick={() => loadFeed(true)}>重试</Button>
				</div>
			{:else if feedMoments.length === 0}
				<div class="flex flex-col items-center justify-center py-16 text-center">
					<Compass class="mb-4 h-12 w-12 text-muted-foreground/50" />
					<h3 class="text-lg font-medium">暂无推荐</h3>
					<p class="text-muted-foreground">
						去发布一些内容，或者探索更多区域吧
					</p>
				</div>
			{:else}
				<div class="columns-1 gap-4 sm:columns-2 lg:columns-3">
					{#each feedMoments as moment (moment.moment_id)}
						<div class="mb-4 break-inside-avoid">
							<MomentCard {moment} />
						</div>
					{/each}
				</div>
				
				{#if feedHasMore}
					<div class="flex justify-center pt-8">
						<Button variant="outline" class="rounded-full px-8" onclick={() => loadFeed(false)} disabled={loadingMoreFeed}>
							{#if loadingMoreFeed}
								<RefreshCw class="mr-2 h-4 w-4 animate-spin" />
								加载中...
							{:else}
								加载更多
							{/if}
						</Button>
					</div>
				{/if}
			{/if}
		</section>
	</div>
</div>
