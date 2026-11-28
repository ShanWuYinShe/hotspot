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
import type { Reaction } from '@/api/types';
import type { ReplyPreview } from '@/lib/chat-reply';
import { applyMyReaction, applyReactionDelta, setMessageReaction } from '@/lib/message-reactions';
import { wsClient } from '@/lib/ws/client';
import { prefersReducedMotion } from '@/lib/motion';
import { loadChatDraft, saveChatDraft } from '@/lib/chat-draft';
import { peekBootstrap } from '@/lib/bootstrap-cache';

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
	/** 图片消息读 URL（type=image）：存在时气泡渲染图片而非文本，text 仅作失败/撤回兜底 */
	imageUrl?: string;
	/** 视频/语音/文件消息（type=video/audio/file），由 lib/chat-media 统一映射 */
	videoUrl?: string;
	audioUrl?: string;
	fileUrl?: string;
	fileName?: string;
	/** 文件消息大小（字节）：本地乐观气泡显示用，历史消息从 media.file_size 取。 */
	fileSize?: number;
	/** 原始发送时间（ISO）；用于按本地日期分组。 */
	sentAt?: string;
	recalled?: boolean;
	editedAt?: string;
	/** 表情反应（spec types.md → Reaction[]；W05 消息点赞） */
	reactions?: Reaction[];
	/** 引用回复的原文预览（服务端 reply_to: MessageBrief 归一，见 lib/chat-reply） */
	replyTo?: ReplyPreview | null;
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
	/** 输入草稿键（如 private:<user_id> / group:<group_id>，见 lib/chat-draft）；不传则不持久化。 */
	draftKey?: () => string;
	/** 加载历史（游标分页）；失败返回 null。实现方负责把 API items 映射为 ChatMessage。 */
	loadHistory: (cursor?: string) => Promise<LoadHistoryResult | null>;
	/** 标记会话已读（幂等语义，失败静默）。加载成功后调用。 */
	markRead: () => void;
	/** 发送文本消息；失败返回非 0 code（可携带 error 文案），成功返回服务端 message_id 替换 temp id。 */
	/** replyToId 对应 spec private_messages.send → reply_to_message_id（群聊同名字段）。 */
	sendMessage: (text: string, mentions?: string[], replyToId?: string) => Promise<SendResult>;
	/** 发送图片消息（file_id 来自 chat_image 上传）；未注入时输入栏不显示媒体入口。 */
	sendImage?: (fileId: string) => Promise<SendResult>;
	/** 发送视频消息（file_id 来自 chat_video 上传）；与 Android 端同一 send(type=video) 语义。 */
	sendVideo?: (fileId: string) => Promise<SendResult>;
	/** 发送文件消息（file_id 来自 chat_file 上传，spec MessageType=file）。 */
	sendFile?: (fileId: string) => Promise<SendResult>;
	/** 发送语音消息（file_id 来自 chat_audio 上传，spec MessageType=audio）。 */
	sendAudio?: (fileId: string) => Promise<SendResult>;
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
	/** 当前用户 ID：用于判定 WS 反应事件是否为自己（Reaction* 事件按 actor 区分）。 */
	selfUserId?: () => string;
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
	// 当前输入框内容归属的草稿键（会话切换时把旧文本写回旧键，见下方 $effect）。
	let draftOwner = '';
	let loading = $state(true);
	// 历史加载失败（网络/服务端错误）。加载失败时必须显式呈现错误态，
	// 不能把失败渲染成"空会话"误导用户（尤其桌面分栏切换会话后）。
	let loadError = $state(false);
	let sending = $state(false);
	let sendError = $state('');
	// 发送失败保留原文与 mentions，供输入栏「重试」一键重发（R21 遗留）。
	let failedSend = $state<{ text: string; mentions?: string[]; replyTo?: string; error: string } | null>(null);
	/** 引用回复目标：非 null 时输入栏显示引用条，发送带 reply_to_message_id。 */
	let replyingTo = $state<ChatMessage | null>(null);
	let scrollEl = $state<HTMLDivElement | null>(null);
	// 滚动位置状态（Round 10）：距底 ≤120px 视为「在底部」；上滑超过 200px 才显示回底按钮；
	// 不在底部时收到的新消息只累加计数，不打断阅读。
	let distanceToBottom = $state(0);
	let nearBottom = $state(true);
	let newMessageCount = $state(0);
	let lastMarkReadAt = 0;

	/** 滚到底（smooth / reduced 直接跳），并同步状态。 */
	function scrollToBottomInternal(el: HTMLElement, animate: boolean) {
		const behavior: ScrollBehavior = animate && !prefersReducedMotion() ? 'smooth' : 'auto';
		el.scrollTo({ top: el.scrollHeight, behavior });
		distanceToBottom = 0;
		nearBottom = true;
		newMessageCount = 0;
	}
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
		// 会话切换：新消息计数/滚动状态清零（旧会话 pending 不带到新会话）
		newMessageCount = 0;
		distanceToBottom = 0;
		nearBottom = true;
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

	// 初始加载 + 会话切换重载（依赖 targetId / draftKey 变化）。
	$effect(() => {
		const id = opts.targetId();
		const key = opts.draftKey?.() ?? '';
		if (!id) {
			messages = [];
			loading = false;
			loadError = false;
			draftOwner = '';
			inputVal = '';
			return;
		}
		// 会话切换：草稿已由下方 inputVal 落盘 effect 逐键写入，这里只切键并载入新会话草稿。
		if (draftOwner !== key) {
			// P0：旧会话的失败重发内容/错误提示不能带到新会话——retrySend 用的是当前注入的
			// sendMessage，带过去就会把 A 的原文发给 B。切换即视为全新会话状态。
			failedSend = null;
			sendError = '';
			replyingTo = null;
		}
		draftOwner = key;
		inputVal = key ? loadChatDraft(key) : '';
		void loadInitial();
	});

	// 输入即时落盘（含清空）：刷新页面 / 切回会话时恢复；发送成功清空输入即清除草稿。
	$effect(() => {
		const text = inputVal;
		if (draftOwner) saveChatDraft(draftOwner, text);
	});

	// 滚动策略（Round 10）：仅在「尾部消息 id 变化」时决策，loadMore 往顶部插入旧消息
	// 不改变尾部 id（视口保持逻辑不受影响）：
	// - 初始加载/切换会话（此前列表为空）→ 直接回底（无动画）；
	// - 自己发送（尾部 sent=true）→ 始终滚底（发送反馈）；
	// - 距底 ≤120px → 自动滚底（smooth，reduced 直接跳）；
	// - 上滑超过阈值收到新消息 → 不跳转，只累加 newMessageCount。
	let prevLastId = '';
	let prevLen = 0;
	$effect(() => {
		const el = scrollEl;
		if (!el) return;
		const len = messages.length;
		const added = Math.max(0, len - prevLen);
		prevLen = len;
		const tail = len ? messages[len - 1] : null;
		const lastId = tail?.id ?? '';
		const prev = prevLastId;
		if (lastId === prev) return;
		prevLastId = lastId;
		if (!lastId) {
			el.scrollTop = el.scrollHeight;
			return;
		}
		const initial = prev === '';
		const own = tail?.sent === true;
		if (initial || own || nearBottom) {
			scrollToBottomInternal(el, !initial);
		} else {
			// 一次批量到达多条（同步注入/合批推送）时按新增条数累加，而不是每 effect 只 +1
			newMessageCount += Math.max(1, added);
		}
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

		// 消息表情反应（spec websocket → ReactionAdded/ReactionRemoved）：事件只带
		// target/emoji/actor，不带计数，本地按增量合并；自己的反应按 actor 标注
		// reacted_by_me，便于气泡反应条高亮。
		const onReaction = (active: boolean) => (e: Event) => {
			const p = (e as CustomEvent<{ target_type?: string; target_id?: string; emoji?: string; actor?: { user_id?: string } }>).detail;
			if (!p || p.target_type !== 'message' || !p.target_id || !p.emoji) return;
			const selfId = opts.selfUserId?.();
			const mine = Boolean(selfId && p.actor?.user_id === selfId);
			messages = messages.map((m) =>
				m.id === p.target_id ? { ...m, reactions: applyReactionDelta(m.reactions, p.emoji as string, active, mine) } : m
			);
		};
		const onReactionAdded = onReaction(true);
		const onReactionRemoved = onReaction(false);

		window.addEventListener(opts.wsNewMessageEvent, onNewMessage as EventListener);
		window.addEventListener(opts.wsMessageUpdateEvent, onMessageUpdate as EventListener);
		window.addEventListener('ReactionAdded', onReactionAdded as EventListener);
		window.addEventListener('ReactionRemoved', onReactionRemoved as EventListener);
		return () => {
			window.removeEventListener(opts.wsNewMessageEvent, onNewMessage as EventListener);
			window.removeEventListener(opts.wsMessageUpdateEvent, onMessageUpdate as EventListener);
			window.removeEventListener('ReactionAdded', onReactionAdded as EventListener);
			window.removeEventListener('ReactionRemoved', onReactionRemoved as EventListener);
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

	/** 近底部时补一次已读（节流 1s）；上滑期间收到的新消息不 markRead，
	 * 由用户滚回底部/点回底按钮后统一补标。 */
	function markReadNow() {
		if (!nearBottom) return;
		const now = Date.now();
		if (now - lastMarkReadAt < 1000) return;
		lastMarkReadAt = now;
		opts.markRead();
	}

	/** 滚到底（按钮点击/外部调用）：滚底 + 清计数 + 补已读。 */
	function scrollToBottom() {
		const el = scrollEl;
		if (!el) return;
		scrollToBottomInternal(el, true);
		markReadNow();
	}

	// 滚动处理：scroll 事件每帧可能触发多次，直接读写 scrollHeight/clientHeight 会强制布局。
	// 与唯一滚动 owner（RoutedContentViewport）同口径：rAF 节流，一帧最多读一次布局。
	let scrollRaf = 0;
	function handleScroll() {
		if (scrollRaf) return;
		scrollRaf = requestAnimationFrame(() => {
			scrollRaf = 0;
			const el = scrollEl;
			if (!el) return;
			distanceToBottom = Math.max(0, el.scrollHeight - el.scrollTop - el.clientHeight);
			const nowNear = distanceToBottom <= 120;
			// 手动滚回底部：清空新消息计数并补已读（恢复自动滚动）
			if (nowNear && !nearBottom) {
				newMessageCount = 0;
				markReadNow();
			}
			nearBottom = nowNear;
			if (!loadingMore && nextCursor && el.scrollTop < 40) void loadMore();
		});
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
		failedSend = null;
		await dispatchSend(text, mentions);
	};

	/**
	 * 发图片：输入栏已完成上传（file_id + 本地 objectURL 预览），这里负责乐观气泡与发送。
	 * 失败只记 sendError（不写 failedSend：file_id 已随请求消耗，重试文本无意义）。
	 */
	const handleSendImage = async (fileId: string, previewUrl: string) => {
		if (sending || !opts.targetId() || !opts.sendImage) return;
		sending = true;
		sendError = '';
		const tempId = `temp-img-${Date.now()}`;
		const sentAt = new Date().toISOString();
		messages = [...messages, { id: tempId, text: '', sent: true, time: formatTime(sentAt), sentAt, imageUrl: previewUrl }];
		try {
			const res = await opts.sendImage(fileId);
			if (res.code !== 0) {
				messages = messages.filter((m) => m.id !== tempId);
				sendError = res.error ?? '图片发送失败';
			} else if (res.message_id) {
				messages = messages
					.filter((m) => m.id !== res.message_id)
					.map((m) => (m.id === tempId ? { ...m, id: res.message_id! } : m));
			}
		} catch {
			messages = messages.filter((m) => m.id !== tempId);
			sendError = '图片发送失败,请稍后重试';
		} finally {
			sending = false;
			cancelTyping();
		}
	};

	/** 发视频：与 handleSendImage 同构（乐观气泡 + 失败回滚；file_id 已消耗故不入 failedSend）。 */
	const handleSendVideo = async (fileId: string, previewUrl: string) => {
		if (sending || !opts.targetId() || !opts.sendVideo) return;
		sending = true;
		sendError = '';
		const tempId = `temp-video-${Date.now()}`;
		const sentAt = new Date().toISOString();
		messages = [...messages, { id: tempId, text: '', sent: true, time: formatTime(sentAt), sentAt, videoUrl: previewUrl }];
		try {
			const res = await opts.sendVideo(fileId);
			if (res.code !== 0) {
				messages = messages.filter((m) => m.id !== tempId);
				sendError = res.error ?? '视频发送失败';
			} else if (res.message_id) {
				messages = messages
					.filter((m) => m.id !== res.message_id)
					.map((m) => (m.id === tempId ? { ...m, id: res.message_id! } : m));
			}
		} catch {
			messages = messages.filter((m) => m.id !== tempId);
			sendError = '视频发送失败,请稍后重试';
		} finally {
			sending = false;
			cancelTyping();
		}
	};

	/** 发文件：乐观气泡带本地 blob 预览（可下载），失败回滚（file_id 已消耗故不入 failedSend）。 */
	const handleSendFile = async (fileId: string, fileName: string, fileSize: number, previewUrl: string) => {
		if (sending || !opts.targetId() || !opts.sendFile) return;
		sending = true;
		sendError = '';
		const tempId = `temp-file-${Date.now()}`;
		const sentAt = new Date().toISOString();
		messages = [...messages, { id: tempId, text: '', sent: true, time: formatTime(sentAt), sentAt, fileUrl: previewUrl, fileName, fileSize }];
		try {
			const res = await opts.sendFile(fileId);
			if (res.code !== 0) {
				messages = messages.filter((m) => m.id !== tempId);
				sendError = res.error ?? '文件发送失败';
			} else if (res.message_id) {
				messages = messages
					.filter((m) => m.id !== res.message_id)
					.map((m) => (m.id === tempId ? { ...m, id: res.message_id! } : m));
			}
		} catch {
			messages = messages.filter((m) => m.id !== tempId);
			sendError = '文件发送失败,请稍后重试';
		} finally {
			sending = false;
			cancelTyping();
		}
	};

	/** 发语音：乐观气泡带本地 blob 预览（<audio> 可直接播），失败回滚。 */
	const handleSendAudio = async (fileId: string, previewUrl: string) => {
		if (sending || !opts.targetId() || !opts.sendAudio) return;
		sending = true;
		sendError = '';
		const tempId = `temp-audio-${Date.now()}`;
		const sentAt = new Date().toISOString();
		messages = [...messages, { id: tempId, text: '', sent: true, time: formatTime(sentAt), sentAt, audioUrl: previewUrl }];
		try {
			const res = await opts.sendAudio(fileId);
			if (res.code !== 0) {
				messages = messages.filter((m) => m.id !== tempId);
				sendError = res.error ?? '语音发送失败';
			} else if (res.message_id) {
				messages = messages
					.filter((m) => m.id !== res.message_id)
					.map((m) => (m.id === tempId ? { ...m, id: res.message_id! } : m));
			}
		} catch {
			messages = messages.filter((m) => m.id !== tempId);
			sendError = '语音发送失败,请稍后重试';
		} finally {
			sending = false;
			cancelTyping();
		}
	};

	/** 开始引用回复（空/已撤回消息不进入引用态）。 */
	const startReply = (message: ChatMessage) => {
		if (message.recalled) return;
		replyingTo = message;
	};

	const cancelReply = () => {
		replyingTo = null;
	};

	/** 重试上一次失败的发送（保留原文与 mentions；成功/再次失败都会更新 failedSend）。 */
	const retrySend = async () => {
		const pending = failedSend;
		if (!pending || sending || !opts.targetId()) return;
		failedSend = null;
		sendError = '';
		await dispatchSend(pending.text, pending.mentions);
	};

	const dispatchSend = async (text: string, mentions?: string[]) => {
		sending = true;
		// 引用回复：乐观气泡带原文预览（发送成功后才清空引用态，失败保留便于重试）。
		const reply = replyingTo;
		const replyPreview = reply
			? { id: reply.id, senderName: reply.senderName || (reply.sent ? '我' : '对方'), preview: reply.text.slice(0, 40) || '[消息]' }
			: null;
		// Optimistic append so the user sees their message instantly.
		const tempId = `temp-${Date.now()}`;
		messages = [
			...messages,
			{ id: tempId, text, sent: true, time: formatTime(new Date().toISOString()), sentAt: new Date().toISOString(), replyTo: replyPreview }
		];
		try {
			// 无引用时保持两参调用（既有注入方/单测断言按参数个数匹配）
			const res = reply ? await opts.sendMessage(text, mentions, reply.id) : await opts.sendMessage(text, mentions);
			if (res.code !== 0) {
				// Roll back the optimistic message on failure, keep text for retry.
				messages = messages.filter((m) => m.id !== tempId);
				sendError = res.error ?? '消息发送失败';
				failedSend = { text, mentions, replyTo: reply?.id, error: sendError };
				// 未发出的正文回写为该会话草稿：切走再回来仍能恢复（失败重发状态本身会被切会话清空）。
				if (draftOwner) saveChatDraft(draftOwner, text);
			} else if (res.message_id) {
				// 成功（含重试路径）：丢弃该会话草稿，避免下次进入又出现已发出的文本。
				if (draftOwner) saveChatDraft(draftOwner, '');
				// 成功：用服务端 message_id 替换临时 id，后续 WS 推送按 id 去重跳过。
				// WS 回显可能先于 HTTP 响应到达并已按真实 id 入列，替换前先去掉同 id 回显，避免重复气泡。
				messages = messages
					.filter((m) => m.id !== res.message_id)
					.map((m) => (m.id === tempId ? { ...m, id: res.message_id! } : m));
				// 引用已随消息发出：清空引用态（WS 回显会带回服务端 reply_to，本地预览对齐）
				if (reply && replyingTo?.id === reply.id) replyingTo = null;
			}
		} catch {
			messages = messages.filter((m) => m.id !== tempId);
			sendError = '消息发送失败,请稍后重试';
			failedSend = { text, mentions, error: sendError };
			if (draftOwner) saveChatDraft(draftOwner, text);
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

	/** 切换消息表情反应（乐观更新 + 失败回滚；服务端语义：同一消息只保留一个 emoji）。 */
	async function react(messageId: string, emoji: string, active: boolean): Promise<void> {
		const before = messages.find((m) => m.id === messageId)?.reactions ?? [];
		const optimistic = applyMyReaction(before, emoji, active);
		messages = messages.map((m) => (m.id === messageId ? { ...m, reactions: optimistic } : m));
		const ok = await setMessageReaction(messageId, emoji, active);
		if (!ok) messages = messages.map((m) => (m.id === messageId ? { ...m, reactions: before } : m));
	}

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
		get failedSend() {
			return failedSend;
		},
		get scrollEl() {
			return scrollEl;
		},
		get distanceToBottom() {
			return distanceToBottom;
		},
		get nearBottom() {
			return nearBottom;
		},
		get newMessageCount() {
			return newMessageCount;
		},
		get showScrollDown() {
			return distanceToBottom > 200;
		},
		scrollToBottom,
		set scrollEl(v: HTMLDivElement | null) {
			scrollEl = v;
			// 卸载/切会话时取消挂起的滚动节流帧，避免对已卸载节点做无谓读布局。
			if (!v && scrollRaf) {
				cancelAnimationFrame(scrollRaf);
				scrollRaf = 0;
			}
		},
		get lastSentId() {
			return lastSentId;
		},
		handleScroll,
		handleInputChange,
		/** 发送图片能力（未注入时为 undefined，输入栏据此隐藏图片入口） */
		sendImage: opts.sendImage,
		sendVideo: opts.sendVideo,
		sendFile: opts.sendFile,
		sendAudio: opts.sendAudio,
		handleSend,
		handleSendImage,
		handleSendVideo,
		handleSendFile,
		handleSendAudio,
		// getter：返回对象若直接复制 $state 值会变成快照，输入栏无法响应绑定
		get replyingTo() {
			return replyingTo;
		},
		startReply,
		cancelReply,
		retrySend,
		handleRecall,
		handleEdit,
		react,
		cancelTyping
	};
}

// 撤回时间窗（spec message_recall_window_seconds，AppConfig 可配；0 = 不允许撤回）。
// 权威来源 client_config.limits（bootstrap 已缓存时同步可取），兜底常量必须与
// 后端默认一致（platform/appconfig/source.go，120s），否则会出现「这里能撤回、
// 服务端回 10400」。私聊/群聊共用。
export const DEFAULT_RECALL_WINDOW_MS = 120_000;

export function resolveRecallWindowMs(injected?: number | null): number {
	const configured =
		injected ?? peekBootstrap()?.client_config?.limits?.message_recall_window_seconds ?? null;
	if (configured === null || configured === undefined) return DEFAULT_RECALL_WINDOW_MS;
	return configured * 1000;
}

export function inRecallWindow(sentAt?: string, windowMs?: number): boolean {
	const window = windowMs ?? resolveRecallWindowMs();
	if (window <= 0) return false;
	if (!sentAt) return true;
	return Date.now() - new Date(sentAt).getTime() < window;
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