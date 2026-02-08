<script lang="ts">
	import { onMount } from 'svelte';
	import { formatRelativeTime } from '$lib/utils/format';
	import { privateChatsService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { presenceStore } from '$lib/stores/presence.svelte';
	import { Card, CardContent } from '$lib/components/ui/card';
	import { Avatar, AvatarFallback, AvatarImage } from '$lib/components/ui/avatar';
	import { Badge } from '$lib/components/ui/badge';
	import { Tabs, TabsContent, TabsList, TabsTrigger } from '$lib/components/ui/tabs';
	import ListSkeleton from '$lib/components/common/list-skeleton.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import type { PrivateChatListItem, MyGroupItem } from '$lib/types/api';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import Users from 'lucide-svelte/icons/users';

	let currentTab = $state<'private' | 'group'>('private');
	let privateChats = $state<PrivateChatListItem[]>([]);
	let groupChats = $state<MyGroupItem[]>([]);
	let isLoadingPrivate = $state(true);
	let isLoadingGroup = $state(false);
	let groupsLoaded = $state(false);

	async function loadPrivateChats() {
		try {
			const response = await privateChatsService.getChats({ limit: 50 });
			if (isSuccess(response) && response.data) {
				privateChats = response.data.items;
			}
		} catch {
			toast.error('加载失败');
		} finally {
			isLoadingPrivate = false;
		}
	}

	async function loadGroupChats() {
		if (groupsLoaded) return;
		
		isLoadingGroup = true;
		try {
			const response = await groupsService.getMyGroups({ limit: 50 });
			if (isSuccess(response) && response.data) {
				groupChats = response.data.items;
				groupsLoaded = true;
			}
		} catch {
			toast.error('加载群聊失败');
		} finally {
			isLoadingGroup = false;
		}
	}

	function handleTabChange(value: string) {
		currentTab = value as 'private' | 'group';
		if (value === 'group' && !groupsLoaded) {
			loadGroupChats();
		}
	}

	onMount(() => {
		loadPrivateChats();
	});
</script>

<svelte:head>
	<title>消息 - HotSpot</title>
</svelte:head>

<div class="space-y-6">
	<h1 class="text-2xl font-bold">消息</h1>

	<Tabs value={currentTab} onValueChange={handleTabChange}>
		<TabsList class="grid w-full grid-cols-2">
			<TabsTrigger value="private">
				<MessageCircle class="mr-2 h-4 w-4" />
				私聊
			</TabsTrigger>
			<TabsTrigger value="group">
				<Users class="mr-2 h-4 w-4" />
				群聊
			</TabsTrigger>
		</TabsList>

		<!-- 私聊列表 -->
		<TabsContent value="private" class="mt-6">
			{#if isLoadingPrivate}
				<ListSkeleton type="chat" count={5} layout="list" />
			{:else if privateChats.length === 0}
				<EmptyState 
					icon={MessageCircle} 
					title="暂无私聊" 
					description="去探索发现感兴趣的人吧" 
				/>
			{:else}
				<div class="space-y-6">
					{#each privateChats as chat (chat.other_user.user_id)}
						{@const isOnline = presenceStore.isOnline(chat.other_user.user_id) || chat.is_online}
						<a href="/chat/users/{chat.other_user.user_id}">
							<Card variant="default" glow="soft" interactive class="cursor-pointer">
								<CardContent class="flex items-center gap-4 p-5">
									<div class="relative">
										<Avatar class="h-14 w-14" glow>
											<AvatarImage src={chat.other_user.avatar_url} alt={chat.other_user.nickname} />
											<AvatarFallback class="bg-linear-to-br from-primary/20 to-secondary/20 text-primary font-bold">
												{chat.other_user.nickname.slice(0, 2)}
											</AvatarFallback>
										</Avatar>
										{#if isOnline}
											<div class="absolute -bottom-0.5 -right-0.5 h-4 w-4 rounded-full border-2 border-background bg-green-500 shadow-sm shadow-green-500/50"></div>
										{/if}
									</div>
									<div class="min-w-0 flex-1">
										<div class="flex items-center justify-between mb-1.5">
											<p class="font-semibold text-base">{chat.other_user.nickname}</p>
											<span class="text-xs text-muted-foreground">
												{formatRelativeTime(chat.last_activity_at)}
											</span>
										</div>
										<div class="flex items-center justify-between gap-2">
											<p class="truncate text-sm text-muted-foreground leading-relaxed">
												{#if chat.last_message}
													{chat.last_message.content || '[媒体消息]'}
												{:else}
													暂无消息
												{/if}
											</p>
											{#if chat.unread_count > 0}
												<Badge variant="destructive" class="ml-2 shrink-0">
													{chat.unread_count > 99 ? '99+' : chat.unread_count}
												</Badge>
											{/if}
										</div>
									</div>
								</CardContent>
							</Card>
						</a>
					{/each}
				</div>
			{/if}
		</TabsContent>

		<!-- 群聊列表 -->
		<TabsContent value="group" class="mt-6">
			{#if isLoadingGroup}
				<ListSkeleton type="chat" count={5} layout="list" />
			{:else if groupChats.length === 0}
				<EmptyState 
					icon={Users} 
					title="暂无群聊" 
					description="去探索加入感兴趣的群组吧" 
				/>
			{:else}
				<div class="space-y-6">
					{#each groupChats as group (group.group_id)}
						<a href="/chat/groups/{group.group_id}">
							<Card variant="default" glow="soft" interactive class="cursor-pointer">
								<CardContent class="flex items-center gap-4 p-5">
									<Avatar class="h-14 w-14" glow>
										<AvatarImage src={group.avatar_url} alt={group.name} />
										<AvatarFallback class="bg-linear-to-br from-primary/20 to-secondary/20 text-primary">
											<Users class="h-7 w-7" />
										</AvatarFallback>
									</Avatar>
									<div class="min-w-0 flex-1">
										<div class="flex items-center justify-between mb-1.5">
											<div class="flex items-center gap-2 min-w-0 flex-1">
											<p class="font-semibold text-base truncate">{group.name}</p>
											{#if group.my_state?.role === 'owner'}
												<Badge variant="default" class="text-xs shrink-0">群主</Badge>
											{:else if group.my_state?.role === 'admin'}
												<Badge variant="secondary" class="text-xs shrink-0">管理员</Badge>
											{/if}
										</div>
											<span class="text-xs text-muted-foreground shrink-0 ml-2">
												{group.members_count} 人
											</span>
										</div>
										<div class="flex items-center justify-between gap-2">
											<p class="text-sm text-muted-foreground leading-relaxed">
												{formatRelativeTime(group.my_state?.joined_at ?? group.created_at ?? '')}
											</p>
											{#if (group.my_state?.unread_count ?? 0) > 0}
												<Badge variant="destructive" class="ml-2 shrink-0">
													{(group.my_state?.unread_count ?? 0) > 99 ? '99+' : (group.my_state?.unread_count ?? 0)}
												</Badge>
											{/if}
										</div>
									</div>
								</CardContent>
							</Card>
						</a>
					{/each}
				</div>
			{/if}
		</TabsContent>
	</Tabs>
</div>
