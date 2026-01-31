<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { feedService, tagsService, usersService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Badge } from '$lib/components/ui/badge';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import type { UserStats, FeedMomentItem, TrendingTag, UserSearchItem, GroupSearchItem } from '$lib/types/api';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Users from 'lucide-svelte/icons/users';
	import Image from 'lucide-svelte/icons/image';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import ArrowRight from 'lucide-svelte/icons/arrow-right';
	import Sparkles from 'lucide-svelte/icons/sparkles';
	import Compass from 'lucide-svelte/icons/compass';
	import TrendingUp from 'lucide-svelte/icons/trending-up';
	import Heart from 'lucide-svelte/icons/heart';
	import Hash from 'lucide-svelte/icons/hash';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';

	const features = [
		{
			icon: MapPin,
			title: '基于位置',
			description: '发现你附近的精彩内容和有趣的人',
			color: 'from-blue-500/20 to-cyan-500/20',
			iconColor: 'text-blue-500'
		},
		{
			icon: Image,
			title: '生活瞬间',
			description: '分享你的生活点滴，记录美好时刻',
			color: 'from-pink-500/20 to-rose-500/20',
			iconColor: 'text-pink-500'
		},
		{
			icon: Users,
			title: '群组互动',
			description: '加入兴趣群组，结识志同道合的朋友',
			color: 'from-purple-500/20 to-violet-500/20',
			iconColor: 'text-purple-500'
		},
		{
			icon: MessageCircle,
			title: '即时聊天',
			description: '与好友实时沟通，保持紧密联系',
			color: 'from-green-500/20 to-emerald-500/20',
			iconColor: 'text-green-500'
		}
	];

	// 已登录用户的状态
	let stats = $state<UserStats | null>(null);
	let statsLoading = $state(false);
	
	// 探索内容状态
	let trendingTags = $state<TrendingTag[]>([]);
	let recommendedUsers = $state<UserSearchItem[]>([]);
	let recommendedGroups = $state<GroupSearchItem[]>([]);
	let feedMoments = $state<FeedMomentItem[]>([]);
	
	// 加载状态
	let loadingTags = $state(false);
	let loadingUsers = $state(false);
	let loadingGroups = $state(false);
	let loadingFeed = $state(false);
	let loadingMoreFeed = $state(false);
	
	// 分页状态
	let feedCursor = $state<string | undefined>(undefined);
	let feedHasMore = $state(true);

	async function loadStats() {
		if (!authStore.isAuthenticated) return;
		statsLoading = true;
		try {
			const response = await usersService.getMyStats();
			if (isSuccess(response) && response.data) {
				stats = response.data;
			}
		} catch {
			// ignore
		} finally {
			statsLoading = false;
		}
	}

	async function loadTrendingTags() {
		loadingTags = true;
		try {
			const response = await tagsService.getTrending({ limit: 10 });
			if (isSuccess(response) && response.data) {
				trendingTags = response.data.items ?? [];
			}
		} catch {
			// ignore
		} finally {
			loadingTags = false;
		}
	}

	async function loadRecommendedUsers() {
		loadingUsers = true;
		try {
			const response = await usersService.searchByLocation({
				reference_point: {
					latitude: mapAnchorStore.latitude,
					longitude: mapAnchorStore.longitude
				},
				radius: mapAnchorStore.radius * 1000,
				limit: 6
			});
			if (isSuccess(response) && response.data) {
				recommendedUsers = response.data.items ?? [];
			}
		} catch {
			// ignore
		} finally {
			loadingUsers = false;
		}
	}

	async function loadRecommendedGroups() {
		loadingGroups = true;
		try {
			const response = await groupsService.searchByLocation({
				reference_point: {
					latitude: mapAnchorStore.latitude,
					longitude: mapAnchorStore.longitude
				},
				radius: mapAnchorStore.radius * 1000,
				limit: 6
			});
			if (isSuccess(response) && response.data) {
				recommendedGroups = response.data.items ?? [];
			}
		} catch {
			// ignore
		} finally {
			loadingGroups = false;
		}
	}

	async function loadFeed(refresh = false) {
		if (refresh) {
			feedMoments = [];
			feedCursor = undefined;
			feedHasMore = true;
			loadingFeed = true;
		} else {
			loadingMoreFeed = true;
		}

		try {
			const response = await feedService.getMoments({
				source: 'all',
				reference_point: {
					latitude: mapAnchorStore.latitude,
					longitude: mapAnchorStore.longitude
				},
				limit: 20,
				cursor: refresh ? undefined : feedCursor
			});

			if (isSuccess(response) && response.data) {
				feedMoments = refresh ? response.data.items ?? [] : [...feedMoments, ...(response.data.items ?? [])];
				feedHasMore = response.data.pagination?.has_more ?? false;
				feedCursor = response.data.pagination?.next_cursor;
			}
		} catch {
			if (refresh) toast.error('加载动态失败');
		} finally {
			loadingFeed = false;
			loadingMoreFeed = false;
		}
	}

	function refreshAll() {
		loadTrendingTags();
		loadRecommendedUsers();
		loadRecommendedGroups();
		loadFeed(true);
	}

	function handleGetStarted() {
		uiStore.openLoginModal(() => refreshAll());
	}

	onMount(() => {
		if (authStore.isAuthenticated) {
			loadStats();
			refreshAll();
		}
	});

	$effect(() => {
		if (authStore.isAuthenticated) {
			loadStats();
		} else {
			stats = null;
		}
	});

	// 监听地图锚点变化
	$effect(() => {
		if (authStore.isAuthenticated) {
			mapAnchorStore.anchor.updatedAt; // Dependency
			refreshAll();
		}
	});
