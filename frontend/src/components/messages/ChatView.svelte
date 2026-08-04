<script lang="ts">
	import { ArrowLeft, Phone, Video, Send, Smile, CheckCheck, MoreHorizontal, MessageCircle } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { getPrivateMessages, sendPrivateMessage, markPrivateRead } from '@/api';
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
	// 历史来自 private_messages/get_history；发送走 private_messages/send。
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

	// spec: private_messages/send 返回 10200 时 data 携带 { reason: BlockReason }。
	const SEND_BLOCK_REASONS: Record<string, string> = {
		privacy: '对方隐私设置不允许发送消息',
		stranger_quota_exceeded: '今日陌生人消息额度已用完',
		account_suspended: '账号被限制'
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
				text: m.content ?? '',
				sent: m.sender.user_id === meId,
				time: formatTime(m.sent_at)
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
			// spec: 10200 时 data = { reason: BlockReason }。
			if (res.code === 10200) {
				const reason = (res.data as { reason?: string } | undefined)?.reason;
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
			return { id, text: payload.content ?? '', sent: false, time: formatTime(payload.sent_at) };
		},
		wsMessageUpdate: (e) => {
			const detail = (e as CustomEvent<{ type: 'edited' | 'recalled' | 'read'; payload: PrivateMessageNewPayload }>).detail;
			if (!detail?.payload || detail.payload.target_user_id !== peerUserId) return null;
			const id = detail.payload.message_id ?? '';
			if (detail.type === 'edited') return { type: 'edited', id, content: detail.payload.content };
			if (detail.type === 'recalled') return { type: 'recalled', id };
			// 'read'：本地列表无已读状态字段，忽略
			return null;
		}
	});

	// 展示用联系人：调用方传入优先，否则用历史推导值。
	const contact = $derived(contactProp ?? derivedContact);

	const peerName = $derived(contact?.name ?? '');
	// 在线状态实时化：优先 WS 推送（presence store），HTTP 快照 contact.online 仅作初始兜底。
	const onlineNow = $derived(presence.isOnline(peerUserId));

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

