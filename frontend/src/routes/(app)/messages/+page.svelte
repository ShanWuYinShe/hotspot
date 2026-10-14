<script lang="ts">
	import { untrack } from 'svelte';
	import { goto } from '$app/navigation';
	import { Plus, MessageSquareDashed } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import ChatView from '@/components/messages/ChatView.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import OnlineStrip from './shared/OnlineStrip.svelte';
	import ChatListBody from './shared/ChatListBody.svelte';
	import GroupChatView from '@/components/messages/GroupChatView.svelte';
	import NewConversationSheet from '@/components/overlays/NewConversationSheet.svelte';
	import CreateGroupSheet from '@/components/overlays/CreateGroupSheet.svelte';
	import ConfirmDialog from '@/components/overlays/ConfirmDialog.svelte';
	import * as Dialog from '@/components/ui/dialog';
	import { getPrivateChats, deletePrivateChat, listMyGroups, markPrivateRead, markGroupRead, updatePrivateChatPreferences, updateGroupPreferences , netError } from '@/api';
	import { showToast } from '@/stores/toast.svelte';
	import { formatChatListTime } from '@/lib/chat.svelte';
	import { compareChatItems, type ChatItem } from '@/lib/chat-list';
	import { ROUTES } from '@/lib/routes';
	import { presence, watchPresence, seedOnline } from '@/stores/presence.svelte';
	import { setChatUnread } from '@/stores/unread.svelte';
	import { WS_PRIVATE_MESSAGE_EVENT, WS_GROUP_MESSAGE_EVENT } from '@/lib/ws/client';

	// 消息列表：在线条 + 会话列表（私聊 + 群聊混合）+ 新会话 Sheet + 建群 Sheet；
	// 桌面左右分栏（内嵌 ChatView / GroupChatView）。
	// WS 实时：ws:private-message / ws:group-message 触发 reloadKey 静默重拉
	// （仅首刷无数据时展示整页 loading，后台刷新不打扰浏览）。

	let chats = $state<ChatItem[]>([]);
	let loading = $state(true);
	let loadError = $state<string | null>(null);
	// 未登录（10100）是正常业务态：显示登录引导而非错误重试
	let needsAuth = $state(false);
	let reloadKey = $state(0);
	let unwatchPresence: (() => void) | null = null;

	// WS 新消息 → 静默重拉会话列表（行级未读/排序/最新一条随服务端真相刷新）
	$effect(() => {
		const onWsMessage = () => {
			reloadKey++;
		};
		window.addEventListener(WS_PRIVATE_MESSAGE_EVENT, onWsMessage);
		window.addEventListener(WS_GROUP_MESSAGE_EVENT, onWsMessage);
		return () => {
			window.removeEventListener(WS_PRIVATE_MESSAGE_EVENT, onWsMessage);
			window.removeEventListener(WS_GROUP_MESSAGE_EVENT, onWsMessage);
		};
	});

	$effect(() => {
		void reloadKey;
		let cancelled = false;
		async function fetchChats() {
			try {
				// 仅首刷（无数据）展示整页 loading；WS 触发的后台刷新静默换数据
				// chats 读取必须 untrack：否则数组回写（新引用）成为 effect 依赖，
				// fetch 完成即重跑自身 → 登录态下无限请求循环（线上 P0）
				if (untrack(() => !chats.length)) loading = true;
				loadError = null;
				needsAuth = false;
				// 并行加载私聊与我的群组，合并为统一会话列表（按置顶 > 未读 > 已读，时间倒序）。
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
				// HTTP 快照种子化：WS 订阅只覆盖之后的变化（spec 无订阅时初始在线快照），
			// 列表渲染的 liveChats 以 presence store 为准，需先注入快照在线态。
			seedOnline(privData.filter((c) => c.presence_status === 'online').map((c) => c.other_user?.user_id ?? ''));
			const privateChats: ChatItem[] = privData.map((c, i) => ({
					id: `chat-${i}`,
					// spec: PrivateChatItem.other_user — the peer; private messages are
					// addressed by target_user_id (the peer's user_id), not a chat id.
					peerUserId: c.other_user?.user_id ?? '',
					name: c.other_user?.nickname ?? '',
					avatar: c.other_user?.avatar_url ?? '',
					lastMessage: c.last_message?.content ?? '',
					time: formatChatListTime(c.last_message?.sent_at ?? c.last_activity_at),
					timestamp: c.last_message?.sent_at ?? c.last_activity_at ?? '',
					unread: c.unread_count ?? 0,
					isPinned: c.is_pinned ?? false,
					isMuted: c.is_muted ?? false,
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
						time: formatChatListTime(c.last_message?.sent_at ?? c.last_activity_at ?? ''),
						timestamp: c.last_message?.sent_at ?? c.last_activity_at ?? '',
						unread: c.viewer_state?.unread_count ?? 0,
						isPinned: c.viewer_state?.is_pinned ?? false,
						isMuted: c.viewer_state?.is_muted ?? false,
						online: false,
						group: true,
						kind: 'group'
					};
				});
				// 私聊与群聊混合，统一按「置顶 > 未读 > 已读，组内时间倒序」自然排序。
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
	let showNewChat = $state(false);
	let newChatTab = $state<'add' | 'chat'>('chat');
	let showCreateGroup = $state(false);
	// 会话 id：私聊为数字索引/chat id，群聊为字符串 group_id。
	let selectedId = $state<number | string | null>(null);

	// 选中会话：设置选中 id + 立即清零本地未读 + 自然重排归位 + 调 markRead API。
	const selectChat = (chat: ChatItem) => {
		if (selectedId !== chat.id) selectedId = chat.id;
		if (chat.unread > 0) {
			chats = chats.map((c) => (c.id === chat.id ? { ...c, unread: 0 } : c)).sort(compareChatItems);
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

	// 免打扰开关：与置顶同款乐观更新（spec: private_chats|groups update_preferences is_muted），
	// 失败回滚并提示。免打扰不改变排序与未读计数，仅弱化提醒语义。
	const toggleMute = (chat: ChatItem) => {
		const next = !chat.isMuted;
		chats = chats.map((c) => (c.id === chat.id ? { ...c, isMuted: next } : c));
		const req =
			chat.kind === 'group'
				? updateGroupPreferences({ group_id: String(chat.id), is_muted: next })
				: updatePrivateChatPreferences({ target_user_id: chat.peerUserId || String(chat.id), is_muted: next });
		void req
			.then((res) => {
				if (res.code !== 0) throw new Error(res.message || '设置失败');
			})
			.catch(() => {
				chats = chats.map((c) => (c.id === chat.id ? { ...c, isMuted: !next } : c));
				showToast('免打扰设置失败，请重试', 'danger');
			});
	};

	// 删除会话（仅私聊；spec: private_chats/delete 只影响自己视角，对方不受影响）：
	// ConfirmDialog 确认后调 API，成功移除本地行；群聊无删除端点（退群走 groups/exit）。
	const handleDeleteChat = () => {
		const chat = deleteTarget;
		if (!chat) return;
		void deletePrivateChat({ target_user_id: chat.peerUserId || String(chat.id) })
			.then((res) => {
				if (res.code === 0) {
					chats = chats.filter((c) => c.id !== chat.id);
					if (selectedId === chat.id) selectedId = null;
					showToast('会话已删除', 'success');
				} else {
					showToast(res.message || '删除失败，请重试', 'danger');
				}
			})
			.catch(() => showToast('删除失败，请重试', 'danger'))
			.finally(() => (deleteTarget = null));
	};
	let deleteTarget = $state<ChatItem | null>(null);

	// 渲染用实时在线判定：WS 推送优先，HTTP 快照（chat.online）仅兜底
	const liveChats = $derived(
		chats.map((c) => ({ ...c, online: presence.isOnline(c.peerUserId) }))
	);

	const filtered = $derived(
		liveChats.filter((c) => c.name.toLowerCase().includes(searchQuery.toLowerCase()))
	);
	const onlineChats = $derived(liveChats.filter((c) => c.online));
	const totalUnread = $derived(chats.reduce((s, c) => s + c.unread, 0));
	// 页面数据是服务端真相：任何变化（拉取/WS 重拉/打开会话清零）同步全局未读徽标
	$effect(() => {
		setChatUnread(totalUnread);
	});
	// 分栏阈值：桌面或所有平板端（≥600px）全面启用 Master-Detail 双栏分栏，彻底消灭单列拉伸与空白
	const useSplitPane = $derived(platform.platform === 'desktop' || platform.platform === 'tablet');
	const selectedChat = $derived(liveChats.find((c) => c.id === selectedId));
</script>
<svelte:head>
	<title>消息 · HotSpot</title>
</svelte:head>


{#if useSplitPane}
	<!-- 未登录：不分栏，通栏单一登录引导（Gemini UI 审查：双栏各自空态+引导互相矛盾且重复） -->
	{#if needsAuth}
	<div class="content-mode-wide flex h-full items-center justify-center">
		<NeedsAuthNotice title="登录后查看" description="登录后即可查看和发起会话" />
	</div>
	{:else}
	<!-- ═══ Desktop / Tablet Messages — split pane (list left + chat right) ═══ -->
	<!-- h-full：RoutedContentViewport 内容链提供显式高度，分栏固定视口高、左栏列表内部独立滚动；平板/桌面画布均撑满。 -->
	<div class="content-mode-wide flex h-full">
		<!-- Left: chat list -->
		<div class="flex flex-col shrink-0 overflow-hidden w-80 border-r border-border-default bg-surface-primary">
			<!-- List header（对齐 Figma：px-4 pt-4 pb-3，标题 18 + Plus 30 + 搜索框） -->
			<div class="shrink-0 border-b border-border-subtle/40">
				<!-- 登记例外：本页三栏面板有意用 surface-primary 实底（与画廊画布
				     契约不同），PageHeader 的 page-header-solid 外壳感知底色（移动端
				     =surface-page）在此会与列表面板错配，故用 ! 覆盖为面板同色。
				     全项目唯一一处 ! 覆盖，勿仿照。 -->
				<PageHeader title="消息" class="!bg-surface-primary" contentClassName="px-4">
					{#snippet trailing()}
						{#if !needsAuth}
						<button
							type="button"
							class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring border border-border-default bg-surface-primary shadow-raised-sm size-9"
							onclick={() => {
								newChatTab = 'chat';
								showNewChat = true;
							}}
							aria-label="发起新会话"
						>
							<Plus class="size-4 text-text-secondary" strokeWidth={1.5} />
						</button>
						{/if}
					{/snippet}
					{#snippet below()}
						{#if !needsAuth}
						<div class="mt-3">
							<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索会话…" class="px-3 py-2" textClass="text-sm" />
						</div>
						{/if}
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
				onToggleMute={toggleMute}
				onDelete={(chat) => (deleteTarget = chat)}
				{selectedId}
				compact
				fill
			/>
		</div>

		<!-- Right: chat view or empty state（底色与面板统一，中间由发丝边框清晰划分） -->
		<div class="flex-1 overflow-hidden bg-surface-primary">
			{#if selectedId !== null && selectedChat}
				<div class="h-full">
					{#if selectedChat.kind === 'group'}
						<GroupChatView groupId={selectedChat.id} compact showBackButton={false} />
					{:else}
						<ChatView peerUserId={selectedChat.peerUserId} compact contact={{ name: selectedChat.name, avatar: selectedChat.avatar, online: selectedChat.online }} showBackButton={false} />
					{/if}
				</div>
			{:else}
				<div class="h-full flex items-center justify-center">
					<EmptyState title="选择一个会话" description="从左侧选择一个会话开始聊天" class="h-full justify-center py-0!">
						{#snippet icon()}
							<MessageSquareDashed class="size-9 opacity-muted text-text-secondary" strokeWidth={1.5} />
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
				style="width:27.5rem;height:var(--cst-sheet-height);max-height:var(--cst-sheet-max-height);"
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
			<Dialog.Content side="center" class="overflow-hidden w-120" aria-label="创建群组"  style="height:var(--cst-sheet-height);max-height:var(--cst-sheet-max-height)">
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
	{/if}
{:else}
	<!-- ═══ Mobile Messages ═══ -->
	<div class="flex flex-col flex-1 min-h-full bg-surface-primary">
		<!-- 同上：登记例外（surface-primary 面板实底，见左栏说明）。 -->
		<PageHeader title="消息" class="!bg-surface-primary" sticky titleSize={platform.platform === 'desktop' ? 'xl' : '2xl'} subtitleSize="sm">
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
						class="hover-btn scale-btn hit-slop flex items-center justify-center rounded-full focus-ring border border-border-default bg-surface-primary shadow-raised-sm size-9"
						onclick={() => {
							newChatTab = 'chat';
							showNewChat = true;
						}}
						aria-label="发起新会话"
					>
						<Plus class="size-4 text-text-secondary" strokeWidth={1.5} />
					</button>
				</div>
			{/snippet}
			{#snippet below()}
				<!-- 搜索框：sticky header 内第二行（对齐 Figma MessagesMobile） -->
				<div class="mt-3">
					<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索会话…" class="rounded-md px-4 py-2.5" />
				</div>
			{/snippet}
		</PageHeader>

		<div class="animate-fade-in flex flex-col flex-1 pb-8 bg-surface-primary">
			{#if onlineChats.length > 0}
				<div class="px-4 pt-2">
					<OnlineStrip chats={onlineChats} onSelect={(c) => void goto(ROUTES.message(c.peerUserId || c.id))} />
				</div>
			{/if}

			<div class="px-4 mt-3 mb-1.5">
				<p class="eyebrow">最近</p>
			</div>

			<ChatListBody
				chats={filtered}
				loadError={loadError}
				needsAuth={needsAuth}
				loading={loading}
				onRetry={() => reloadKey++}
				onSelect={(chat) => void (chat.kind === 'group' ? goto(ROUTES.groupMessage(chat.id)) : goto(ROUTES.message(chat.peerUserId || chat.id)))}
				onTogglePin={togglePin}
				onToggleMute={toggleMute}
				onDelete={(chat) => (deleteTarget = chat)}
			/>
		</div>

		<Dialog.Root open={showNewChat} onOpenChange={(next) => (next ? undefined : (showNewChat = false))}>
			<Dialog.Content side="bottom" class="overflow-hidden" style="height:var(--cst-sheet-height-lg);box-shadow:var(--cst-elevation-sheet);">
				<NewConversationSheet
					defaultTab={newChatTab}
					onClose={() => (showNewChat = false)}
					onNewChat={(id) => void goto(ROUTES.message(id))}
					onCreateGroup={() => (showCreateGroup = true)}
				/>
			</Dialog.Content>
		</Dialog.Root>

		<Dialog.Root open={showCreateGroup} onOpenChange={(next) => (next ? undefined : (showCreateGroup = false))}>
			<Dialog.Content side="bottom" class="overflow-hidden" style="height:var(--cst-sheet-height-2xl);box-shadow:var(--cst-elevation-sheet);">
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
{/if}

<!-- 删除会话确认（危险操作防误触；仅删除自己视角的私聊会话，对方不受影响） -->
<ConfirmDialog
	open={deleteTarget !== null}
	title={`删除与「${deleteTarget?.name ?? ''}」的会话？`}
	description="删除后将清空本机的聊天记录，对方不受影响。"
	icon={MessageSquareDashed}
	confirmLabel="删除会话"
	onCancel={() => (deleteTarget = null)}
	onConfirm={handleDeleteChat}
/>
