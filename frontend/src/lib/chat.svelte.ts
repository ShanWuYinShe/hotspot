// 私聊/群聊共享的聊天逻辑工厂（迁移自 ChatView/GroupChatView 的重复逻辑）。
//
// 两视图此前各自维护一份几乎相同的状态机：消息加载/游标分页/滚动到底/typing
// 发送节流/乐观发送+回滚/WS 实时接收/编辑撤回/已读标记。差异仅在端点、WS 事件
// 名与 payload 形状、错误文案。此处以依赖注入（options）参数化差异，视图只保留
// 自己的 header 与 typing 接收（两者机制不同：私聊 window event / 群聊 wsClient.on）。
//
// 调用方式：组件 <script> 顶层 const chat = useChat({...})；工厂内的 $state/$effect
// 生命周期绑定组件实例，卸载自动清理。

import type { PrivateMessageNewPayload } from '@/lib/ws/types';

export interface ChatMessage {
	id: string;
	text: string;
	sent: boolean; // true = outgoing (from current user)
	time: string;
	recalled?: boolean;
}

export interface ChatMessageUpdate {
	type: 'edited' | 'recalled';
	id: string;
	content?: string;
}

export interface LoadHistoryResult {
	messages: ChatMessage[];
	next_cursor?: string | null;
}

export interface SendResult {
	code: number;
	message_id?: string;
	error?: string;
}

export interface UseChatOptions {
	/** 会话标识（私聊=peerUserId，群聊=groupId）；变化时自动重载历史并重绑 WS 监听。 */
	targetId: () => string;
	/** 加载历史（游标分页）；失败返回 null。实现方负责把 API items 映射为 ChatMessage。 */
	loadHistory: (cursor?: string) => Promise<LoadHistoryResult | null>;
	/** 标记会话已读（幂等语义，失败静默）。加载成功后调用。 */
	markRead: () => void;
	/** 发送文本消息；失败返回非 0 code（可携带 error 文案），成功返回服务端 message_id 替换 temp id。 */
	sendMessage: (text: string) => Promise<SendResult>;
	/** 发送侧 typing 信号（未连接时实现方静默丢弃）。 */
	sendTypingSignal: (isTyping: boolean) => void;
	/** WS 新消息事件名（window event）。 */
	wsNewMessageEvent: string;
	/** WS 消息更新（编辑/撤回）事件名。 */
	wsMessageUpdateEvent: string;
	/** 从 WS 新消息事件构造 ChatMessage；返回 null 表示忽略（非本会话/自己发送的消息）。 */
	wsNewMessage: (e: Event) => ChatMessage | null;
	/** 从 WS 更新事件提取编辑/撤回；返回 null 表示忽略（非本会话/无关类型）。 */
	wsMessageUpdate: (e: Event) => ChatMessageUpdate | null;
}