<div class="flex min-h-0 flex-col h-full bg-[var(--surface-primary)]">
	<!-- Header -->
	<div class="flex-shrink-0 flex items-center gap-3 px-4 bg-[var(--surface-primary)]" style="height:{compact ? 52 : 60}px;border-bottom:1px solid var(--border-default);">
		{#if showBackButton}
			<button
				type="button"
				class="hover-btn scale-btn tap-88 flex items-center justify-center rounded-full outline-none flex-shrink-0 bg-[var(--surface-primary)]"
				style="width:36px;height:36px;box-shadow:4px 4px 9px var(--elevation-shadow-dark), -4px -4px 9px var(--elevation-shadow-light);border:1px solid var(--border-default);"
				onclick={onBack}
			>
				<ArrowLeft style="width:15px;height:15px;color:var(--text-secondary);" strokeWidth={1.5} />
			</button>
		{/if}

		<!-- Avatar -->
		<Avatar
			src={contact?.avatar}
			name={peerName}
			size={compact ? 34 : 38}
			style="border:1.5px solid var(--border-default);box-shadow:3px 3px 7px var(--elevation-shadow-dark), -3px -3px 7px var(--elevation-shadow-light);"
		>
			{#snippet badge()}
				{#if onlineNow}
					<span class="absolute bottom-0 right-0 rounded-full bg-[var(--status-online)]" style="width:9px;height:9px;border:2px solid var(--surface-primary);"></span>
				{/if}
			{/snippet}
		</Avatar>

		<!-- Name + status -->
		<div class="flex-1 min-w-0">
			<p class="truncate text-[var(--text-primary)]" style="font-size:{compact ? 13 : 14}px;">
				{peerName || '会话'}
			</p>
			<p class="{peerTyping ? 'text-[var(--accent-default)]' : onlineNow ? 'text-[var(--status-online)]' : 'text-[var(--text-secondary)]'}" style="font-size:11px;">
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
			{#each [Phone, Video] as Icon, i}
				<button
					type="button"
					class="hover-btn scale-btn tap-88 flex items-center justify-center rounded-full outline-none bg-[var(--surface-primary)]"
					style="width:{compact ? 32 : 36}px;height:{compact ? 32 : 36}px;box-shadow:4px 4px 9px var(--elevation-shadow-dark), -4px -4px 9px var(--elevation-shadow-light);border:1px solid var(--border-default);"
				>
					<Icon style="width:14px;height:14px;color:var(--text-secondary);" strokeWidth={1.5} />
				</button>
			{/each}
			<button
				type="button"
				class="hover-btn scale-btn tap-88 flex items-center justify-center rounded-full outline-none bg-[var(--surface-primary)]"
				style="width:{compact ? 32 : 36}px;height:{compact ? 32 : 36}px;box-shadow:4px 4px 9px var(--elevation-shadow-dark), -4px -4px 9px var(--elevation-shadow-light);border:1px solid var(--border-default);"
			>
				<MoreHorizontal style="width:14px;height:14px;color:var(--text-secondary);" strokeWidth={1.5} />
			</button>
		</div>
	</div>

	<!-- Messages -->
	<div bind:this={chat.scrollEl} onscroll={chat.handleScroll} class="flex-1 overflow-y-auto no-scrollbar" style="overscroll-behavior:contain;">
		{#if !chat.loading && chat.messages.length === 0}
			<!-- 空会话：还没有消息时的引导空态 -->
			<div class="flex h-full flex-col items-center justify-center gap-2 px-6" style="min-height:240px;">
				<div
					class="flex items-center justify-center rounded-full"
					style="width:56px;height:56px;border:1px solid var(--border-default);box-shadow:6px 6px 14px var(--elevation-shadow-dark), -6px -6px 14px var(--elevation-shadow-light);"
				>
					<MessageCircle style="width:22px;height:22px;color:var(--text-secondary);" strokeWidth={1.5} />
				</div>
				<p class="text-[var(--text-primary)]" style="font-size:14px;font-weight:500;">还没有消息</p>
				<p class="text-center text-[var(--text-secondary)]" style="font-size:12px;max-width:220px;">打个招呼，开始这段对话吧</p>
			</div>
		{:else}
		<div class="animate-fade-in flex flex-col gap-2 px-4 pt-4 pb-3">
			<!-- Date pill -->
			<div class="flex justify-center mb-2">
				<span class="px-3 py-1 rounded-full bg-[var(--surface-primary)] text-[var(--text-secondary)]" style="font-size:10px;border:1px solid var(--border-default);letter-spacing:0.05em;opacity:0.8;">
					今天
				</span>
			</div>

			{#each chat.messages as msg (msg.id)}
				<div class="animate-fade-in flex {msg.sent ? 'justify-end' : 'justify-start'}">
					<div
						class="{msg.sent ? 'bg-[var(--accent-default)]' : 'bg-[var(--surface-primary)]'}"
						style="max-width:76%;padding:10px 14px;border-radius:{msg.sent ? '18px 4px 18px 18px' : '4px 18px 18px 18px'};{msg.sent
							? 'box-shadow:0 4px 14px var(--glow-accent);'
							: 'box-shadow:4px 4px 10px var(--elevation-shadow-dark), -4px -4px 10px var(--elevation-shadow-light);border:1px solid var(--border-default);'}"
					>
						<p class="{msg.sent ? 'text-white' : 'text-[var(--text-primary)]'}" style="font-size:13px;line-height:1.55;">
							{msg.text}
						</p>
						<p
							class="{msg.sent ? 'text-[var(--content-inverse-muted)]' : 'text-[var(--text-secondary)]'}"
							style="font-size:10px;margin-top:4px;display:flex;align-items:center;gap:3px;justify-content:{msg.sent ? 'flex-end' : 'flex-start'};"
						>
							{msg.time}
							{#if msg.sent && msg.id === chat.lastSentId}
								<CheckCheck style="width:12px;height:12px;opacity:0.8;" strokeWidth={2} />
							{/if}
						</p>
					</div>
				</div>
			{/each}
		</div>
		{/if}
	</div>

	<!-- Input bar -->
	{#if chat.sendError}
		<p class="px-4 pb-1.5 text-[var(--status-warning)]" style="font-size:11px;">{chat.sendError}</p>
	{/if}
	<div class="flex-shrink-0 flex items-center gap-2.5 px-4 py-3 bg-[var(--surface-primary)]" style="border-top:1px solid var(--border-default);">
		<button
			type="button"
			class="hover-btn scale-btn tap-88 flex items-center justify-center rounded-full outline-none flex-shrink-0 bg-[var(--surface-primary)]"
			style="width:36px;height:36px;box-shadow:4px 4px 8px var(--elevation-shadow-dark), -4px -4px 8px var(--elevation-shadow-light);border:1px solid var(--border-default);"
		>
			<Smile style="width:15px;height:15px;color:var(--text-secondary);" strokeWidth={1.5} />
		</button>

		<div class="flex-1 flex items-center px-4 rounded-[18px] input-surface" style="height:38px;">
			<input
				type="text"
				bind:value={chat.inputVal}
				oninput={(e) => chat.handleInputChange(e)}
				onkeydown={(e) => {
					if (e.key === 'Enter') void chat.handleSend();
				}}
				placeholder="发消息…"
				class="flex-1 bg-transparent outline-none text-[var(--text-primary)]"
				style="font-size:13px;"
			/>
		</div>

		<button
			type="button"
			class="hover-btn scale-btn tap-88 flex items-center justify-center rounded-full outline-none flex-shrink-0 transition-all duration-200 {chat.inputVal.trim() ? 'bg-[var(--accent-default)]' : 'bg-[var(--surface-primary)]'}"
			style="width:36px;height:36px;box-shadow:{chat.inputVal.trim() ? '0 4px 14px var(--glow-accent)' : '4px 4px 8px var(--elevation-shadow-dark), -4px -4px 8px var(--elevation-shadow-light)'};border:1px solid var(--border-default);"
			onclick={() => void chat.handleSend()}
		>
			<Send style="width:14px;height:14px;color:{chat.inputVal.trim() ? 'var(--text-on-accent)' : 'var(--text-secondary)'};transform:translateX(1px);" strokeWidth={2} />
		</button>
	</div>
</div>