</script>

<svelte:head>
	<title>HotSpot - 发现附近的精彩</title>
</svelte:head>

<div class="space-y-12">
	{#if !authStore.isAuthenticated}
		<!-- Hero section for guests -->
		<section class="relative overflow-hidden rounded-3xl bg-linear-to-br from-primary/10 via-primary/15 to-secondary/10 p-8 shadow-xl shadow-primary/10 ring-1 ring-primary/10 md:p-16">
			<!-- Decorative elements -->
			<div class="absolute -right-20 -top-20 h-64 w-64 rounded-full bg-primary/10 blur-3xl"></div>
			<div class="absolute -bottom-20 -left-20 h-64 w-64 rounded-full bg-secondary/10 blur-3xl"></div>
			
			<div class="relative mx-auto max-w-3xl text-center">
				<div class="mb-6 inline-flex items-center gap-2 rounded-full bg-primary/10 px-4 py-2 text-sm font-medium text-primary backdrop-blur-sm">
					<Sparkles class="h-4 w-4" />
					基于位置的社交平台
				</div>
				<h1 class="mb-6 text-4xl font-bold tracking-tight md:text-5xl lg:text-6xl">
					<span class="bg-linear-to-r from-primary via-primary/80 to-primary/60 bg-clip-text text-transparent">
						发现附近的精彩
					</span>
				</h1>
				<p class="mx-auto mb-10 max-w-xl text-lg text-muted-foreground md:text-xl">
					HotSpot 帮助你发现身边有趣的人、精彩的瞬间和活跃的群组。
					开始探索你的世界吧！
				</p>
				<div class="flex flex-col gap-4 sm:flex-row sm:justify-center">
					<Button size="lg" class="gap-2 rounded-full px-8 shadow-lg shadow-primary/25 transition-all hover:shadow-xl hover:shadow-primary/30" onclick={handleGetStarted}>
						立即开始
						<ArrowRight class="h-4 w-4" />
					</Button>
					<Button size="lg" variant="outline" class="rounded-full px-8" onclick={() => goto('/explore')}>
						先逛逛看
					</Button>
				</div>
			</div>
		</section>

		<!-- Features -->
		<section>
			<div class="mb-8 text-center">
				<h2 class="mb-2 text-2xl font-bold md:text-3xl">核心功能</h2>
				<p class="text-muted-foreground">探索 HotSpot 为你提供的精彩体验</p>
			</div>
			<div class="grid gap-6 sm:grid-cols-2 lg:grid-cols-4">
				{#each features as feature}
					<Card class="group cursor-pointer border-0 bg-linear-to-br {feature.color} backdrop-blur-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-primary/10">
						<CardHeader>
							<div class="mb-3 inline-flex h-12 w-12 items-center justify-center rounded-2xl bg-background/80 shadow-sm transition-transform group-hover:scale-110">
								<feature.icon class="h-6 w-6 {feature.iconColor}" />
							</div>
							<CardTitle class="text-lg">{feature.title}</CardTitle>
						</CardHeader>
						<CardContent>
							<CardDescription class="text-foreground/70">{feature.description}</CardDescription>
						</CardContent>
					</Card>
				{/each}
			</div>
		</section>

		<!-- Stats section -->
		<section class="rounded-2xl bg-linear-to-r from-primary/5 to-secondary/5 p-8 md:p-12">
			<div class="grid gap-8 text-center sm:grid-cols-3">
				<div>
					<div class="mb-2 text-3xl font-bold text-primary md:text-4xl">10K+</div>
					<div class="text-sm text-muted-foreground">活跃用户</div>
				</div>
				<div>
					<div class="mb-2 text-3xl font-bold text-primary md:text-4xl">50K+</div>
					<div class="text-sm text-muted-foreground">精彩瞬间</div>
				</div>
				<div>
					<div class="mb-2 text-3xl font-bold text-primary md:text-4xl">1K+</div>
					<div class="text-sm text-muted-foreground">活跃群组</div>
				</div>
			</div>
		</section>
	{:else}
		<!-- Dashboard for authenticated users -->
		<div class="space-y-8">
			<!-- Header with stats -->
			<section>
				<div class="mb-6 flex items-center justify-between">
					<div>
						<h1 class="mb-2 text-2xl font-bold md:text-3xl">
							欢迎回来，<span class="text-primary">{authStore.currentAccount?.nickname}</span>！
						</h1>
						<p class="text-muted-foreground">发现附近的精彩内容</p>
					</div>
					<Button variant="ghost" size="icon" class="rounded-full" onclick={refreshAll} disabled={loadingFeed}>
						<RefreshCw class="h-4 w-4 {loadingFeed ? 'animate-spin' : ''}" />
					</Button>
				</div>

				<!-- Quick stats -->
				<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
					<Card class="border-0 bg-linear-to-br from-pink-500/10 to-rose-500/10">
						<CardContent class="flex items-center gap-4 p-6">
							<div class="rounded-2xl bg-pink-500/20 p-3">
								<Heart class="h-6 w-6 text-pink-500" />
							</div>
							<div>
								{#if statsLoading}
									<Skeleton class="h-7 w-10" />
								{:else}
									<p class="text-2xl font-bold">{stats?.moments_count ?? 0}</p>
								{/if}
								<p class="text-sm text-muted-foreground">我的动态</p>
							</div>
						</CardContent>
					</Card>
					<Card class="border-0 bg-linear-to-br from-blue-500/10 to-cyan-500/10">
						<CardContent class="flex items-center gap-4 p-6">
							<div class="rounded-2xl bg-blue-500/20 p-3">
								<Users class="h-6 w-6 text-blue-500" />
							</div>
							<div>
								{#if statsLoading}
									<Skeleton class="h-7 w-10" />
								{:else}
									<p class="text-2xl font-bold">{stats?.followers_count ?? 0}</p>
								{/if}
								<p class="text-sm text-muted-foreground">关注者</p>
							</div>
						</CardContent>
					</Card>
					<Card class="border-0 bg-linear-to-br from-purple-500/10 to-violet-500/10">
						<CardContent class="flex items-center gap-4 p-6">
							<div class="rounded-2xl bg-purple-500/20 p-3">
								<TrendingUp class="h-6 w-6 text-purple-500" />
							</div>
							<div>
								{#if statsLoading}
									<Skeleton class="h-7 w-10" />
								{:else}
									<p class="text-2xl font-bold">{stats?.friends_count ?? 0}</p>
								{/if}
								<p class="text-sm text-muted-foreground">好友数量</p>
							</div>
						</CardContent>
					</Card>
					<Card class="border-0 bg-linear-to-br from-emerald-500/10 to-teal-500/10">
						<CardContent class="flex items-center gap-4 p-6">
							<div class="rounded-2xl bg-emerald-500/20 p-3">
								<Users class="h-6 w-6 text-emerald-500" />
							</div>
							<div>
								{#if statsLoading}
									<Skeleton class="h-7 w-10" />
								{:else}
									<p class="text-2xl font-bold">{stats?.groups_count ?? 0}</p>
								{/if}
								<p class="text-sm text-muted-foreground">我的群组</p>
							</div>
						</CardContent>
					</Card>
				</div>
			</section>

			<!-- Trending Topics -->
			{#if trendingTags.length > 0}
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

			<!-- Recommended Users -->
			{#if loadingUsers}
				<section class="space-y-4">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2 px-1">
							<Users class="h-5 w-5 text-primary" />
							<h2 class="text-lg font-semibold tracking-tight">推荐用户</h2>
						</div>
					</div>
					<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
						{#each Array(6) as _}
							<Card class="border-0 bg-card/50">
								<CardContent class="p-6">
									<div class="flex items-center gap-4">
										<Skeleton class="h-12 w-12 rounded-full" />
										<div class="flex-1 space-y-2">
											<Skeleton class="h-4 w-24" />
											<Skeleton class="h-3 w-16" />
										</div>
									</div>
									<Skeleton class="mt-3 h-4 w-full" />
									<Skeleton class="mt-1.5 h-4 w-3/4" />
								</CardContent>
							</Card>
						{/each}
					</div>
				</section>
			{:else if recommendedUsers.length > 0}
				<section class="space-y-4">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2 px-1">
							<Users class="h-5 w-5 text-primary" />
							<h2 class="text-lg font-semibold tracking-tight">推荐用户</h2>
						</div>
						<a href="/users/nearby" class="text-sm font-medium text-muted-foreground hover:underline">
							查看更多
						</a>
					</div>
					<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
						{#each recommendedUsers as user}
							<a href="/users/{user.user_id}" class="group">
								<Card class="group/card h-full cursor-pointer border-0 bg-card/50 backdrop-blur-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-primary/5">
									<CardContent class="p-6">
										<div class="flex items-center gap-4">
											<Avatar class="h-12 w-12 ring-2 ring-transparent transition-all duration-300 group-hover/card:ring-primary/20">
												<AvatarImage src={user.avatar_url} alt={user.nickname} />
												<AvatarFallback>{(user.nickname || '用户').slice(0, 2)}</AvatarFallback>
											</Avatar>
											<div class="min-w-0 flex-1">
												<h3 class="truncate font-semibold">{user.nickname}</h3>
												{#if user.location?.distance}
													<p class="flex items-center gap-1 text-xs text-muted-foreground">
														<MapPin class="h-3 w-3" />
														{(user.location.distance / 1000).toFixed(1)} km
													</p>
												{/if}
											</div>
										</div>
										{#if user.bio}
											<p class="mt-3 line-clamp-2 text-sm text-muted-foreground">{user.bio}</p>
										{/if}
									</CardContent>
								</Card>
							</a>
						{/each}
					</div>
				</section>
			{/if}

			<!-- Recommended Groups -->
			{#if loadingGroups}
				<section class="space-y-4">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2 px-1">
							<Users class="h-5 w-5 text-primary" />
							<h2 class="text-lg font-semibold tracking-tight">推荐群组</h2>
						</div>
					</div>
					<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
						{#each Array(6) as _}
							<Card class="border-0 bg-card/50">
								<CardContent class="p-6">
									<div class="flex items-center gap-4">
										<Skeleton class="h-12 w-12 rounded-full" />
										<div class="flex-1 space-y-2">
											<Skeleton class="h-4 w-24" />
											<Skeleton class="h-3 w-20" />
										</div>
									</div>
									<Skeleton class="mt-3 h-4 w-full" />
									<Skeleton class="mt-1.5 h-4 w-3/4" />
								</CardContent>
							</Card>
						{/each}
					</div>
				</section>
			{:else if recommendedGroups.length > 0}
				<section class="space-y-4">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2 px-1">
							<Users class="h-5 w-5 text-primary" />
							<h2 class="text-lg font-semibold tracking-tight">推荐群组</h2>
						</div>
						<a href="/groups/nearby" class="text-sm font-medium text-muted-foreground hover:underline">
							查看更多
						</a>
					</div>
					<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
						{#each recommendedGroups as group}
							<a href="/groups/{group.group_chat_id}" class="group">
								<Card class="group/card h-full cursor-pointer border-0 bg-card/50 backdrop-blur-sm transition-all duration-300 hover:-translate-y-1 hover:shadow-xl hover:shadow-primary/5">
									<CardContent class="p-6">
										<div class="flex items-center gap-4">
											<Avatar class="h-12 w-12 ring-2 ring-transparent transition-all duration-300 group-hover/card:ring-primary/20">
												<AvatarImage src={group.avatar_url} alt={group.name} />
												<AvatarFallback>{(group.name || '群组').slice(0, 2)}</AvatarFallback>
											</Avatar>
											<div class="min-w-0 flex-1">
												<h3 class="truncate font-semibold">{group.name}</h3>
												<p class="flex items-center gap-1 text-xs text-muted-foreground">
													<Users class="h-3 w-3" />
													{group.members_count} 位成员
												</p>
											</div>
										</div>
										{#if group.description}
											<p class="mt-3 line-clamp-2 text-sm text-muted-foreground">{group.description}</p>
										{/if}
									</CardContent>
								</Card>
							</a>
						{/each}
					</div>
				</section>
			{/if}

			<!-- Moments Feed -->
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
									<CardContent class="space-y-3 p-4">
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
						<div class="flex justify-center pt-4">
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
	{/if}
</div>
