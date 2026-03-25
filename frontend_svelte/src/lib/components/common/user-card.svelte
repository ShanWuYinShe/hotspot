<script lang="ts">
	/**
	 * 通用用户卡片组件
	 * 
	 * 用于显示用户信息的统一UI组件
	 * 支持多种显示模式和自定义选项
	 */
	import { dev } from '$app/environment';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { presenceStore } from '$lib/stores/presence.svelte';
	import { isSuccess } from '$lib/api/client';
	import { reportsService, usersService } from '$lib/api/services';
	import { formatDistance } from '$lib/utils/format';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Card, CardContent, CardFooter } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import {
		DropdownMenu,
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuSeparator,
		DropdownMenuTrigger
	} from '$lib/components/ui/dropdown-menu';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';
	import User from 'lucide-svelte/icons/user';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Share2 from 'lucide-svelte/icons/share-2';
	import Flag from 'lucide-svelte/icons/flag';
	import type { UserInfo, UserItem } from '$lib/types/api';

	interface Props {
		/** 用户数据 */
		user: UserInfo | UserItem;
		/** 是否显示距离 */
		showDistance?: boolean;
		/** 是否显示简介 */
		showBio?: boolean;
		/** 是否显示标签 */
		showTags?: boolean;
		/** 是否显示在线状态 */
		showOnlineStatus?: boolean;
		/** 是否显示操作菜单 */
		showMenu?: boolean;
		/** 链接地址（默认跳转到用户详情） */
		href?: string;
		/** 自定义CSS类 */
		class?: string;
		/** 卡片变体 */
		variant?: 'default' | 'compact' | 'detailed' | 'square';
	}

	let {
		user,
		showDistance = false,
		showBio = false,
		showTags = false,
		showOnlineStatus = false,
		showMenu = true,
		href,
		class: className = '',
		variant = 'default'
	}: Props = $props();

	// Extract UserInfo from either UserInfo or UserItem
	const userInfo = $derived('user' in user ? user.user : user);

	// 计算实际链接
	const actualHref = $derived(href ?? `/users/${userInfo.user_id}`);

	// 格式化距离

	// 获取距离信息
	const distance = $derived(
		showDistance && 'location' in user && user.location?.distance
			? user.location.distance
			: null
	);

	// 获取在线状态
	// 优先从 presence store 读取实时状态，如果没有则从 user 对象读取
	const isOnline = $derived.by(() => {
		if (!showOnlineStatus) return false;
		
		// 优先使用 presence store 的实时状态
		const presenceStatus = presenceStore.getStatus(userInfo.user_id);
		if (presenceStatus !== 'offline') {
			// 如果 presence store 有状态且不是 offline，使用它
			return presenceStore.isOnline(userInfo.user_id);
		}
		
		// 降级：从 user 对象读取（API 返回的静态状态）
		return 'presence_status' in user && user.presence_status === 'online';
	});

	// 获取标签列表（最多显示3个）
	const displayTags = $derived(
		showTags && 'tags' in user && user.tags ? user.tags.slice(0, 3) : []
	);

	// 检查是否是自己
	const isMe = $derived(authStore.currentUserId === userInfo.user_id);
	let isFollowing = $state(false);
	let isFollowPending = $state(false);
	let isReporting = $state(false);

	$effect(() => {
		const relation = 'relation' in user ? (user as UserItem).relation : undefined;
		isFollowing = relation?.is_following ?? false;
	});

	function isInteractiveTarget(target: EventTarget | null): boolean {
		return target instanceof HTMLElement
			? Boolean(
				target.closest(
					'button, a, input, select, textarea, summary, [role="button"], [role="menuitem"], [data-card-action]'
				)
			)
			: false;
	}

	function handleCardClick(event: MouseEvent) {
		if (event.defaultPrevented || isInteractiveTarget(event.target)) {
			return;
		}

		goto(actualHref);
	}

	function handleCardKeydown(event: KeyboardEvent) {
		if (event.defaultPrevented || isInteractiveTarget(event.target)) {
			return;
		}

		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			goto(actualHref);
		}
	}

	// 菜单操作处理函数
	function handleViewProfile(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		goto(actualHref);
	}

	function handleSendMessage(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModal();
			return;
		}
		goto(`/chat/users/${userInfo.user_id}`);
	}

	async function handleFollow(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModal();
			return;
		}
		if (isFollowPending) return;

		isFollowPending = true;
		try {
			const response = isFollowing
				? await usersService.unfollow({ user_id: userInfo.user_id })
				: await usersService.follow({ user_id: userInfo.user_id, source: 'profile' });
			if (!isSuccess(response)) {
				toast.error(response.message || '操作失败');
				return;
			}
			isFollowing = !isFollowing;
			toast.success(isFollowing ? '关注成功' : '已取消关注');
		} catch {
			toast.error(isFollowing ? '取消关注失败' : '关注失败');
		} finally {
			isFollowPending = false;
		}
	}

	async function handleShare(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		const shareData = {
			title: `${userInfo.nickname} - HotSpot`,
			text: `查看 ${userInfo.nickname} 的个人主页`,
			url: `${window.location.origin}/users/${userInfo.user_id}`
		};

		if (navigator.share) {
			try {
				await navigator.share(shareData);
			} catch (err) {
				if ((err as Error).name !== 'AbortError') {
					if (dev) console.error('分享失败:', err);
				}
			}
		} else {
			// 降级：复制链接
			await navigator.clipboard.writeText(shareData.url);
			toast.success('链接已复制到剪贴板');
		}
	}

	async function handleReport(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModal();
			return;
		}
		if (isReporting) return;
		isReporting = true;
		try {
			const response = await reportsService.create({
				target_type: 'user',
				target_id: userInfo.user_id,
				reason: 'other'
			});
			if (!isSuccess(response)) {
				toast.error(response.message || '举报提交失败');
				return;
			}
			toast.success('举报已提交，我们会尽快处理');
		} catch {
			toast.error('举报提交失败');
		} finally {
			isReporting = false;
		}
	}
