<script lang="ts">
	import { goto } from '$app/navigation';
	import { Plus, MessageSquareDashed } from '@lucide/svelte';
	import { platformStore } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import ChatView from '@/components/messages/ChatView.svelte';
	import GroupChatView from '@/components/messages/GroupChatView.svelte';
	import ChatListItem, { type ChatItem } from '@/components/messages/ChatListItem.svelte';
	import NewConversationSheet from '@/components/overlays/NewConversationSheet.svelte';
	import EmbeddedList from '@/components/data-display/EmbeddedList.svelte';
	import CreateGroupSheet from '@/components/overlays/CreateGroupSheet.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { getPrivateChats, listMyGroups } from '@/api';
	import { ROUTES } from '@/lib/routes';
	import { presence, watchPresence } from '@/stores/presence.svelte';

	// 消息列表：在线条 + 会话列表（私聊 + 群聊混合）+ 新会话 Sheet + 建群 Sheet；
	// 桌面左右分栏（内嵌 ChatView / GroupChatView）。迁移自旧版 MessagesPage。

	/** 时间戳兜底：无效日期按 0 处理（用于私聊/群聊混合排序）。 */
	function tsOf(t: string): number {
		const n = new Date(t).getTime();
		return isNaN(n) ? 0 : n;
	}

	let chats = $state<ChatItem[]>([]);
	let loading = $state(true);
	let unwatchPresence: (() => void) | null = null;

	$effect(() => {
		let cancelled = false;
		async function fetchChats() {
			try {
				loading = true;
				// 并行加载私聊与我的群组，合并为统一会话列表（按时间倒序）。
				const [privRes, groupRes] = await Promise.all([
					getPrivateChats().catch(() => ({ code: -1, data: { items: [] } })),
					listMyGroups().catch(() => ({ code: -1, data: { items: [] } }))
				]);
				if (cancelled) return;
				const privData = (privRes.code === 0 && privRes.data ? (privRes.data as any).items : []) as any[];
				const groupData = (groupRes.code === 0 && groupRes.data ? (groupRes.data as any).items : []) as any[];
				const privateChats: ChatItem[] = privData.map((c: any, i: number) => ({
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
					group: c.group ?? c.is_group ?? false,
					kind: 'private'
				}));
				// spec: GroupItem { group: GroupInfo, my_state?: MyGroupState, last_activity_at?, last_message? }
				const groupChats: ChatItem[] = groupData.map((c: any, i: number) => {
					const g = c.group ?? {};
					return {
						id: g.group_id ?? `group-${i}`,
						peerUserId: '',
						name: g.name ?? '',
						avatar: g.avatar_url ?? '',
						// spec: last_message 仅 groups/list_my 返回,非文本消息预览为空。
						lastMessage: c.last_message?.content ?? '',
						time: c.last_activity_at ?? '',
						unread: c.my_state?.unread_count ?? 0,
						online: false,
						group: true,
						kind: 'group'
					};
				});
				// 私聊与群聊混合，统一按最近活跃时间倒序。
				const items = [...privateChats, ...groupChats].sort((a, b) => tsOf(b.time) - tsOf(a.time));
				chats = items;
				// 在线状态实时化：仅对非 group 会话订阅全部对端（chats.id 语义即 peer user_id，
				// 但显式用 peerUserId 更稳）；旧订阅先释放再建立新订阅。
				unwatchPresence?.();
				unwatchPresence = watchPresence(
					items.filter((c) => !c.group).map((c) => c.peerUserId).filter(Boolean)
				);
			} catch {
				if (!cancelled) chats = [];
			} finally {
				if (!cancelled) loading = false;
			}
		}
		void fetchChats();
		return () => {
			cancelled = true;
			// 组件卸载：释放全部在线订阅
			unwatchPresence?.();
			unwatchPresence = null;
		};
	});

	// 共享状态：搜索/新会话/建群（移动与桌面共用）
	let searchVal = $state('');
	let showNewChat = $state(false);
	let newChatTab = $state<'add' | 'chat'>('chat');
	let showCreateGroup = $state(false);
	// 会话 id：私聊为数字索引/chat id，群聊为字符串 group_id。
	let selectedId = $state<number | string | null>(null);

	// 渲染用实时在线判定：WS 推送优先，HTTP 快照（chat.online）仅兜底
	const liveChats = $derived(
		chats.map((c) => ({ ...c, online: presence.isOnline(c.peerUserId) }))
	);

	const filtered = $derived(
		liveChats.filter((c) => c.name.toLowerCase().includes(searchVal.toLowerCase()))
	);
	const onlineChats = $derived(liveChats.filter((c) => c.online));
	const totalUnread = $derived(chats.reduce((s, c) => s + c.unread, 0));
	// 平板与桌面共用分栏（对齐 Figma Make：platform !== 'mobile' 即桌面分栏）
	const isDesktop = $derived(platformStore.platform !== 'mobile');
	const selectedChat = $derived(liveChats.find((c) => c.id === selectedId));
