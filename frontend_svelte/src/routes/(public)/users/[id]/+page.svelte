<script lang="ts">
	import { goto } from '$app/navigation';
	import { untrack } from 'svelte';
	import { usersService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { usePagination } from '$lib/hooks/use-pagination.svelte';
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
	let isActionLoading = $state(false);
	let currentTab = $state<'moments' | 'followers' | 'following'>('moments');

	// userId 变化时重置子 tab 分页状态并触发首次加载
	$effect(() => {
		userId; // 依赖声明
		untrack(() => {
			user = data.user;
			isFollowing = data.user?.relation?.is_following ?? false;
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
			uiStore.openLoginModalWithRedirect({ onSuccess: handleFollow });
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
	<!-- Sticky header -->
	<div
		class="sticky top-0 z-10 flex items-center gap-4 px-4 py-3"
		style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
	>
		<button
			class="flex items-center justify-center rounded-full outline-none"
			style="width: 36px; height: 36px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
			onclick={() => history.back()}
			aria-label="返回"
		>
			<ArrowLeft size={18} color="var(--neu-text)" />
		</button>
		<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">用户主页</h1>
	</div>

	{#if !user}
		<div
			class="rounded-[24px] p-6"
			style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
		>
			<div class="flex flex-col items-center gap-4">
				<p style="color: var(--neu-text-muted);">
					{data.error || '用户不存在'}
				</p>
			</div>
		</div>
	{:else}
		<div
			class="rounded-[24px] p-6"
			style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
		>
			<div class="space-y-6">
				<div class="flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
					<div class="flex flex-col gap-4 sm:flex-row sm:items-center">
						<div class="overflow-hidden rounded-full" style="width: 96px; height: 96px; border: 2px solid var(--neu-accent-primary); opacity: 0.9;">
							<img src={user.user.avatar_url} alt={user.user.nickname} class="h-full w-full object-cover" />
						</div>
						<div>
							<h2 style="font-size: 24px; font-weight: 700; color: var(--neu-text);">{user.user.nickname}</h2>
							{#if user.bio}
								<p style="margin-top: 8px; max-width: 36rem; font-size: 14px; color: var(--neu-text-muted);">{user.bio}</p>
							{/if}
							{#if user.location}
								<p class="mt-2 flex items-center gap-1" style="font-size: 14px; color: var(--neu-text-muted);">
									<MapPin size={16} />
									{user.location.location_name}
								</p>
							{/if}
							{#if user.tags && user.tags.length > 0}
								<div class="mt-3 flex flex-wrap gap-2">
									{#each user.tags as tag (tag)}
										<span
											class="rounded-full px-2.5 py-1"
											style="font-size: 11px; color: var(--neu-accent-primary); background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
										>
											#{tag}
										</span>
									{/each}
								</div>
							{/if}
						</div>
					</div>

					{#if authStore.isAuthenticated && authStore.currentUserId !== userId}
						<div class="flex flex-wrap gap-3">
							<button
								class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none"
								style={isFollowing
									? 'box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-accent-primary); background: var(--neu-bg);'
									: 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 14px var(--neu-glow);'}
								onclick={handleFollow}
								disabled={isActionLoading}
							>
								{#if isActionLoading}
									<Loader2 size={16} class="animate-spin" />
								{:else if isFollowing}
									<UserCheck size={16} />
									已关注
								{:else}
									<UserPlus size={16} />
									关注
								{/if}
							</button>
							<button
								class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none"
								style="box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
								onclick={() => goto(`/chat/users/${userId}`)}
							>
								<MessageCircle size={16} />
								发消息
							</button>
						</div>
					{/if}
				</div>

				{#if user.social_context?.mutual_friends_count}
					<p style="font-size: 14px; color: var(--neu-text-muted);">
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
			</div>
		</div>

		<!-- Tab bar -->
		<div class="flex gap-2">
			{#each [
				{ value: 'moments', label: 'TA的瞬间', icon: Image },
				{ value: 'followers', label: '粉丝列表', icon: Users },
				{ value: 'following', label: '关注列表', icon: Users }
			] as tab (tab.value)}
				<button
					class="flex items-center gap-2 rounded-full px-4 py-2 font-medium outline-none transition-all"
					style={currentTab === tab.value
						? 'background: var(--neu-accent-primary); color: var(--neu-active-text); box-shadow: 0 4px 12px var(--neu-glow);'
						: 'background: var(--neu-bg); color: var(--neu-text-muted); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);'}
					onclick={() => handleTabChange(tab.value)}
				>
					<tab.icon size={16} />
					<span style="font-size: 13px;">{tab.label}</span>
				</button>
			{/each}
		</div>

		<!-- Tab content: Moments -->
		{#if currentTab === 'moments'}
			{#if momentsPagination.loading}
				<div class="space-y-4">
					{#each Array(3) as _, i (i)}
						<div
							class="rounded-[24px] p-4 space-y-3"
							style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
						>
							<div class="rounded" style="width: 100%; height: 14px; background: var(--neu-border);"></div>
							<div class="rounded" style="width: 75%; height: 14px; background: var(--neu-border);"></div>
							<div class="rounded" style="width: 100%; height: 128px; background: var(--neu-border);"></div>
						</div>
					{/each}
				</div>
			{:else if momentsPagination.items.length === 0}
				<div
					class="flex flex-col items-center justify-center rounded-[24px] py-12"
					style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
				>
					<Image size={40} color="var(--neu-text-muted)" />
					<p style="margin-top: 16px; color: var(--neu-text-muted);">TA 还没有发布瞬间</p>
				</div>
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
						<button
							class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none"
							style="box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
							onclick={() => loadMoments(false)}
							disabled={momentsPagination.loadingMore}
						>
							{#if momentsPagination.loadingMore}
								<Loader2 size={16} class="animate-spin" />
							{/if}
							加载更多
						</button>
					</div>
				{/if}
			{/if}
		{/if}

		<!-- Tab content: Followers -->
		{#if currentTab === 'followers'}
			{#if followersPagination.loading}
				<div class="space-y-2">
					{#each Array(5) as _, i (i)}
						<div
							class="flex items-center gap-4 rounded-[24px] p-4"
							style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
						>
							<div class="rounded-full" style="width: 48px; height: 48px; background: var(--neu-border);"></div>
							<div class="flex-1 space-y-2">
								<div class="rounded" style="width: 96px; height: 14px; background: var(--neu-border);"></div>
								<div class="rounded" style="width: 128px; height: 12px; background: var(--neu-border);"></div>
							</div>
						</div>
					{/each}
				</div>
			{:else if followersPagination.items.length === 0}
				<div
					class="flex flex-col items-center justify-center rounded-[24px] py-12"
					style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
				>
					<Users size={40} color="var(--neu-text-muted)" />
					<p style="margin-top: 16px; color: var(--neu-text-muted);">暂无粉丝</p>
				</div>
			{:else}
				<div class="grid gap-3 sm:grid-cols-2">
					{#each followersPagination.items as follower (follower.user.user_id)}
						<a href="/users/{follower.user.user_id}">
							<div
								class="flex cursor-pointer items-center gap-3 rounded-[24px] p-4 transition-all"
								style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
							>
								<div class="overflow-hidden rounded-full" style="width: 48px; height: 48px; border: 1.5px solid var(--neu-border);">
									<img src={follower.user.avatar_url} alt={follower.user.nickname} class="h-full w-full object-cover" />
								</div>
								<div class="min-w-0">
									<p class="truncate" style="font-weight: 500; font-size: 14px; color: var(--neu-text);">{follower.user.nickname}</p>
									<p style="font-size: 12px; color: var(--neu-text-muted);">查看主页</p>
								</div>
							</div>
						</a>
					{/each}
				</div>
				{#if followersPagination.hasMore}
					<div class="flex justify-center pt-4">
						<button
							class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none"
							style="box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
							onclick={() => loadFollowers(false)}
							disabled={followersPagination.loadingMore}
						>
							{#if followersPagination.loadingMore}
								<Loader2 size={16} class="animate-spin" />
							{/if}
							加载更多
						</button>
					</div>
				{/if}
			{/if}
		{/if}

		<!-- Tab content: Following -->
		{#if currentTab === 'following'}
			{#if followingPagination.loading}
				<div class="space-y-2">
					{#each Array(5) as _, i (i)}
						<div
							class="flex items-center gap-4 rounded-[24px] p-4"
							style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
						>
							<div class="rounded-full" style="width: 48px; height: 48px; background: var(--neu-border);"></div>
							<div class="flex-1 space-y-2">
								<div class="rounded" style="width: 96px; height: 14px; background: var(--neu-border);"></div>
								<div class="rounded" style="width: 128px; height: 12px; background: var(--neu-border);"></div>
							</div>
						</div>
					{/each}
				</div>
			{:else if followingPagination.items.length === 0}
				<div
					class="flex flex-col items-center justify-center rounded-[24px] py-12"
					style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
				>
					<Users size={40} color="var(--neu-text-muted)" />
					<p style="margin-top: 16px; color: var(--neu-text-muted);">暂无关注</p>
				</div>
			{:else}
				<div class="grid gap-3 sm:grid-cols-2">
					{#each followingPagination.items as follow (follow.user.user_id)}
						<a href="/users/{follow.user.user_id}">
							<div
								class="flex cursor-pointer items-center gap-3 rounded-[24px] p-4 transition-all"
								style="background: var(--neu-bg); box-shadow: 8px 8px 20px var(--neu-shadow-dark), -8px -8px 20px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
							>
								<div class="overflow-hidden rounded-full" style="width: 48px; height: 48px; border: 1.5px solid var(--neu-border);">
									<img src={follow.user.avatar_url} alt={follow.user.nickname} class="h-full w-full object-cover" />
								</div>
								<div class="min-w-0">
									<p class="truncate" style="font-weight: 500; font-size: 14px; color: var(--neu-text);">{follow.user.nickname}</p>
									<p style="font-size: 12px; color: var(--neu-text-muted);">查看主页</p>
								</div>
							</div>
						</a>
					{/each}
				</div>
				{#if followingPagination.hasMore}
					<div class="flex justify-center pt-4">
						<button
							class="flex items-center gap-2 rounded-full px-5 py-2.5 font-medium outline-none"
							style="box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border); color: var(--neu-text-muted); background: var(--neu-bg);"
							onclick={() => loadFollowing(false)}
							disabled={followingPagination.loadingMore}
						>
							{#if followingPagination.loadingMore}
								<Loader2 size={16} class="animate-spin" />
							{/if}
							加载更多
						</button>
					</div>
				{/if}
			{/if}
		{/if}
	{/if}
</div>
