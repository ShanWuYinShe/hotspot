// 私聊/群聊共享的聊天逻辑工厂（收敛 ChatView/GroupChatView 的重复逻辑）。
//
// 两视图此前各自维护一份几乎相同的状态机：消息加载/游标分页/滚动到底/typing
// 发送节流/乐观发送+回滚/WS 实时接收/编辑撤回/已读标记。差异仅在端点、WS 事件
// 名与 payload 形状、错误文案。此处以依赖注入（options）参数化差异，视图只保留
// 自己的 header 与 typing 接收（两者机制不同：私聊 window event / 群聊 wsClient.on）。
//
// 调用方式：组件 <script> 顶层 const chat = useChat({...})；工厂内的 $state/$effect
// 生命周期绑定组件实例，卸载自动清理。

import type { PrivateMessageNewPayload } from '@/lib/ws/types';
import { wsClient } from '@/lib/ws/client';

// 断线补拉阈值（spec L163-165）：WS 断开超过该时长后，重连时主动拉取历史（private_messages/fetch / group_messages/fetch）
// 首页补拉（HTTP 与 WS 推送重复按主键去重，spec L169）；未超阈值直接依赖
// WS 推送 / resume 续传。
const RECONNECT_HISTORY_FETCH_THRESHOLD_MS = 10_000;

export interface ChatMessage {
	id: string;
	text: string;
	sent: boolean; // true = outgoing (from current user)
	time: string;
	/** 群聊场景的发送者昵称（incoming 气泡上方展示；私聊不需要） */
	senderName?: string;
	/** 原始发送时间（ISO）；用于按本地日期分组。 */
	sentAt?: string;
	recalled?: boolean;
	editedAt?: string;
}

export interface ChatMessageUpdate {
	type: 'edited' | 'recalled';
	id: string;
	content?: string;
	editedAt?: string;
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
	sendMessage: (text: string, mentions?: string[]) => Promise<SendResult>;
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
	/** 撤回消息（spec：发送者窗口内 / 群管理员豁免）；返回 code，非 0 失败。可选，未注入则不提供撤回。 */
	recallMessage?: (id: string) => Promise<{ code: number; error?: string }>;
	/** 编辑消息（spec：仅本人 + type=text + 未撤回）；返回 code，非 0 失败。可选，未注入则不提供编辑。 */
	updateMessage?: (id: string, content: string) => Promise<{ code: number; error?: string }>;
}

