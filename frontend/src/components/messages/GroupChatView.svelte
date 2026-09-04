<script lang="ts">
	import { goto } from '$app/navigation';
	import { ROUTES } from '@/lib/routes';
	import { ArrowLeft } from '@lucide/svelte';
	import Avatar from '@/components/shared/Avatar.svelte';
	import ChatMessageList from '@/components/messages/ChatMessageList.svelte';
	import ChatInputBar from '@/components/messages/ChatInputBar.svelte';
	import { auth } from '@/stores/auth.svelte';
	import { getGroupDetail, getGroupMessages, sendGroupMessage, markGroupRead, recallGroupMessage, recallGroupMessageAsAdmin, updateGroupMessage } from '@/api';
	import { WS_GROUP_MESSAGE_EVENT, WS_GROUP_MESSAGE_UPDATE_EVENT, wsClient } from '@/lib/ws/client';
	import type { UserInfo } from '@/lib/ws/types';
	import { useChat, formatTime, createTypingIndicator, inRecallWindow, type ChatMessage } from '@/lib/chat.svelte';

	// 服务端推送版输入状态（spec GroupTypingPayload，types.ts 未单独导出）。
	interface GroupTypingPayload {
		group_id: string;
		actor: UserInfo;
		is_typing: boolean;
	}

	// GroupChatView — 群聊窗口（spec: 以 group_id 寻址）。
	// 历史来自 group_messages/fetch;发送走 group_messages/send。
	// 消息方向由 sender.user_id 与当前用户比较得出;共享状态机由 useChat 工厂承载;
	// 消息区与输入栏由 ChatMessageList/ChatInputBar 共享组件渲染。

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
	let membersCount = $state<number | null>(null);

	const meId = $derived(auth.user?.user_id ?? '__me__');
	let myRole = $state<'owner' | 'admin' | 'member'>('member');

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
				membersCount = d.group.members_count ?? null;
				myRole = d.viewer_state?.role ?? 'member';
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
				text: m.recalled_at ? '消息已撤回' : (m.content ?? ''),
				recalled: Boolean(m.recalled_at),
				editedAt: m.edited_at,
				sent: m.sender.user_id === meId,
				time: formatTime(m.sent_at),
				sentAt: m.sent_at
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
			// sentAt 供 useChat 按服务端时间保持正序（resume 回放/实时交错不乱序）
			return { id, text: payload.content ?? '', sent, time: formatTime(payload.sent_at), sentAt: payload.sent_at };
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
		},
		recallMessage: async (id) => {
			// 视角分离（spec: api_spec_chat.md / api_spec_group.md）：
			// 撤回自己的消息走 group_messages/recall（仅作者，时间窗内）；
			// 群主/群管理员撤回他人消息走 groups/admin/messages/recall（守卫子树，无时间窗）。
			const msg = chat.messages.find((m) => m.id === id);
			if (!msg) return { code: 1, error: '消息不存在' };
			const res = msg.sent
				? await recallGroupMessage({ message_id: id })
				: await recallGroupMessageAsAdmin({ group_id: groupId, message_id: id });
			return { code: res.code, error: res.code !== 0 ? '撤回失败' : undefined };
		},
		updateMessage: async (id, content) => {
			const res = await updateGroupMessage({ message_id: id, content });
			return { code: res.code, error: res.code !== 0 ? '编辑失败' : undefined };
		}
	});
	// 撤回时间窗仅作者适用（spec message_recall_window_seconds，默认 300s）；
	// 群主/群管理员无时间窗。
	const canRecallMessage = (msg: ChatMessage) =>
		(msg.sent && inRecallWindow(msg.sentAt)) || myRole === 'admin' || myRole === 'owner';

	// 最后一条非本人消息，供屏幕阅读器 aria-live 播报（只播文本，避免整表重读）
	const lastIncoming = $derived([...chat.messages].reverse().find((m) => !m.sent));

	// --- typing 接收：仅处理本群（payload.group_id === gid）推送。 ---
	const typing = createTypingIndicator();

	$effect(() => {
		const gid = groupId;
		if (!gid) return;

		const off = wsClient.on('GroupTyping', (payload) => {
			// ws/types 中 GroupTyping 为宽松 Record，按 spec 形状收窄读取。
			const p = payload as unknown as GroupTypingPayload;
			if (!p || p.group_id !== gid) return;
			typing.notify(p.is_typing, p.is_typing ? (p.actor?.nickname ?? '') : '');
		});
		return () => {
			off();
			typing.reset();
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

		<!-- 头像 + 名称/状态：点击进入群详情（移动端群聊页的管理/公告入口） -->
		<button
			type="button"
			class="avatar-entry hover-bg focus-ring flex min-w-0 flex-1 items-center gap-3 rounded-md text-left"
			onclick={() => void goto(ROUTES.group(groupId))}
			aria-label={`查看群「${groupName || '群聊'}」详情`}
		>
			<Avatar
				src={groupAvatar}
				name={groupName}
				size={compact ? 34 : 38}
				shape="rounded"
				style="border:var(--cst-border-width-regular) solid var(--cst-border-default);box-shadow:var(--shadow-raised-sm);"
			/>

			<!-- Name + status -->
			<div class="flex-1 min-w-0">
				<p class="truncate text-text-primary {compact ? 'text-sm' : 'text-md'}">
					{groupName || '群聊'}
				</p>
				<p class="{typing.active ? 'text-accent-default' : 'text-text-secondary'} text-xs" >
					{#if typing.active}
						{typing.actorName || '有人'}正在输入…
					{:else if membersCount != null}
						{membersCount} 位成员
					{:else}
						群聊
					{/if}
				</p>
			</div>
		</button>
	</div>

	<!-- Messages -->
	<ChatMessageList {chat} canRecall={canRecallMessage} />

	<!-- Input bar -->
	<ChatInputBar {chat} />
</div>
