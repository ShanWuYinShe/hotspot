// 消息页会话列表的数据模型与排序。
//
// 排序（用户要求 + spec 置顶语义）：
//   置顶的聊天优先，其余一律按最新一条消息时间倒序；时间相同用 id 做稳定 tie-breaker。
// 已读/未读只用于过滤，不参与排序——避免阅读后会话因分组变化在列表内跳动。

export interface ChatItem {
	id: string;
	peerUserId: string;
	name: string;
	avatar: string;
	lastMessage: string;
	time: string;
	/** 排序用原始时间戳（ISO 字符串）；time 仅用于展示。 */
	timestamp: string;
	unread: number;
	isPinned: boolean;
	online: boolean;
	kind?: 'private' | 'group';
}

/** 时间戳兜底：无效日期按 0 处理（用于私聊/群聊混合排序）。 */
function tsOf(t: string): number {
	const n = new Date(t).getTime();
	return isNaN(n) ? 0 : n;
}

export function compareChatItems(a: ChatItem, b: ChatItem): number {
	if (a.isPinned !== b.isPinned) return Number(b.isPinned) - Number(a.isPinned);
	const byTime = tsOf(b.timestamp) - tsOf(a.timestamp);
	if (byTime !== 0) return byTime;
	return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
}
