<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
	import { Button } from '$lib/components/ui/button';
	import { Card, CardContent, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Skeleton } from '$lib/components/ui/skeleton';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import MomentCard from '$lib/components/moments/moment-card.svelte';
	import type { UserCard, UserInfo, UserMomentListItem } from '$lib/types/api';
	import ArrowLeft from 'lucide-svelte/icons/arrow-left';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Users from 'lucide-svelte/icons/users';
	import Image from 'lucide-svelte/icons/image';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import UserCheck from 'lucide-svelte/icons/user-check';
	import UserMinus from 'lucide-svelte/icons/user-minus';
	import Loader2 from 'lucide-svelte/icons/loader-2';

	const userId = parseInt($page.params.id!);

	let user = $state<UserCard | null>(null);
	let isLoading = $state(true);
	let isFollowing = $state(false);
	let isFollowPending = $state(false);
	let isActionLoading = $state(false);
	let currentTab = $state<'moments' | 'followers' | 'following'>('moments');

	// 使用 usePagination hook 统一管理三个 tab 的分页状态
	const momentsPagination = usePagination<UserMomentListItem>({
		initialLoading: false,
		uniqueKey: 'moment_id'
	});

	const followersPagination = usePagination<UserInfo>({
		initialLoading: false,
		uniqueKey: 'user_id'
	});

	const followingPagination = usePagination<UserInfo>({
		initialLoading: false,
		uniqueKey: 'user_id'
	});

	async function loadUser() {
		try {
			const response = await usersService.getUser({ user_id: userId });
			if (isSuccess(response) && response.data) {
				user = response.data;
				isFollowing = user.relation?.is_following ?? false;
				isFollowPending = user.relation?.has_pending_follow_request ?? false;
			} else {
				toast.error(response.message || '用户不存在');
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoading = false;
		}
	}

	async function handleFollow() {
		if (!authStore.isAuthenticated) {
			uiStore.saveRedirectUrl();
			uiStore.openLoginModal(handleFollow);
			return;
		}

		isActionLoading = true;
		try {
		if (isFollowing || isFollowPending) {
			const response = await usersService.unfollow({ user_id: userId });
			if (isSuccess(response) && response.data) {
				// 使用后端返回的 UserRelation 更新状态
				isFollowing = response.data.is_following;
				isFollowPending = response.data.has_pending_follow_request;
				toast.success('已取消关注');
			} else {
				toast.error(response.message || '操作失败');
			}
		} else {
			const response = await usersService.follow({ user_id: userId });
			if (isSuccess(response) && response.data) {
				// 使用后端返回的 UserRelation 判断关注状态
				if (response.data.has_pending_follow_request) {
					isFollowPending = true;
					toast.success('已发送关注请求，等待对方审批');
				} else if (response.data.is_following) {
					isFollowing = true;
					isFollowPending = false;
					toast.success('关注成功');
				}
			} else {
				// 处理具体的错误码
				if (response.code === 10213) {
					toast.error('对方不允许被关注');
				} else if (response.code === 10211) {
					toast.error('已被对方拉黑');
				} else if (response.code === 10411) {
					toast.error('不能关注自己');
				} else {
					toast.error(response.message || '操作失败');
				}
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
					limit: 12,
					cursor
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
					limit: 20,
					cursor
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
					limit: 20,
					cursor
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

	onMount(() => {
		loadUser();
		loadMoments(true);
	});
</script>

<svelte:head>
	<title>{user?.nickname ?? '用户'} - HotSpot</title>
</svelte:head>

<div class="mx-auto max-w-5xl space-y-8">
	<!-- Header -->
	<div class="flex items-center gap-4">
		<Button variant="ghost" size="icon" onclick={() => history.back()}>
			<ArrowLeft class="h-5 w-5" />
		</Button>
		<h1 class="text-2xl font-bold">用户主页</h1>
	</div>

	{#if isLoading}
		<Card>
			<CardContent class="p-6">
				<div class="flex flex-col items-center gap-4">
					<Skeleton class="h-24 w-24 rounded-full" />
					<Skeleton class="h-6 w-32" />
					<Skeleton class="h-4 w-48" />
					<div class="flex gap-8">
						<Skeleton class="h-12 w-20" />
						<Skeleton class="h-12 w-20" />
						<Skeleton class="h-12 w-20" />
					</div>
				</div>
			</CardContent>
		</Card>
	{:else if user}
		<Card class="border-0 bg-card/70 shadow-sm shadow-black/5 ring-1 ring-border/40">
			<CardContent class="space-y-6 p-6">
				<div class="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
					<div class="flex flex-col gap-4 sm:flex-row sm:items-center">
						<Avatar class="h-24 w-24 ring-2 ring-primary/15">
							<AvatarImage src={user.avatar_url} alt={user.nickname} />
							<AvatarFallback class="text-2xl">{user.nickname.slice(0, 2)}</AvatarFallback>
						</Avatar>
						<div>
							<h2 class="text-2xl font-bold">{user.nickname}</h2>
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
									{#each user.tags as tag}
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
								{:else if isFollowPending}
									<UserMinus class="mr-2 h-4 w-4" />
									等待确认
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
					<button
						type="button"
						class="rounded-2xl border border-border/60 bg-background/70 p-4 text-left transition-all hover:-translate-y-0.5 hover:shadow-lg hover:shadow-black/10"
						onclick={() => handleTabChange('moments')}
					>
						<div class="text-2xl font-bold">{user.stats.moments_count}</div>
						<div class="text-sm text-muted-foreground">瞬间</div>
					</button>
					<button
						type="button"
						class="rounded-2xl border border-border/60 bg-background/70 p-4 text-left transition-all hover:-translate-y-0.5 hover:shadow-lg hover:shadow-black/10"
						onclick={() => handleTabChange('followers')}
					>
						<div class="text-2xl font-bold">{user.stats.followers_count}</div>
						<div class="text-sm text-muted-foreground">粉丝</div>
					</button>
					<button
						type="button"
						class="rounded-2xl border border-border/60 bg-background/70 p-4 text-left transition-all hover:-translate-y-0.5 hover:shadow-lg hover:shadow-black/10"
						onclick={() => handleTabChange('following')}
					>
						<div class="text-2xl font-bold">{user.stats.following_count}</div>
						<div class="text-sm text-muted-foreground">关注</div>
					</button>
				</div>
			</CardContent>
		</Card>

		<Tabs value={currentTab} onValueChange={handleTabChange}>
			<TabsList class="grid w-full grid-cols-3 rounded-2xl bg-card/70 p-1.5 shadow-sm shadow-black/5 ring-1 ring-border/40">
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
								<MomentCard moment={{
									...moment,
									user: user!,
									is_liked: false,
									is_favorited: false
								}} />
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
						{#each Array(5) as _}
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
						{#each followersPagination.items as follower (follower.user_id)}
							<a href="/users/{follower.user_id}">
								<Card class="cursor-pointer border-0 bg-card/70 shadow-sm shadow-black/5 transition-shadow hover:shadow-lg hover:shadow-black/10">
									<CardContent class="flex items-center gap-3 p-4">
										<Avatar class="h-12 w-12">
											<AvatarImage src={follower.avatar_url} alt={follower.nickname} />
											<AvatarFallback>{follower.nickname.slice(0, 2)}</AvatarFallback>
										</Avatar>
										<div class="min-w-0">
											<p class="truncate font-medium">{follower.nickname}</p>
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
						{#each Array(5) as _}
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
						{#each followingPagination.items as follow (follow.user_id)}
							<a href="/users/{follow.user_id}">
								<Card class="cursor-pointer border-0 bg-card/70 shadow-sm shadow-black/5 transition-shadow hover:shadow-lg hover:shadow-black/10">
									<CardContent class="flex items-center gap-3 p-4">
										<Avatar class="h-12 w-12">
											<AvatarImage src={follow.avatar_url} alt={follow.nickname} />
											<AvatarFallback>{follow.nickname.slice(0, 2)}</AvatarFallback>
										</Avatar>
										<div class="min-w-0">
											<p class="truncate font-medium">{follow.nickname}</p>
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
	{:else}
		<Card>
			<CardContent class="py-12 text-center">
				<p class="text-muted-foreground">用户不存在</p>
				<Button variant="outline" class="mt-4" onclick={() => goto('/')}>
					返回首页
				</Button>
			</CardContent>
		</Card>
	{/if}
</div>
