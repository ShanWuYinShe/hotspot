// 消息页会话列表的数据模型与排序。
//
// 排序优先级（用户要求 + spec 置顶语义）：
//   置顶的聊天 > 后到的未读 > 先到的未读 > 后到的已读 > 先到的已读
// 即：置顶优先 → 未读优先 → 组内按最近活跃时间倒序；时间相同用 id 做稳定 tie-breaker。

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
	const aUnread = a.unread > 0 ? 1 : 0;
	const bUnread = b.unread > 0 ? 1 : 0;
	if (aUnread !== bUnread) return bUnread - aUnread;
	const byTime = tsOf(b.timestamp) - tsOf(a.timestamp);
	if (byTime !== 0) return byTime;
	return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
}