export function formatTime(iso?: string): string {
	if (!iso) return '';
	const d = new Date(iso);
	if (isNaN(d.getTime())) return '';
	return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

export function useChat(opts: UseChatOptions) {
	let messages = $state<ChatMessage[]>([]);
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

	// 初始加载 + 会话切换重载（依赖 targetId 变化）。
	$effect(() => {
		const id = opts.targetId();
		if (!id) {
			messages = [];
			loading = false;
			return;
		}
		let cancelled = false;
		async function load() {
			loading = true;
			const res = await opts.loadHistory();
			if (cancelled) return;
			if (res) {
				messages = res.messages;
				nextCursor = res.next_cursor ?? null;
			}
			loadingMore = false;
			loading = false;
			// 标记已读；幂等语义，失败静默不打扰会话浏览。
			opts.markRead();
		}
		void load();
		return () => {
			cancelled = true;
		};
	});

	// 滚动到底部（初始/会话切换立即定位；消息追加平滑滚动）
	$effect(() => {
		const el = scrollEl;
		if (el) el.scrollTop = el.scrollHeight;
	});
	$effect(() => {
		const el = scrollEl;
		if (el) el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
	});

	// WS 实时接收：新消息追加（按 id 去重）；更新处理编辑/撤回。
	// 依赖 targetId：桌面分栏切换会话时自动重新绑定监听；切换时清理 typing 发送定时器。
	$effect(() => {
		const id = opts.targetId();
		if (!id) return;

		const onNewMessage = (e: Event) => {
			const msg = opts.wsNewMessage(e);
			if (!msg || !msg.id) return;
			// 按 message_id 去重（乐观追加已替换为真实 id / 自己发送的推送由实现方返回 null）
			if (messages.some((m) => m.id === msg.id)) return;
			messages = [...messages, msg];
		};

		const onMessageUpdate = (e: Event) => {
			const upd = opts.wsMessageUpdate(e);
			if (!upd) return;
			// 消息不在本地列表则忽略（保持空操作）
			if (upd.type === 'edited') {
				messages = messages.map((m) => (m.id === upd.id ? { ...m, text: upd.content ?? m.text } : m));
			} else if (upd.type === 'recalled') {
				messages = messages.map((m) => (m.id === upd.id ? { ...m, text: '[消息已撤回]', recalled: true } : m));
			}
		};

		window.addEventListener(opts.wsNewMessageEvent, onNewMessage as EventListener);
		window.addEventListener(opts.wsMessageUpdateEvent, onMessageUpdate as EventListener);
		return () => {
			window.removeEventListener(opts.wsNewMessageEvent, onNewMessage as EventListener);
			window.removeEventListener(opts.wsMessageUpdateEvent, onMessageUpdate as EventListener);
			// 会话切换时清理发送侧 typing 定时器
			if (typingStopTimer) {
				clearTimeout(typingStopTimer);
				typingStopTimer = null;
			}
			lastTypingSentAt = 0;
		};
	});

	/** 发送输入状态（未连接由实现方静默丢弃）。 */
	function sendTyping(isTyping: boolean) {
		opts.sendTypingSignal(isTyping);
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
		const target = opts.targetId();
		const el = scrollEl;
		const prevHeight = el?.scrollHeight ?? 0;
		const prevTop = el?.scrollTop ?? 0;
		const res = await opts.loadHistory(nextCursor);
		// 会话已切换则丢弃结果(竞态保护)
		if (opts.targetId() !== target) return;
		loadingMore = false;
		if (!res) return;
		messages = [...res.messages, ...messages];
		nextCursor = res.next_cursor ?? null;
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

	/** 输入内容变化：非空且距上次发送 ≥3 秒发 true；清空立即发 false；停止输入 3 秒后发 false。
	 * 注意：显式 oninput 先于 bind:value 的赋值执行，必须先从事件目标读取最新值。 */
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
		if (!text || sending || !opts.targetId()) return;
		inputVal = '';
		sendError = '';
		sending = true;
		// Optimistic append so the user sees their message instantly.
		const tempId = `temp-${Date.now()}`;
		messages = [...messages, { id: tempId, text, sent: true, time: formatTime(new Date().toISOString()) }];
		try {
			const res = await opts.sendMessage(text);
			if (res.code !== 0) {
				// Roll back the optimistic message on failure.
				messages = messages.filter((m) => m.id !== tempId);
				sendError = res.error ?? '消息发送失败';
			} else if (res.message_id) {
				// 成功：用服务端 message_id 替换临时 id，后续 WS 推送按 id 去重跳过。
				messages = messages.map((m) => (m.id === tempId ? { ...m, id: res.message_id! } : m));
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

	return {
		get messages() {
			return messages;
		},
		get inputVal() {
			return inputVal;
		},
		set inputVal(v: string) {
			inputVal = v;
		},
		get loading() {
			return loading;
		},
		get sending() {
			return sending;
		},
		get sendError() {
			return sendError;
		},
		get scrollEl() {
			return scrollEl;
		},
		set scrollEl(v: HTMLDivElement | null) {
			scrollEl = v;
		},
		get lastSentId() {
			return lastSentId;
		},
		handleScroll,
		handleInputChange,
		handleSend,
		cancelTyping
	};
}