</script>


<div class="group block animate-bounce-in">
	<Card
		variant="default"
		glow="soft"
		interactive
		tabindex={0}
		role="link"
		onclick={handleCardClick}
		onkeydown={handleCardKeydown}
		class="group/card flex h-full flex-col overflow-hidden {className}"
	>
		<CardContent class="flex flex-1 flex-col p-6">
			{#if variant === 'square'}
				<!-- 方块模式：信息密度更高，卡片高度随内容自适应 -->
				<div class="flex flex-1 flex-col gap-3">
					<div
						class="relative overflow-hidden rounded-2xl border border-border/40 bg-linear-to-br from-primary/15 via-secondary/10 to-muted/30 p-3.5"
					>
						<div class="pointer-events-none absolute inset-0 bg-linear-to-t from-black/5 to-transparent dark:from-white/5"></div>

						<div class="relative flex items-start justify-between gap-3">
							<div class="flex min-w-0 items-center gap-2.5">
								<div class="relative shrink-0">
									<Avatar glow class="size-12">
										<AvatarImage src={userInfo.avatar_url} alt={userInfo.nickname} />
										<AvatarFallback class="bg-linear-to-br from-primary/25 to-secondary/25 text-primary font-bold text-sm">
											{(userInfo.nickname || '用户').slice(0, 2)}
										</AvatarFallback>
									</Avatar>
									{#if isOnline}
										<span
											class="absolute -bottom-0.5 -right-0.5 size-3.5 rounded-full border-2 border-background bg-green-500 shadow-sm shadow-green-500/50"
										></span>
									{/if}
								</div>

								<div class="min-w-0 flex-1">
									<h3 class="truncate text-sm font-semibold">{userInfo.nickname}</h3>
									<p class="mt-0.5 text-[11px] text-muted-foreground">
										{isOnline ? '在线中' : '最近活跃'}
									</p>
								</div>
							</div>

							{#if distance}
								<div
									class="flex shrink-0 items-center gap-1 rounded-2xl bg-background/70 px-2 py-1 text-[10px] text-muted-foreground"
								>
									<MapPin class="size-3" />
									{formatDistance(distance)}
								</div>
							{/if}
						</div>

						{#if showBio && 'bio' in user && user.bio}
							<p
								class="relative mt-3 text-sm leading-relaxed text-muted-foreground"
								style="display: -webkit-box; -webkit-line-clamp: {user.bio.length > 120 ? 4 : user.bio.length > 60 ? 3 : 2}; -webkit-box-orient: vertical; overflow: hidden;"
							>
								{user.bio}
							</p>
						{:else if showBio}
							<p class="relative mt-3 text-xs italic text-muted-foreground/60">
								这个人很神秘，暂时还没有留下简介
							</p>
						{/if}
					</div>

					{#if displayTags.length > 0}
						<div class="flex flex-wrap gap-1.5">
							{#each displayTags as tag (tag)}
								<Badge variant="secondary" class="rounded-2xl border border-border/50 bg-muted/50 px-2 py-0.5 text-[10px]">
									#{tag}
								</Badge>
							{/each}
						</div>
					{/if}
				</div>
			{:else if variant === 'compact'}
				<!-- 紧凑模式：水平布局，适合列表 -->
				<div class="flex items-center gap-4">
					<div class="relative shrink-0">
						<Avatar
							glow
							class="h-12 w-12"
						>
							<AvatarImage src={userInfo.avatar_url} alt={userInfo.nickname} />
							<AvatarFallback class="bg-linear-to-br from-primary/20 to-secondary/20 text-primary font-bold">{(userInfo.nickname || '用户').slice(0, 2)}</AvatarFallback>
						</Avatar>
					{#if isOnline}
						<span
							class="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-background bg-green-500 shadow-sm shadow-green-500/50"
						></span>
					{/if}
					</div>
					<div class="min-w-0 flex-1">
						<div class="flex items-center justify-between gap-2">
							<h3 class="truncate font-semibold">{userInfo.nickname}</h3>
							{#if distance}
								<span
									class="flex shrink-0 items-center gap-0.5 rounded-2xl bg-muted/50 px-1.5 py-0.5 text-[10px] text-muted-foreground"
								>
									<MapPin class="h-3 w-3" />
									{formatDistance(distance)}
								</span>
							{/if}
						</div>
						{#if showBio && 'bio' in user && user.bio}
							<p class="mt-1 line-clamp-1 text-sm text-muted-foreground">{user.bio}</p>
						{/if}
					</div>
				</div>
			{:else if variant === 'detailed'}
				<!-- 详细模式：垂直布局，显示更多信息 -->
				<div class="space-y-4">
					<div class="flex items-center gap-4">
						<div class="relative shrink-0">
							<Avatar
								glow
								class="h-16 w-16"
							>
								<AvatarImage src={userInfo.avatar_url} alt={userInfo.nickname} />
								<AvatarFallback class="text-lg bg-linear-to-br from-primary/20 to-secondary/20 text-primary font-bold">{(userInfo.nickname || '用户').slice(0, 2)}</AvatarFallback>
							</Avatar>
						{#if isOnline}
							<span
								class="absolute bottom-0 right-0 h-4 w-4 rounded-full border-2 border-background bg-green-500 shadow-sm shadow-green-500/50"
							></span>
						{/if}
						</div>
						<div class="min-w-0 flex-1">
							<h3 class="truncate text-lg font-semibold">{userInfo.nickname}</h3>
							{#if distance}
								<p class="mt-1 flex items-center gap-1 text-sm text-muted-foreground">
									<MapPin class="h-3.5 w-3.5" />
									{formatDistance(distance)}
								</p>
							{/if}
						</div>
					</div>
					{#if showBio && 'bio' in user && user.bio}
						<p class="text-sm text-muted-foreground">{user.bio}</p>
					{/if}
					{#if displayTags.length > 0}
						<div class="flex flex-wrap gap-2">
							{#each displayTags as tag (tag)}
								<Badge variant="secondary" class="rounded-2xl text-xs">#{tag}</Badge>
							{/each}
						</div>
					{/if}
				</div>
			{:else}
				<!-- 默认模式：平衡的布局 -->
				<div class="flex items-center gap-4">
					<div class="relative shrink-0">
						<Avatar
							glow
							class="h-12 w-12"
						>
							<AvatarImage src={userInfo.avatar_url} alt={userInfo.nickname} />
							<AvatarFallback class="bg-linear-to-br from-primary/20 to-secondary/20 text-primary font-bold">{(userInfo.nickname || '用户').slice(0, 2)}</AvatarFallback>
						</Avatar>
					{#if isOnline}
						<span
							class="absolute bottom-0 right-0 h-3 w-3 rounded-full border-2 border-background bg-green-500 shadow-sm shadow-green-500/50"
						></span>
					{/if}
					</div>
					<div class="min-w-0 flex-1">
						<h3 class="truncate font-semibold">{userInfo.nickname}</h3>
						{#if distance}
							<p class="flex items-center gap-1 text-xs text-muted-foreground">
								<MapPin class="h-3 w-3" />
								{formatDistance(distance)}
							</p>
						{/if}
					</div>
				</div>
				{#if showBio && 'bio' in user && user.bio}
					<p class="mt-3 line-clamp-2 text-sm text-muted-foreground">{user.bio}</p>
				{/if}
				{#if displayTags.length > 0}
					<div class="mt-3 flex flex-wrap gap-2">
						{#each displayTags as tag (tag)}
							<Badge variant="secondary" class="rounded-2xl text-xs">#{tag}</Badge>
						{/each}
					</div>
				{/if}
			{/if}
		</CardContent>

		<!-- 操作按钮区域 -->
		{#if !isMe}
			<CardFooter class="border-t border-border/40 bg-muted/10 p-3 md:bg-muted/5 md:transition-colors md:group-hover/card:bg-muted/20">
				<div class="flex w-full flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
					<Button
						variant="ghost"
						size="sm"
						data-card-action
						class="justify-start sm:justify-center"
						onclick={handleSendMessage}
					>
						<MessageCircle class="h-4 w-4" />
						<span class="font-medium text-xs">消息</span>
					</Button>
					<div class="flex flex-wrap items-center justify-end gap-2">
						<Button
							variant="ghost"
							size="sm"
							data-card-action
							onclick={handleFollow}
							disabled={isFollowPending}
						>
							{#if isFollowPending}
								<Loader2 class="h-4 w-4 animate-spin" />
							{:else}
								<UserPlus class="h-4 w-4" />
							{/if}
							<span class="font-medium text-xs">{isFollowing ? '取消关注' : '关注'}</span>
						</Button>
						{#if showMenu}
							<DropdownMenu>
								<DropdownMenuTrigger>
									<Button
										variant="ghost"
										size="icon"
										data-card-action
										onclick={(e: MouseEvent) => {
											e.preventDefault();
											e.stopPropagation();
										}}
										aria-label="更多操作"
									>
										<MoreVertical class="h-4 w-4" />
									</Button>
								</DropdownMenuTrigger>
								<DropdownMenuContent align="end" class="w-48">
									<DropdownMenuItem onclick={handleViewProfile}>
										<User class="mr-2 h-4 w-4" />
										<span>查看主页</span>
									</DropdownMenuItem>
									<DropdownMenuItem onclick={handleShare}>
										<Share2 class="mr-2 h-4 w-4" />
										<span>分享</span>
									</DropdownMenuItem>
									<DropdownMenuSeparator />
									<DropdownMenuItem onclick={handleReport} class="text-destructive">
										<Flag class="mr-2 h-4 w-4" />
										<span>举报</span>
									</DropdownMenuItem>
								</DropdownMenuContent>
							</DropdownMenu>
						{/if}
					</div>
				</div>
			</CardFooter>
		{/if}
	</Card>
</div>
