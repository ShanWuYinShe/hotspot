<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { feedService, tagsService, momentsService, usersService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { mapAnchorStore } from '$lib/stores/map-anchor.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Badge } from '$lib/components/ui/badge';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import LocationPickerModal from '$lib/components/common/location-picker-modal.svelte';
	import CreateMomentModal from '$lib/components/common/create-moment-modal.svelte';
	import CreateGroupModal from '$lib/components/common/create-group-modal.svelte';
	import type {
		FeedMomentItem,
		FeedUserItem,
		FeedGroupItem,
		MomentSearchItem,
		TrendingTag,
		TrendingBundle,
		UserSearchItem,
		GroupSearchItem
	} from '$lib/types/api';
	import TrendingUp from 'lucide-svelte/icons/trending-up';
	import Users from 'lucide-svelte/icons/users';
	import UsersRound from 'lucide-svelte/icons/users-round';
	import Hash from 'lucide-svelte/icons/hash';
	import RefreshCw from 'lucide-svelte/icons/refresh-cw';
	import Compass from 'lucide-svelte/icons/compass';
	import Flame from 'lucide-svelte/icons/flame';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import SlidersHorizontal from 'lucide-svelte/icons/sliders-horizontal';
	import Image from 'lucide-svelte/icons/image';
	import Navigation from 'lucide-svelte/icons/navigation';
	import Lock from 'lucide-svelte/icons/lock';
	import Plus from 'lucide-svelte/icons/plus';
	import Sparkles from 'lucide-svelte/icons/sparkles';
	import ArrowRight from 'lucide-svelte/icons/arrow-right';

	type MainTab = 'nearby' | 'recommend' | 'following';
	type NearbyTab = 'moments' | 'users' | 'groups';

	let currentTab = $state<MainTab>('nearby');
	let nearbyTab = $state<NearbyTab>('moments');
	let showLocationPicker = $state(false);
	let showCreateMoment = $state(false);
	let showCreateGroup = $state(false);
	let radiusKm = $state(5);
	let debounceTimer: ReturnType<typeof setTimeout> | null = null;
	let recommendationsTimer: ReturnType<typeof setTimeout> | null = null;

	// Feed moments (login required)
	let feedMoments = $state<FeedMomentItem[]>([]);
	let feedLoading = $state(true);
	let feedLoadingMore = $state(false);
	let feedHasMore = $state(true);
	let feedCursor = $state<string | undefined>(undefined);

	// Nearby moments/users/groups (public)
	let nearbyMoments = $state<MomentSearchItem[]>([]);
	let nearbyUsers = $state<UserSearchItem[]>([]);
	let nearbyGroups = $state<GroupSearchItem[]>([]);
	let nearbyMomentsLoading = $state(false);
	let nearbyUsersLoading = $state(false);
	let nearbyGroupsLoading = $state(false);
	let nearbyMomentsLoadingMore = $state(false);
	let nearbyUsersLoadingMore = $state(false);
	let nearbyGroupsLoadingMore = $state(false);
	let nearbyMomentsHasMore = $state(true);
	let nearbyUsersHasMore = $state(true);
	let nearbyGroupsHasMore = $state(true);
	let nearbyMomentsCursor = $state<string | undefined>(undefined);
	let nearbyUsersCursor = $state<string | undefined>(undefined);
	let nearbyGroupsCursor = $state<string | undefined>(undefined);
	let nearbyLoading = $derived(
		nearbyTab === 'moments' ? nearbyMomentsLoading : nearbyTab === 'users' ? nearbyUsersLoading : nearbyGroupsLoading
	);

	let trendingTags = $state<TrendingTag[]>([]);
	let isLoadingTags = $state(true);
	let trendingBundle = $state<TrendingBundle | null>(null);
	let isLoadingTrending = $state(true);

	// Personalized recommendations (login required)
	let recommendedUsers = $state<FeedUserItem[]>([]);
	let recommendedGroups = $state<FeedGroupItem[]>([]);
	let recommendedUsersLoading = $state(false);
	let recommendedGroupsLoading = $state(false);

	function formatDistance(meters: number): string {
		if (meters < 1000) return `${Math.round(meters)}m`;
		return `${(meters / 1000).toFixed(1)}km`;
	}

	function getAnchor() {
		return {
			latitude: mapAnchorStore.latitude,
			longitude: mapAnchorStore.longitude
		};
	}

	async function loadFeed(refresh = false) {
		if (!authStore.isAuthenticated) return;
		if (refresh) {
			feedMoments = [];
			feedCursor = undefined;
			feedHasMore = true;
			feedLoading = true;
		} else {
			feedLoadingMore = true;
		}

		try {
			const response = await feedService.getMoments({
				source: currentTab === 'recommend' ? 'all' : 'following',
				reference_point: getAnchor(),
				limit: 20,
				cursor: refresh ? undefined : feedCursor
			});

			if (isSuccess(response) && response.data) {
				feedMoments = refresh ? response.data.items : [...feedMoments, ...response.data.items];
				feedHasMore = response.data.pagination.has_more;
				feedCursor = response.data.pagination.next_cursor;
			}
		} catch {
			toast.error('加载失败');
		} finally {
			feedLoading = false;
			feedLoadingMore = false;
		}
	}

	async function loadNearbyMoments(refresh = false) {
		if (refresh) {
			nearbyMoments = [];
			nearbyMomentsCursor = undefined;
			nearbyMomentsHasMore = true;
			nearbyMomentsLoading = true;
		} else {
			nearbyMomentsLoadingMore = true;
		}

		try {
			const response = await momentsService.searchByLocation({
				reference_point: getAnchor(),
				radius: radiusKm * 1000,
				limit: 20,
				cursor: refresh ? undefined : nearbyMomentsCursor
			});

			if (isSuccess(response) && response.data) {
				nearbyMoments = refresh ? response.data.items : [...nearbyMoments, ...response.data.items];
				nearbyMomentsHasMore = response.data.pagination.has_more;
				nearbyMomentsCursor = response.data.pagination.next_cursor;
			}
		} catch {
			toast.error('加载附近动态失败');
		} finally {
			nearbyMomentsLoading = false;
			nearbyMomentsLoadingMore = false;
		}
	}

	async function loadNearbyUsers(refresh = false) {
		if (refresh) {
			nearbyUsers = [];
			nearbyUsersCursor = undefined;
			nearbyUsersHasMore = true;
			nearbyUsersLoading = true;
		} else {
			nearbyUsersLoadingMore = true;
		}

		try {
			const response = await usersService.searchByLocation({
				reference_point: getAnchor(),
				radius: radiusKm * 1000,
				limit: 20,
				cursor: refresh ? undefined : nearbyUsersCursor
			});

			if (isSuccess(response) && response.data) {
				nearbyUsers = refresh ? response.data.items : [...nearbyUsers, ...response.data.items];
				nearbyUsersHasMore = response.data.pagination.has_more;
				nearbyUsersCursor = response.data.pagination.next_cursor;
			}
		} catch {
			toast.error('加载附近的人失败');
		} finally {
			nearbyUsersLoading = false;
			nearbyUsersLoadingMore = false;
		}
	}

	async function loadNearbyGroups(refresh = false) {
		if (refresh) {
			nearbyGroups = [];
			nearbyGroupsCursor = undefined;
			nearbyGroupsHasMore = true;
			nearbyGroupsLoading = true;
		} else {
			nearbyGroupsLoadingMore = true;
		}

		try {
			const response = await groupsService.searchByLocation({
				reference_point: getAnchor(),
				radius: radiusKm * 1000,
				limit: 20,
				cursor: refresh ? undefined : nearbyGroupsCursor
			});

			if (isSuccess(response) && response.data) {
				nearbyGroups = refresh ? response.data.items : [...nearbyGroups, ...response.data.items];
				nearbyGroupsHasMore = response.data.pagination.has_more;
				nearbyGroupsCursor = response.data.pagination.next_cursor;
			}
		} catch {
			toast.error('加载附近群组失败');
		} finally {
			nearbyGroupsLoading = false;
			nearbyGroupsLoadingMore = false;
		}
	}

	function loadNearbyCurrent(refresh = false) {
		if (nearbyTab === 'moments') loadNearbyMoments(refresh);
		if (nearbyTab === 'users') loadNearbyUsers(refresh);
		if (nearbyTab === 'groups') loadNearbyGroups(refresh);
	}

	function loadMain(refresh = false) {
		if (currentTab === 'nearby') {
			loadNearbyCurrent(refresh);
		} else {
			loadFeed(refresh);
		}
	}

	function openCreateMoment() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		showCreateMoment = true;
	}

	function openCreateGroup() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal();
			return;
		}
		showCreateGroup = true;
	}

	function handleMainTabChange(tab: string) {
		const next = tab as MainTab;
		currentTab = next;
		if (!authStore.isAuthenticated && next !== 'nearby') {
			feedLoading = false;
			feedMoments = [];
			feedHasMore = false;
			feedCursor = undefined;
			return;
		}
		loadMain(true);
	}

	function handleNearbyTabChange(tab: string) {
		nearbyTab = tab as NearbyTab;
		loadNearbyCurrent(true);
	}

	function scheduleReload() {
		if (debounceTimer) clearTimeout(debounceTimer);
		debounceTimer = setTimeout(() => {
			if (currentTab === 'nearby') {
				loadNearbyCurrent(true);
			}
		}, 300);
	}

	async function loadTrendingTags() {
		isLoadingTags = true;
		try {
			const response = await tagsService.getTrending({ limit: 10 });
			if (isSuccess(response) && response.data) {
				trendingTags = response.data.items;
			}
		} catch {
			// ignore
		} finally {
			isLoadingTags = false;
		}
	}

	async function loadTrendingBundle() {
		if (!authStore.isAuthenticated) {
			trendingBundle = null;
			isLoadingTrending = false;
			return;
		}
		isLoadingTrending = true;
		try {
			const response = await feedService.getTrending({ limit: 6 });
			if (isSuccess(response) && response.data) {
				trendingBundle = response.data;
			}
		} catch {
			// ignore
		} finally {
			isLoadingTrending = false;
		}
	}

	async function loadRecommendations() {
		if (!authStore.isAuthenticated) {
			recommendedUsers = [];
			recommendedGroups = [];
			return;
		}

		recommendedUsersLoading = true;
		recommendedGroupsLoading = true;

		try {
			const [usersRes, groupsRes] = await Promise.all([
				feedService.getUsers({
					reference_point: getAnchor(),
					radius: radiusKm * 1000,
					limit: 6
				}),
				feedService.getGroups({
					reference_point: getAnchor(),
					radius: radiusKm * 1000,
					limit: 6
				})
			]);

			if (isSuccess(usersRes) && usersRes.data) {
				recommendedUsers = usersRes.data.items;
			}
			if (isSuccess(groupsRes) && groupsRes.data) {
				recommendedGroups = groupsRes.data.items;
			}
		} catch {
			// ignore
		} finally {
			recommendedUsersLoading = false;
			recommendedGroupsLoading = false;
		}
	}

	function scheduleRecommendations() {
		if (!authStore.isAuthenticated) return;
		if (recommendationsTimer) clearTimeout(recommendationsTimer);
		recommendationsTimer = setTimeout(() => {
			loadRecommendations();
		}, 400);
	}

	$effect(() => {
		if (currentTab === 'nearby') {
			mapAnchorStore.anchor.updatedAt;
			radiusKm;
			scheduleReload();
		}
	});

	$effect(() => {
		if (authStore.isAuthenticated) {
			mapAnchorStore.anchor.updatedAt;
			radiusKm;
			scheduleRecommendations();
		}
	});

	$effect(() => {
		if (authStore.isAuthenticated && (currentTab === 'recommend' || currentTab === 'following') && feedMoments.length === 0) {
			loadFeed(true);
		}
	});

	$effect(() => {
		if (authStore.isAuthenticated) {
			if (!trendingBundle && !isLoadingTrending) {
				loadTrendingBundle();
			}
			if (recommendedUsers.length === 0 && recommendedGroups.length === 0) {
				loadRecommendations();
			}
		} else {
			recommendedUsers = [];
			recommendedGroups = [];
		}
	});

	onMount(() => {
		const tabFromUrl = $page.url.searchParams.get('tab') as MainTab | null;
		const nearbyFromUrl = $page.url.searchParams.get('nearby') as NearbyTab | null;
		if (tabFromUrl && ['nearby', 'recommend', 'following'].includes(tabFromUrl)) {
			currentTab = tabFromUrl;
		}
		if (nearbyFromUrl && ['moments', 'users', 'groups'].includes(nearbyFromUrl)) {
			nearbyTab = nearbyFromUrl;
		}
		if (!authStore.isAuthenticated && currentTab !== 'nearby') {
			currentTab = 'nearby';
		}
		loadMain(true);
		loadTrendingTags();
		loadTrendingBundle();
		if (authStore.isAuthenticated) {
			loadRecommendations();
		}
	});
