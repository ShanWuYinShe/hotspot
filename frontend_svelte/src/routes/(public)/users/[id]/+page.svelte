<script lang="ts">
	import { goto } from '$app/navigation';
	import { untrack } from 'svelte';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent } from '$lib/components/ui/card';
	import PageHeader from '$lib/components/layout/page-header.svelte';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import ProfileStatCard from '$lib/components/profile/profile-stat-card.svelte';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import type { UserItem, MomentItem, FollowerItem, FollowingItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Users from 'lucide-svelte/icons/users';
	import Image from 'lucide-svelte/icons/image';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import UserCheck from 'lucide-svelte/icons/user-check';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	const userId = $derived(data.userId);

	let user = $state<UserItem | null>(null);
	let isFollowing = $state(false);
	$effect(() => {
		user = data.user;
		isFollowing = data.user?.relation?.is_following ?? false;
	});
	let isActionLoading = $state(false);
	let currentTab = $state<'moments' | 'followers' | 'following'>('moments');

	// userId 变化时重置子 tab 分页状态并触发首次加载
	// 注意：上方 data→user 同步 effect 先于此 effect 执行，因此此处 user 已是最新值
	$effect(() => {
		userId; // 依赖声明
		untrack(() => {
			currentTab = 'moments';
			momentsPagination.reset();
			followersPagination.reset();
			followingPagination.reset();
			// 首次加载 moments（仅当 user 数据可用时）
			if (user) {
				loadMoments(true);
			}
		});
	});

	// 使用 usePagination hook 统一管理三个 tab 的分页状态
	const momentsPagination = usePagination<MomentItem>({
		initialLoading: false,
		uniqueKey: 'moment_id'
	});

	const followersPagination = usePagination<FollowerItem>({
		initialLoading: false,
		uniqueKey: (item) => item.user.user_id
	});

	const followingPagination = usePagination<FollowingItem>({
		initialLoading: false,
		uniqueKey: (item) => item.user.user_id
	});


	async function handleFollow() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal(handleFollow);
			return;
		}

		isActionLoading = true;
		try {
		if (isFollowing) {
			const response = await usersService.unfollow({ user_id: userId });
			if (isSuccess(response)) {
				isFollowing = false;
				// 更新 user.relation
				if (user) {
					user = {
						...user,
						relation: user.relation ? { ...user.relation, is_following: false } : undefined
					};
				}
				toast.success('已取消关注');
			} else {
				toast.error(response.message || '操作失败');
			}
		} else {
			const response = await usersService.follow({ user_id: userId });
			if (isSuccess(response)) {
				isFollowing = true;
				// 更新 user.relation
				if (user) {
					user = {
						...user,
						relation: user.relation ? { ...user.relation, is_following: true } : { is_following: true, is_followed_by: false, is_blocked: false }
					};
				}
				toast.success('关注成功');
			} else {
				// 统一使用 response.message 展示错误（spec 哲学：code 驱动行为，message 描述上下文）
				toast.error(response.message || '操作失败');
			}
		}
		} catch {
			toast.error('操作失败');
		} finally {
			isActionLoading = false;
		}
	}

	// 使用 usePagination hook，大大简化分页逻辑
	async function loadMoments(refresh = false) {
		await momentsPagination.load(
			(cursor) =>
				usersService.getMoments({
					user_id: userId,
					pagination: { limit: 12, cursor }
				}),
			refresh
		);

		if (momentsPagination.error) {
			toast.error('加载瞬间失败');
		}
	}

	async function loadFollowers(refresh = false) {
		await followersPagination.load(
			(cursor) =>
				usersService.getFollowers({
					user_id: userId,
					pagination: { limit: 20, cursor }
				}),
			refresh
		);

		if (followersPagination.error) {
			toast.error('加载粉丝失败');
		}
	}

	async function loadFollowing(refresh = false) {
		await followingPagination.load(
			(cursor) =>
				usersService.getFollowing({
					user_id: userId,
					pagination: { limit: 20, cursor }
				}),
			refresh
		);

		if (followingPagination.error) {
			toast.error('加载关注失败');
		}
	}

	function handleTabChange(next: string) {
		const target = next as 'moments' | 'followers' | 'following';
		currentTab = target;
		if (target === 'moments' && momentsPagination.items.length === 0) loadMoments(true);
		if (target === 'followers' && followersPagination.items.length === 0) loadFollowers(true);
		if (target === 'following' && followingPagination.items.length === 0) loadFollowing(true);
	}

	// 数据错误提示（独立 effect，避免 user 变化导致重复触发）
	$effect(() => {
		if (data.error) {
			toast.error(data.error);
		}
	});
</script>

