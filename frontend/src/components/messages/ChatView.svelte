<script lang="ts">
	import { ArrowLeft, Phone, Video, Send, Smile, MoreHorizontal, MessageCircle } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import OnlineDot from '@/components/data-display/OnlineDot.svelte';
	import EmptyState from '@/components/data-display/EmptyState.svelte';
	import LoadError from '@/components/data-display/LoadError.svelte';
	import MessageBubble from '@/components/messages/MessageBubble.svelte';
	import ChatDatePill from '@/components/messages/ChatDatePill.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { getPrivateMessages, sendPrivateMessage, markPrivateRead, recallPrivateMessage, updatePrivateMessage } from '@/api';
	import {
		WS_PRIVATE_MESSAGE_EVENT,
		WS_PRIVATE_MESSAGE_UPDATE_EVENT,
		WS_PRIVATE_TYPING_EVENT,
		wsClient
	} from '@/lib/ws/client';
	import type { PrivateMessageNewPayload, PrivateTypingPayload } from '@/lib/ws/types';
	import { presence, watchPresence } from '@/stores/presence.svelte';
	import { useChat, formatTime, type ChatMessage } from '@/lib/chat.svelte';

	// ChatView — 私聊窗口（spec: 以对端 user_id 寻址）。
	// 历史来自 private_messages/fetch；发送走 private_messages/send。
	// 消息方向由 sender.user_id 与当前用户比较得出。
	// 共享状态机（加载/分页/滚动/typing/发送/WS 接收）由 useChat 工厂承载。

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
			if (res.code === 10200) {
				const reason = (res.data as { forbidden_reason?: string } | undefined)?.forbidden_reason;
				return { code: res.code, error: (reason && SEND_BLOCK_REASONS[reason]) || '消息发送失败' };
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
			// 自己发送的消息跳过（乐观追加 + HTTP 回执已处理）
			if (payload.sender?.user_id === meId) return null;
			const id = payload.message_id ?? '';
			if (!id) return null;
			// 会话有新消息即视为已读（与加载行为一致）
			void markPrivateRead({ scope: 'session', target_user_id: peerUserId }).catch(() => {});
			// sentAt 供 useChat 按服务端时间保持正序（resume 回放/实时交错不乱序）
			return { id, text: payload.content ?? '', sent: false, time: formatTime(payload.sent_at), sentAt: payload.sent_at };
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

	// 撤回时间窗（spec message_recall_window_seconds，默认 300s；服务端 10400 兜底，ClientConfig 接入留后续）
	const RECALL_WINDOW_MS = 300_000;
	const inRecallWindow = (sentAt?: string) => {
		if (!sentAt) return true;
		return Date.now() - new Date(sentAt).getTime() < RECALL_WINDOW_MS;
	};

	// 展示用联系人：调用方传入优先，否则用历史推导值。
	const contact = $derived(contactProp ?? derivedContact);

	const peerName = $derived(contact?.name ?? '');
	// 在线状态实时化：优先 WS 推送（presence store），HTTP 快照 contact.online 仅作初始兜底。
	const onlineNow = $derived(presence.isOnline(peerUserId));
	// 最后一条非本人消息，供屏幕阅读器 aria-live 播报（只播文本，避免整表重读）
	const lastIncoming = $derived([...chat.messages].reverse().find((m) => !m.sent));

	// --- typing 接收：仅处理对端（payload.actor.user_id === peer）推送；true 时 3 秒超时自动重置。 ---
	let peerTyping = $state(false);
	let peerTypingTimer: ReturnType<typeof setTimeout> | null = null;

	$effect(() => {
		const peer = peerUserId;
		if (!peer) return;

		const onTyping = (e: Event) => {
			const payload = (e as CustomEvent<PrivateTypingPayload>).detail;
			if (!payload || payload.target_user_id !== peer) return;
			if (payload.actor?.user_id !== peer) return;
			if (payload.is_typing) {
				peerTyping = true;
				if (peerTypingTimer) clearTimeout(peerTypingTimer);
				peerTypingTimer = setTimeout(() => {
					peerTypingTimer = null;
					peerTyping = false;
				}, 3000);
			} else {
				peerTyping = false;
				if (peerTypingTimer) {
					clearTimeout(peerTypingTimer);
					peerTypingTimer = null;
				}
			}
		};

		window.addEventListener(WS_PRIVATE_TYPING_EVENT, onTyping as EventListener);
		return () => {
			window.removeEventListener(WS_PRIVATE_TYPING_EVENT, onTyping as EventListener);
			if (peerTypingTimer) {
				clearTimeout(peerTypingTimer);
				peerTypingTimer = null;
			}
			peerTyping = false;
		};
	});

	// 在线状态订阅：切换会话时自动重新订阅。
	$effect(() => {
		const peer = peerUserId;
		if (!peer) return;
		const unwatch = watchPresence([peer]);
		return () => {
			unwatch();
		};
	});
</script>

<div class="flex min-h-0 flex-col h-full bg-surface-primary">
	<p class="sr-only" aria-live="polite">{lastIncoming ? `新消息：${lastIncoming.text}` : ''}</p>
	<!-- Header -->
	<div class="shrink-0 flex items-center gap-3 border-b border-border-default bg-surface-primary px-4 {compact ? 'h-13' : 'h-15'}">
		{#if showBackButton}
			<button
				type="button"
				class="hover-btn scale-btn flex items-center justify-center rounded-full border border-border-default bg-surface-primary size-9 shrink-0 shadow-raised-sm focus-ring"
				onclick={onBack}
				aria-label="返回"
			>
				<ArrowLeft class="size-4 text-text-secondary"   strokeWidth={1.5} />
			</button>
		{/if}

		<!-- Avatar -->
		<Avatar
			src={contact?.avatar}
			name={peerName}
			size={compact ? 34 : 38}
			style="border:var(--cst-border-width-regular) solid var(--cst-border-default);box-shadow:var(--shadow-raised-sm);"
		>
			{#snippet badge()}
				{#if onlineNow}
					<OnlineDot size="sm" />
				{/if}
			{/snippet}
		</Avatar>

		<!-- Name + status -->
		<div class="flex-1 min-w-0">
			<p class="truncate text-text-primary {compact ? 'text-sm' : 'text-md'}">
				{peerName || '会话'}
			</p>
			<p class="{peerTyping ? 'text-accent-default' : onlineNow ? 'text-status-online' : 'text-text-secondary'} text-xs" >
				{#if peerTyping}
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

		<!-- Actions -->
		<div class="flex gap-1.5">
				{#each [{ icon: Phone, label: '语音通话' }, { icon: Video, label: '视频通话' }] as { icon: Icon, label }}
					<button
						type="button"
						class="hover-btn scale-btn flex items-center justify-center rounded-full border border-border-default bg-surface-primary shadow-raised-sm focus-ring disabled:opacity-muted {compact ? 'size-8' : 'size-9'}"
						aria-label={label}
						disabled
						title="暂未开放"
					>
						<Icon class="size-3.5 text-text-secondary"   strokeWidth={1.5} />
					</button>
				{/each}
				<button
					type="button"
					class="hover-btn scale-btn flex items-center justify-center rounded-full border border-border-default bg-surface-primary shadow-raised-sm focus-ring disabled:opacity-muted {compact ? 'size-8' : 'size-9'}"
					aria-label="更多操作"
					disabled
					title="暂未开放"
				>
				<MoreHorizontal class="size-3.5 text-text-secondary"   strokeWidth={1.5} />
			</button>
		</div>
	</div>

	<!-- Messages -->
	<div bind:this={chat.scrollEl} onscroll={chat.handleScroll} class="flex-1 overflow-y-auto no-scrollbar" style="overscroll-behavior:contain;">
		{#if chat.loading}
			<div class="flex h-full items-center justify-center" role="status" aria-label="加载中">
				<div class="spinner size-7"></div>
			</div>
		{:else if chat.loadError}
			<!-- 错误态统一 LoadError（与全站一致），不再各页手搓文案 + 按钮 -->
			<LoadError message="消息加载失败，请检查网络后重试" onRetry={() => void chat.reload()} class="h-full" />
		{:else if chat.messages.length === 0}
			<!-- 空会话：统一 EmptyState 空态（对齐 GroupChatView/全局空态契约） -->
			<EmptyState iconSize={56} class="h-full" title="还没有消息" description="打个招呼，开始这段对话吧">
				{#snippet icon()}
					<MessageCircle class="size-5.5 text-text-secondary" strokeWidth={1.5} />
				{/snippet}
			</EmptyState>
		{:else}
		<div class="animate-fade-in flex flex-col gap-2 px-4 pt-4 pb-3">
			{#each chat.dateGroups as group (group.label)}
				<ChatDatePill label={group.label} />
				{#each group.messages as msg (msg.id)}
					<MessageBubble
						text={msg.text}
						sent={msg.sent}
						time={msg.time}
						showCheck={msg.id === chat.lastSentId}
						messageId={msg.id}
						recalled={msg.recalled}
						editedAt={msg.editedAt}
						canRecall={msg.sent && inRecallWindow(msg.sentAt)}
						canEdit={msg.sent}
						onRecall={(id) => void chat.handleRecall(id)}
						onEdit={(id, content) => void chat.handleEdit(id, content)}
					/>
				{/each}
			{/each}
		</div>
		{/if}
	</div>

	<!-- Input bar -->
	{#if chat.sendError}
		<p class="px-4 pb-1.5 text-status-warning text-xs" >{chat.sendError}</p>
	{/if}
	<div class="shrink-0 flex items-center gap-2.5 px-4 py-3 bg-surface-primary border-t border-border-default" >
		<button
			type="button"
			class="hover-btn scale-btn flex items-center justify-center rounded-full border border-border-default bg-surface-primary size-9 shrink-0 shadow-raised-sm focus-ring disabled:opacity-muted"
			aria-label="表情"
			disabled
			title="暂未开放"
		>
			<Smile class="size-4 text-text-secondary"   strokeWidth={1.5} />
		</button>

		<div class="flex-1 flex items-center px-4 rounded-md input-surface h-9.5" >
			<input
				type="text"
				aria-label="消息输入框"
				bind:value={chat.inputVal}
				oninput={(e) => chat.handleInputChange(e)}
				onkeydown={(e) => {
					// IME 组合中的回车是选词确认，不能当发送（中文输入法致命）
					if (e.key === 'Enter' && !e.isComposing) void chat.handleSend();
				}}
				placeholder="发消息…"
				class="flex-1 bg-transparent text-text-primary text-sm"
			/>
		</div>

		<button
			type="button"
			class="hover-btn scale-btn flex items-center justify-center rounded-full focus-ring shrink-0 transition-all duration-standard {chat.inputVal.trim() ? 'bg-accent-default' : 'bg-surface-primary'} size-9"
			style="box-shadow:{chat.inputVal.trim() ? 'var(--shadow-glow-md)' : 'var(--shadow-raised-sm)'};border:var(--cst-border-width-hairline) solid var(--cst-border-default);"
			onclick={() => void chat.handleSend()}
			aria-label="发送"
		>
			<Send class="size-3.5" style="color:{chat.inputVal.trim() ? 'var(--cst-text-on-accent)' : 'var(--cst-text-secondary)'};transform:translateX(1px);" strokeWidth={2} />
		</button>
	</div>
</div>
