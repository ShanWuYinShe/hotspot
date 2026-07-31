<script lang="ts">
	import { goto } from '$app/navigation';
	import { Plus, MessageSquareDashed } from '@lucide/svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import PageHeader from '@/components/shared/PageHeader.svelte';
	import SearchInput from '@/components/shared/SearchInput.svelte';
	import EmptyState from '@/components/shared/EmptyState.svelte';
	import OnlineDot from '@/components/shared/OnlineDot.svelte';
	import ChatView from '@/components/messages/ChatView.svelte';
	import ChatListItem, { type ChatItem } from '@/components/messages/ChatListItem.svelte';
	import NewConversationSheet from '@/components/messages/NewConversationSheet.svelte';
	import CreateGroupSheet from '@/components/groups/CreateGroupSheet.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { getPrivateChats } from '@/api';
	import { ROUTES } from '@/lib/routes';

	// 消息列表：在线条 + 会话列表 + 新会话 Sheet + 建群 Sheet；
	// 桌面左右分栏（内嵌 ChatView）。迁移自旧版 MessagesPage。

	let chats = $state<ChatItem[]>([]);
	let loading = $state(true);

	$effect(() => {
		let cancelled = false;
		async function fetchChats() {
			try {
				loading = true;
				const res = await getPrivateChats().catch(() => ({ code: -1, data: { items: [] } }));
				if (cancelled) return;
				if (res.code === 0 && res.data) {
					const data = res.data as any;
					const items: ChatItem[] = (data.items || []).map((c: any, i: number) => ({
						id: c.id ?? i + 1,
						// spec: PrivateChatItem.other_user — the peer; private messages are
						// addressed by target_user_id (the peer's user_id), not a chat id.
						peerUserId: c.other_user?.user_id ?? c.peer_user_id ?? c.target_user_id ?? '',
						name: c.other_user?.nickname ?? c.name ?? c.nickname ?? c.peer_name ?? '',
						avatar: c.other_user?.avatar_url ?? c.avatar ?? c.peer_avatar ?? '',
						lastMessage: c.last_message?.content ?? c.last_message ?? c.last_msg ?? '',
						time: c.last_message?.sent_at ?? c.last_activity_at ?? c.time ?? c.updated_at ?? '',
						unread: c.unread_count ?? c.unread ?? 0,
						online: c.other_user?.presence_status === 'online' || c.online || false,
						group: c.group ?? c.is_group ?? false
					}));
					chats = items;
				}
			} catch {
				if (!cancelled) chats = [];
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchChats();
		return () => {
			cancelled = true;
		};
	});

	// 共享状态：搜索/新会话/建群（移动与桌面共用）
	let searchVal = $state('');
	let showNewChat = $state(false);
	let newChatTab = $state<'add' | 'chat'>('chat');
	let showCreateGroup = $state(false);
	let selectedId = $state<number | null>(null);

	const filtered = $derived(chats.filter((c) => c.name.toLowerCase().includes(searchVal.toLowerCase())));
	const onlineChats = $derived(chats.filter((c) => c.online));
	const totalUnread = $derived(chats.reduce((s, c) => s + c.unread, 0));
	const isDesktop = $derived(platformStore.platform === 'desktop');
	const selectedChat = $derived(chats.find((c) => c.id === selectedId));
</script>

{#if isDesktop}
	<!-- ═══ Desktop Messages — split pane (list left + chat right) ═══ -->
	<div class="content-mode-wide flex h-full">
		<!-- Left: chat list -->
		<div class="flex flex-col flex-shrink-0 overflow-hidden" style="width:300px;border-right:1px solid var(--neu-border);">
			<!-- List header -->
			<div class="flex-shrink-0" style="border-bottom:1px solid var(--neu-border);">
				<PageHeader title="消息" className="!py-2">
					{#snippet actions()}
						<button
							type="button"
							class="neu-hover-btn neu-scale-btn flex items-center justify-center rounded-full outline-none neu-surface-flat"
							style="width:30px;height:30px;"
							onclick={() => {
								newChatTab = 'chat';
								showNewChat = true;
							}}
						>
							<Plus style="width:14px;height:14px;color:var(--neu-text-muted);" strokeWidth={1.5} />
						</button>
					{/snippet}
				</PageHeader>
				<div class="px-4 pb-3">
					<SearchInput value={searchVal} onChange={(v) => (searchVal = v)} placeholder="搜索…" compact />
				</div>
			</div>

			<!-- 在线 strip -->
			{#if onlineChats.length > 0}
				<div class="flex-shrink-0 px-4 py-2.5" style="border-bottom:1px solid var(--neu-border);">
					<p class="uppercase mb-2 text-[var(--neu-text-muted)] text-[9px] tracking-[0.12em]">在线</p>
					<div class="flex gap-3 overflow-x-auto no-scrollbar">
						{#each onlineChats as c (c.id)}
							<button
								type="button"
								class="neu-hover-btn flex flex-col items-center gap-1 flex-shrink-0 cursor-pointer"
								onclick={() => (selectedId = c.id)}
							>
								<div class="relative">
									<div class="rounded-full overflow-hidden" style="width:36px;height:36px;border:{selectedId === c.id ? '2px solid var(--neu-accent-primary)' : '1.5px solid var(--neu-border)'};">
										<img src={c.avatar} alt={c.name} class="w-full h-full object-cover" />
									</div>
									<OnlineDot size="sm" />
								</div>
								<span class="text-[var(--neu-text-muted)] text-[10px]">{c.name}</span>
							</button>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Chat list -->
			<div class="flex-1 overflow-y-auto no-scrollbar px-2 py-2">
				{#if totalUnread > 0}
					<p class="uppercase px-2 mb-1 text-[var(--neu-text-muted)] text-[9px] tracking-[0.12em]">未读</p>
				{/if}
				{#each filtered as chat (chat.id)}
					<ChatListItem
						chat={chat}
						isSelected={selectedId === chat.id}
						onClick={() => (selectedId = chat.id)}
						compact
					/>
				{/each}
			</div>
		</div>

		<!-- Right: chat view or empty state -->
		<div class="flex min-w-0 flex-1 bg-[color-mix(in_srgb,var(--page-bg)_55%,transparent)] p-4">
			<div class="min-h-0 flex-1 overflow-hidden rounded-[22px] neu-ring-border bg-[var(--neu-bg)] shadow-[0_10px_28px_color-mix(in_srgb,var(--neu-shadow-dark)_32%,transparent)]">
				<div class="h-full" style="overflow:hidden;border-radius:22px;">
					{#if selectedId !== null && selectedChat}
						<div class="h-full">
							<ChatView peerUserId={selectedChat.peerUserId} contact={{ name: selectedChat.name, avatar: selectedChat.avatar, online: selectedChat.online }} showBackButton={false} compact />
						</div>
					{:else}
						<div class="h-full">
							<EmptyState title="选择一个会话" description="从左侧选择联系人开始聊天" className="h-full justify-center !py-0">
								{#snippet icon()}
									<MessageSquareDashed style="width:28px;height:28px;color:var(--neu-text-muted);opacity:0.5;" strokeWidth={1.25} />
								{/snippet}
							</EmptyState>
						</div>
					{/if}
				</div>
			</div>
		</div>

		<Dialog.Root open={showNewChat} onOpenChange={(next) => (next ? undefined : (showNewChat = false))}>
			<Dialog.Content
				side="center"
				class="overflow-hidden"
				aria-label="发起新会话"
				style="width:440px;height:68vh;max-height:600px;"
			>
				<NewConversationSheet
					defaultTab={newChatTab}
					onClose={() => (showNewChat = false)}
					onNewChat={(id) => {
						selectedId = id;
						showNewChat = false;
					}}
					onCreateGroup={() => {
						showNewChat = false;
						showCreateGroup = true;
					}}
				/>
			</Dialog.Content>
		</Dialog.Root>

		<Dialog.Root open={showCreateGroup} onOpenChange={(next) => (next ? undefined : (showCreateGroup = false))}>
			<Dialog.Content side="center" class="overflow-hidden" aria-label="创建群组" style="width:480px;height:78vh;max-height:700px;">
				<CreateGroupSheet onClose={() => (showCreateGroup = false)} />
			</Dialog.Content>
		</Dialog.Root>
	</div>
{:else}
	<!-- ═══ Mobile Messages ═══ -->
	<PageHeader title="消息" sticky subtitle={totalUnread > 0 ? `${totalUnread} 条未读` : '全部已读'}>
		{#snippet actions()}
			<button
				type="button"
				class="neu-hover-btn neu-scale-btn flex items-center justify-center rounded-full outline-none neu-surface-flat"
				style="width:36px;height:36px;"
				onclick={() => {
					newChatTab = 'chat';
					showNewChat = true;
				}}
			>
				<Plus style="width:16px;height:16px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</button>
		{/snippet}
	</PageHeader>
	<SearchInput value={searchVal} onChange={(v) => (searchVal = v)} placeholder="搜索会话…" className="mx-5 mb-3" />

	<div class="animate-fade-in flex flex-col px-5 pt-4 pb-4" style="max-width:640px;margin-left:auto;margin-right:auto;width:100%;">
		{#if onlineChats.length > 0}
			<div class="mb-4">
				<p class="uppercase mb-2.5 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em]">在线</p>
				<div class="flex gap-4 overflow-x-auto no-scrollbar pb-1">
					{#each onlineChats as chat (chat.id)}
						<button
							type="button"
							class="neu-hover-btn flex flex-col items-center gap-1.5 flex-shrink-0 cursor-pointer"
							onclick={() => void goto(ROUTES.message(chat.peerUserId || chat.id))}
						>
							<div class="relative">
								<div class="rounded-full overflow-hidden bg-[var(--neu-bg)]" style="width:52px;height:52px;padding:2px;border:2px solid var(--neu-accent-primary);">
									<img src={chat.avatar} alt={chat.name} class="w-full h-full object-cover rounded-full" />
								</div>
								<OnlineDot size="lg" />
							</div>
							<span class="text-[var(--neu-text-muted)] text-[11px]">{chat.name}</span>
						</button>
					{/each}
				</div>
			</div>
		{/if}

		<p class="uppercase mb-2.5 text-[var(--neu-text-muted)] text-[10px] tracking-[0.14em]">最近</p>

		<div class="rounded-[18px] overflow-hidden neu-surface-flat">
			{#if filtered.length === 0}
				<EmptyState title="没有找到会话">
					{#snippet icon()}
						<MessageSquareDashed style="width:28px;height:28px;color:var(--neu-text-muted);opacity:0.5;" strokeWidth={1.25} />
					{/snippet}
				</EmptyState>
			{:else}
				{#each filtered as chat, i (chat.id)}
					<div class="px-2 py-1">
						<ChatListItem chat={chat} onClick={() => void goto(ROUTES.message(chat.peerUserId || chat.id))} />
					</div>
					{#if i < filtered.length - 1}
						<div class="mx-4 bg-[var(--neu-border)]" style="height:1px;"></div>
					{/if}
				{/each}
			{/if}
		</div>
	</div>

	<Dialog.Root open={showNewChat} onOpenChange={(next) => (next ? undefined : (showNewChat = false))}>
		<Dialog.Content side="bottom" class="overflow-hidden" style="height:85%;box-shadow:0 -8px 40px rgba(0,0,0,0.18);">
			<NewConversationSheet
				defaultTab={newChatTab}
				onClose={() => (showNewChat = false)}
				onNewChat={(id) => void goto(ROUTES.message(id))}
				onCreateGroup={() => (showCreateGroup = true)}
			/>
		</Dialog.Content>
	</Dialog.Root>

	<Dialog.Root open={showCreateGroup} onOpenChange={(next) => (next ? undefined : (showCreateGroup = false))}>
		<Dialog.Content side="bottom" class="overflow-hidden" style="height:92%;box-shadow:0 -8px 40px rgba(0,0,0,0.18);">
			<CreateGroupSheet onClose={() => (showCreateGroup = false)} />
		</Dialog.Content>
	</Dialog.Root>
{/if}
