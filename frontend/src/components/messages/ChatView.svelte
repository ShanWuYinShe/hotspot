<script lang="ts">
	import { ArrowLeft, UserRound } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import Avatar from '@/components/shared/Avatar.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import ChatMessageList from '@/components/messages/ChatMessageList.svelte';
	import ChatInputBar from '@/components/messages/ChatInputBar.svelte';
	import ChatColumn from '@/components/messages/ChatColumn.svelte';
	import ActionIconButton from '@/components/navigation/ActionIconButton.svelte';
	import NeedsAuthNotice from '@/components/auth/NeedsAuthNotice.svelte';
	import ReportDialog from '@/components/overlays/ReportDialog.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { getPrivateMessages, sendPrivateMessage, markPrivateRead, recallPrivateMessage, updatePrivateMessage, openTheDoor } from '@/api';
	import {
		WS_PRIVATE_MESSAGE_EVENT,
		WS_PRIVATE_MESSAGE_UPDATE_EVENT,
		WS_PRIVATE_TYPING_EVENT,
		wsClient
	} from '@/lib/ws/client';
	import type { PrivateMessageNewPayload, PrivateTypingPayload } from '@/lib/ws/types';
	import { presence, watchPresence, seedOnline } from '@/stores/presence.svelte';
	import { useChat, formatTime, createTypingIndicator, inRecallWindow, type ChatMessage } from '@/lib/chat.svelte';

	// ChatView — 私聊窗口（spec: 以对端 user_id 寻址）。
	// 历史来自 private_messages/fetch；发送走 private_messages/send。
	// 消息方向由 sender.user_id 与当前用户比较得出。
	// 共享状态机（加载/分页/滚动/typing/发送/WS 接收）由 useChat 工厂承载；
	// 消息区与输入栏由 ChatMessageList/ChatInputBar 共享组件渲染。

	interface ChatContact {
		name: string;
		avatar: string;
		online: boolean;
		lastSeen?: string;
	}

	let {
		peerUserId,
		contact: contactProp,
		showBackButton = false,
		onBack,
		compact = false
	}: {
		/** The peer's user_id (spec: target_user_id). */
		peerUserId: string;
		/** Optional peer display info from the chat list; fetched if absent. */
		contact?: ChatContact;
		showBackButton?: boolean;
		onBack?: () => void;
		compact?: boolean; // for desktop split-pane (smaller header)
	} = $props();

	// contactProp 仅作为初始联系人信息（此后可能由消息记录推导），有意只读取一次。
	// 无 contact 时从历史中第一条非自己消息的 sender 推导。
	let derivedContact = $state<ChatContact | null>(null);
	// 10100（未登录/会话失效）是正常业务态：给登录引导而非死胡同重试（对齐 notify/me）。
	let needsAuth = $state(false);

	// spec: private_messages/send 返回 10200 时 data 携带 { forbidden_reason: ForbiddenReason }。
	const SEND_BLOCK_REASONS: Record<string, string> = {
		blocked_by_user: '你已被对方拉黑',
		privacy_setting: '对方隐私设置不允许发送消息',
		stranger_quota_exceeded: '今日陌生人消息额度已用完'
	};

	const meId = $derived(auth.user?.user_id ?? '__me__');

	const chat = useChat({
		targetId: () => peerUserId,
		loadHistory: async (cursor) => {
			const res = await getPrivateMessages({
				target_user_id: peerUserId,
				pagination: { limit: 50, cursor }
			}).catch(() => null);
			if (res?.code === 10100) {
				// 未登录：给登录引导而非死胡同重试
				needsAuth = true;
				return null;
			}
			needsAuth = false;
			if (!res || res.code !== 0 || !res.data) return null;
			const items = res.data.items;
			const mapped: ChatMessage[] = items.map((m) => ({
				id: m.message_id,
				text: m.recalled_at ? '消息已撤回' : (m.content ?? ''),
				recalled: Boolean(m.recalled_at),
				editedAt: m.edited_at,
				sent: m.sender.user_id === meId,
				time: formatTime(m.sent_at),
				sentAt: m.sent_at
			}));
			// 首屏加载且调用方未传 contact 时，从最新一条非自己消息的 sender 推导。
			if (!cursor && !contactProp && !derivedContact) {
				const peer = items.map((m) => m.sender).find((s) => s.user_id !== meId);
				if (peer) {
					derivedContact = { name: peer.nickname, avatar: peer.avatar_url ?? '', online: false };
				}
			}
			return { messages: mapped, next_cursor: res.data.next_cursor ?? null };
		},
		markRead: () => {
			// spec: private_messages/mark_read (scope='session') 标记该会话全部已读;
			// 幂等语义,失败静默不打扰会话浏览。
			void markPrivateRead({ scope: 'session', target_user_id: peerUserId }).catch(() => {});
		},
		sendMessage: async (text) => {
			const res = await sendPrivateMessage({ target_user_id: peerUserId, type: 'text', content: text });
			if (res.code === 0) return { code: 0, message_id: res.data?.message_id };
			// spec: 10200 时 data = { forbidden_reason: ForbiddenReason }。
			// 后端 message 方向敏感（如拉黑方/被拉黑方两套文案），优先直显；
			// 本地映射仅作缺失兜底。
			if (res.code === 10200) {
				const reason = (res.data as { forbidden_reason?: string } | undefined)?.forbidden_reason;
				return { code: res.code, error: res.message || (reason && SEND_BLOCK_REASONS[reason]) || '消息发送失败' };
			}
			return { code: res.code, error: '消息发送失败' };
		},
		sendTypingSignal: (isTyping) => {
			// 未连接静默丢弃。
			if (!peerUserId || !wsClient.isConnected) return;
			wsClient.sendPrivateTyping(peerUserId, isTyping);
		},
		wsNewMessageEvent: WS_PRIVATE_MESSAGE_EVENT,
		wsMessageUpdateEvent: WS_PRIVATE_MESSAGE_UPDATE_EVENT,
		wsNewMessage: (e) => {
			const payload = (e as CustomEvent<PrivateMessageNewPayload>).detail;
			if (!payload || payload.target_user_id !== peerUserId) return null;
			const id = payload.message_id ?? '';
			if (!id) return null;
			// 按 message_id 去重：本标签页发送的消息在 HTTP 回执后已替换为真实 id，
			// WS 推送到达时自然去重；其它设备/标签页发出的消息则被正确同步显示。
			const sent = payload.sender?.user_id === meId;
			// markRead 延后（Round 10）：由 useChat 在「近底部/用户回底」时统一补标，
			// 上滑读历史时收到的新消息不再被立即标记已读。
			void sent;
			// sentAt 供 useChat 按服务端时间保持正序（resume 回放/实时交错不乱序）
			return { id, text: payload.content ?? '', sent, time: formatTime(payload.sent_at), sentAt: payload.sent_at };
		},
		wsMessageUpdate: (e) => {
			const detail = (e as CustomEvent<{ type: 'edited' | 'recalled' | 'read'; payload: PrivateMessageNewPayload }>).detail;
			if (!detail?.payload || detail.payload.target_user_id !== peerUserId) return null;
			const id = detail.payload.message_id ?? '';
			if (detail.type === 'edited') return { type: 'edited', id, content: detail.payload.content };
			if (detail.type === 'recalled') return { type: 'recalled', id };
			// 'read'：本地列表无已读状态字段，忽略
			return null;
		},
		recallMessage: async (id) => {
			const res = await recallPrivateMessage({ message_id: id });
			return { code: res.code, error: res.code !== 0 ? '撤回失败' : undefined };
		},
		updateMessage: async (id, content) => {
			const res = await updatePrivateMessage({ message_id: id, content });
			return { code: res.code, error: res.code !== 0 ? '编辑失败' : undefined };
		}
	});

	// 展示用联系人：调用方传入优先，否则用历史推导值。
	const contact = $derived(contactProp ?? derivedContact);

	// 对端资料主动解析：新会话历史里没有对端 sender 时兜底（头部显示「?」、
	// 标题回退「会话」）。有调用方传入/已推导/匿名/无 peerUserId 时不重复拉取。
	let peerProfileLoaded = $state(false);
	$effect(() => {
		if (peerProfileLoaded || derivedContact || contactProp || !peerUserId) return;
		if (!auth.isAuthenticated) return;
		peerProfileLoaded = true;
		void openTheDoor(peerUserId)
			.then((res) => {
				if (res.code !== 0 || derivedContact) return;
				const u = res.data?.user?.user;
				if (u?.nickname) {
					derivedContact = {
						name: u.nickname,
						avatar: u.avatar_url ?? '',
						online: res.data?.user?.presence_status === 'online'
					};
				}
			})
			.catch(() => {
				// 拉取失败静默：头部回退「?/会话」，不影响消息收发；重进页面可重试。
				peerProfileLoaded = false;
			});
	});

	const peerName = $derived(contact?.name ?? '');
	// 在线状态实时化：优先 WS 推送（presence store），HTTP 快照 contact.online 仅作初始兜底。
	const onlineNow = $derived(presence.isOnline(peerUserId));
	// 最后一条非本人消息，供屏幕阅读器 aria-live 播报（只播文本，避免整表重读）
	const lastIncoming = $derived([...chat.messages].reverse().find((m) => !m.sent));

	// --- typing 接收：仅处理对端（payload.actor.user_id === peer）推送。 ---
	const typing = createTypingIndicator();

	$effect(() => {
		const peer = peerUserId;
		if (!peer) return;

		const onTyping = (e: Event) => {
			const payload = (e as CustomEvent<PrivateTypingPayload>).detail;
			if (!payload || payload.target_user_id !== peer) return;
			if (payload.actor?.user_id !== peer) return;
			typing.notify(payload.is_typing);
		};

		window.addEventListener(WS_PRIVATE_TYPING_EVENT, onTyping as EventListener);
		return () => {
			window.removeEventListener(WS_PRIVATE_TYPING_EVENT, onTyping as EventListener);
			typing.reset();
		};
	});

	// 在线状态订阅：切换会话时自动重新订阅。
	$effect(() => {
		const peer = peerUserId;
		if (!peer) return;
		const unwatch = watchPresence([peer]);
		// 订阅只覆盖「订阅之后的变化」：订阅前已在线的对端不会再收到 UserOnline
		// 事件（spec 无订阅时初始在线快照），用 HTTP 快照 contact.online 补初始态。
		if (contact?.online) seedOnline([peer]);
		return () => {
			unwatch();
		};
	});

	// 收到消息的举报（reports/create target_type='message'）
	let reportMessageId = $state<string | null>(null);