</script>

{#if isDesktop}
	<!-- ═══ Desktop Messages — split pane (list left + chat right) ═══ -->
	<div class="content-mode-wide flex h-full">
		<!-- Left: chat list -->
		<div class="flex flex-col flex-shrink-0 overflow-hidden" style="width:300px;border-right:1px solid var(--neu-border);">
			<!-- List header（对齐 Figma：px-4 pt-4 pb-3，标题 18 + Plus 30 + 搜索框） -->
			<div class="flex-shrink-0" style="border-bottom:1px solid var(--neu-border);">
				<PageHeader title="消息" titleSize={18} contentClassName="px-4 pt-4 pb-0">
					{#snippet trailing()}
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
					{#snippet below()}
						<div class="mt-3 pb-3">
							<SearchInput value={searchVal} onChange={(v) => (searchVal = v)} placeholder="搜索…" class="px-3 py-2" fontSize={12} />
						</div>
					{/snippet}
				</PageHeader>
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

			<!-- Chat list（EmbeddedList 契约：分栏内连接列表，行无独立边框） -->
			<EmbeddedList class="flex-1 overflow-y-auto no-scrollbar px-2 py-2">
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
			</EmbeddedList>
		</div>

		<!-- Right: chat view or empty state（对齐 Figma：无卡片包裹，直接平铺） -->
		<div class="flex-1 overflow-hidden">
			{#if selectedId !== null && selectedChat}
				<div class="h-full">
					{#if selectedChat.kind === 'group'}
						<GroupChatView groupId={String(selectedChat.id)} showBackButton={false} />
					{:else}
						<ChatView peerUserId={selectedChat.peerUserId} contact={{ name: selectedChat.name, avatar: selectedChat.avatar, online: selectedChat.online }} showBackButton={false} />
					{/if}
				</div>
			{:else}
				<div class="h-full">
					<EmptyState title="选择一个会话" description="从左侧选择联系人开始聊天" class="h-full justify-center py-0!" iconSize={72}>
						{#snippet icon()}
							<MessageSquareDashed style="width:28px;height:28px;color:var(--neu-text-muted);opacity:0.5;" strokeWidth={1.25} />
						{/snippet}
					</EmptyState>
				</div>
			{/if}
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
				<CreateGroupSheet
					onClose={() => (showCreateGroup = false)}
					onCreated={(groupId) => {
						showCreateGroup = false;
						void goto(ROUTES.groupMessage(groupId));
					}}
				/>
			</Dialog.Content>
		</Dialog.Root>
	</div>
{:else}
	<!-- ═══ Mobile Messages ═══ -->
	<PageHeader title="消息" sticky titleSize={22} subtitleSize={12} contentClassName="pt-4 pb-0" subtitle={totalUnread > 0 ? `${totalUnread} 条未读` : '全部已读'}>
		{#snippet trailing()}
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
		{#snippet below()}
			<!-- 搜索框：sticky header 内第二行（对齐 Figma MessagesMobile） -->
			<div class="mt-3 pb-3">
				<SearchInput value={searchVal} onChange={(v) => (searchVal = v)} placeholder="搜索会话…" class="rounded-relaxed px-4 py-2.5" />
			</div>
		{/snippet}
	</PageHeader>

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

		<!-- 列表：无外层容器，EmbeddedList 直接裸排（对齐 Figma） -->
		{#if filtered.length === 0}
			<EmptyState bare class="py-12!" title="没有找到会话">
				{#snippet icon()}
					<MessageSquareDashed style="width:36px;height:36px;color:var(--neu-text-muted);opacity:0.5;" strokeWidth={1.25} />
				{/snippet}
			</EmptyState>
		{:else}
			<EmbeddedList class="px-2 py-1">
				{#each filtered as chat (chat.id)}
					<ChatListItem
						chat={chat}
						onClick={() => void (chat.kind === 'group' ? goto(ROUTES.groupMessage(chat.id)) : goto(ROUTES.message(chat.peerUserId || chat.id)))}
					/>
				{/each}
			</EmbeddedList>
		{/if}
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
			<CreateGroupSheet
				onClose={() => (showCreateGroup = false)}
				onCreated={(groupId) => {
					showCreateGroup = false;
					void goto(ROUTES.groupMessage(groupId));
				}}
			/>
		</Dialog.Content>
	</Dialog.Root>
{/if}
