<script lang="ts">
	import { goto } from '$app/navigation';
	import { Plus, MessageSquareDashed } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import ChatView from '@/components/messages/ChatView.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import OnlineStrip from './shared/OnlineStrip.svelte';
	import ChatListBody from './shared/ChatListBody.svelte';
	import GroupChatView from '@/components/messages/GroupChatView.svelte';
	import NewConversationSheet from '@/components/overlays/NewConversationSheet.svelte';
	import TabBar from '@/components/navigation/TabBar.svelte';
	import CreateGroupSheet from '@/components/overlays/CreateGroupSheet.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { getPrivateChats, listMyGroups, markPrivateRead, markGroupRead, updatePrivateChatPreferences, updateGroupPreferences , netError } from '@/api';
	import { formatTime } from '@/lib/chat.svelte';
	import { compareChatItems, type ChatItem } from '@/lib/chat-list';
	import { ROUTES } from '@/lib/routes';
	import { presence, watchPresence } from '@/stores/presence.svelte';

	// 消息列表：在线条 + 会话列表（私聊 + 群聊混合）+ 新会话 Sheet + 建群 Sheet；
	// 桌面左右分栏（内嵌 ChatView / GroupChatView）。

	let chats = $state<ChatItem[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	// 未登录（10100）是正常业务态：显示登录引导而非错误重试
	let needsAuth = $state(false);
	let reloadKey = $state(0);
	let unwatchPresence: (() => void) | null = null;

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchChats() {
			try {
				loading = true;
				loadError = null;
				needsAuth = false;
				// 并行加载私聊与我的群组，合并为统一会话列表（按时间倒序）。
				const [privRes, groupRes] = await Promise.all([
					getPrivateChats().catch(() => netError({ items: [] })),
					listMyGroups().catch(() => netError({ items: [] }))
				]);
				if (cancelled) return;
				const privData = privRes.code === 0 && privRes.data ? privRes.data.items : [];
				const groupData = groupRes.code === 0 && groupRes.data ? groupRes.data.items : [];
				if (privRes.code === 10100 || groupRes.code === 10100) {
					if (!cancelled) needsAuth = true;
					return;
				}
				if (privRes.code !== 0 || groupRes.code !== 0) {
					if (!cancelled) loadError = privRes.message ?? groupRes.message ?? '加载失败';
					return;
				}
				const privateChats: ChatItem[] = privData.map((c, i) => ({
					id: `chat-${i}`,
					// spec: PrivateChatItem.other_user — the peer; private messages are
					// addressed by target_user_id (the peer's user_id), not a chat id.
					peerUserId: c.other_user?.user_id ?? '',
					name: c.other_user?.nickname ?? '',
					avatar: c.other_user?.avatar_url ?? '',
					lastMessage: c.last_message?.content ?? '',
					time: formatTime(c.last_message?.sent_at ?? c.last_activity_at),
					timestamp: c.last_message?.sent_at ?? c.last_activity_at ?? '',
					unread: c.unread_count ?? 0,
					isPinned: c.is_pinned ?? false,
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
						// spec: last_message 仅 groups/fetch_joined 返回,非文本消息预览为空。
						// 排序/展示统一以最新一条消息时间为准，无消息时回退群组活跃时间
						lastMessage: c.last_message?.content ?? '',
						time: formatTime(c.last_message?.sent_at ?? c.last_activity_at ?? ''),
						timestamp: c.last_message?.sent_at ?? c.last_activity_at ?? '',
						unread: c.viewer_state?.unread_count ?? 0,
						isPinned: c.viewer_state?.is_pinned ?? false,
						online: false,
						group: true,
						kind: 'group'
					};
				});
				// 私聊与群聊混合，统一按「置顶 > 未读 > 已读，组内时间倒序」排序。
				const items = [...privateChats, ...groupChats].sort(compareChatItems);
				chats = items;
				// 在线状态实时化：仅对非 group 会话订阅全部对端（chats.id 语义即 peer user_id，
				// 但显式用 peerUserId 更稳）；旧订阅先释放再建立新订阅。
				unwatchPresence?.();
				unwatchPresence = watchPresence(
					items.filter((c) => c.kind !== "group").map((c) => c.peerUserId).filter(Boolean)
				);
			} catch {
				if (!cancelled) loadError = '加载失败，请稍后重试';
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
	// 已读/未读只做过滤，不参与排序（避免读后列表跳动）
	let chatFilter = $state<'all' | 'unread' | 'read'>('all');
	const FILTER_TABS: { id: 'all' | 'unread' | 'read'; label: string }[] = [
		{ id: 'all', label: '全部' },
		{ id: 'unread', label: '未读' },
		{ id: 'read', label: '已读' }
	];
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
			// 只清零本地未读徽章；列表按时间排序，位置不跳动
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

	// 置顶/取消置顶：乐观更新本地状态并立即重排（置顶优先），API 失败回滚。
	const togglePin = (chat: ChatItem) => {
		const next = !chat.isPinned;
		chats = chats.map((c) => (c.id === chat.id ? { ...c, isPinned: next } : c)).sort(compareChatItems);
		const req =
			chat.kind === 'group'
				? updateGroupPreferences({ group_id: String(chat.id), is_pinned: next })
				: updatePrivateChatPreferences({ target_user_id: chat.peerUserId || String(chat.id), is_pinned: next });
		void req.catch(() => {
			// 失败回滚：置顶状态还原并恢复原排序
			chats = chats.map((c) => (c.id === chat.id ? { ...c, isPinned: !next } : c)).sort(compareChatItems);
		});
	};

	// 渲染用实时在线判定：WS 推送优先，HTTP 快照（chat.online）仅兜底
	const liveChats = $derived(
		chats.map((c) => ({ ...c, online: presence.isOnline(c.peerUserId) }))
	);

	const filtered = $derived(
		liveChats.filter((c) => {
			const byFilter =
				chatFilter === 'all' || (chatFilter === 'unread' ? c.unread > 0 : c.unread === 0);
			return byFilter && c.name.toLowerCase().includes(searchQuery.toLowerCase());
		})
	);
	const onlineChats = $derived(liveChats.filter((c) => c.online));
	const totalUnread = $derived(chats.reduce((s, c) => s + c.unread, 0));
	// 分栏阈值 = 桌面 OR 宽平板（≥960px / isWideTablet）。
	// 紧凑平板（600-959）走移动端单列流程：w-75=300px 左列 + 余宽聊天的分栏在
	// 600-700vp 视口下会让右侧聊天被压到 300-400px，Tap & 键入都难用。
	// 与 harmonyapp WIDE_MIN_VP=960 一致，让两端在"宽平板才分栏"上对齐。
	const isDesktop = $derived(platform.platform === 'desktop' || platform.isWideTablet);
	const selectedChat = $derived(liveChats.find((c) => c.id === selectedId));
</script>
<svelte:head>
	<title>消息 · HotSpot</title>
</svelte:head>


{#if isDesktop}
	<!-- ═══ Desktop Messages — split pane (list left + chat right) ═══ -->
	<!-- h-full：RoutedContentViewport 内容链现提供显式高度（见该组件注释），
	     分栏固定视口高、左栏列表内部滚动；平板/桌面画布均撑满。 -->
	<div class="content-mode-wide flex h-full">
		<!-- Left: chat list -->
		<div class="flex flex-col shrink-0 overflow-hidden w-75 border-r border-border-default"  >
			<!-- List header（对齐 Figma：px-4 pt-4 pb-3，标题 18 + Plus 30 + 搜索框） -->
			<div class="shrink-0">
				<PageHeader title="消息"  contentClassName="px-4 pt-4 pb-0">
					{#snippet trailing()}
						<button
							type="button"
						class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring surface-flat size-9"
						onclick={() => {
							newChatTab = 'chat';
							showNewChat = true;
						}}
						aria-label="发起新会话"
					>
						<Plus class="size-3.5 text-text-secondary"   strokeWidth={1.5} />
						</button>
					{/snippet}
					{#snippet below()}
						<div class="mt-3 pb-3">
							<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索…" class="px-3 py-2" textClass="text-sm" />
							<div class="mt-3">
								<TabBar
									items={FILTER_TABS}
									value={chatFilter}
									onChange={(id) => (chatFilter = id as typeof chatFilter)}
								/>
							</div>
						</div>
					{/snippet}
				</PageHeader>
			</div>

			<!-- 在线 strip -->
			{#if onlineChats.length > 0}
				<OnlineStrip chats={onlineChats} variant="sm" {selectedId} onSelect={selectChat} />
			{/if}

			<ChatListBody
				chats={filtered}
				loadError={loadError}
				needsAuth={needsAuth}
				loading={loading}
				onRetry={() => reloadKey++}
				onSelect={selectChat}
				onTogglePin={togglePin}
				{selectedId}
				compact
				fill
			/>
		</div>

		<!-- Right: chat view or empty state（对齐 Figma：无卡片包裹，直接平铺） -->
		<div class="flex-1 overflow-hidden">
			{#if selectedId !== null && selectedChat}
				<div class="h-full">
					{#if selectedChat.kind === 'group'}
						<GroupChatView groupId={selectedChat.id} compact showBackButton={false} />
					{:else}
						<ChatView peerUserId={selectedChat.peerUserId} compact contact={{ name: selectedChat.name, avatar: selectedChat.avatar, online: selectedChat.online }} showBackButton={false} />
					{/if}
				</div>
			{:else}
				<div class="h-full">
					<EmptyState title="选择一个会话" description="从左侧选择联系人开始聊天" class="h-full justify-center py-0!" iconSize={72}>
						{#snippet icon()}
							<MessageSquareDashed class="size-7 opacity-muted text-text-secondary"  strokeWidth={1.5} />
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
				style="width:27.5rem;height:78vh;max-height:700px;"
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
	<PageHeader title="消息" sticky titleSize="3xl" subtitleSize="sm" contentClassName="pt-4 pb-0">
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
					class="hover-btn scale-btn hit-slop flex items-center justify-center rounded-full focus-ring surface-flat size-9"
					onclick={() => {
						newChatTab = 'chat';
						showNewChat = true;
					}}
					aria-label="发起新会话"
				>
					<Plus class="size-4 text-text-secondary"   strokeWidth={1.5} />
				</button>
			</div>
		{/snippet}
		{#snippet below()}
			<!-- 搜索框：sticky header 内第二行（对齐 Figma MessagesMobile） -->
			<div class="mt-3 pb-3">
				<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索会话…" class="rounded-md px-4 py-2.5" />
				<div class="mt-3">
					<TabBar
						items={FILTER_TABS}
						value={chatFilter}
						onChange={(id) => (chatFilter = id as typeof chatFilter)}
					/>
				</div>
			</div>
		{/snippet}
	</PageHeader>

	<ContentColumn class="animate-fade-in flex flex-col min-h-full pt-4 pb-6">
		{#if onlineChats.length > 0}
			<OnlineStrip chats={onlineChats} onSelect={(c) => void goto(ROUTES.message(c.peerUserId || c.id))} />
		{/if}

		<p class="eyebrow mb-2.5">最近</p>

		<ChatListBody
			chats={filtered}
			loadError={loadError}
			needsAuth={needsAuth}
			loading={loading}
			onRetry={() => reloadKey++}
			onSelect={(chat) => void (chat.kind === 'group' ? goto(ROUTES.groupMessage(chat.id)) : goto(ROUTES.message(chat.peerUserId || chat.id)))}
			onTogglePin={togglePin}
		/>
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