</script>

<div class="flex min-h-0 flex-col h-full bg-surface-chat">
	<p class="sr-only" aria-live="polite">{lastIncoming ? `新消息：${lastIncoming.text}` : ''}</p>
	<!-- Header -->
	<!-- 去硬边线（Round 2）：聊天画布自身底色承担分隔；
	     头部层级（R13）：返回（移动）→ 头像+昵称/状态（可点进资料）→ 资料入口图标钮 -->
	<div class="shrink-0 bg-surface-chat {compact ? 'h-13' : 'h-15'}">
		<!-- 头部内容经 ChatColumn 与消息列/输入栏同轴（R16）；px-4 是列内 gutter -->
		<ChatColumn class="flex h-full items-center gap-2 px-4">
		{#if showBackButton}
			<button
				type="button"
				class="hover-btn hit-slop scale-btn flex items-center justify-center rounded-full border border-border-default bg-surface-primary size-10.5 shrink-0 shadow-raised-sm focus-ring"
				onclick={onBack}
				aria-label="返回"
			>
				<ArrowLeft class="size-4.5 text-text-secondary"   strokeWidth={1.5} />
			</button>
		{/if}

		<!-- 头像 + 名称/状态：点击进入对方主页（移动端私聊页的资料入口，Telegram/iMessage 惯例） -->
		<button
			type="button"
			class="avatar-entry hit-slop-y hover-bg focus-ring flex min-w-0 flex-1 items-center gap-2.5 rounded-md text-left"
			onclick={() => void goto(ROUTES.user(peerUserId))}
			aria-label={`查看 ${peerName || '对方'} 的资料`}
			title="查看资料"
		>
			<Avatar
				src={contact?.avatar}
				name={peerName}
				decorative
				size={compact ? 36 : 40}
				class="border-regular border-border-default shadow-raised-sm"
			>
				{#snippet badge()}
					{#if onlineNow}
						<OnlineDot size="sm" />
					{/if}
				{/snippet}
			</Avatar>

			<!-- Name + status -->
			<div class="flex-1 min-w-0">
				<!-- 会话对端名：移动端独立页 = 页面唯一 h1；桌面/宽平板分栏（compact）
				     时页面已有「消息」PageHeader h1，降为 h2，避免一页两个 h1。
				     font-normal 抵消 base h1 的 medium 字重，保持与原 p 一致的视觉。 -->
				<svelte:element
					this={compact ? 'h2' : 'h1'}
					class="truncate font-semibold text-text-primary {compact ? 'text-base' : 'text-lg'}"
				>
					{peerName || '会话'}
				</svelte:element>
			<p class="mt-0.5 flex items-center gap-1 text-sm {typing.active ? 'text-accent-default' : onlineNow ? 'text-status-online' : 'text-text-secondary'}">
				{#if typing.active}
					正在输入…
				{:else if onlineNow}
					在线
				{:else if contact?.lastSeen}
					活跃于{contact.lastSeen}
				{:else if contact}
					离线
				{/if}
			</p>
			</div>
		</button>

		<!-- 资料入口：与「点头像/昵称进资料」同一跳转，给桌面/平板一个显式、可发现的入口 -->
		<ActionIconButton
			icon={UserRound}
			label="查看资料"
			variant="ghost"
			size={36}
			onClick={() => void goto(ROUTES.user(peerUserId))}
		/>
		</ChatColumn>

	</div>

	{#if needsAuth}
		<!-- 未登录：登录引导（替代死胡同重试；输入栏一并隐藏，匿名不可发送） -->
		<div class="flex flex-1 items-center justify-center">
			<NeedsAuthNotice spotlight title="登录后查看" description="登录后即可查看和发起会话" />
		</div>
	{:else}
		<!-- Messages -->
		<ChatMessageList {chat} canRecall={(msg) => msg.sent && inRecallWindow(msg.sentAt)} onReport={(id) => (reportMessageId = id)} />

		<!-- Input bar -->
		<ChatInputBar {chat} />
	{/if}
</div>

<ReportDialog
	open={reportMessageId !== null}
	targetType="message"
	targetId={reportMessageId ?? ''}
	targetLabel="这条消息"
	onClose={() => (reportMessageId = null)}
/>
