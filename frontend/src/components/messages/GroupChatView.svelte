<script lang="ts">
	import { ArrowLeft, Send, CheckCheck } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { getGroupDetail, getGroupMessages, sendGroupMessage, markGroupRead } from '@/api';
	import {
		WS_GROUP_MESSAGE_EVENT,
		WS_GROUP_MESSAGE_UPDATE_EVENT,
		wsClient
	} from '@/lib/ws/client';
	import type { UserInfo } from '@/lib/ws/types';

	// 服务端推送版输入状态（spec GroupTypingPayload，types.ts 未单独导出）。
	interface GroupTypingPayload {
		group_id: string;
		actor: UserInfo;
		is_typing: boolean;
	}

	// GroupChatView — 群聊窗口（spec: 以 group_id 寻址）。
	// 历史来自 group_messages/get_history;发送走 group_messages/send。
	// 消息方向由 sender.user_id 与当前用户比较得出;结构对齐 ChatView(私聊)。

	interface GroupMessage {
		id: string;
		text: string;
		sent: boolean; // true = outgoing (from current user)
		time: string;
		recalled?: boolean;
	}

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

	function formatTime(iso?: string) {
		if (!iso) return '';
		const d = new Date(iso);
		if (isNaN(d.getTime())) return '';
		return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
	}

	let groupName = $state('');
	let groupAvatar = $state('');
	let messages = $state<GroupMessage[]>([]);
	let inputVal = $state('');
	let loading = $state(true);
	let sending = $state(false);
	let sendError = $state('');
	let scrollEl = $state<HTMLDivElement | null>(null);
	// --- 历史分页(契约:游标分页,next_cursor 缺失即无更多) ---
	let nextCursor: string | null = null;
	let loadingMore = false;

	// --- typing 发送状态（契约：客户端不超过 3 秒发一次；服务端权威节流）---
	let lastTypingSentAt = 0;
	let typingStopTimer: ReturnType<typeof setTimeout> | null = null;
	// --- typing 接收状态（群内任一成员正在输入）---
	let groupTyping = $state(false);
	let typingName = $state('');
	let groupTypingTimer: ReturnType<typeof setTimeout> | null = null;

	// 加载群信息 + 历史;切换 groupId(桌面分栏)自动重新加载。
	$effect(() => {
		const gid = groupId;
		if (!gid) {
			messages = [];
			loading = false;
			return;
		}
		let cancelled = false;
		async function load() {
			loading = true;
			// 群信息：失败静默，标题回退"群聊"。
			getGroupDetail(gid)
				.then((res) => {
					if (cancelled || res.code !== 0 || !res.data) return;
					const d = res.data as any;
					groupName = d.name ?? d.group?.name ?? '';
					groupAvatar = d.avatar_url ?? d.group?.avatar_url ?? '';
				})
				.catch(() => {});
			const res = await getGroupMessages({ group_id: gid, pagination: { limit: 50 } }).catch(() => ({
				code: -1,
				data: { items: [] }
			}));
			if (cancelled) return;
			const items = (res.code === 0 && res.data ? (res.data as any).items : []) as any[];
			const meId = auth.user?.user_id ?? '__me__';
			messages = items.map((m: any) => ({
				id: String(m.message_id ?? ''),
				text: m.recalled_at ? '[消息已撤回]' : (m.content ?? m.text ?? ''),
				recalled: Boolean(m.recalled_at),
				sent: m.sender?.user_id ? m.sender.user_id === meId : Boolean(m.sent),
				time: formatTime(m.sent_at ?? m.created_at)
			}));
			loading = false;
			nextCursor = (res.data as any)?.next_cursor ?? null;
			loadingMore = false;
			// spec: group_messages/mark_read (scope='group') 标记该群全部已读;幂等语义,失败静默。
			void markGroupRead({ scope: 'group', group_id: gid }).catch(() => {});
		}
		void load();
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

	/** 发送输入状态（未连接静默丢弃）。 */
	function sendTyping(isTyping: boolean) {
		if (!groupId || !wsClient.isConnected) return;
		wsClient.sendGroupTyping(groupId, isTyping);
	}

	/** 立即结束输入状态（清空输入框 / 发送成功后调用）。 */
	function cancelTyping() {
		if (typingStopTimer) {
			clearTimeout(typingStopTimer);
			typingStopTimer = null;
		}
		lastTypingSentAt = 0;
		sendTyping(false);
	}

	/** 滚动到顶加载更早的历史(游标分页;契约:服务端按 sent_at 倒序,next_cursor 指向更早批次)。 */
	async function loadMore() {
		if (loadingMore || !nextCursor) return;
		loadingMore = true;
		const gid = groupId;
		const el = scrollEl;
		const prevHeight = el?.scrollHeight ?? 0;
		const prevTop = el?.scrollTop ?? 0;
		const res = await getGroupMessages({ group_id: gid, pagination: { limit: 50, cursor: nextCursor } }).catch(
			() => ({ code: -1, data: null })
		);
		// 群已切换则丢弃结果(竞态保护)
		if (groupId !== gid) return;
		loadingMore = false;
		if (!res || res.code !== 0 || !res.data) return;
		const items = (res.data as any).items ?? [];
		const meId = auth.user?.user_id ?? '__me__';
		const mapped: GroupMessage[] = items.map((m: any) => ({
			id: String(m.message_id ?? ''),
			text: m.recalled_at ? '[消息已撤回]' : (m.content ?? m.text ?? ''),
			recalled: Boolean(m.recalled_at),
			sent: m.sender?.user_id ? m.sender.user_id === meId : Boolean(m.sent),
			time: formatTime(m.sent_at ?? m.created_at)
		}));
		messages = [...mapped, ...messages];
		nextCursor = (res.data as any)?.next_cursor ?? null;
		// 新内容插在顶部:把可视位置下移新内容高度
		requestAnimationFrame(() => {
			const el2 = scrollEl;
			if (el2) el2.scrollTop = el2.scrollHeight - prevHeight + prevTop;
		});
	}

	function handleScroll() {
		const el = scrollEl;
		if (!el || loadingMore || !nextCursor) return;
		if (el.scrollTop < 40) void loadMore();
	}

	/** 输入内容变化：非空且距上次发送 ≥3 秒发 true；清空立即发 false；停止输入 3 秒后发 false。 */
	function handleInputChange(e: Event) {
		inputVal = (e.target as HTMLInputElement).value;
		sendError = '';
		if (inputVal.trim()) {
			const now = Date.now();
			if (now - lastTypingSentAt >= 3000) {
				lastTypingSentAt = now;
				sendTyping(true);
			}
			if (typingStopTimer) clearTimeout(typingStopTimer);
			typingStopTimer = setTimeout(() => {
				typingStopTimer = null;
				lastTypingSentAt = 0;
				sendTyping(false);
			}, 3000);
		} else {
			cancelTyping();
		}
	}

	const handleSend = async () => {
		const text = inputVal.trim();
		if (!text || sending || !groupId) return;
		inputVal = '';
		sendError = '';
		sending = true;
		// Optimistic append so the user sees their message instantly.
		const tempId = `temp-${Date.now()}`;
		messages = [...messages, { id: tempId, text, sent: true, time: formatTime(new Date().toISOString()) }];
		try {
			const res = await sendGroupMessage({ group_id: groupId, type: 'text', content: text });
			if (res.code !== 0) {
				// Roll back the optimistic message on failure.
				messages = messages.filter((m) => m.id !== tempId);
				sendError = '消息发送失败';
			} else {
				// 成功：用服务端 message_id 替换临时 id，后续 WS 推送按 id 去重跳过。
				const realId = (res.data as any)?.message_id;
				if (realId) {
					messages = messages.map((m) => (m.id === tempId ? { ...m, id: String(realId) } : m));
				}
			}
		} catch {
			messages = messages.filter((m) => m.id !== tempId);
			sendError = '消息发送失败,请稍后重试';
		} finally {
			sending = false;
			// 发送完成（无论成败）立即结束输入状态，避免对端残留"正在输入"
			cancelTyping();
		}
	};

	const lastSentId = $derived(
		messages
			.filter((m) => m.sent)
			.map((m) => m.id)
			.pop()
	);

	// WS 实时接收：ws:group-message 追加本群新消息；ws:group-message-update 处理编辑/撤回。
	// 依赖 groupId：桌面分栏切换会话时自动重新绑定监听。
	$effect(() => {
		const gid = groupId;
		if (!gid) return;

		const onNewMessage = (e: Event) => {
			const payload = (e as CustomEvent<{
				message_id: string;
				group_id: string;
				sender: UserInfo;
				type: string;
				content?: string;
				sent_at: string;
			}>).detail;
			if (!payload || payload.group_id !== gid) return;
			const id = String(payload.message_id ?? '');
			// 按 message_id 去重：乐观追加已替换为真实 id，自己发送的推送自然跳过。
			if (!id || messages.some((m) => m.id === id)) return;
			const meId = auth.user?.user_id ?? '__me__';
			const sent = payload.sender?.user_id === meId;
			messages = [
				...messages,
				{ id, text: payload.content ?? '', sent, time: formatTime(payload.sent_at) }
			];
			// 收到非自己消息即视为已读（与 load() 行为一致）
			if (!sent) void markGroupRead({ scope: 'group', group_id: gid }).catch(() => {});
		};

		const onMessageUpdate = (e: Event) => {
			const detail = (e as CustomEvent<{
				type: 'edited' | 'recalled' | 'read_count';
				payload: { message_id: string; group_id: string; content?: string };
			}>).detail;
			if (!detail?.payload || detail.payload.group_id !== gid) return;
			const id = String(detail.payload.message_id ?? '');
			// 消息不在本地列表则忽略（保持空操作）
			if (detail.type === 'edited') {
				messages = messages.map((m) =>
					m.id === id ? { ...m, text: detail.payload.content ?? m.text } : m
				);
			} else if (detail.type === 'recalled') {
				messages = messages.map((m) => (m.id === id ? { ...m, text: '[消息已撤回]', recalled: true } : m));
			}
			// 'read_count'：本地无已读数展示字段，忽略
		};

		window.addEventListener(WS_GROUP_MESSAGE_EVENT, onNewMessage as EventListener);
		window.addEventListener(WS_GROUP_MESSAGE_UPDATE_EVENT, onMessageUpdate as EventListener);
		return () => {
			window.removeEventListener(WS_GROUP_MESSAGE_EVENT, onNewMessage as EventListener);
			window.removeEventListener(WS_GROUP_MESSAGE_UPDATE_EVENT, onMessageUpdate as EventListener);
		};
	});

	// typing 接收：仅处理本群（payload.group_id === gid）推送；true 时 3 秒超时自动重置。
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
			// 切换会话时清理发送侧 typing 定时器
			if (typingStopTimer) {
				clearTimeout(typingStopTimer);
				typingStopTimer = null;
			}
			lastTypingSentAt = 0;
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
			src={groupAvatar}
			name={groupName}
			size={compact ? 34 : 38}
			shape="rounded"
			style="border:1.5px solid var(--neu-border);box-shadow:3px 3px 7px var(--neu-shadow-dark), -3px -3px 7px var(--neu-shadow-light);"
		/>

		<!-- Name + status -->
		<div class="flex-1 min-w-0">
			<p class="truncate text-[var(--neu-text)]" style="font-size:{compact ? 13 : 14}px;">
				{groupName || '群聊'}
			</p>
			<p class="{groupTyping ? 'text-[var(--neu-accent-primary)]' : 'text-[var(--neu-text-muted)]'}" style="font-size:11px;">
				{#if groupTyping}
					{typingName || '有人'}正在输入…
				{:else}
					群聊
				{/if}
			</p>
		</div>
	</div>

	<!-- Messages -->
	<div bind:this={scrollEl} onscroll={handleScroll} class="flex-1 overflow-y-auto no-scrollbar" style="overscroll-behavior:contain;">
		{#if loading}
			<div class="flex h-full items-center justify-center">
				<p class="text-[var(--neu-text-muted)]" style="font-size:12px;">加载中…</p>
			</div>
		{:else if messages.length === 0}
			<div class="flex h-full items-center justify-center">
				<p class="text-[var(--neu-text-muted)]" style="font-size:12px;">还没有消息，来打个招呼吧</p>
			</div>
		{:else}
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
								class="{msg.sent ? 'text-[var(--content-inverse-muted)]' : 'text-[var(--neu-text-muted)]'}"
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
		{/if}
	</div>

	<!-- Input bar -->
	{#if sendError}
		<p class="px-4 pb-1.5 text-[var(--semantic-warning)]" style="font-size:11px;">{sendError}</p>
	{/if}
	<div class="flex-shrink-0 flex items-center gap-2.5 px-4 py-3 bg-[var(--neu-bg)]" style="border-top:1px solid var(--neu-border);">
		<div class="flex-1 flex items-center px-4 rounded-[18px] neu-input-surface" style="height:38px;">
			<input
				type="text"
				bind:value={inputVal}
				oninput={(e) => handleInputChange(e)}
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
			<Send style="width:14px;height:14px;color:{inputVal.trim() ? 'var(--text-on-accent)' : 'var(--neu-text-muted)'};transform:translateX(1px);" strokeWidth={2} />
		</button>
	</div>
</div>
