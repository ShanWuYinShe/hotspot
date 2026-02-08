<script lang="ts">
	/**
	 * 通用群组卡片组件
	 * 
	 * 用于显示群组信息的统一UI组件
	 * 支持多种显示模式和自定义选项
	 */
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth.svelte';
	import { uiStore } from '$lib/stores/ui.svelte';
	import { toast } from '$lib/stores/toast.svelte';
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
	import Share2 from 'lucide-svelte/icons/share-2';
	import Flag from 'lucide-svelte/icons/flag';
	import type { GroupInfo } from '$lib/types/api';

	interface Props {
		/** 群组数据 */
		group: GroupInfo;
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

	// 计算实际链接
	const actualHref = $derived(href ?? `/groups/${group.group_id}`);

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

	// 菜单操作处理函数
	function handleViewDetail(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		goto(actualHref);
	}

	function handleJoinGroup(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModal();
			return;
		}
		// TODO: 实现加入群组功能
		toast.info('加入群组功能开发中');
	}

	async function handleShare(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		const shareData = {
			title: `${group.name} - HotSpot`,
			text: `加入 ${group.name} 群组`,
			url: `${window.location.origin}/groups/${group.group_id}`
		};

		if (navigator.share) {
			try {
				await navigator.share(shareData);
			} catch (err) {
				if ((err as Error).name !== 'AbortError') {
					console.error('分享失败:', err);
				}
			}
		} else {
			// 降级：复制链接
			await navigator.clipboard.writeText(shareData.url);
			toast.success('链接已复制到剪贴板');
		}
	}

	function handleReport(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		if (!authStore.isAuthenticated) {
			uiStore.openLoginModal();
			return;
		}
		// TODO: 实现举报功能
		toast.info('举报功能开发中');
	}
</script>

<a href={actualHref} class="group block animate-bounce-in">
	<Card
		variant="secondary"
		glow="soft"
		interactive
		class="group/card h-full {className}"
	>
		<CardContent class="p-5">
			{#if variant === 'square'}
				<!-- 方块模式：Pinterest/小红书风格，高度自适应 -->
				<!-- 使用group_id的哈希值生成稳定的随机高度 -->
				{@const heightVariants = [125, 145, 165, 135, 155, 140, 150, 160]}
				{@const idHash = group.group_id.split('').reduce((acc, char) => acc + char.charCodeAt(0), 0)}
				{@const minHeight = heightVariants[idHash % heightVariants.length]}
				<div class="flex flex-col justify-between space-y-2.5" style="min-height: {minHeight}px;">
					<!-- 头像和基本信息 -->
					<div class="flex items-center gap-2.5">
						<div class="relative shrink-0">
							<Avatar
								glow
								class="size-12"
							>
								<AvatarImage src={group.avatar_url} alt={group.name} />
								<AvatarFallback class="bg-linear-to-br from-secondary/20 to-primary/20 text-primary font-bold text-sm">
									{group.name.slice(0, 2)}
								</AvatarFallback>
							</Avatar>
							{#if unreadCount > 0}
								<Badge 
									variant="accent" 
									class="absolute -top-1 -right-1 size-5 rounded-full p-0 flex items-center justify-center text-[9px]"
								>
									{unreadCount > 99 ? '99+' : unreadCount}
								</Badge>
							{/if}
						</div>
						<div class="min-w-0 flex-1">
							<div class="flex items-center gap-1">
								<h3 class="truncate font-semibold text-sm">{group.name}</h3>
								{#if isPrivate}
									<Lock class="size-3 text-muted-foreground shrink-0" />
								{/if}
							</div>
							<div class="flex items-center gap-2 text-xs text-muted-foreground">
								{#if showMembers}
									<span class="flex items-center gap-0.5">
										<Users class="size-3" />
										{group.members_count}人
									</span>
								{/if}
								{#if distance}
									<span class="flex items-center gap-0.5">
										<MapPin class="size-3" />
										{formatDistance(distance)}
									</span>
								{/if}
							</div>
						</div>
					</div>

					<div class="space-y-2">
						<!-- 描述（根据长度自适应行数：1-4行） -->
						{#if 'description' in group}
							{@const groupWithDesc = group as { description?: string }}
							{#if groupWithDesc.description}
								{@const desc = groupWithDesc.description}
								<p class="text-sm leading-relaxed text-muted-foreground" style="display: -webkit-box; -webkit-line-clamp: {desc.length > 100 ? 4 : desc.length > 50 ? 3 : 2}; -webkit-box-orient: vertical; overflow: hidden;">
									{desc}
								</p>
							{:else}
								<!-- 没有描述时显示占位符 -->
								<p class="text-xs text-muted-foreground/50 italic">群主很懒，还没有添加群组描述</p>
							{/if}
						{/if}

						<!-- 标签（如果有） -->
						{#if 'tags' in group}
							{@const groupWithTags = group as { tags?: string[] }}
							{#if groupWithTags.tags && groupWithTags.tags.length > 0}
								<div class="flex flex-wrap gap-1.5">
									{#each groupWithTags.tags.slice(0, 5) as tag}
										<Badge variant="secondary" class="rounded-full text-[10px] px-2 py-0.5">#{tag}</Badge>
									{/each}
								</div>
							{/if}
						{/if}
					</div>
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
							<AvatarImage src={group.avatar_url} alt={group.name} />
							<AvatarFallback class="bg-gradient-to-br from-secondary/20 to-primary/20 text-primary font-bold text-base">
								{group.name.slice(0, 2)}
							</AvatarFallback>
						</Avatar>
					</div>
					
					<!-- 群组信息 -->
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2">
							<h3 class="font-bold text-foreground truncate">{group.name}</h3>
							{#if isPrivate}
								<Lock class="h-4 w-4 text-muted-foreground shrink-0" />
							{/if}
						</div>
						
						<div class="flex items-center gap-3 mt-1">
							{#if showMembers}
								<span class="flex items-center gap-1 text-xs text-muted-foreground">
									<Users class="h-3.5 w-3.5" />
									{group.members_count}人
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
							class="shrink-0 h-7 w-7 rounded-full p-0 flex items-center justify-center"
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
			>
				<UserPlus class="h-4 w-4" />
				<span class="font-medium text-xs">加入群组</span>
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
