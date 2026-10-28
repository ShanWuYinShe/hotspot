<script lang="ts">
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { Plus, MessageSquareDashed, Pin, BellOff, Trash2, X, MessageCircle, Users, Zap } from '@lucide/svelte';
	import { platform } from '@/stores/platform.svelte';
	import PageHeader from '@/components/layout/page/PageHeader.svelte';
	import ContentColumn from '@/components/layout/page/ContentColumn.svelte';
	import SearchInput from '@/components/forms/SearchInput.svelte';
	import ChatView from '@/components/messages/ChatView.svelte';
	import Button from '@/components/ui/button.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import OnlineStrip from './shared/OnlineStrip.svelte';
	import ChatListBody from './shared/ChatListBody.svelte';
	import ChatEmptyPanel from './shared/ChatEmptyPanel.svelte';
	import { canUseSplitPane, resolvePushChatHref } from './shared/deep-link';
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

	const queryPeer = $derived(page.url.searchParams.get('peer'));
	const queryGroup = $derived(page.url.searchParams.get('group'));

	// 选中会话：设置选中 id + 立即清零本地未读 + 自然重排归位 + 调 markRead API。
	const selectChat = (chat: ChatItem) => {
		if (selectedId !== chat.id) selectedId = chat.id;
		if (useSplitPane) {
			const targetParam = chat.kind === 'group' ? `group=${encodeURIComponent(String(chat.id))}` : `peer=${encodeURIComponent(chat.peerUserId || String(chat.id))}`;
			const currentParam = chat.kind === 'group' ? (queryGroup ? `group=${queryGroup}` : '') : (queryPeer ? `peer=${queryPeer}` : '');
			if (targetParam !== currentParam) {
				void goto(`?${targetParam}`, { replaceState: true, noScroll: true, keepFocus: true });
			}
		}
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
	// 移动端会话操作弹层目标（s2-01：置顶/免打扰/删除在移动端原本无入口）
	let moreTarget = $state<ChatItem | null>(null);

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
	// 分栏阈值：桌面与宽平板（≥960px，如横屏 iPad）启用 Master-Detail 双栏；
	// 窄平板（820 级纵向）保留单栏列表，点开会话再进详情（判定见 shared/deep-link.ts）。
	const useSplitPane = $derived(canUseSplitPane(platform));

	// 窄平板/移动不可分栏：把 ?peer= / ?group= 深链落到单栏推入页。
	// 否则（桌面分享/历史/通知带过来的深链）在平板上只会停在列表页——
	// 用户看到的是一块没有会话内容的空白面板。replaceState 不新增历史项。
	$effect(() => {
		if (useSplitPane) return;
		const href = resolvePushChatHref(queryPeer, queryGroup);
		if (href) void goto(href, { replaceState: true });
	});
	const selectedChat = $derived(
		liveChats.find((c) => {
			if (selectedId !== null && c.id === selectedId) return true;
			if (queryGroup && c.kind === 'group' && String(c.id) === queryGroup) return true;
			if (queryPeer && c.kind !== 'group' && (c.peerUserId === queryPeer || String(c.id) === queryPeer)) return true;
			return false;
		})
	);

	// 桌面端深链参数（?peer=... 或 ?group=...）自动对齐选中行
	$effect(() => {
		if (!useSplitPane) return;
		if (queryGroup) {
			const found = liveChats.find((c) => c.kind === 'group' && String(c.id) === queryGroup);
			if (found && selectedId !== found.id) {
				selectedId = found.id;
			}
		} else if (queryPeer) {
			const found = liveChats.find((c) => c.kind !== 'group' && (c.peerUserId === queryPeer || String(c.id) === queryPeer));
			if (found && selectedId !== found.id) {
				selectedId = found.id;
			}
		}
	});
</script>
<svelte:head>
	<title>消息 · HotSpot</title>
</svelte:head>


{#if useSplitPane}
	<!-- 未登录：不分栏，通栏单一登录引导（Gemini UI 审查：双栏各自空态+引导互相矛盾且重复） -->
	{#if needsAuth}
	<!-- 访客态补页头：此前分栏分支直接渲染 NeedsAuthNotice，桌面/平板访客整页没有
	     PageHeader/无 h1，与移动端（有「消息」页头）不一致（全站复审 s2-15）。 -->
	<div class="content-mode-wide flex h-full flex-col">
		<!-- 访客态页头通栏铺在画布上：不能沿用分栏左栏的 surface-primary 实底
		     （会在画布上形成一条白色横带），顶部保持透明、与连续画布融合。 -->
		<PageHeader title="消息" sticky titleSize={platform.platform === 'desktop' ? 'xl' : '2xl'} />
		<div class="flex flex-1 items-center justify-center">
			<NeedsAuthNotice
				spotlight
				title="登录后查看消息"
				description="登录后即可查看和发起会话"
				highlights={[
					{ icon: MessageCircle, text: '私聊与群聊会话' },
					{ icon: Users, text: '在线好友与群组' },
					{ icon: Zap, text: '消息实时同步' }
				]}
			/>
		</div>
	</div>
	{:else}
	<!-- ═══ Desktop / Tablet Messages — split pane (list left + chat right) ═══ -->
	<!-- h-full：RoutedContentViewport 内容链提供显式高度，分栏固定视口高、左栏列表内部独立滚动；平板/桌面画布均撑满。 -->
	<div class="content-mode-wide flex h-full">
		<!-- Left: chat list -->
		<!-- 去硬分隔线（Round 2）：分栏边界改极浅发丝线，避免一条明显竖线贯穿整页 -->
		<div class="flex flex-col shrink-0 overflow-hidden w-80 border-r border-border-subtle/40 bg-surface-primary">
			<!-- List header（对齐 Figma：px-4 pt-4 pb-3，标题 18 + Plus 30 + 搜索框） -->
			<div class="shrink-0">
				<!-- 登记例外：本页三栏面板有意用 surface-primary 实底（与画廊画布
				     契约不同），PageHeader 的 page-header-solid 外壳感知底色（移动端
				     =surface-page）在此会与列表面板错配，故用 ! 覆盖为面板同色。
				     全项目唯一一处 ! 覆盖，勿仿照。 -->
				<PageHeader title="消息" class="!bg-surface-primary" contentClassName="px-4" titleSize="xl">
					{#snippet trailing()}
						{#if !needsAuth}
						<!-- 带文字标签（2026-10-19 UI 审查）：全站底栏中央是「+ 发布瞬间」FAB，
						     本页头部若也用纯「+」图标，同屏会出现两个「+」做两件事、语义无法区分。
						     加「新会话」文字后一眼可辨；高度仍 h-9，与页头对齐。 -->
						<Button
							type="button"
							variant="soft"
							size="sm"
							class="hit-slop-44 h-9 rounded-full px-3.5"
							aria-label="发起新会话"
							onclick={() => {
								newChatTab = 'chat';
								showNewChat = true;
							}}
						>
							<Plus class="size-4 text-text-secondary" strokeWidth={1.5} aria-hidden="true" />
							新会话
						</Button>
						{/if}
					{/snippet}
					{#snippet below()}
						{#if !needsAuth}
						<div class="mt-3">
							<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索会话…" class="px-3 py-2.5" textClass="text-md" />
						</div>
						{/if}
					{/snippet}
				</PageHeader>
			</div>

			<!-- 在线 strip -->
			{#if onlineChats.length > 0}
				<OnlineStrip chats={onlineChats} variant="sm" {selectedId} onSelect={selectChat} />
			{:else if loading && !needsAuth && !loadError}
				<!-- N2：分栏骨架期在线条占位（sm 几何），避免加载后列表被在线条下推 -->
				<div class="shrink-0 px-4 py-2.5" aria-hidden="true">
					<div class="eyebrow mb-2 h-3 w-8 rounded-sm bg-surface-subtle"></div>
					<div class="flex gap-3">
						{#each Array(3) as _, i (i)}
							<div class="flex flex-col items-center gap-1">
								<div class="skeleton-block size-9 rounded-full"></div>
								<div class="h-2.5 w-7 rounded-sm bg-surface-subtle"></div>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<ChatListBody
				chats={filtered}
				loadError={loadError}
				needsAuth={needsAuth}
				loading={loading}
				onRetry={() => reloadKey++}
				onSelect={selectChat}
				onMore={(chat) => (moreTarget = chat)}
				searchQuery={searchQuery}
				onNewChat={() => {
					newChatTab = 'chat';
					showNewChat = true;
				}}
				{selectedId}
				compact
				fill
			/>
		</div>

		<!-- Right: chat view or empty state：右栏统一用聊天画布（--cst-surface-chat），
		     与左栏列表的 surface-primary 形成一档微差，靠画布本身分区、不额外加边框 -->
		<div class="flex-1 overflow-hidden bg-surface-chat">
			{#if queryGroup || (selectedChat && selectedChat.kind === 'group')}
				<div class="h-full">
					<GroupChatView groupId={queryGroup || String(selectedChat?.id ?? '')} compact showBackButton={false} />
				</div>
			{:else if queryPeer || (selectedChat && selectedChat.kind !== 'group')}
				<div class="h-full">
					<ChatView
						peerUserId={queryPeer || selectedChat?.peerUserId || String(selectedChat?.id ?? '')}
						compact
						contact={selectedChat ? { name: selectedChat.name, avatar: selectedChat.avatar, online: selectedChat.online } : undefined}
						showBackButton={false}
					/>
				</div>
			{:else}
				<!-- 空会话面板（R13）：引导 + 最近 3 条会话建议（纯前端取自已拉到的列表）。
				     小面板不开 spotlight；发新聊天入口内置在面板底部，动线闭环。 -->
				<ChatEmptyPanel
					recent={liveChats}
					onSelect={selectChat}
					onNewChat={() => {
						newChatTab = 'chat';
						showNewChat = true;
					}}
				/>
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
		<!-- gutter 走 --cst-page-gutter token（移动 16px）；页头/正文不再硬编码 px-4 -->
		<PageHeader title="消息" class="!bg-surface-primary" sticky titleSize={platform.platform === 'desktop' ? 'xl' : '2xl'}>
			{#snippet trailing()}
				<!-- 未读计数统一在标题右侧（对齐首页/附近瞬间样板），与「新会话」同排；
				     未登录隐藏无效控件（对齐桌面分栏守卫；Gemini UI 审查 r24） -->
				{#if !needsAuth}
				<div class="flex items-center gap-2">
					{#if totalUnread > 0}
						<span class="text-text-secondary text-xs">{totalUnread} 条未读</span>
					{:else}
						<!-- 全零态用中性灰（status-online 绿是「在线」语义，不作正向文案） -->
						<span class="text-text-secondary text-xs">全部已读</span>
					{/if}
					<!-- 移动端单栏页头（本分支）：与底栏中央「+ 发布瞬间」FAB 同屏，
					     纯「+」图标会与之语义混同，故加「新会话」文字标签（同分栏分支口径）。 -->
					<Button
						type="button"
						variant="soft"
						size="sm"
						class="hit-slop-44 h-9 rounded-full px-3.5"
						aria-label="发起新会话"
						onclick={() => {
							newChatTab = 'chat';
							showNewChat = true;
						}}
					>
						<Plus class="size-4 text-text-secondary" strokeWidth={1.5} aria-hidden="true" />
						新会话
					</Button>
				</div>
				{/if}
			{/snippet}
			{#snippet below()}
				<!-- 搜索框：sticky header 内第二行（对齐 Figma MessagesMobile）；未登录隐藏（r24） -->
				{#if !needsAuth}
				<div class="mt-3">
					<SearchInput value={searchQuery} onChange={(v) => (searchQuery = v)} placeholder="搜索会话…" class="rounded-md px-4 py-2.5" textClass="text-md" />
				</div>
				{/if}
			{/snippet}
		</PageHeader>

		<div class="animate-fade-in flex flex-col flex-1 pb-6 bg-surface-primary">
			{#if onlineChats.length > 0}
				<div class="px-[var(--cst-page-gutter)] pt-2">
					<OnlineStrip chats={onlineChats} onSelect={(c) => void goto(ROUTES.message(c.peerUserId || c.id))} />
				</div>
			{:else if loading && !needsAuth && !loadError}
				<!-- N2 修复：骨架期在线条占位必须渲染在「最近」标签**之前**（与真实内容顺序：
				     在线条 → 最近 → 列表 一致），高度 = 真实 lg 条 92px，避免加载后整体下移。 -->
				<div class="px-[var(--cst-page-gutter)] pt-2" aria-hidden="true">
					<div class="mb-3 flex h-[92px] flex-col overflow-hidden">
						<div class="eyebrow mb-2.5 h-3 w-8 rounded-sm bg-surface-subtle"></div>
						<div class="flex gap-3">
							{#each Array(4) as _, i (i)}
								<div class="flex flex-col items-center gap-1">
									<div class="skeleton-block size-11 rounded-full"></div>
									<div class="h-2.5 w-8 rounded-sm bg-surface-subtle"></div>
								</div>
							{/each}
						</div>
					</div>
				</div>
			{/if}

			{#if !needsAuth}
			<div class="px-[var(--cst-page-gutter)] mt-3 mb-1.5">
				<p class="eyebrow">最近</p>
			</div>
			{/if}

			<ChatListBody
				chats={filtered}
				loadError={loadError}
				needsAuth={needsAuth}
				loading={loading}
				onRetry={() => reloadKey++}
				onSelect={(chat) => void (chat.kind === 'group' ? goto(ROUTES.groupMessage(chat.id)) : goto(ROUTES.message(chat.peerUserId || chat.id)))}
				onMore={(chat) => (moreTarget = chat)}
				searchQuery={searchQuery}
				onNewChat={() => {
					newChatTab = 'chat';
					showNewChat = true;
				}}
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
<!-- 移动端会话操作（s2-01）：置顶/免打扰/删除在移动端没有 hover 显现入口，
     此前只能「取消」不能「新增」。形态走全站「更多操作」族的底部浮动卡
     （与用户详情页更多操作同语言）；删除仍交给下方 ConfirmDialog 二次确认。 -->
<Dialog.Root open={!!moreTarget} onOpenChange={(next) => (next ? undefined : (moreTarget = null))}>
	<Dialog.Content
		side="bottom"
		class="inset-x-4 bottom-4 mx-auto max-w-md rounded-xl p-5"
		aria-label={moreTarget ? `${moreTarget.name} 的会话操作` : '会话操作'}
		style="box-shadow:var(--cst-elevation-modal);"
	>
		{#if moreTarget}
			<div class="flex items-center justify-between">
				<div>
					<p class="eyebrow">会话操作</p>
					<p class="mt-1 text-md text-text-primary">{moreTarget.name}</p>
				</div>
				<ActionIconButton icon={X} label="关闭" onClick={() => (moreTarget = null)} />
			</div>
			<div class="mt-4 flex flex-col gap-2">
				<Button
					type="button"
					variant="secondary"
					size="lg"
					class="w-full justify-start"
					onclick={() => {
						const chat = moreTarget;
						moreTarget = null;
						if (chat) togglePin(chat);
					}}
				>
					<Pin class="size-4" strokeWidth={1.75} />
					{moreTarget.isPinned ? '取消置顶' : '置顶会话'}
				</Button>
				<Button
					type="button"
					variant="secondary"
					size="lg"
					class="w-full justify-start"
					onclick={() => {
						const chat = moreTarget;
						moreTarget = null;
						if (chat) toggleMute(chat);
					}}
				>
					<BellOff class="size-4" strokeWidth={1.75} />
					{moreTarget.isMuted ? '关闭免打扰' : '开启免打扰'}
				</Button>
				{#if moreTarget.kind !== 'group'}
					<Button
						type="button"
						variant="secondary"
						size="lg"
						class="w-full justify-start text-status-danger"
						onclick={() => {
							deleteTarget = moreTarget;
							moreTarget = null;
						}}
					>
						<Trash2 class="size-4" strokeWidth={1.75} />
						删除会话
					</Button>
				{/if}
			</div>
		{/if}
	</Dialog.Content>
</Dialog.Root>

<ConfirmDialog
	open={deleteTarget !== null}
	title={`删除与「${deleteTarget?.name ?? ''}」的会话？`}
	description="删除后将清空本机的聊天记录，对方不受影响。"
	icon={MessageSquareDashed}
	confirmLabel="删除会话"
	onCancel={() => (deleteTarget = null)}
	onConfirm={handleDeleteChat}
/>