<svelte:head>
	<title>{user?.user.nickname ?? '用户'} - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-5xl space-y-8">
	<PageHeader title="用户主页">
		{#snippet actions()}
			<Button
				variant="ghost"
				size="icon"
				onclick={() => history.back()}
				aria-label="返回"
			>
				<ArrowLeft class="h-5 w-5" />
			</Button>
		{/snippet}
	</PageHeader>

	{#if !user}
		<Card>
			<CardContent class="p-6">
				<div class="flex flex-col items-center gap-4">
					<p class="text-muted-foreground">
						{data.error || '用户不存在'}
					</p>
				</div>
			</CardContent>
		</Card>
	{:else}
		<Card class="rounded-3xl border border-border/60 bg-card/85 shadow-sm backdrop-blur-md">
			<CardContent class="space-y-6 p-6">
				<div class="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
					<div class="flex flex-col gap-4 sm:flex-row sm:items-center">
						<Avatar class="h-24 w-24 ring-2 ring-primary/15">
							<AvatarImage src={user.user.avatar_url} alt={user.user.nickname} />
							<AvatarFallback class="text-2xl">{user.user.nickname.slice(0, 2)}</AvatarFallback>
						</Avatar>
						<div>
							<h2 class="text-2xl font-bold">{user.user.nickname}</h2>
							{#if user.bio}
								<p class="mt-2 max-w-xl text-sm text-muted-foreground">{user.bio}</p>
							{/if}
							{#if user.location}
								<p class="mt-2 flex items-center gap-1 text-sm text-muted-foreground">
									<MapPin class="h-4 w-4" />
									{user.location.location_name}
								</p>
							{/if}
							{#if user.tags && user.tags.length > 0}
								<div class="mt-3 flex flex-wrap gap-2">
									{#each user.tags as tag (tag)}
										<Badge variant="secondary">#{tag}</Badge>
									{/each}
								</div>
							{/if}
						</div>
					</div>

					{#if authStore.isAuthenticated && authStore.currentUserId !== userId}
						<div class="flex flex-wrap gap-3">
							<Button
								variant={isFollowing ? 'outline' : 'default'}
								onclick={handleFollow}
								disabled={isActionLoading}
							>
								{#if isActionLoading}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{:else if isFollowing}
									<UserCheck class="mr-2 h-4 w-4" />
									已关注
								{:else}
									<UserPlus class="mr-2 h-4 w-4" />
									关注
								{/if}
							</Button>
							<Button variant="outline" onclick={() => goto(`/chat/users/${userId}`)}>
								<MessageCircle class="mr-2 h-4 w-4" />
								发消息
							</Button>
						</div>
					{/if}
				</div>

				{#if user.social_context?.mutual_friends_count}
					<p class="text-sm text-muted-foreground">
						你们有 {user.social_context.mutual_friends_count} 个共同好友
					</p>
				{/if}

				<div class="grid gap-3 sm:grid-cols-3">
					<ProfileStatCard
						value={user.stats?.moments_count ?? 0}
						label="瞬间"
						onclick={() => handleTabChange('moments')}
					/>
					<ProfileStatCard
						value={user.stats?.followers_count ?? 0}
						label="粉丝"
						onclick={() => handleTabChange('followers')}
					/>
					<ProfileStatCard
						value={user.stats?.following_count ?? 0}
						label="关注"
						onclick={() => handleTabChange('following')}
					/>
				</div>
			</CardContent>
		</Card>

		<Tabs value={currentTab} onValueChange={handleTabChange}>
			<TabsList class="grid w-full grid-cols-3 rounded-3xl bg-card/80 p-1.5 shadow-sm shadow-black/5 dark:shadow-black/20 ring-1 ring-border/60 backdrop-blur-sm">
				<TabsTrigger value="moments" class="gap-2 rounded-xl">
					<Image class="h-4 w-4" />
					TA的瞬间
				</TabsTrigger>
				<TabsTrigger value="followers" class="gap-2 rounded-xl">
					<Users class="h-4 w-4" />
					粉丝列表
				</TabsTrigger>
				<TabsTrigger value="following" class="gap-2 rounded-xl">
					<Users class="h-4 w-4" />
					关注列表
				</TabsTrigger>
			</TabsList>

			<TabsContent value="moments" class="mt-6">
				{#if momentsPagination.loading}
					<div class="space-y-4">
						{#each Array(3) as _, i (i)}
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
				{:else if momentsPagination.items.length === 0}
					<Card>
						<CardContent class="flex flex-col items-center justify-center py-12">
							<Image class="mb-4 h-10 w-10 text-muted-foreground" />
							<p class="text-muted-foreground">TA 还没有发布瞬间</p>
						</CardContent>
					</Card>
				{:else}
					<div class="columns-1 gap-4 sm:columns-2 lg:columns-3">
						{#each momentsPagination.items as moment (moment.moment_id)}
							<div class="mb-4 break-inside-avoid">
								<MomentCard {moment} />
							</div>
						{/each}
					</div>
					{#if momentsPagination.hasMore}
						<div class="flex justify-center pt-4">
							<Button variant="outline" onclick={() => loadMoments(false)} disabled={momentsPagination.loadingMore}>
								{#if momentsPagination.loadingMore}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{/if}
								加载更多
							</Button>
						</div>
					{/if}
				{/if}
			</TabsContent>

			<TabsContent value="followers" class="mt-6">
				{#if followersPagination.loading}
					<div class="space-y-2">
						{#each Array(5) as _, i (i)}
							<Card>
								<CardContent class="flex items-center gap-4 p-4">
									<Skeleton class="h-12 w-12 rounded-full" />
									<div class="flex-1 space-y-2">
										<Skeleton class="h-4 w-24" />
										<Skeleton class="h-3 w-32" />
									</div>
								</CardContent>
							</Card>
						{/each}
					</div>
				{:else if followersPagination.items.length === 0}
					<Card>
						<CardContent class="flex flex-col items-center justify-center py-12">
							<Users class="mb-4 h-10 w-10 text-muted-foreground" />
							<p class="text-muted-foreground">暂无粉丝</p>
						</CardContent>
					</Card>
				{:else}
					<div class="grid gap-3 sm:grid-cols-2">
						{#each followersPagination.items as follower (follower.user.user_id)}
							<a href="/users/{follower.user.user_id}">
								<Card class="cursor-pointer rounded-2xl border border-border/60 bg-card/80 shadow-sm transition-shadow hover:shadow-lg hover:shadow-black/10 dark:hover:shadow-black/30">
									<CardContent class="flex items-center gap-3 p-4">
										<Avatar class="h-12 w-12">
											<AvatarImage src={follower.user.avatar_url} alt={follower.user.nickname} />
											<AvatarFallback>{follower.user.nickname.slice(0, 2)}</AvatarFallback>
										</Avatar>
										<div class="min-w-0">
											<p class="truncate font-medium">{follower.user.nickname}</p>
											<p class="text-xs text-muted-foreground">查看主页</p>
										</div>
									</CardContent>
								</Card>
							</a>
						{/each}
					</div>
					{#if followersPagination.hasMore}
						<div class="flex justify-center pt-4">
							<Button variant="outline" onclick={() => loadFollowers(false)} disabled={followersPagination.loadingMore}>
								{#if followersPagination.loadingMore}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{/if}
								加载更多
							</Button>
						</div>
					{/if}
				{/if}
			</TabsContent>

			<TabsContent value="following" class="mt-6">
				{#if followingPagination.loading}
					<div class="space-y-2">
						{#each Array(5) as _, i (i)}
							<Card>
								<CardContent class="flex items-center gap-4 p-4">
									<Skeleton class="h-12 w-12 rounded-full" />
									<div class="flex-1 space-y-2">
										<Skeleton class="h-4 w-24" />
										<Skeleton class="h-3 w-32" />
									</div>
								</CardContent>
							</Card>
						{/each}
					</div>
				{:else if followingPagination.items.length === 0}
					<Card>
						<CardContent class="flex flex-col items-center justify-center py-12">
							<Users class="mb-4 h-10 w-10 text-muted-foreground" />
							<p class="text-muted-foreground">暂无关注</p>
						</CardContent>
					</Card>
				{:else}
					<div class="grid gap-3 sm:grid-cols-2">
						{#each followingPagination.items as follow (follow.user.user_id)}
							<a href="/users/{follow.user.user_id}">
								<Card class="cursor-pointer rounded-2xl border border-border/60 bg-card/80 shadow-sm transition-shadow hover:shadow-lg hover:shadow-black/10 dark:hover:shadow-black/30">
									<CardContent class="flex items-center gap-3 p-4">
										<Avatar class="h-12 w-12">
											<AvatarImage src={follow.user.avatar_url} alt={follow.user.nickname} />
											<AvatarFallback>{follow.user.nickname.slice(0, 2)}</AvatarFallback>
										</Avatar>
										<div class="min-w-0">
											<p class="truncate font-medium">{follow.user.nickname}</p>
											<p class="text-xs text-muted-foreground">查看主页</p>
										</div>
									</CardContent>
								</Card>
							</a>
						{/each}
					</div>
					{#if followingPagination.hasMore}
						<div class="flex justify-center pt-4">
							<Button variant="outline" onclick={() => loadFollowing(false)} disabled={followingPagination.loadingMore}>
								{#if followingPagination.loadingMore}
									<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								{/if}
								加载更多
							</Button>
						</div>
					{/if}
				{/if}
			</TabsContent>
		</Tabs>
	{/if}
</div>
