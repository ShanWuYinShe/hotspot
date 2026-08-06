<script lang="ts">
	import { ArrowLeft, Send } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import MessageBubble from '@/components/messages/MessageBubble.svelte';
	import ChatDatePill from '@/components/messages/ChatDatePill.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { getGroupDetail, getGroupMessages, sendGroupMessage, markGroupRead } from '@/api';
	import { WS_GROUP_MESSAGE_EVENT, WS_GROUP_MESSAGE_UPDATE_EVENT, wsClient } from '@/lib/ws/client';
	import type { UserInfo } from '@/lib/ws/types';
	import { useChat, formatTime, type ChatMessage } from '@/lib/chat.svelte';

	// 服务端推送版输入状态（spec GroupTypingPayload，types.ts 未单独导出）。
	interface GroupTypingPayload {
		group_id: string;
		actor: UserInfo;
		is_typing: boolean;
	}

	// GroupChatView — 群聊窗口（spec: 以 group_id 寻址）。
	// 历史来自 group_messages/get_history;发送走 group_messages/send。
	// 消息方向由 sender.user_id 与当前用户比较得出;共享状态机由 useChat 工厂承载。

	let {
		groupId,
		showBackButton = false,
		onBack,
		compact = false
	}: {
		/** The group's id (spec: group_id). */
		groupId: string;
		showBackButton?: boolean;
		onBack?: () => void;
		compact?: boolean; // for desktop split-pane (smaller header)
	} = $props();

	let groupName = $state('');
	let groupAvatar = $state('');

	const meId = $derived(auth.user?.user_id ?? '__me__');

	// 群信息：失败静默，标题回退"群聊"。
	$effect(() => {
		const gid = groupId;
		if (!gid) return;
		let cancelled = false;
		getGroupDetail(gid)
			.then((res) => {
				if (cancelled || res.code !== 0 || !res.data) return;
				const d = res.data;
				groupName = d.group.name;
				groupAvatar = d.group.avatar_url ?? '';
			})
			.catch(() => {});
		return () => {
			cancelled = true;
		};
	});

	const chat = useChat({
		targetId: () => groupId,
		loadHistory: async (cursor) => {
			const res = await getGroupMessages({ group_id: groupId, pagination: { limit: 50, cursor } }).catch(
				() => null
			);
			if (!res || res.code !== 0 || !res.data) return null;
			const items = res.data.items;
			const mapped: ChatMessage[] = items.map((m) => ({
				id: m.message_id,
				text: m.recalled_at ? '[消息已撤回]' : (m.content ?? ''),
				recalled: Boolean(m.recalled_at),
				sent: m.sender.user_id === meId,
				time: formatTime(m.sent_at)
			}));
			return { messages: mapped, next_cursor: res.data.next_cursor ?? null };
		},
		markRead: () => {
			// spec: group_messages/mark_read (scope='group') 标记该群全部已读;幂等语义,失败静默。
			void markGroupRead({ scope: 'group', group_id: groupId }).catch(() => {});
		},
		sendMessage: async (text) => {
			const res = await sendGroupMessage({ group_id: groupId, type: 'text', content: text });
			if (res.code === 0) return { code: 0, message_id: res.data?.message_id };
			return { code: res.code, error: '消息发送失败' };
		},
		sendTypingSignal: (isTyping) => {
			// 未连接静默丢弃。
			if (!groupId || !wsClient.isConnected) return;
			wsClient.sendGroupTyping(groupId, isTyping);
		},
		wsNewMessageEvent: WS_GROUP_MESSAGE_EVENT,
		wsMessageUpdateEvent: WS_GROUP_MESSAGE_UPDATE_EVENT,
		wsNewMessage: (e) => {
			const payload = (e as CustomEvent<{
				message_id: string;
				group_id: string;
				sender: UserInfo;
				type: string;
				content?: string;
				sent_at: string;
			}>).detail;
			if (!payload || payload.group_id !== groupId) return null;
			const id = payload.message_id ?? '';
			if (!id) return null;
			// 按 message_id 去重：乐观追加已替换为真实 id，自己发送的推送自然跳过。
			const sent = payload.sender?.user_id === meId;
			// 收到非自己消息即视为已读（与加载行为一致）
			if (!sent) void markGroupRead({ scope: 'group', group_id: groupId }).catch(() => {});
			return { id, text: payload.content ?? '', sent, time: formatTime(payload.sent_at) };
		},
		wsMessageUpdate: (e) => {
			const detail = (e as CustomEvent<{
				type: 'edited' | 'recalled' | 'read_count';
				payload: { message_id: string; group_id: string; content?: string };
			}>).detail;
			if (!detail?.payload || detail.payload.group_id !== groupId) return null;
			const id = detail.payload.message_id ?? '';
			if (detail.type === 'edited') return { type: 'edited', id, content: detail.payload.content };
			if (detail.type === 'recalled') return { type: 'recalled', id };
			// 'read_count'：本地无已读数展示字段，忽略
			return null;
		}
	});

	// --- typing 接收：仅处理本群（payload.group_id === gid）推送；true 时 3 秒超时自动重置。 ---
	let groupTyping = $state(false);
	let typingName = $state('');
	let groupTypingTimer: ReturnType<typeof setTimeout> | null = null;

	$effect(() => {
		const gid = groupId;
		if (!gid) return;

		const off = wsClient.on('GroupTyping', (payload) => {
			// ws/types 中 GroupTyping 为宽松 Record，按 spec 形状收窄读取。
			const p = payload as unknown as GroupTypingPayload;
			if (!p || p.group_id !== gid) return;
			if (p.is_typing) {
				groupTyping = true;
				typingName = p.actor?.nickname ?? '';
				if (groupTypingTimer) clearTimeout(groupTypingTimer);
				groupTypingTimer = setTimeout(() => {
					groupTypingTimer = null;
					groupTyping = false;
					typingName = '';
				}, 3000);
			} else {
				groupTyping = false;
				typingName = '';
				if (groupTypingTimer) {
					clearTimeout(groupTypingTimer);
					groupTypingTimer = null;
				}
			}
		});
		return () => {
			off();
			if (groupTypingTimer) {
				clearTimeout(groupTypingTimer);
				groupTypingTimer = null;
			}
			groupTyping = false;
			typingName = '';
		};
	});
