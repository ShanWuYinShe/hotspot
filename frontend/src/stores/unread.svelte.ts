// 全局未读 store（runes 模块单例）。
// 职责：为侧栏「消息」、顶栏「通知」铃铛与移动底部导航提供全局未读徽标。
//
// 未读计数按 spec 由客户端维护（backend-go AGENTS.md「未读计数（聊天）」：
// 服务端不维护独立未读字段，仅提供冷启动校准）。三层模型：
// 1. 冷启动校准：system/bootstrap 的 unread_summary（notify_total +
//    private/group_chat_unread_count，服务端按已读游标现算的真相）；
// 2. WS 增量：ws:notify → 通知 +1；ws:private-message / ws:group-message →
//    聊天 +1（过滤自己发送的回显，多端同步场景）；
// 3. 页面回写：notify 页 / messages 页拉取后把服务端真相写回（含打开会话
//    清零），徽标允许短暂漂移，进入对应页面即收敛。
//
// initUnread() 必须在组件初始化期间同步调用一次（+layout.svelte script 顶层，
// 与 initWs()/initPresence() 一致），内部才完成事件绑定。

import { WS_NOTIFY_EVENT, wsClient } from '@/lib/ws/client';
import { auth } from '@/stores/auth.svelte';
import type { UnreadSummary } from '@/api/types';

/** 通知未读数（unread_summary.notify_total 同语义）。 */
let notifyUnread = $state(0);

/** 聊天未读数（私聊 + 群聊会话未读之和）。 */
let chatUnread = $state(0);

/** 响应式读取入口。 */
export const unread = {
	get notify() {
		return notifyUnread;
	},
	get chat() {
		return chatUnread;
	},
	get total() {
		return notifyUnread + chatUnread;
	},
};

/** 回写通知未读（notify 页拉取 getUnreadCount 后调用）。 */
export function setNotifyUnread(n: number): void {
	notifyUnread = Math.max(0, n);
}

/** 回写聊天未读（messages 页拉取/打开会话清零后调用；传页面算出的服务端真相）。 */
export function setChatUnread(n: number): void {
	chatUnread = Math.max(0, n);
}

/** 由 system/bootstrap 的 unread_summary 校准（spec api_spec_system.md → bootstrap）。 */
export function applyBootstrapUnread(s: UnreadSummary | null | undefined): void {
	if (!s) return;
	notifyUnread = Math.max(0, s.notify_total);
	chatUnread = Math.max(0, s.private_chat_unread_count + s.group_chat_unread_count);
}

/** 登出/会话失效/封禁时清零，避免跨账号残留。 */
export function resetUnread(): void {
	notifyUnread = 0;
	chatUnread = 0;
}

/**
 * 初始化未读事件监听。返回清理函数（解绑事件）。
 * 增量规则：ws:notify（client.ts NOTIFY_EVENT_TYPES 派发的通知类事件）→ 通知 +1；
 * 私聊/群聊新消息 → 聊天 +1，但过滤自己发送的回显。匿名态不计数（WS 需鉴权，
 * 正常收不到；防御登出瞬间的残留推送）。
 */
export function initUnread(): () => void {
	const onNotify = () => {
		if (auth.status !== 'authenticated') return;
		notifyUnread += 1;
	};
	const onPrivateMessage = (p: unknown) => {
		if (auth.status !== 'authenticated') return;
		const sender = (p as { sender?: { user_id?: string } } | null)?.sender?.user_id;
		if (sender && sender === auth.user?.user_id) return;
		chatUnread += 1;
	};
	const onGroupMessage = (p: unknown) => {
		if (auth.status !== 'authenticated') return;
		const sender = (p as { sender?: { user_id?: string } } | null)?.sender?.user_id;
		if (sender && sender === auth.user?.user_id) return;
		chatUnread += 1;
	};
	window.addEventListener(WS_NOTIFY_EVENT, onNotify);
	const offPrivate = wsClient.on('PrivateMessageNew', onPrivateMessage);
	const offGroup = wsClient.on('GroupMessageNew', onGroupMessage);
	return () => {
		window.removeEventListener(WS_NOTIFY_EVENT, onNotify);
		offPrivate();
		offGroup();
	};
}
