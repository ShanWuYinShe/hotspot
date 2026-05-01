<script lang="ts">
	import { onMount } from 'svelte';
	import { withMinimumLoadingTime } from '$lib/utils/loading';
	import { formatRelativeTime } from '$lib/utils/format';
	import { privateChatsService, groupsService } from '$lib/api/services';
	import { isSuccess } from '$lib/api/client';
	import { toast } from '$lib/stores/toast.svelte';
	import { presenceStore } from '$lib/stores/presence.svelte';
	import { messageStore } from '$lib/stores/message.svelte';
	import EmptyState from '$lib/components/common/empty-state.svelte';
	import type { PrivateChatItem, GroupItem } from '$lib/types/api';
	import MessageCircle from 'lucide-svelte/icons/message-circle';
	import Users from 'lucide-svelte/icons/users';
	import Search from 'lucide-svelte/icons/search';
	import UserPlus from 'lucide-svelte/icons/user-plus';
	import Plus from 'lucide-svelte/icons/plus';


	let currentTab = $state<'private' | 'group'>('private');
	let privateChats = $state.raw<PrivateChatItem[]>([]);
	let groupChats = $state.raw<GroupItem[]>([]);
	let isLoadingPrivate = $state(true);
	let isLoadingGroup = $state(false);
	let groupsLoaded = $state(false);
	let searchVal = $state('');

	async function loadPrivateChats() {
		try {
			const response = await withMinimumLoadingTime(() =>
				privateChatsService.getChats({ pagination: { limit: 50 } })
			);
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
			const response = await withMinimumLoadingTime(() =>
				groupsService.getMyGroups({ pagination: { limit: 50 } })
			);
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
		if (value === 'group' && !groupsLoaded) loadGroupChats();
	}

	const totalUnread = $derived(messageStore.totalUnreadCount);

	const filteredPrivateChats = $derived(
		searchVal.trim()
			? privateChats.filter(c => c.other_user.nickname.toLowerCase().includes(searchVal.toLowerCase()))
			: privateChats
	);

	const filteredGroupChats = $derived(
		searchVal.trim()
			? groupChats.filter(g => g.group.name.toLowerCase().includes(searchVal.toLowerCase()))
			: groupChats
	);

	onMount(() => { loadPrivateChats(); });
</script>

<svelte:head>
	<title>消息 - HotSpot</title>
</svelte:head>

<div class="flex flex-col">
	<!-- Sticky header -->
	<div
		class="sticky top-0 z-10 px-5 pt-3 pb-3"
		style="background-color: var(--neu-bg); border-bottom: 1px solid var(--neu-border);"
	>
		<div class="mb-3 flex items-center justify-between">
			<div>
				<h1 style="font-size: 20px; font-weight: 600; color: var(--neu-text);">消息</h1>
				<p style="font-size: 12px; color: var(--neu-text-muted); margin-top: 2px;">
					{totalUnread > 0 ? `${totalUnread} 条未读` : '全部已读'}
				</p>
			</div>
			<div class="flex gap-2.5">
				<button
					class="flex items-center justify-center rounded-full outline-none"
					style="width: 38px; height: 38px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
				>
					<UserPlus size={16} strokeWidth={1.5} color="var(--neu-text-muted)" />
				</button>
				<button
					class="flex items-center justify-center rounded-full outline-none"
					style="width: 38px; height: 38px; background: var(--neu-bg); box-shadow: 5px 5px 10px var(--neu-shadow-dark), -5px -5px 10px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
				>
					<Plus size={16} strokeWidth={1.5} color="var(--neu-text-muted)" />
				</button>
			</div>
		</div>

		<!-- Search bar -->
		<div
			class="mb-3 flex items-center gap-3 rounded-[18px] px-4 py-2.5"
			style="box-shadow: inset 6px 6px 12px var(--neu-shadow-dark), inset -6px -6px 12px var(--neu-shadow-light); border: 1px solid var(--neu-border);"
		>
			<Search size={15} strokeWidth={1.5} color="var(--neu-text-muted)" />
			<input
				type="text"
				bind:value={searchVal}
				placeholder="搜索会话…"
				class="flex-1 bg-transparent outline-none"
				style="font-size: 13px; color: var(--neu-text);"
			/>
		</div>

		<!-- Tabs -->
		<div class="flex" style="border-bottom: 1px solid var(--neu-border);">
			{#each [{ id: 'private', label: '私聊' }, { id: 'group', label: '群聊' }] as tab (tab.id)}
				<button
					class="relative flex-1 pb-3 pt-1 text-center outline-none"
					style="color: {currentTab === tab.id ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'}; font-weight: {currentTab === tab.id ? 500 : 400}; font-size: 13px; transition: color 0.2s;"
					onclick={() => handleTabChange(tab.id)}
				>
					{tab.label}
					{#if currentTab === tab.id}
						<div
							style="position: absolute; bottom: -1px; left: 12%; right: 12%; height: 2px; border-radius: 1px; background: var(--neu-accent-primary);"
						></div>
					{/if}
				</button>
			{/each}
		</div>
	</div>

	<!-- Content -->
	<div class="flex flex-col px-2 pt-2 pb-4">
		{#if currentTab === 'private'}
			{#if isLoadingPrivate}
				{#each Array(5) as _}
					<div class="flex items-center gap-3.5 rounded-[18px] px-4 py-3">
						<div class="rounded-full" style="width: 44px; height: 44px; background: var(--neu-border);"></div>
						<div class="flex-1">
							<div class="mb-1.5 rounded" style="width: 70px; height: 12px; background: var(--neu-border);"></div>
							<div class="rounded" style="width: 120px; height: 10px; background: var(--neu-border);"></div>
						</div>
					</div>
				{/each}
			{:else if filteredPrivateChats.length === 0}
				<EmptyState icon={MessageCircle} title="暂无私聊" description="去探索发现感兴趣的人吧" />
			{:else}
				{#each filteredPrivateChats as chat (chat.other_user.user_id)}
					{@const isOnline = presenceStore.isOnline(chat.other_user.user_id) || chat.presence_status === 'online'}
					<a
						href="/chat/users/{chat.other_user.user_id}"
						class="flex items-center gap-3.5 rounded-[18px] px-4 py-3 transition-all"
					>
						<div class="relative flex-shrink-0">
							<div class="overflow-hidden rounded-full" style="width: 44px; height: 44px; border: 1.5px solid var(--neu-border);">
								<img src={chat.other_user.avatar_url || ''} alt={chat.other_user.nickname} class="h-full w-full object-cover" />
							</div>
							{#if isOnline}
								<span class="absolute bottom-0 right-0 rounded-full" style="width: 10px; height: 10px; background: var(--neu-success); border: 2px solid var(--neu-bg);"></span>
							{/if}
						</div>
						<div class="min-w-0 flex-1">
							<div class="mb-0.5 flex items-center justify-between">
								<span style="font-size: 14px; color: var(--neu-text); font-weight: {chat.unread_count > 0 ? 600 : 400};">
									{chat.other_user.nickname}
								</span>
								<span style="font-size: 11px; color: var(--neu-text-muted);">
									{formatRelativeTime(chat.last_activity_at)}
								</span>
							</div>
							<p class="truncate pr-4" style="font-size: 12px; color: var(--neu-text-muted);">
								{#if chat.last_message}
									{chat.last_message.content || '[媒体消息]'}
								{:else}
									暂无消息
								{/if}
							</p>
						</div>
						{#if chat.unread_count > 0}
							<div
								class="flex flex-shrink-0 items-center justify-center rounded-full"
								style="width: 20px; height: 20px; background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 11px; font-weight: 600; box-shadow: 0 2px 8px var(--neu-glow);"
							>
								{chat.unread_count > 99 ? '99+' : chat.unread_count}
							</div>
						{/if}
					</a>
				{/each}
			{/if}
		{:else}
			{#if isLoadingGroup}
				{#each Array(5) as _}
					<div class="flex items-center gap-3.5 rounded-[18px] px-4 py-3">
						<div class="rounded-full" style="width: 44px; height: 44px; background: var(--neu-border);"></div>
						<div class="flex-1">
							<div class="mb-1.5 rounded" style="width: 80px; height: 12px; background: var(--neu-border);"></div>
							<div class="rounded" style="width: 100px; height: 10px; background: var(--neu-border);"></div>
						</div>
					</div>
				{/each}
			{:else if filteredGroupChats.length === 0}
				<EmptyState icon={Users} title="暂无群聊" description="去探索加入感兴趣的群组吧" />
			{:else}
				{#each filteredGroupChats as group (group.group.group_id)}
					<a
						href="/chat/groups/{group.group.group_id}"
						class="flex items-center gap-3.5 rounded-[18px] px-4 py-3 transition-all"
					>
						<div class="flex-shrink-0 overflow-hidden rounded-full" style="width: 44px; height: 44px; border: 1.5px solid var(--neu-border);">
							{#if group.group.avatar_url}
								<img src={group.group.avatar_url} alt={group.group.name} class="h-full w-full object-cover" />
							{:else}
								<div class="flex h-full w-full items-center justify-center" style="background: var(--neu-fallback-bg);">
									<Users size={18} color="var(--neu-fallback-text)" />
								</div>
							{/if}
						</div>
						<div class="min-w-0 flex-1">
							<div class="mb-0.5 flex items-center justify-between">
								<span style="font-size: 14px; color: var(--neu-text); font-weight: 500;">{group.group.name}</span>
								<span style="font-size: 11px; color: var(--neu-text-muted);">{group.group.members_count}人</span>
							</div>
							<p style="font-size: 12px; color: var(--neu-text-muted);">
								{formatRelativeTime(group.last_activity_at ?? group.my_state?.joined_at ?? '')}
							</p>
						</div>
						{#if (group.my_state?.unread_count ?? 0) > 0}
							<div
								class="flex flex-shrink-0 items-center justify-center rounded-full"
								style="width: 20px; height: 20px; background: var(--neu-accent-primary); color: var(--neu-active-text); font-size: 11px; font-weight: 600; box-shadow: 0 2px 8px var(--neu-glow);"
							>
								{(group.my_state?.unread_count ?? 0) > 99 ? '99+' : (group.my_state?.unread_count ?? 0)}
							</div>
						{/if}
					</a>
				{/each}
			{/if}
		{/if}
	</div>
</div>
