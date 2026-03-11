<script lang="ts">
	/**
	 * 通用群组卡片组件
	 * 
	 * 用于显示群组信息的统一UI组件
	 * 支持多种显示模式和自定义选项
	 */
	import { dev } from '$app/environment';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
	import { isSuccess } from '$lib/api/client';
	import { groupsService, reportsService } from '$lib/api/services';
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
	import Users from 'lucide-svelte/icons/users';
	import MapPin from 'lucide-svelte/icons/map-pin';
	import Lock from 'lucide-svelte/icons/lock';
	import MoreVertical from 'lucide-svelte/icons/more-vertical';
	import Info from 'lucide-svelte/icons/info';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import Loader2 from 'lucide-svelte/icons/loader-2';
	import Share2 from 'lucide-svelte/icons/share-2';
	import Flag from 'lucide-svelte/icons/flag';
	import type { GroupItem, GroupInfo } from '$lib/types/api';

	interface Props {
		/** 群组数据 */
		group: GroupItem | GroupInfo;
		/** 是否显示距离 */
		showDistance?: boolean;
		/** 是否显示成员数 */
		showMembers?: boolean;
		/** 是否显示未读消息 */
		showUnread?: boolean;
		/** 是否显示操作菜单 */
		showMenu?: boolean;
		/** 链接地址（默认跳转到群组详情） */
		href?: string;
		/** 自定义CSS类 */
		class?: string;
		/** 卡片变体 */
		variant?: 'default' | 'compact' | 'square';
	}

	let {
		group,
		showDistance = false,
		showMembers = true,
		showUnread = false,
		showMenu = true,
		href,
		class: className = '',
		variant = 'default'
	}: Props = $props();

	// Extract GroupInfo from either GroupItem or GroupInfo
	const groupInfo = $derived('group' in group ? group.group : group);
	
	// 计算实际链接
	const actualHref = $derived(href ?? `/groups/${groupInfo.group_id}`);

	// 获取距离信息（单位：米）
	const distance = $derived.by(() => {
		if (!showDistance || !('location' in group)) return null;
		const loc = group.location as { distance?: number };
		return loc?.distance ?? null;
	});

	// 获取未读数
	const unreadCount = $derived.by(() => {
		if (!showUnread) return 0;
		const groupWithState = group as { my_state?: { unread_count?: number } };
		if (groupWithState.my_state?.unread_count != null) {
			return groupWithState.my_state.unread_count;
		}
		const groupWithUnread = group as { unread_count?: number };
		return groupWithUnread.unread_count ?? 0;
	});

	// 获取是否私密
	const isPrivate = $derived('has_password' in group ? group.has_password : false);
	let joinedOverride = $state<boolean | null>(null);
	let isJoining = $state(false);
	let isReporting = $state(false);
	const isJoined = $derived.by(() => {
		if (joinedOverride != null) return joinedOverride;
		const groupWithState = group as { my_state?: { is_joined?: boolean } };
		return groupWithState.my_state?.is_joined ?? false;
	});

	// 菜单操作处理函数
	function handleViewDetail(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		goto(actualHref);
	}

	async function handleJoinGroup(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModal();
			return;
		}
		if (isJoining) return;
		if (isJoined) {
			await goto(`/chat/groups/${groupInfo.group_id}`);
			return;
		}

		isJoining = true;
		try {
			const response = await groupsService.join({ group_id: groupInfo.group_id });
			if (!isSuccess(response)) {
				toast.error(response.message || '加入群组失败');
				return;
			}

			if (response.data?.status === 'pending_approval') {
				toast.info('已提交入群申请，等待管理员审核');
				return;
			}

			joinedOverride = true;
			toast.success('加入群组成功');
			await goto(`/chat/groups/${groupInfo.group_id}`);
		} catch {
			toast.error('加入群组失败');
		} finally {
			isJoining = false;
		}
	}

	async function handleShare(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		const shareData = {
			title: `${groupInfo.name} - HotSpot`,
			text: `加入 ${groupInfo.name} 群组`,
			url: `${window.location.origin}/groups/${groupInfo.group_id}`
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
				target_type: 'group',
				target_id: groupInfo.group_id,
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

<a href={actualHref} class="group block animate-bounce-in">
	<Card
		variant="secondary"
		glow="soft"
		interactive
		class="group/card flex h-full flex-col {className}"
	>
		<CardContent class="flex flex-1 flex-col p-5">
			{#if variant === 'square'}
				<!-- 方块模式：强调群组信息，同时让卡片高度随内容自适应 -->
				<div class="flex flex-1 flex-col gap-3">
					<div
						class="relative overflow-hidden rounded-2xl border border-border/40 bg-linear-to-br from-secondary/15 via-primary/10 to-muted/30 p-3.5"
					>
						<div class="pointer-events-none absolute inset-0 bg-linear-to-t from-black/5 to-transparent dark:from-white/5"></div>

						<div class="relative flex items-start justify-between gap-3">
							<div class="flex min-w-0 items-center gap-2.5">
								<div class="relative shrink-0">
									<Avatar glow class="size-12">
										<AvatarImage src={groupInfo.avatar_url} alt={groupInfo.name} />
										<AvatarFallback class="bg-linear-to-br from-secondary/25 to-primary/25 text-primary font-bold text-sm">
											{groupInfo.name.slice(0, 2)}
										</AvatarFallback>
									</Avatar>
									{#if unreadCount > 0}
										<Badge
											variant="accent"
											class="absolute -top-1 -right-1 size-5 rounded-xl p-0 flex items-center justify-center text-[9px]"
										>
											{unreadCount > 99 ? '99+' : unreadCount}
										</Badge>
									{/if}
								</div>

								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-1">
										<h3 class="truncate text-sm font-semibold">{groupInfo.name}</h3>
										{#if isPrivate}
											<Lock class="size-3 shrink-0 text-muted-foreground" />
										{/if}
									</div>
									<div class="mt-0.5 flex items-center gap-2 text-[11px] text-muted-foreground">
										{#if showMembers}
											<span class="flex items-center gap-0.5">
												<Users class="size-3" />
												{groupInfo.members_count}人
											</span>
										{/if}
									</div>
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

						{#if 'description' in group}
							{@const groupWithDesc = group as { description?: string }}
							{#if groupWithDesc.description}
								{@const desc = groupWithDesc.description}
								<p
									class="relative mt-3 text-sm leading-relaxed text-muted-foreground"
									style="display: -webkit-box; -webkit-line-clamp: {desc.length > 120 ? 4 : desc.length > 60 ? 3 : 2}; -webkit-box-orient: vertical; overflow: hidden;"
								>
									{desc}
								</p>
							{:else}
								<p class="relative mt-3 text-xs italic text-muted-foreground/60">
									群主很懒，还没有添加群组描述
								</p>
							{/if}
						{/if}
					</div>

					{#if 'tags' in group}
						{@const groupWithTags = group as { tags?: string[] }}
						{#if groupWithTags.tags && groupWithTags.tags.length > 0}
							<div class="flex flex-wrap gap-1.5">
								{#each groupWithTags.tags.slice(0, 5) as tag (tag)}
									<Badge variant="secondary" class="rounded-2xl border border-border/50 bg-muted/50 px-2 py-0.5 text-[10px]">
										#{tag}
									</Badge>
								{/each}
							</div>
						{/if}
					{/if}
				</div>
			{:else}
				<!-- 默认模式：水平布局 -->
				<div class="flex items-center gap-4">
					<!-- 群组头像（发光） -->
					<div class="relative shrink-0">
						<Avatar
							glow
							class="h-14 w-14"
						>
							<AvatarImage src={groupInfo.avatar_url} alt={groupInfo.name} />
							<AvatarFallback class="bg-linear-to-br from-secondary/20 to-primary/20 text-primary font-bold text-base">
								{groupInfo.name.slice(0, 2)}
							</AvatarFallback>
						</Avatar>
					</div>
					
					<!-- 群组信息 -->
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2">
							<h3 class="font-bold text-foreground truncate">{groupInfo.name}</h3>
							{#if isPrivate}
								<Lock class="h-4 w-4 text-muted-foreground shrink-0" />
							{/if}
						</div>
						
						<div class="flex items-center gap-3 mt-1">
							{#if showMembers}
								<span class="flex items-center gap-1 text-xs text-muted-foreground">
									<Users class="h-3.5 w-3.5" />
									{groupInfo.members_count}人
								</span>
							{/if}
							
							{#if distance}
								<Badge variant="accent" class="text-[10px] px-2 py-0.5">
									<MapPin class="h-3 w-3" />
									{formatDistance(distance)}
								</Badge>
							{/if}
						</div>
					</div>
					
					<!-- 未读消息（脉冲发光） -->
					{#if unreadCount > 0}
						<Badge 
							variant="accent" 
							class="shrink-0 h-7 w-7 rounded-xl p-0 flex items-center justify-center"
						>
							{unreadCount > 99 ? '99+' : unreadCount}
						</Badge>
					{/if}
				</div>
			{/if}
		</CardContent>

		<!-- 操作按钮区域 -->
		<CardFooter class="border-t border-border/40 bg-muted/5 p-3 transition-colors group-hover/card:bg-muted/20">
			<div class="flex w-full items-center justify-between">
			<Button
				variant="ghost"
				size="sm"
				onclick={handleJoinGroup}
				disabled={isJoining}
			>
				{#if isJoining}
					<Loader2 class="h-4 w-4 animate-spin" />
				{:else}
					<UserPlus class="h-4 w-4" />
				{/if}
				<span class="font-medium text-xs">
					{#if isJoining}
						加入中
					{:else if isJoined}
						进入群聊
					{:else}
						加入群组
					{/if}
				</span>
			</Button>
			{#if showMenu}
				<DropdownMenu>
					<DropdownMenuTrigger>
						<Button
							variant="ghost"
							size="icon"
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
							<DropdownMenuItem onclick={handleViewDetail}>
								<Info class="mr-2 h-4 w-4" />
								<span>查看详情</span>
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
		</CardFooter>
	</Card>
</a>
