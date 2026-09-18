// 消息页会话列表的数据模型与排序。
//
// 排序契约（符合主流即时通讯直觉）：
//   1. 置顶会话（isPinned）排在最前面；
//   2. 在同为置顶或同为普通的分组中，未读会话（unread > 0）优先排在已读会话前面；
//   3. 组内统一按最新一条消息/活跃时间戳倒序排列；
//   4. 时间相同用 id 做稳定 tie-breaker。

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
	/** 免打扰（spec PrivateChatItem.is_muted / GroupViewerState.is_muted）；不阻塞未读徽章，仅弱化提醒。 */
	isMuted: boolean;
	online: boolean;
	kind?: 'private' | 'group';
}

/** 时间戳兜底：无效日期按 0 处理（用于私聊/群聊混合排序）。 */
function tsOf(t: string): number {
	const n = new Date(t).getTime();
	return isNaN(n) ? 0 : n;
}

export function compareChatItems(a: ChatItem, b: ChatItem): number {
	// 1. 置顶优先
	if (a.isPinned !== b.isPinned) return Number(b.isPinned) - Number(a.isPinned);
	// 2. 未读优先（置顶内未读 > 置顶内已读；普通内未读 > 普通内已读）
	const aUnread = a.unread > 0;
	const bUnread = b.unread > 0;
	if (aUnread !== bUnread) return Number(bUnread) - Number(aUnread);
	// 3. 最新消息时间倒序
	const byTime = tsOf(b.timestamp) - tsOf(a.timestamp);
	if (byTime !== 0) return byTime;
	// 4. 稳定 tie-breaker
	return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
}