export function formatTime(iso?: string): string {
	if (!iso) return '';
	const d = new Date(iso);
	if (isNaN(d.getTime())) return '';
	return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

/**
 * 会话列表时间戳（审查 M1）：此前恒用 formatTime 的 HH:mm，三天前的会话
 * 显示「17:20」式误导。按时间差切换——今天→HH:mm / 昨天→「昨天」 /
 * 今年→M月D日 / 跨年→YYYY年M月D日。
 */
export function formatChatListTime(iso?: string): string {
	if (!iso) return '';
	const d = new Date(iso);
	if (isNaN(d.getTime())) return '';
	const now = new Date();
	const startOfDay = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
	const dayDiff = Math.round((startOfDay(now) - startOfDay(d)) / 86400000);
	if (dayDiff === 0) return formatTime(iso);
	if (dayDiff === 1) return '昨天';
	if (d.getFullYear() === now.getFullYear()) return `${d.getMonth() + 1}月${d.getDate()}日`;
	return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

/** 日期分组胶囊文案：今天 / 昨天 / M月D日（跨年带年份）。 */
export function formatDateLabel(iso?: string): string {
	if (!iso) return '今天';
	const d = new Date(iso);
	if (isNaN(d.getTime())) return '今天';
	const startOfDay = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
	const dayDiff = Math.round((startOfDay(new Date()) - startOfDay(d)) / 86400000);
	if (dayDiff === 0) return '今天';
	if (dayDiff === 1) return '昨天';
	if (d.getFullYear() === new Date().getFullYear()) return `${d.getMonth() + 1}月${d.getDate()}日`;
	return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

export interface ChatDateGroup {
	label: string;
	messages: ChatMessage[];
}

/** 把按时间正序（旧→新）的消息按本地日期切分为分组，保持组内顺序。 */
export function groupMessagesByDay(messages: ChatMessage[]): ChatDateGroup[] {
	const groups: ChatDateGroup[] = [];
	let currentDay = '';
	for (const msg of messages) {
		const day = msg.sentAt ? new Date(msg.sentAt).toDateString() : '';
		if (day !== currentDay) {
			currentDay = day;
			groups.push({ label: formatDateLabel(msg.sentAt), messages: [] });
		}
		groups[groups.length - 1].messages.push(msg);
	}
	return groups;
}

/** 按服务端 sent_at 保持时间正序（旧→新）。sent_at 均为服务端下发的 ISO 8601
 * （同格式字典序等价于时间序）；无 sentAt 的乐观消息排最后。V8 sort 稳定，
 * 同刻消息保持到达序。 */
function sortBySentAt(list: ChatMessage[]): ChatMessage[] {
	return list.sort((a, b) => {
		const x = a.sentAt ?? '\uffff';
		const y = b.sentAt ?? '\uffff';
		return x < y ? -1 : x > y ? 1 : 0;
	});
}

export function useChat(opts: UseChatOptions) {
	let messages = $state<ChatMessage[]>([]);
	let inputVal = $state('');
	let loading = $state(true);
	// 历史加载失败（网络/服务端错误）。加载失败时必须显式呈现错误态，
	// 不能把失败渲染成"空会话"误导用户（尤其桌面分栏切换会话后）。
	let loadError = $state(false);
	let sending = $state(false);
	let sendError = $state('');
	let scrollEl = $state<HTMLDivElement | null>(null);
	// --- 历史分页(契约:游标分页,next_cursor 缺失即无更多) ---
	// $state：加载中/失败/到底三态要暴露给视图做指示（layout 第 6 条），须可响应。
	let nextCursor = $state<string | null>(null);
	let loadingMore = $state(false);
	let loadingMoreFailed = $state(false);

	// --- typing 发送状态（契约：客户端不超过 3 秒发一次；服务端权威节流）---
	let lastTypingSentAt = 0;
	let typingStopTimer: ReturnType<typeof setTimeout> | null = null;

	// 初始加载 + 会话切换重载。以「发起时的会话 id」做竞态保护：响应晚到时
	// 若已切走则丢弃，避免旧会话内容渲染进新会话。
	async function loadInitial() {
		const target = opts.targetId();
		loading = true;
		loadError = false;
		// 切换会话时清空旧消息：分栏复用实例时不清理会残留上一个会话的内容
		messages = [];
		nextCursor = null;
		const res = await opts.loadHistory();
		if (opts.targetId() !== target) return;
		if (res) {
			// 契约：服务端历史按 sent_at 倒序（新→旧）返回；渲染需要旧→新，
			// 首屏翻转为正序，保证越新的消息在下方。
			messages = res.messages.slice().reverse();
			nextCursor = res.next_cursor ?? null;
		} else {
			loadError = true;
		}
		loadingMore = false;
		loadingMoreFailed = false;
		loading = false;
		// 标记已读；幂等语义，失败静默不打扰会话浏览。
		opts.markRead();
	}

	// 初始加载 + 会话切换重载（依赖 targetId 变化）。
	$effect(() => {
		const id = opts.targetId();
		if (!id) {
			messages = [];
			loading = false;
			loadError = false;
			return;
		}
		void loadInitial();
	});

	// 滚动到底部：仅在「尾部消息 id 变化」（初始加载/新消息 append/会话切换）时触发；
	// loadMore 往顶部插入旧消息不改变尾部 id，避免与顶部位置还原互相打架。
	let prevLastId = '';
	$effect(() => {
		const el = scrollEl;
		if (!el) return;
		const lastId = messages.length ? messages[messages.length - 1].id : '';
		if (lastId === prevLastId) return;
		prevLastId = lastId;
		if (lastId) el.scrollTo({ top: el.scrollHeight, behavior: 'smooth' });
		else el.scrollTop = el.scrollHeight;
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
			// 按 sent_at 保持时间正序：resume 续传回放（旧事件）与实时事件交错到达时，
			// 简单 append 会把回放消息排到实时消息之后（spec L173 按 seq 排序语义，
			// 以服务端 sent_at 同源时间实现）。
			messages = sortBySentAt([...messages, msg]);
		};

		const onMessageUpdate = (e: Event) => {
			const upd = opts.wsMessageUpdate(e);
			if (!upd) return;
			// 消息不在本地列表则忽略（保持空操作）
			if (upd.type === 'edited') {
				messages = messages.map((m) => (m.id === upd.id ? { ...m, text: upd.content ?? m.text, editedAt: upd.editedAt ?? new Date().toISOString() } : m));
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

	// 断线补拉（spec L163-165）：WS 断开记录时间点，重连后断开时长超阈值（10s）时
	// 主动拉取历史（private_messages/fetch / group_messages/fetch） 首页补拉并按 id 去重合并——resume 失效/降级时断线期间消息
	// 只能靠 HTTP 补齐；未超阈值直接依赖 WS 推送 / resume 续传。resume 成功时补拉
	// 结果与续传事件按主键去重（spec L169），冗余无害。
	let disconnectedAt = 0;
	$effect(() => {
		return wsClient.onStatusChange((s) => {
			if (s === 'disconnected') {
				disconnectedAt = Date.now();
			} else if (s === 'connected' && disconnectedAt > 0) {
				const gap = Date.now() - disconnectedAt;
				disconnectedAt = 0;
				if (gap > RECONNECT_HISTORY_FETCH_THRESHOLD_MS) void refetchHistory();
			}
		});
	});

	/** 重连补拉：重新拉取首屏历史，与本地消息按 id 去重合并后按 sent_at 正序。
	 * 会话切换竞态用 targetId 前后比对丢弃（与 loadMore 同模式）。 */
	async function refetchHistory() {
		const target = opts.targetId();
		if (!target) return;
		const res = await opts.loadHistory();
		if (opts.targetId() !== target) return;
		if (!res) return;
		const fresh = res.messages.slice().reverse(); // 服务端倒序（新→旧）→ 正序
		const freshIds = new Set(fresh.map((m) => m.id));
		const kept = messages.filter((m) => !freshIds.has(m.id)); // 本地独有（乐观/更早历史）
		messages = sortBySentAt([...fresh, ...kept]);
		nextCursor = res.next_cursor ?? null;
		opts.markRead();
	}

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

	/** 滚动到顶加载更早的历史(游标分页;契约:服务端按 sent_at 倒序返回,next_cursor
	    指向更早批次;每批内部同样翻转为正序后插到顶部,保持旧上新下)。 */
	async function loadMore() {
		if (loadingMore || !nextCursor) return;
		loadingMore = true;
		loadingMoreFailed = false;
		const target = opts.targetId();
		const el = scrollEl;
		const prevHeight = el?.scrollHeight ?? 0;
		const prevTop = el?.scrollTop ?? 0;
		const res = await opts.loadHistory(nextCursor);
		// 会话已切换则丢弃结果(竞态保护)
		if (opts.targetId() !== target) return;
		loadingMore = false;
		// 失败可见化（layout 第 6 条：禁止静默转「无更多」），由消息区渲染重试入口。
		if (!res) {
			loadingMoreFailed = true;
			return;
		}
		messages = [...res.messages.slice().reverse(), ...messages];
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

	const handleSend = async (mentions?: string[]) => {
		const text = inputVal.trim();
		if (!text || sending || !opts.targetId()) return;
		inputVal = '';
		sendError = '';
		sending = true;
		// Optimistic append so the user sees their message instantly.
		const tempId = `temp-${Date.now()}`;
		messages = [
			...messages,
			{ id: tempId, text, sent: true, time: formatTime(new Date().toISOString()), sentAt: new Date().toISOString() }
		];
		try {
			const res = await opts.sendMessage(text, mentions);
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

	/** 撤回消息：调注入函数，成功后本地标记已撤回（WS 回环到达再幂等覆盖）。 */
	async function handleRecall(id: string) {
		if (!opts.recallMessage) return;
		try {
			const res = await opts.recallMessage(id);
			if (res.code === 0) {
				messages = messages.map((m) => (m.id === id ? { ...m, text: '[消息已撤回]', recalled: true } : m));
			} else {
				sendError = res.error ?? '撤回失败';
			}
		} catch {
			sendError = '撤回失败,请稍后重试';
		}
	}

	/** 编辑消息：调注入函数，成功后本地更新文本 + 标记已编辑（WS 回环到达再幂等覆盖）。 */
	async function handleEdit(id: string, content: string) {
		if (!opts.updateMessage) return;
		try {
			const res = await opts.updateMessage(id, content);
			if (res.code === 0) {
				messages = messages.map((m) => (m.id === id ? { ...m, text: content, editedAt: new Date().toISOString() } : m));
			} else {
				sendError = res.error ?? '编辑失败';
			}
		} catch {
			sendError = '编辑失败,请稍后重试';
		}
	}

	const lastSentId = $derived(
		messages
			.filter((m) => m.sent)
			.map((m) => m.id)
			.pop()
	);
	const dateGroups = $derived(groupMessagesByDay(messages));

	return {
		get messages() {
			return messages;
		},
		get dateGroups() {
			return dateGroups;
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
		get loadError() {
			return loadError;
		},
		get loadingMore() {
			return loadingMore;
		},
		get loadingMoreFailed() {
			return loadingMoreFailed;
		},
		get hasMore() {
			return nextCursor !== null;
		},
		loadMore,
		reload: loadInitial,
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
		handleRecall,
		handleEdit,
		cancelTyping
	};
}

// 撤回时间窗（spec message_recall_window_seconds，默认 300s；服务端 10400
// 兜底，ClientConfig 接入留后续）。私聊/群聊共用。
export const RECALL_WINDOW_MS = 300_000;

export function inRecallWindow(sentAt?: string): boolean {
	if (!sentAt) return true;
	return Date.now() - new Date(sentAt).getTime() < RECALL_WINDOW_MS;
}

// createTypingIndicator —— 私聊/群聊共用的「正在输入」状态机：
// notify(true) 激活并启动 3 秒超时自动重置；notify(false) 立即重置；
// 群聊场景携带 actor 昵称展示「某某正在输入…」。
export function createTypingIndicator() {
	let active = $state(false);
	let actorName = $state('');
	let timer: ReturnType<typeof setTimeout> | null = null;

	function reset() {
		if (timer) {
			clearTimeout(timer);
			timer = null;
		}
		active = false;
		actorName = '';
	}

	function notify(isTyping: boolean, name = '') {
		if (isTyping) {
			active = true;
			actorName = name;
			if (timer) clearTimeout(timer);
			timer = setTimeout(() => {
				timer = null;
				active = false;
				actorName = '';
			}, 3000);
		} else {
			reset();
		}
	}

	return {
		notify,
		reset,
		get active() {
			return active;
		},
		get actorName() {
			return actorName;
		}
	};
}

// ChatController —— useChat 返回值的类型出口，供共享子组件标注 props。
export type ChatController = ReturnType<typeof useChat>;
