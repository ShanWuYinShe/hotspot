<script lang="ts">
	import { goto } from '$app/navigation';
	import { Plus, MessageSquareDashed } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import ChatView from '@/components/messages/ChatView.svelte';
	import GroupChatView from '@/components/messages/GroupChatView.svelte';
	import ChatListItem, { type ChatItem } from '@/components/messages/ChatListItem.svelte';
	import NewConversationSheet from '@/components/overlays/NewConversationSheet.svelte';
	import ListContainer from '@/components/list/ListContainer.svelte';
	import CreateGroupSheet from '@/components/overlays/CreateGroupSheet.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { getPrivateChats, listMyGroups, markPrivateRead, markGroupRead } from '@/api';
	import { formatTime } from '@/lib/chat.svelte';
	import { ROUTES } from '@/lib/routes';
	import { presence, watchPresence } from '@/stores/presence.svelte';

	// 消息列表：在线条 + 会话列表（私聊 + 群聊混合）+ 新会话 Sheet + 建群 Sheet；
	// 桌面左右分栏（内嵌 ChatView / GroupChatView）。

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
				const privData = privRes.code === 0 && privRes.data ? privRes.data.items : [];
				const groupData = groupRes.code === 0 && groupRes.data ? groupRes.data.items : [];
				const privateChats: ChatItem[] = privData.map((c, i) => ({
					id: `chat-${i}`,
					// spec: PrivateChatItem.other_user — the peer; private messages are
					// addressed by target_user_id (the peer's user_id), not a chat id.
					peerUserId: c.other_user?.user_id ?? '',
					name: c.other_user?.nickname ?? '',
					avatar: c.other_user?.avatar_url ?? '',
					lastMessage: c.last_message?.content ?? '',
					time: formatTime(c.last_message?.sent_at ?? c.last_activity_at),
					unread: c.unread_count ?? 0,
					online: c.presence_status === 'online',
					kind: 'private'
				}));
				// spec: GroupItem { group: GroupInfo, my_state?: MyGroupState, last_activity_at?, last_message? }
				const groupChats: ChatItem[] = groupData.map((c, i) => {
					const g = c.group;
					return {
						id: g.group_id ?? `group-${i}`,
						peerUserId: '',
						name: g.name,
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
					items.filter((c) => c.kind !== "group").map((c) => c.peerUserId).filter(Boolean)
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
	let searchQuery = $state('');
	let showNewChat = $state(false);
	let newChatTab = $state<'add' | 'chat'>('chat');
	let showCreateGroup = $state(false);
	// 会话 id：私聊为数字索引/chat id，群聊为字符串 group_id。
	let selectedId = $state<number | string | null>(null);

	// 选中会话：设置选中 id + 立即清零本地未读（徽章消失）+ 调 markRead API。
	// 未读状态不随 ChatView 内部处理（那里只标记"已读"不影响列表徽章），
	// 列表层必须自己联动，否则点击未读会话后徽章不消失。
	const selectChat = (chat: ChatItem) => {
		if (selectedId !== chat.id) selectedId = chat.id;
		if (chat.unread > 0) {
			chats = chats.map((c) => (c.id === chat.id ? { ...c, unread: 0 } : c));
			// spec: private_messages/mark_read(scope=session) / group_messages/mark_read(scope=group)；
			// 幂等语义，失败静默（徽章已本地清零，不打扰浏览）。
			if (chat.kind === 'group') {
				void markGroupRead({ scope: 'group', group_id: String(chat.id) }).catch(() => {});
			} else {
				void markPrivateRead({ scope: 'session', target_user_id: chat.peerUserId || String(chat.id) }).catch(() => {});
			}
		}
	};

	// 渲染用实时在线判定：WS 推送优先，HTTP 快照（chat.online）仅兜底
	const liveChats = $derived(
		chats.map((c) => ({ ...c, online: presence.isOnline(c.peerUserId) }))
	);

	const filtered = $derived(
		liveChats.filter((c) => c.name.toLowerCase().includes(searchQuery.toLowerCase()))
	);
	const onlineChats = $derived(liveChats.filter((c) => c.online));
	const totalUnread = $derived(chats.reduce((s, c) => s + c.unread, 0));
	// 平板与桌面共用分栏（对齐 Figma Make：platform !== 'mobile' 即桌面分栏）
	const isDesktop = $derived(platform.platform !== 'mobile');
	const selectedChat = $derived(liveChats.find((c) => c.id === selectedId));
</script>

{#if isDesktop}
	<!-- ═══ Desktop Messages — split pane (list left + chat right) ═══ -->
	<!-- h-full：RoutedContentViewport 内容链现提供显式高度（见该组件注释），
	     分栏固定视口高、左栏列表内部滚动；平板/桌面画布均撑满。 -->
	<div class="content-mode-wide flex h-full">
		<!-- Left: chat list -->
		<div class="flex flex-col shrink-0 overflow-hidden w-75 border-r border-border-default"  >
			<!-- List header（对齐 Figma：px-4 pt-4 pb-3，标题 18 + Plus 30 + 搜索框） -->
			<div class="shrink-0 border-b border-border-default" >
				<PageHeader title="消息" titleSize={18} contentClassName="px-4 pt-4 pb-0">
					{#snippet trailing()}
						<button
							type="button"
							class="hover-btn scale-btn flex items-center justify-center rounded-full outline-none surface-flat"
							style="width:30px;height:30px;"
							onclick={() => {
								newChatTab = 'chat';
								showNewChat = true;
							}}
						>
							<Plus class="size-3.5"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
						</button>
					{/snippet}
					{#snippet below()}
						<div class="mt-3 pb-3">
							<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索…" class="px-3 py-2" fontSize={12} />
						</div>
					{/snippet}
				</PageHeader>
			</div>

			<!-- 在线 strip -->
			{#if onlineChats.length > 0}
				<div class="shrink-0 px-4 py-2.5 border-b border-border-default" >
					<p class="uppercase mb-2 text-text-secondary text-[9px] tracking-[0.12em]">在线</p>
					<div class="flex gap-3 overflow-x-auto no-scrollbar">
						{#each onlineChats as c (c.id)}
							<button
								type="button"
								class="avatar-entry tap-93 flex flex-col items-center gap-1 shrink-0 cursor-pointer"
								onclick={() => selectChat(c)}
							>
								<div class="relative">
									<div class="avatar-entry__avatar rounded-full overflow-hidden" style="width:36px;height:36px;border:{selectedId === c.id ? '2px solid var(--cst-accent-default)' : '1.5px solid var(--cst-border-default)'};">
										<img src={c.avatar} alt={c.name} class="w-full h-full object-cover" />
									</div>
									<OnlineDot size="sm" />
								</div>
								<span class="text-text-secondary text-xxs">{c.name}</span>
							</button>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Chat list（List 契约：分栏内连接列表，行无独立边框。
			     每行外层 px-3 py-1 包裹——对齐 Figma MessagesPage：行间有呼吸间距，
			     选中态 glow 不与相邻行重叠，分隔线嵌在间距中不显突兀） -->
			<ListContainer class="flex-1 overflow-y-auto no-scrollbar px-2 py-2">
				{#if totalUnread > 0}
					<p class="uppercase px-2 mb-1 text-text-secondary text-[9px] tracking-[0.12em]">未读</p>
				{/if}
				{#each filtered as chat (chat.id)}
					<div class="px-3 py-1">
						<ChatListItem
							chat={chat}
							isSelected={selectedId === chat.id}
							onClick={() => selectChat(chat)}
							compact
						/>
					</div>
				{/each}
			</ListContainer>
		</div>

		<!-- Right: chat view or empty state（对齐 Figma：无卡片包裹，直接平铺） -->
		<div class="flex-1 overflow-hidden">
			{#if selectedId !== null && selectedChat}
				<div class="h-full">
					{#if selectedChat.kind === 'group'}
						<GroupChatView groupId={selectedChat.id} showBackButton={false} />
					{:else}
						<ChatView peerUserId={selectedChat.peerUserId} contact={{ name: selectedChat.name, avatar: selectedChat.avatar, online: selectedChat.online }} showBackButton={false} />
					{/if}
				</div>
			{:else}
				<div class="h-full">
					<EmptyState title="选择一个会话" description="从左侧选择联系人开始聊天" class="h-full justify-center py-0!" iconSize={72}>
						{#snippet icon()}
							<MessageSquareDashed class="size-7"  style="color:var(--cst-text-secondary);opacity:0.5" strokeWidth={1.25} />
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
			<Dialog.Content side="center" class="overflow-hidden w-120" aria-label="创建群组"  style="height:78vh;max-height:700px">
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
	<PageHeader title="消息" sticky titleSize={22} subtitleSize={12} contentClassName="pt-4 pb-0">
		{#snippet trailing()}
			<!-- 未读计数统一在标题右侧（对齐首页/附近动态样板），与「新会话」同排 -->
			<div class="flex items-center gap-2">
				{#if totalUnread > 0}
					<span class="text-text-secondary text-xs">{totalUnread} 条未读</span>
				{:else}
					<span class="text-status-online text-xs">全部已读</span>
				{/if}
				<button
					type="button"
					class="hover-btn scale-btn flex items-center justify-center rounded-full outline-none surface-flat"
					style="width:36px;height:36px;"
					onclick={() => {
						newChatTab = 'chat';
						showNewChat = true;
					}}
				>
					<Plus class="size-4"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
				</button>
			</div>
		{/snippet}
		{#snippet below()}
			<!-- 搜索框：sticky header 内第二行（对齐 Figma MessagesMobile） -->
			<div class="mt-3 pb-3">
				<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索会话…" class="rounded-relaxed px-4 py-2.5" />
			</div>
		{/snippet}
	</PageHeader>

	<ContentColumn class="animate-fade-in flex flex-col min-h-full pt-4 pb-4">
		{#if onlineChats.length > 0}
			<div class="mb-4">
				<p class="uppercase mb-2.5 text-text-secondary text-xxs tracking-[0.14em]">在线</p>
				<div class="flex gap-4 overflow-x-auto no-scrollbar pb-1">
					{#each onlineChats as chat (chat.id)}
						<button
							type="button"
							class="avatar-entry tap-93 flex flex-col items-center gap-1.5 shrink-0 cursor-pointer"
							onclick={() => void goto(ROUTES.message(chat.peerUserId || chat.id))}
						>
							<div class="relative">
								<div class="avatar-entry__avatar rounded-full overflow-hidden bg-surface-primary size-13"  style="padding:2px;border:2px solid var(--cst-accent-default)">
									<img src={chat.avatar} alt={chat.name} class="w-full h-full object-cover rounded-full" />
								</div>
								<OnlineDot size="lg" />
							</div>
							<span class="text-text-secondary text-xs">{chat.name}</span>
						</button>
					{/each}
				</div>
			</div>
		{/if}

		<p class="uppercase mb-2.5 text-text-secondary text-xxs tracking-[0.14em]">最近</p>

		<!-- 列表：无外层容器，List 直接裸排（对齐 Figma）；每行 px-3 py-1 包裹 -->
		{#if filtered.length === 0}
			<EmptyState bare class="py-12!" title="没有找到会话">
				{#snippet icon()}
					<MessageSquareDashed class="size-9"  style="color:var(--cst-text-secondary);opacity:0.5" strokeWidth={1.25} />
				{/snippet}
			</EmptyState>
		{:else}
			<ListContainer class="px-2 py-1">
				{#each filtered as chat (chat.id)}
					<div class="px-3 py-1">
						<ChatListItem
							chat={chat}
							onClick={() => void (chat.kind === 'group' ? goto(ROUTES.groupMessage(chat.id)) : goto(ROUTES.message(chat.peerUserId || chat.id)))}
						/>
					</div>
				{/each}
			</ListContainer>
		{/if}
	</ContentColumn>

	<Dialog.Root open={showNewChat} onOpenChange={(next) => (next ? undefined : (showNewChat = false))}>
		<Dialog.Content side="bottom" class="overflow-hidden" style="height:85%;box-shadow:var(--cst-elevation-sheet);">
			<NewConversationSheet
				defaultTab={newChatTab}
				onClose={() => (showNewChat = false)}
				onNewChat={(id) => void goto(ROUTES.message(id))}
				onCreateGroup={() => (showCreateGroup = true)}
			/>
		</Dialog.Content>
	</Dialog.Root>

	<Dialog.Root open={showCreateGroup} onOpenChange={(next) => (next ? undefined : (showCreateGroup = false))}>
		<Dialog.Content side="bottom" class="overflow-hidden" style="height:92%;box-shadow:var(--cst-elevation-sheet);">
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