</script>

<svelte:head>
	<title>探索 - HotSpot</title>
</svelte:head>

<div class="space-y-8">
	<!-- Header -->
	<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
		<div class="flex items-center gap-3">
			<div class="rounded-xl bg-linear-to-br from-primary/20 to-primary/10 p-2.5">
				<Compass class="h-6 w-6 text-primary" />
			</div>
			<div>
				<h1 class="text-3xl font-bold tracking-tight">探索</h1>
				<p class="text-base text-muted-foreground">发现精彩内容与附近动态</p>
			</div>
		</div>
		<div class="flex flex-wrap items-center gap-2">
			<Button size="sm" class="gap-2 rounded-full" onclick={openCreateMoment}>
				<Image class="h-4 w-4" />
				发布动态
			</Button>
			<Button variant="outline" size="sm" class="gap-2 rounded-full" onclick={openCreateGroup}>
				<UsersRound class="h-4 w-4" />
				创建群组
			</Button>
			<Button
				variant="outline"
				size="sm"
				class="gap-2 rounded-full"
				onclick={() => {
					loadMain(true);
					loadTrendingBundle();
				}}
				disabled={currentTab === 'nearby' ? nearbyLoading : feedLoading}
			>
				<RefreshCw class="h-4 w-4 {currentTab === 'nearby' ? (nearbyLoading ? 'animate-spin' : '') : (feedLoading ? 'animate-spin' : '')}" />
				刷新
			</Button>
		</div>
	</div>

	<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_240px] xl:grid-cols-[minmax(0,1fr)_260px] xl:gap-10">
		<!-- Main content -->
		<div class="space-y-8">
			<Card class="border-0 bg-linear-to-br from-primary/20 via-background to-primary/5 shadow-xl shadow-primary/15 ring-1 ring-primary/15">
				<CardHeader class="pb-3">
					<CardTitle class="flex items-center gap-2 text-base">
						<div class="rounded-lg bg-primary/15 p-1.5 text-primary">
							<Sparkles class="h-4 w-4" />
						</div>
						本周热榜
					</CardTitle>
					<p class="text-sm text-muted-foreground">热门瞬间与趋势标签正在升温</p>
				</CardHeader>
				<CardContent>
					{#if !authStore.isAuthenticated}
						<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_1.3fr]">
							<div class="space-y-4">
								<div class="text-sm font-semibold text-foreground">趋势标签</div>
								{#if isLoadingTags}
									<div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
										{#each Array(6) as _}
											<Skeleton class="h-10 w-full rounded-xl" />
										{/each}
									</div>
								{:else if trendingTags.length === 0}
									<p class="text-sm text-muted-foreground">暂无热门标签</p>
								{:else}
									<div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
										{#each trendingTags.slice(0, 8) as tag}
											<a
												href="/search?tag={encodeURIComponent(tag.name)}"
												class="group flex items-center justify-between rounded-xl border border-border/60 bg-background/80 px-3 py-2 text-xs font-medium text-foreground shadow-sm shadow-black/5 ring-1 ring-border/40 transition-all hover:-translate-y-0.5 hover:border-primary/40 hover:bg-primary/10 hover:shadow-md hover:shadow-primary/15 hover:ring-primary/30"
											>
												<span class="truncate">#{tag.name}</span>
												<span class="text-[10px] text-muted-foreground">{tag.usage_count}</span>
											</a>
										{/each}
									</div>
								{/if}
							</div>
							<div class="rounded-2xl border border-border/50 bg-card/70 p-6 text-sm text-muted-foreground">
								<div class="flex items-center gap-2 text-foreground">
									<TrendingUp class="h-4 w-4 text-primary" />
									<span class="font-medium">登录后解锁热榜推荐</span>
								</div>
								<p class="mt-2">
									登录即可查看热门动态、个性化推荐与好友点赞内容。
								</p>
								<Button variant="outline" size="sm" class="mt-4 rounded-full" onclick={() => uiStore.openLoginModal()}>
									去登录
								</Button>
							</div>
						</div>
					{:else if isLoadingTrending}
						<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_1.6fr]">
							<div class="space-y-3">
								<Skeleton class="h-6 w-40" />
								<div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
									{#each Array(6) as _}
										<Skeleton class="h-10 w-full rounded-xl" />
									{/each}
								</div>
							</div>
							<div class="columns-1 gap-4 sm:columns-2">
								{#each Array(4) as _}
									<div class="mb-4 break-inside-avoid">
										<Skeleton class="h-40 w-full rounded-xl" />
									</div>
								{/each}
							</div>
						</div>
					{:else if trendingBundle}
						<div class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_1.6fr]">
							<div class="space-y-4">
								<div class="text-sm font-semibold text-foreground">趋势标签</div>
								<div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
									{#each trendingBundle.trending_tags.slice(0, 8) as tag}
										<a
											href="/search?tag={encodeURIComponent(tag.name)}"
											class="group flex items-center justify-between rounded-xl border border-border/60 bg-background/80 px-3 py-2 text-xs font-medium text-foreground shadow-sm shadow-black/5 ring-1 ring-border/40 transition-all hover:-translate-y-0.5 hover:border-primary/40 hover:bg-primary/10 hover:shadow-md hover:shadow-primary/15 hover:ring-primary/30"
										>
											<span class="truncate">#{tag.name}</span>
											<span class="text-[10px] text-muted-foreground">{tag.usage_count}</span>
										</a>
									{/each}
								</div>
								{#if trendingBundle.trending_tags.length === 0}
									<p class="text-sm text-muted-foreground">暂无热门标签</p>
								{/if}
							</div>
							<div class="space-y-4">
								<div class="text-sm font-semibold text-foreground">热门推荐</div>
								{#if trendingBundle.hot_moments.length === 0}
									<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
										<CardContent class="flex items-center justify-center py-12 text-sm text-muted-foreground">
											暂无热门内容
										</CardContent>
									</Card>
								{:else}
									<div class="columns-1 gap-4 sm:columns-2">
										{#each trendingBundle.hot_moments.slice(0, 4) as moment (moment.moment_id)}
											<div class="mb-4 break-inside-avoid">
												<MomentCard {moment} />
											</div>
										{/each}
									</div>
								{/if}
							</div>
						</div>
					{:else}
						<div class="py-8 text-center text-sm text-muted-foreground">暂无热门内容</div>
					{/if}
				</CardContent>
			</Card>

			{#if authStore.isAuthenticated}
				<Card class="border-0 bg-card/80 shadow-lg shadow-black/5 ring-1 ring-border/50">
					<CardHeader class="pb-3">
						<CardTitle class="flex items-center gap-2 text-base">
							<div class="rounded-lg bg-primary/15 p-1.5 text-primary">
								<Compass class="h-4 w-4" />
							</div>
							为你推荐
						</CardTitle>
						<p class="text-sm text-muted-foreground">基于位置与兴趣的精选人群与群组</p>
					</CardHeader>
					<CardContent class="grid gap-8 xl:grid-cols-2">
						<div class="space-y-3">
							<div class="flex items-center justify-between text-sm font-semibold">
								<span>推荐用户</span>
								<Button variant="outline" size="sm" class="h-7 rounded-full px-3 shadow-sm" onclick={loadRecommendations}>
									<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
									换一批
								</Button>
							</div>
							{#if recommendedUsersLoading}
								<div class="columns-1 gap-3 sm:columns-2">
									{#each Array(3) as _}
										<div class="mb-3 break-inside-avoid">
											<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
												<CardContent class="flex items-center gap-3 p-3">
													<Skeleton class="h-10 w-10 rounded-full" />
													<div class="flex-1 space-y-2">
														<Skeleton class="h-3 w-24" />
														<Skeleton class="h-3 w-40" />
													</div>
												</CardContent>
											</Card>
										</div>
									{/each}
								</div>
							{:else if recommendedUsers.length === 0}
								<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
									<CardContent class="py-8 text-center text-sm text-muted-foreground">
										暂无推荐用户
									</CardContent>
								</Card>
							{:else}
								<div class="columns-1 gap-3 sm:columns-2">
									{#each recommendedUsers as user (user.user_id)}
										<div class="mb-3 break-inside-avoid">
											<a href="/users/{user.user_id}">
												<Card class="cursor-pointer border border-border/60 bg-card/80 shadow-sm shadow-black/5 ring-1 ring-border/40 transition-all hover:-translate-y-0.5 hover:bg-accent/50 hover:shadow-md hover:shadow-primary/10 hover:ring-primary/20">
													<CardContent class="flex items-center gap-3 p-3">
														<Avatar class="h-10 w-10">
															<AvatarImage src={user.avatar_url} alt={user.nickname} />
															<AvatarFallback>{user.nickname.slice(0, 2)}</AvatarFallback>
														</Avatar>
														<div class="min-w-0 flex-1">
															<div class="flex items-center gap-2">
																<p class="truncate font-medium">{user.nickname}</p>
																{#if user.mutual_friends_count}
																	<Badge variant="secondary" class="rounded-full text-[10px]">
																		共同好友 {user.mutual_friends_count}
																	</Badge>
																{/if}
															</div>
															{#if user.common_tags && user.common_tags.length > 0}
																<p class="mt-1 truncate text-xs text-muted-foreground">
																	共同兴趣：{user.common_tags.slice(0, 3).join(' / ')}
																</p>
															{:else if user.bio}
																<p class="mt-1 truncate text-xs text-muted-foreground">{user.bio}</p>
															{/if}
														</div>
														<ArrowRight class="h-4 w-4 text-muted-foreground" />
													</CardContent>
												</Card>
											</a>
										</div>
									{/each}
								</div>
							{/if}
						</div>

						<div class="space-y-3">
							<div class="flex items-center justify-between text-sm font-semibold">
								<span>推荐群组</span>
								<Button variant="outline" size="sm" class="h-7 rounded-full px-3 shadow-sm" onclick={loadRecommendations}>
									<RefreshCw class="mr-1.5 h-3.5 w-3.5" />
									换一批
								</Button>
							</div>
							{#if recommendedGroupsLoading}
								<div class="columns-1 gap-3 sm:columns-2">
									{#each Array(3) as _}
										<div class="mb-3 break-inside-avoid">
											<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
												<CardContent class="flex items-center gap-3 p-3">
													<Skeleton class="h-10 w-10 rounded-full" />
													<div class="flex-1 space-y-2">
														<Skeleton class="h-3 w-24" />
														<Skeleton class="h-3 w-40" />
													</div>
												</CardContent>
											</Card>
										</div>
									{/each}
								</div>
							{:else if recommendedGroups.length === 0}
								<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
									<CardContent class="py-8 text-center text-sm text-muted-foreground">
										暂无推荐群组
									</CardContent>
								</Card>
							{:else}
								<div class="columns-1 gap-3 sm:columns-2">
									{#each recommendedGroups as group (group.group_chat_id)}
										<div class="mb-3 break-inside-avoid">
											<a href="/groups/{group.group_chat_id}">
												<Card class="cursor-pointer border border-border/60 bg-card/80 shadow-sm shadow-black/5 ring-1 ring-border/40 transition-all hover:-translate-y-0.5 hover:bg-accent/50 hover:shadow-md hover:shadow-primary/10 hover:ring-primary/20">
													<CardContent class="flex items-center gap-3 p-3">
														<Avatar class="h-10 w-10">
															<AvatarImage src={group.avatar_url} alt={group.name} />
															<AvatarFallback>{group.name.slice(0, 2)}</AvatarFallback>
														</Avatar>
														<div class="min-w-0 flex-1">
															<div class="flex items-center gap-2">
																<p class="truncate font-medium">{group.name}</p>
																{#if group.has_password}
																	<Lock class="h-3 w-3 text-muted-foreground" />
																{/if}
															</div>
															<div class="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
																<span>{group.members_count} 成员</span>
																{#if group.friends_in_group}
																	<span>好友 {group.friends_in_group}</span>
																{/if}
															</div>
															{#if group.tags && group.tags.length > 0}
																<div class="mt-1 flex flex-wrap gap-1">
																	{#each group.tags.slice(0, 2) as tag}
																		<Badge variant="secondary" class="text-[10px]">#{tag}</Badge>
																	{/each}
																</div>
															{/if}
														</div>
														<ArrowRight class="h-4 w-4 text-muted-foreground" />
													</CardContent>
												</Card>
											</a>
										</div>
									{/each}
								</div>
							{/if}
						</div>
					</CardContent>
				</Card>
			{:else}
				<Card class="border-0 bg-linear-to-r from-primary/5 to-primary/10">
					<CardContent class="flex flex-col gap-3 p-4 text-sm text-muted-foreground">
						<div class="flex items-center gap-2 text-foreground">
							<Sparkles class="h-4 w-4 text-primary" />
							<span class="font-medium">登录后解锁个性化推荐</span>
						</div>
						<div>注册/登录后可获得更符合兴趣的内容与人群推荐。</div>
						<Button variant="outline" size="sm" class="w-fit rounded-full" onclick={() => uiStore.openLoginModal()}>
							去登录
						</Button>
					</CardContent>
				</Card>
			{/if}

			<Tabs value={currentTab} onValueChange={handleMainTabChange}>
				<TabsList class="h-auto gap-1.5 rounded-full bg-card/80 p-1.5 shadow-md shadow-black/10 ring-1 ring-border/50 backdrop-blur-sm">
					<TabsTrigger value="nearby" class="gap-2 rounded-full data-[state=active]:bg-background data-[state=active]:shadow-sm">
						<MapPin class="h-4 w-4" />
						附近
					</TabsTrigger>
					<TabsTrigger value="recommend" class="gap-2 rounded-full data-[state=active]:bg-background data-[state=active]:shadow-sm">
						<TrendingUp class="h-4 w-4" />
						推荐
					</TabsTrigger>
					<TabsTrigger value="following" class="gap-2 rounded-full data-[state=active]:bg-background data-[state=active]:shadow-sm">
						<Users class="h-4 w-4" />
						关注
					</TabsTrigger>
				</TabsList>

				<!-- Nearby -->
				<TabsContent value="nearby" class="mt-6 space-y-5">
					<Card class="border-0 bg-card/80 shadow-md shadow-black/10 ring-1 ring-border/40 backdrop-blur-sm">
						<CardContent class="space-y-4 p-5">
							<div class="flex items-center justify-between">
								<span class="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
									附近设置
								</span>
								<Badge variant="secondary" class="rounded-full">
									范围 {radiusKm} km
								</Badge>
							</div>
							<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
								<div class="min-w-0">
									<div class="text-xs text-muted-foreground">浏览锚点</div>
									<div class="truncate text-lg font-semibold">{mapAnchorStore.displayName}</div>
									<div class="text-xs text-muted-foreground">
										{mapAnchorStore.latitude.toFixed(4)}, {mapAnchorStore.longitude.toFixed(4)}
									</div>
								</div>
								<div class="flex items-center gap-2">
									<Button variant="outline" size="sm" class="gap-2 rounded-full" onclick={() => (showLocationPicker = true)}>
										<Navigation class="h-4 w-4" />
										设置锚点
									</Button>
									<Button variant="outline" size="sm" class="gap-2 rounded-full" onclick={() => loadNearbyCurrent(true)}>
										<RefreshCw class="h-4 w-4" />
										刷新
									</Button>
								</div>
							</div>
							<div class="flex flex-col gap-3 sm:flex-row sm:items-center">
								<div class="flex items-center gap-2 text-sm font-medium">
									<SlidersHorizontal class="h-4 w-4 text-primary" />
									<span>调整范围</span>
								</div>
								<input
									type="range"
									min="1"
									max="50"
									step="1"
									class="h-2 w-full cursor-pointer appearance-none rounded-full bg-muted/70 accent-primary sm:flex-1"
									bind:value={radiusKm}
								/>
							</div>
						</CardContent>
					</Card>

					<Tabs value={nearbyTab} onValueChange={handleNearbyTabChange}>
						<TabsList class="h-auto gap-1.5 rounded-full bg-card/80 p-1.5 shadow-md shadow-black/10 ring-1 ring-border/50 backdrop-blur-sm">
							<TabsTrigger value="moments" class="gap-2 rounded-full data-[state=active]:bg-background data-[state=active]:shadow-sm">
								<Image class="h-4 w-4" />
								动态
							</TabsTrigger>
							<TabsTrigger value="users" class="gap-2 rounded-full data-[state=active]:bg-background data-[state=active]:shadow-sm">
								<Users class="h-4 w-4" />
								附近的人
							</TabsTrigger>
							<TabsTrigger value="groups" class="gap-2 rounded-full data-[state=active]:bg-background data-[state=active]:shadow-sm">
								<UsersRound class="h-4 w-4" />
								附近群组
							</TabsTrigger>
						</TabsList>

						<!-- Nearby moments -->
						<TabsContent value="moments" class="mt-6">
							{#if nearbyMomentsLoading}
								<div class="space-y-4">
									{#each Array(3) as _}
										<Card class="border-0 bg-card/50 backdrop-blur-sm">
											<CardContent class="p-5">
												<div class="flex gap-4">
													<Skeleton class="h-12 w-12 shrink-0 rounded-full" />
													<div class="flex-1 space-y-3">
														<Skeleton class="h-4 w-1/4" />
														<Skeleton class="h-4 w-full" />
														<Skeleton class="h-4 w-3/4" />
													</div>
												</div>
											</CardContent>
										</Card>
									{/each}
								</div>
							{:else if nearbyMoments.length === 0}
								<Card class="border-0 bg-linear-to-br from-muted/50 to-muted/30">
									<CardContent class="flex flex-col items-center justify-center py-16">
										<MapPin class="mb-4 h-10 w-10 text-muted-foreground" />
										<p class="text-muted-foreground">附近暂无动态</p>
									</CardContent>
								</Card>
							{:else}
								<div class="space-y-4">
									<div class="columns-1 gap-4 sm:columns-2 xl:columns-3">
										{#each nearbyMoments as moment (moment.moment_id)}
											<div class="mb-4 break-inside-avoid">
												<MomentCard {moment} />
											</div>
										{/each}
									</div>
									{#if nearbyMomentsHasMore}
										<div class="flex justify-center pt-4">
											<Button variant="outline" class="rounded-full px-8" onclick={() => loadNearbyMoments(false)} disabled={nearbyMomentsLoadingMore}>
												{#if nearbyMomentsLoadingMore}
													<RefreshCw class="mr-2 h-4 w-4 animate-spin" />
													加载中...
												{:else}
													加载更多
												{/if}
											</Button>
										</div>
									{/if}
								</div>
							{/if}
						</TabsContent>

						<!-- Nearby users -->
						<TabsContent value="users" class="mt-6">
							{#if nearbyUsersLoading}
								<div class="space-y-2">
									{#each Array(5) as _}
										<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
											<CardContent class="flex items-center gap-4 p-4">
												<Skeleton class="h-14 w-14 rounded-full" />
												<div class="flex-1 space-y-2">
													<Skeleton class="h-4 w-24" />
													<Skeleton class="h-3 w-48" />
													<Skeleton class="h-3 w-16" />
												</div>
											</CardContent>
										</Card>
									{/each}
								</div>
							{:else if nearbyUsers.length === 0}
								<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
									<CardContent class="flex flex-col items-center justify-center py-12">
										<MapPin class="mb-4 h-12 w-12 text-muted-foreground" />
										<p class="text-muted-foreground">附近暂无其他用户</p>
									</CardContent>
								</Card>
							{:else}
								<div class="space-y-2">
									{#each nearbyUsers as user (user.user_id)}
										<a href="/users/{user.user_id}">
											<Card class="cursor-pointer border-0 bg-card/60 shadow-sm shadow-black/5 transition-colors hover:bg-accent/40">
												<CardContent class="flex items-center gap-4 p-4">
													<div class="relative">
														<Avatar class="h-14 w-14">
															<AvatarImage src={user.avatar_url} alt={user.nickname} />
															<AvatarFallback>{user.nickname.slice(0, 2)}</AvatarFallback>
														</Avatar>
														{#if user.is_online}
															<div class="absolute bottom-0 right-0 h-4 w-4 rounded-full border-2 border-background bg-green-500"></div>
														{/if}
													</div>
													<div class="min-w-0 flex-1">
														<div class="flex items-center gap-2">
															<p class="font-medium">{user.nickname}</p>
															{#if user.location}
																<span class="flex items-center gap-0.5 text-xs text-muted-foreground">
																	<MapPin class="h-3 w-3" />
																	{formatDistance(user.location.distance)}
																</span>
															{/if}
														</div>
														{#if user.bio}
															<p class="truncate text-sm text-muted-foreground">{user.bio}</p>
														{/if}
														{#if user.tags && user.tags.length > 0}
															<div class="mt-1 flex flex-wrap gap-1">
																{#each user.tags.slice(0, 3) as tag}
																	<Badge variant="secondary" class="text-xs">#{tag}</Badge>
																{/each}
															</div>
														{/if}
													</div>
												</CardContent>
											</Card>
										</a>
									{/each}
									{#if nearbyUsersHasMore}
										<div class="flex justify-center pt-4">
											<Button variant="outline" class="rounded-full px-8" onclick={() => loadNearbyUsers(false)} disabled={nearbyUsersLoadingMore}>
												{#if nearbyUsersLoadingMore}
													<RefreshCw class="mr-2 h-4 w-4 animate-spin" />
													加载中...
												{:else}
													加载更多
												{/if}
											</Button>
										</div>
									{/if}
								</div>
							{/if}
						</TabsContent>

						<!-- Nearby groups -->
						<TabsContent value="groups" class="mt-6">
							{#if nearbyGroupsLoading}
								<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
									{#each Array(6) as _}
										<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
											<CardContent class="p-4">
												<div class="flex items-center gap-3">
													<Skeleton class="h-12 w-12 rounded-full" />
													<div class="flex-1 space-y-2">
														<Skeleton class="h-4 w-24" />
														<Skeleton class="h-3 w-16" />
													</div>
												</div>
											</CardContent>
										</Card>
									{/each}
								</div>
							{:else if nearbyGroups.length === 0}
								<Card class="border-0 bg-card/60 shadow-sm shadow-black/5">
									<CardContent class="flex flex-col items-center justify-center py-12">
										<UsersRound class="mb-4 h-12 w-12 text-muted-foreground" />
										<p class="text-muted-foreground">附近暂无群组</p>
									</CardContent>
								</Card>
							{:else}
								<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
									{#each nearbyGroups as group (group.group_chat_id)}
										<a href="/groups/{group.group_chat_id}">
											<Card class="cursor-pointer border-0 bg-card/60 shadow-sm shadow-black/5 transition-shadow hover:shadow-lg hover:shadow-black/10">
												<CardContent class="p-4">
													<div class="flex items-center gap-3">
														<Avatar class="h-12 w-12">
															<AvatarImage src={group.avatar_url} alt={group.name} />
															<AvatarFallback>{group.name.slice(0, 2)}</AvatarFallback>
														</Avatar>
														<div class="min-w-0 flex-1">
															<div class="flex items-center gap-2">
																<p class="truncate font-medium">{group.name}</p>
																{#if group.has_password}
																	<Lock class="h-3 w-3 text-muted-foreground" />
																{/if}
															</div>
															<div class="flex items-center gap-2 text-xs text-muted-foreground">
																<span>{group.members_count} 成员</span>
																{#if group.location}
																	<span class="flex items-center gap-0.5">
																		<MapPin class="h-3 w-3" />
																		{formatDistance(group.location.distance)}
																	</span>
																{/if}
															</div>
														</div>
													</div>
													{#if group.description}
														<p class="mt-2 line-clamp-2 text-sm text-muted-foreground">{group.description}</p>
													{/if}
													{#if group.tags && group.tags.length > 0}
														<div class="mt-2 flex flex-wrap gap-1">
															{#each group.tags.slice(0, 3) as tag}
																<Badge variant="secondary" class="text-xs">#{tag}</Badge>
															{/each}
														</div>
													{/if}
												</CardContent>
											</Card>
										</a>
									{/each}
									{#if nearbyGroupsHasMore}
										<div class="col-span-full flex justify-center pt-4">
											<Button variant="outline" class="rounded-full px-8" onclick={() => loadNearbyGroups(false)} disabled={nearbyGroupsLoadingMore}>
												{#if nearbyGroupsLoadingMore}
													<RefreshCw class="mr-2 h-4 w-4 animate-spin" />
													加载中...
												{:else}
													加载更多
												{/if}
											</Button>
										</div>
									{/if}
								</div>
							{/if}
						</TabsContent>
					</Tabs>
				</TabsContent>

				<!-- Recommend / Following -->
				<TabsContent value="recommend" class="mt-6">
					{#if !authStore.isAuthenticated}
						<Card class="border-0 bg-linear-to-r from-primary/5 to-primary/10">
							<CardContent class="flex flex-col gap-3 p-4 text-sm text-muted-foreground">
								<div class="flex items-center gap-2 text-foreground">
									<TrendingUp class="h-4 w-4 text-primary" />
									<span class="font-medium">登录后查看推荐内容</span>
								</div>
								<Button variant="outline" size="sm" class="w-fit rounded-full" onclick={() => uiStore.openLoginModal()}>
									去登录
								</Button>
							</CardContent>
						</Card>
					{:else}
						{#if feedLoading}
							<div class="space-y-4">
								{#each Array(3) as _}
									<Card class="border-0 bg-card/50 backdrop-blur-sm">
										<CardContent class="p-5">
											<div class="flex gap-4">
												<Skeleton class="h-12 w-12 shrink-0 rounded-full" />
												<div class="flex-1 space-y-3">
													<Skeleton class="h-4 w-1/4" />
													<Skeleton class="h-4 w-full" />
													<Skeleton class="h-4 w-3/4" />
												</div>
											</div>
										</CardContent>
									</Card>
								{/each}
							</div>
						{:else if feedMoments.length === 0}
							<Card class="border-0 bg-linear-to-br from-muted/50 to-muted/30">
								<CardContent class="flex flex-col items-center justify-center py-16">
									<Compass class="mb-4 h-10 w-10 text-muted-foreground" />
									<p class="text-muted-foreground">暂无推荐内容</p>
								</CardContent>
							</Card>
						{:else}
							<div class="space-y-4">
								<div class="columns-1 gap-4 sm:columns-2 xl:columns-3">
									{#each feedMoments as moment (moment.moment_id)}
										<div class="mb-4 break-inside-avoid">
											<MomentCard {moment} />
										</div>
									{/each}
								</div>
								{#if feedHasMore}
									<div class="flex justify-center pt-4">
										<Button variant="outline" class="rounded-full px-8" onclick={() => loadFeed(false)} disabled={feedLoadingMore}>
											{#if feedLoadingMore}
												<RefreshCw class="mr-2 h-4 w-4 animate-spin" />
												加载中...
											{:else}
												加载更多
											{/if}
										</Button>
									</div>
								{/if}
							</div>
						{/if}
					{/if}
				</TabsContent>

				<TabsContent value="following" class="mt-6">
					{#if !authStore.isAuthenticated}
						<Card class="border-0 bg-linear-to-r from-primary/5 to-primary/10">
							<CardContent class="flex flex-col gap-3 p-4 text-sm text-muted-foreground">
								<div class="flex items-center gap-2 text-foreground">
									<Users class="h-4 w-4 text-primary" />
									<span class="font-medium">登录后查看关注动态</span>
								</div>
								<Button variant="outline" size="sm" class="w-fit rounded-full" onclick={() => uiStore.openLoginModal()}>
									去登录
								</Button>
							</CardContent>
						</Card>
					{:else}
						{#if feedLoading}
							<div class="space-y-4">
								{#each Array(3) as _}
									<Card class="border-0 bg-card/50 backdrop-blur-sm">
										<CardContent class="p-5">
											<div class="flex gap-4">
												<Skeleton class="h-12 w-12 shrink-0 rounded-full" />
												<div class="flex-1 space-y-3">
													<Skeleton class="h-4 w-1/4" />
													<Skeleton class="h-4 w-full" />
													<Skeleton class="h-4 w-3/4" />
												</div>
											</div>
										</CardContent>
									</Card>
								{/each}
							</div>
						{:else if feedMoments.length === 0}
							<Card class="border-0 bg-linear-to-br from-muted/50 to-muted/30">
								<CardContent class="flex flex-col items-center justify-center py-16">
									<Users class="mb-4 h-10 w-10 text-muted-foreground" />
									<p class="text-muted-foreground">还没有关注内容</p>
								</CardContent>
							</Card>
						{:else}
							<div class="space-y-4">
								<div class="columns-1 gap-4 sm:columns-2 xl:columns-3">
									{#each feedMoments as moment (moment.moment_id)}
										<div class="mb-4 break-inside-avoid">
											<MomentCard {moment} />
										</div>
									{/each}
								</div>
								{#if feedHasMore}
									<div class="flex justify-center pt-4">
										<Button variant="outline" class="rounded-full px-8" onclick={() => loadFeed(false)} disabled={feedLoadingMore}>
											{#if feedLoadingMore}
												<RefreshCw class="mr-2 h-4 w-4 animate-spin" />
												加载中...
											{:else}
												加载更多
											{/if}
										</Button>
									</div>
								{/if}
							</div>
						{/if}
					{/if}
				</TabsContent>
			</Tabs>
		</div>

		<!-- Sidebar -->
		<div class="hidden space-y-6 lg:block">
		<Card class="border-0 bg-card/60 shadow-sm shadow-black/5 ring-1 ring-border/40">
			<CardHeader class="pb-3">
				<CardTitle class="flex items-center gap-2 text-base">
					<div class="rounded-lg bg-primary/15 p-1.5 text-primary">
						<Plus class="h-4 w-4" />
					</div>
					快捷操作
				</CardTitle>
			</CardHeader>
			<CardContent class="space-y-3">
				<Button variant="outline" size="sm" class="w-full gap-2" onclick={openCreateMoment}>
					<Image class="h-4 w-4" />
					发布动态
				</Button>
				<Button variant="outline" size="sm" class="w-full gap-2" onclick={openCreateGroup}>
					<UsersRound class="h-4 w-4" />
					创建群组
				</Button>
				<Button variant="ghost" size="sm" class="w-full gap-2" onclick={() => (showLocationPicker = true)}>
					<MapPin class="h-4 w-4" />
					设置浏览锚点
				</Button>
			</CardContent>
		</Card>
		<Card class="border-0 bg-card/60 shadow-sm shadow-black/5 ring-1 ring-border/40">
				<CardHeader class="pb-3">
					<CardTitle class="flex items-center gap-2 text-base">
						<div class="rounded-lg bg-linear-to-br from-orange-500 to-red-500 p-1.5 text-white">
							<Flame class="h-4 w-4" />
						</div>
						热门标签
					</CardTitle>
				</CardHeader>
				<CardContent>
					{#if isLoadingTags}
						<div class="space-y-2">
							{#each Array(5) as _}
								<Skeleton class="h-8 w-full rounded-full" />
							{/each}
						</div>
					{:else if trendingTags.length === 0}
						<div class="py-4 text-center text-sm text-muted-foreground">
							暂无热门标签
						</div>
					{:else}
						<div class="flex flex-wrap gap-2">
						{#each trendingTags as tag, i}
							<a href="/search?tag={encodeURIComponent(tag.name)}" class="group">
									<Badge
										variant="secondary"
										class="cursor-pointer rounded-full px-3 py-1.5 transition-all hover:bg-primary/10 hover:text-primary {i < 3 ? 'bg-primary/10 text-primary' : ''}"
									>
										<Hash class="mr-1 h-3 w-3" />
									{tag.name}
										<span class="ml-1.5 text-xs opacity-60">
										{tag.usage_count}
										</span>
									</Badge>
								</a>
							{/each}
						</div>
					{/if}
				</CardContent>
			</Card>
		</div>
	</div>
</div>

<LocationPickerModal bind:open={showLocationPicker} />
<CreateMomentModal bind:open={showCreateMoment} onCreated={() => loadMain(true)} />
<CreateGroupModal bind:open={showCreateGroup} />
