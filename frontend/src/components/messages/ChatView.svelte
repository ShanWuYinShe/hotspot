<script lang="ts">
	import { ArrowLeft, Phone, Video, Send, Smile, CheckCheck, MoreHorizontal } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { getPrivateMessages, sendPrivateMessage, markPrivateRead } from '@/api';
	import {
		WS_PRIVATE_MESSAGE_EVENT,
		WS_PRIVATE_MESSAGE_UPDATE_EVENT
	} from '@/lib/ws/client';
	import type { PrivateMessageNewPayload } from '@/lib/ws/types';

	// ChatView — 私聊窗口（spec: 以对端 user_id 寻址）。
	// 历史来自 private_messages/get_history；发送走 private_messages/send。
	// 消息方向由 sender.user_id 与当前用户比较得出。
	// 迁移自旧版 ChatView。

	interface ChatContact {
		name: string;
		avatar: string;
		online: boolean;
		lastSeen?: string;
	}

	interface ChatMessage {
		id: string;
		text: string;
		sent: boolean; // true = outgoing (from current user)
		time: string;
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

	function formatTime(iso?: string) {
		if (!iso) return '';
		const d = new Date(iso);
		if (isNaN(d.getTime())) return '';
		return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
	}

	// contactProp 仅作为初始联系人信息（此后可能由消息记录推导），有意只读取一次。
	function initialContact(): ChatContact | null {
		return contactProp ?? null;
	}
	let contact = $state<ChatContact | null>(initialContact());
	let messages = $state<ChatMessage[]>([]);
	let inputVal = $state('');
	let loading = $state(true);
	let sending = $state(false);
	let sendError = $state('');
	let scrollEl = $state<HTMLDivElement | null>(null);

	// spec: private_messages/send 返回 10200 时 data 携带 { reason: BlockReason }。
	const SEND_BLOCK_REASONS: Record<string, string> = {
		privacy: '对方隐私设置不允许发送消息',
		stranger_quota_exceeded: '今日陌生人消息额度已用完',
		account_suspended: '账号被限制'
	};

	$effect(() => {
		let cancelled = false;
		async function load() {
			loading = true;
			// Fetch history by target_user_id; derive direction from sender.
			const res = await getPrivateMessages({ target_user_id: peerUserId, pagination: { limit: 50 } }).catch(() => ({
				code: -1,
				data: { items: [] }
			}));
			if (cancelled) return;
			const items = (res.code === 0 && res.data ? (res.data as any).items : []) as any[];
			const mapped: ChatMessage[] = items.map((m: any) => ({
				id: String(m.message_id ?? ''),
				text: m.content ?? m.text ?? '',
				sent: m.sender?.user_id ? m.sender.user_id === (auth.user?.user_id ?? '__me__') : Boolean(m.sent),
				time: formatTime(m.sent_at ?? m.created_at)
			}));
			messages = mapped;
			loading = false;

			// spec: private_messages/mark_read (scope='session') 标记该会话全部已读;
			// 幂等语义,失败静默不打扰会话浏览。
			void markPrivateRead({ scope: 'session', target_user_id: peerUserId }).catch(() => {});

			// If no peer display info was passed, try to read it from the latest
			// incoming message's sender (or any message sender that isn't me).
			if (!contactProp) {
				const peer = items.map((m: any) => m.sender).find((s: any) => s && s.user_id !== (auth.user?.user_id ?? '__me__'));
				if (peer) {
					contact = {
						name: peer.nickname ?? peer.name ?? '',
						avatar: peer.avatar_url ?? peer.avatar ?? '',
						online: peer.presence_status === 'online'
					};
				}
			}
		}
		if (peerUserId) void load();
		else {
			messages = [];
			loading = false;
		}
		return () => {
			cancelled = true;
		};
	});

	// 滚动到底部
	$effect(() => {
		const el = scrollEl;
		if (el) el.scrollTop = el.scrollHeight;
	});
	$effect(() => {
		const el = scrollEl;
		if (el) el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
	});

	const peerName = $derived(contact?.name ?? '');

	const handleSend = async () => {
		const text = inputVal.trim();
		if (!text || sending || !peerUserId) return;
		inputVal = '';
		sendError = '';
		sending = true;
		// Optimistic append so the user sees their message instantly.
		const tempId = `temp-${Date.now()}`;
		messages = [...messages, { id: tempId, text, sent: true, time: formatTime(new Date().toISOString()) }];
		try {
			const res = await sendPrivateMessage({ target_user_id: peerUserId, type: 'text', content: text });
			if (res.code !== 0) {
				// Roll back the optimistic message on failure.
				messages = messages.filter((m) => m.id !== tempId);
				// spec: 发送被拒绝返回 10200 时 data = { reason: BlockReason }。
				if (res.code === 10200) {
					const reason = (res.data as any)?.reason;
					sendError = (reason && SEND_BLOCK_REASONS[reason]) || '消息发送失败';
				}
			}
		} catch {
			messages = messages.filter((m) => m.id !== tempId);
			sendError = '消息发送失败,请稍后重试';
		} finally {
			sending = false;
		}
	};

	const lastSentId = $derived(
		messages
			.filter((m) => m.sent)
			.map((m) => m.id)
			.pop()
	);

	// WS 实时接收：ws:private-message 追加对端新消息；ws:private-message-update 处理撤回/编辑。
	// 依赖 peerUserId：桌面分栏切换会话时自动重新绑定监听。
	$effect(() => {
		const peer = peerUserId;
		if (!peer) return;

		const onNewMessage = (e: Event) => {
			const payload = (e as CustomEvent<PrivateMessageNewPayload>).detail;
			if (!payload || payload.target_user_id !== peer) return;
			// 自己发送的消息跳过（乐观追加 + HTTP 回执已处理）；按 message_id 去重
			if (payload.sender?.user_id === (auth.user?.user_id ?? '__me__')) return;
			const id = String(payload.message_id ?? '');
			if (!id || messages.some((m) => m.id === id)) return;
			messages = [
				...messages,
				{ id, text: payload.content ?? '', sent: false, time: formatTime(payload.sent_at) }
			];
			// 会话有新消息即视为已读（与 load() 行为一致）
			void markPrivateRead({ scope: 'session', target_user_id: peer }).catch(() => {});
		};

		const onMessageUpdate = (e: Event) => {
			const detail = (e as CustomEvent<{ type: 'edited' | 'recalled' | 'read'; payload: PrivateMessageNewPayload }>).detail;
			if (!detail?.payload || detail.payload.target_user_id !== peer) return;
			const id = String(detail.payload.message_id ?? '');
			// 消息不在本地列表则忽略（保持空操作）
			if (detail.type === 'edited') {
				messages = messages.map((m) =>
					m.id === id ? { ...m, text: detail.payload.content ?? m.text } : m
				);
			} else if (detail.type === 'recalled') {
				messages = messages.map((m) => (m.id === id ? { ...m, text: '[消息已撤回]' } : m));
			}
			// 'read'：本地列表无已读状态字段，忽略
		};

		window.addEventListener(WS_PRIVATE_MESSAGE_EVENT, onNewMessage as EventListener);
		window.addEventListener(WS_PRIVATE_MESSAGE_UPDATE_EVENT, onMessageUpdate as EventListener);
		return () => {
			window.removeEventListener(WS_PRIVATE_MESSAGE_EVENT, onNewMessage as EventListener);
			window.removeEventListener(WS_PRIVATE_MESSAGE_UPDATE_EVENT, onMessageUpdate as EventListener);
		};
	});
</script>

<div class="flex min-h-0 flex-col h-full bg-[var(--neu-bg)]">
	<!-- Header -->
	<div class="flex-shrink-0 flex items-center gap-3 px-4 bg-[var(--neu-bg)]" style="height:{compact ? 52 : 60}px;border-bottom:1px solid var(--neu-border);">
		{#if showBackButton}
			<button
				type="button"
				class="neu-hover-btn neu-scale-btn flex items-center justify-center rounded-full outline-none flex-shrink-0 bg-[var(--neu-bg)]"
				style="width:36px;height:36px;box-shadow:4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light);border:1px solid var(--neu-border);"
				onclick={onBack}
			>
				<ArrowLeft style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</button>
		{/if}

		<!-- Avatar -->
		<Avatar
			src={contact?.avatar}
			name={peerName}
			size={compact ? 34 : 38}
			style="border:1.5px solid var(--neu-border);box-shadow:3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light);"
		>
			{#snippet badge()}
				{#if contact?.online}
					<span class="absolute bottom-0 right-0 rounded-full bg-[var(--semantic-online)]" style="width:9px;height:9px;border:2px solid var(--neu-bg);"></span>
				{/if}
			{/snippet}
		</Avatar>

		<!-- Name + status -->
		<div class="flex-1 min-w-0">
			<p class="truncate text-[var(--neu-text)]" style="font-size:{compact ? 13 : 14}px;">
				{peerName || '会话'}
			</p>
			<p class="{contact?.online ? 'text-[var(--semantic-online)]' : 'text-[var(--neu-text-muted)]'}" style="font-size:11px;">
				{contact ? (contact.online ? '在线' : contact.lastSeen ? `活跃于${contact.lastSeen}` : '离线') : ''}
			</p>
		</div>

		<!-- Actions -->
		<div class="flex gap-1.5">
			{#each [Phone, Video] as Icon, i}
				<button
					type="button"
					class="neu-hover-btn neu-scale-btn flex items-center justify-center rounded-full outline-none bg-[var(--neu-bg)]"
					style="width:{compact ? 32 : 36}px;height:{compact ? 32 : 36}px;box-shadow:4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light);border:1px solid var(--neu-border);"
				>
					<Icon style="width:14px;height:14px;color:var(--neu-text-muted);" strokeWidth={1.5} />
				</button>
			{/each}
			<button
				type="button"
				class="neu-hover-btn neu-scale-btn flex items-center justify-center rounded-full outline-none bg-[var(--neu-bg)]"
				style="width:{compact ? 32 : 36}px;height:{compact ? 32 : 36}px;box-shadow:4px 4px 9px var(--neu-shadow-dark), -4px -4px 9px var(--neu-shadow-light);border:1px solid var(--neu-border);"
			>
				<MoreHorizontal style="width:14px;height:14px;color:var(--neu-text-muted);" strokeWidth={1.5} />
			</button>
		</div>
	</div>

	<!-- Messages -->
	<div bind:this={scrollEl} class="flex-1 overflow-y-auto no-scrollbar" style="overscroll-behavior:contain;">
		<div class="animate-fade-in flex flex-col gap-2 px-4 pt-4 pb-3">
			<!-- Date pill -->
			<div class="flex justify-center mb-2">
				<span class="px-3 py-1 rounded-full bg-[var(--neu-bg)] text-[var(--neu-text-muted)]" style="font-size:10px;border:1px solid var(--neu-border);letter-spacing:0.05em;opacity:0.8;">
					今天
				</span>
			</div>

			{#each messages as msg (msg.id)}
				<div class="animate-fade-in flex {msg.sent ? 'justify-end' : 'justify-start'}">
					<div
						class="{msg.sent ? 'bg-[var(--neu-accent-primary)]' : 'bg-[var(--neu-bg)]'}"
						style="max-width:76%;padding:10px 14px;border-radius:{msg.sent ? '18px 4px 18px 18px' : '4px 18px 18px 18px'};{msg.sent
							? 'box-shadow:0 4px 14px var(--neu-glow);'
							: 'box-shadow:4px 4px 10px var(--neu-shadow-dark), -4px -4px 10px var(--neu-shadow-light);border:1px solid var(--neu-border);'}"
					>
						<p class="{msg.sent ? 'text-white' : 'text-[var(--neu-text)]'}" style="font-size:13px;line-height:1.55;">
							{msg.text}
						</p>
						<p
							class="{msg.sent ? 'text-[rgba(255,255,255,0.65)]' : 'text-[var(--neu-text-muted)]'}"
							style="font-size:10px;margin-top:4px;display:flex;align-items:center;gap:3px;justify-content:{msg.sent ? 'flex-end' : 'flex-start'};"
						>
							{msg.time}
							{#if msg.sent && msg.id === lastSentId}
								<CheckCheck style="width:12px;height:12px;opacity:0.8;" strokeWidth={2} />
							{/if}
						</p>
					</div>
				</div>
			{/each}
		</div>
	</div>

	<!-- Input bar -->
	{#if sendError}
		<p class="px-4 pb-1.5 text-[var(--semantic-warning)]" style="font-size:11px;">{sendError}</p>
	{/if}
	<div class="flex-shrink-0 flex items-center gap-2.5 px-4 py-3 bg-[var(--neu-bg)]" style="border-top:1px solid var(--neu-border);">
		<button
			type="button"
			class="neu-hover-btn neu-scale-btn flex items-center justify-center rounded-full outline-none flex-shrink-0 bg-[var(--neu-bg)]"
			style="width:36px;height:36px;box-shadow:4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light);border:1px solid var(--neu-border);"
		>
			<Smile style="width:15px;height:15px;color:var(--neu-text-muted);" strokeWidth={1.5} />
		</button>

		<div class="flex-1 flex items-center px-4 rounded-[18px] neu-input-surface" style="height:38px;">
			<input
				type="text"
				bind:value={inputVal}
				oninput={() => (sendError = '')}
				onkeydown={(e) => {
					if (e.key === 'Enter') void handleSend();
				}}
				placeholder="发消息…"
				class="flex-1 bg-transparent outline-none text-[var(--neu-text)]"
				style="font-size:13px;"
			/>
		</div>

		<button
			type="button"
			class="neu-hover-btn neu-scale-btn flex items-center justify-center rounded-full outline-none flex-shrink-0 transition-all duration-200 {inputVal.trim() ? 'bg-[var(--neu-accent-primary)]' : 'bg-[var(--neu-bg)]'}"
			style="width:36px;height:36px;box-shadow:{inputVal.trim() ? '0 4px 14px var(--neu-glow)' : '4px 4px 8px var(--neu-shadow-dark), -4px -4px 8px var(--neu-shadow-light)'};border:1px solid var(--neu-border);"
			onclick={() => void handleSend()}
		>
			<Send style="width:14px;height:14px;color:{inputVal.trim() ? 'white' : 'var(--neu-text-muted)'};transform:translateX(1px);" strokeWidth={2} />
		</button>
	</div>
</div>