</script>

<div class="flex min-h-0 flex-col h-full bg-surface-primary">
	<!-- Header -->
	<div class="shrink-0 flex items-center gap-3 px-4 bg-surface-primary" style="height:{compact ? 52 : 60}px;border-bottom:1px solid var(--cst-border-default);">
		{#if showBackButton}
			<button
				type="button"
				class="hover-btn scale-btn flex items-center justify-center rounded-full outline-none shrink-0 bg-surface-primary"
				style="width:36px;height:36px;box-shadow:var(--shadow-neu-md);border:1px solid var(--cst-border-default);"
				onclick={onBack}
			>
				<ArrowLeft class="size-3.75"  style="color:var(--cst-text-secondary)" strokeWidth={1.5} />
			</button>
		{/if}

		<!-- Avatar -->
		<Avatar
			src={groupAvatar}
			name={groupName}
			size={compact ? 34 : 38}
			shape="rounded"
			style="border:1.5px solid var(--cst-border-default);box-shadow:var(--shadow-neu-xs);"
		/>

		<!-- Name + status -->
		<div class="flex-1 min-w-0">
			<p class="truncate text-text-primary" style="font-size:{compact ? 13 : 14}px;">
				{groupName || '群聊'}
			</p>
			<p class="{groupTyping ? 'text-accent-default' : 'text-text-secondary'} text-xs" >
				{#if groupTyping}
					{typingName || '有人'}正在输入…
				{:else}
					群聊
				{/if}
			</p>
		</div>
	</div>

	<!-- Messages -->
	<div bind:this={chat.scrollEl} onscroll={chat.handleScroll} class="flex-1 overflow-y-auto no-scrollbar" style="overscroll-behavior:contain;">
		{#if chat.loading}
			<div class="flex h-full items-center justify-center">
				<p class="text-text-secondary text-xs" >加载中…</p>
			</div>
		{:else if chat.messages.length === 0}
			<div class="flex h-full items-center justify-center">
				<p class="text-text-secondary text-xs" >还没有消息，来打个招呼吧</p>
			</div>
		{:else}
			<div class="animate-fade-in flex flex-col gap-2 px-4 pt-4 pb-3">
				<ChatDatePill />

				{#each chat.messages as msg (msg.id)}
					<MessageBubble text={msg.text} sent={msg.sent} time={msg.time} showCheck={msg.id === chat.lastSentId} />
				{/each}
			</div>
		{/if}
	</div>

	<!-- Input bar -->
	{#if chat.sendError}
		<p class="px-4 pb-1.5 text-status-warning text-xs" >{chat.sendError}</p>
	{/if}
	<div class="shrink-0 flex items-center gap-2.5 px-4 py-3 bg-surface-primary border-t border-border-default" >
		<div class="flex-1 flex items-center px-4 rounded-relaxed input-surface h-9.5" >
			<input
				type="text"
				bind:value={chat.inputVal}
				oninput={(e) => chat.handleInputChange(e)}
				onkeydown={(e) => {
					if (e.key === 'Enter') void chat.handleSend();
				}}
				placeholder="发消息…"
				class="flex-1 bg-transparent outline-none text-text-primary"
				style="font-size:13px;"
			/>
		</div>

		<button
			type="button"
			class="hover-btn scale-btn flex items-center justify-center rounded-full outline-none shrink-0 transition-all duration-200 {chat.inputVal.trim() ? 'bg-accent-default' : 'bg-surface-primary'}"
			style="width:36px;height:36px;box-shadow:{chat.inputVal.trim() ? '0 4px 14px var(--cst-glow-accent)' : 'var(--shadow-neu-md)'};border:1px solid var(--cst-border-default);"
			onclick={() => void chat.handleSend()}
		>
			<Send style="width:14px;height:14px;color:{chat.inputVal.trim() ? 'var(--cst-text-on-accent)' : 'var(--cst-text-secondary)'};transform:translateX(1px);" strokeWidth={2} />
		</button>
	</div>
</div>
