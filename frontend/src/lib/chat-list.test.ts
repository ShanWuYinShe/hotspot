// 会话列表排序契约：置顶优先，同级下未读优先，组内按最新消息时间倒序。
import { describe, it, expect } from 'vitest';
import { compareChatItems, type ChatItem } from './chat-list';

function chat(partial: Omit<Partial<ChatItem>, 'timestamp' | 'time'> & { id: string; timestamp: string }): ChatItem {
	return {
		peerUserId: '',
		name: '',
		avatar: '',
		lastMessage: '',
		time: '',
		unread: 0,
		isPinned: false,
		online: false,
		...partial
	};
}

describe('compareChatItems', () => {
	it('置顶优先：置顶会话排在所有非置顶之前', () => {
		const pinned = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z', isPinned: true, unread: 0 });
		const newest = chat({ id: 'b', timestamp: '2026-01-03T00:00:00Z', unread: 2 });
		const middle = chat({ id: 'c', timestamp: '2026-01-02T00:00:00Z' });
		expect([middle, newest, pinned].sort(compareChatItems).map((c) => c.id)).toEqual(['a', 'b', 'c']);
	});

	it('未读优先：同级内有未读的会话排在已读会话之前', () => {
		const olderUnread = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z', unread: 1 });
		const newerRead = chat({ id: 'b', timestamp: '2026-01-02T00:00:00Z', unread: 0 });
		expect([olderUnread, newerRead].sort(compareChatItems).map((c) => c.id)).toEqual(['a', 'b']);
	});

	it('置顶组内同样满足未读优先，未读置顶 > 已读置顶 > 未读普通 > 已读普通', () => {
		const pinnedRead = chat({ id: 'pr', timestamp: '2026-01-03T00:00:00Z', isPinned: true, unread: 0 });
		const pinnedUnread = chat({ id: 'pu', timestamp: '2026-01-01T00:00:00Z', isPinned: true, unread: 3 });
		const normalUnread = chat({ id: 'nu', timestamp: '2026-01-04T00:00:00Z', isPinned: false, unread: 1 });
		const normalRead = chat({ id: 'nr', timestamp: '2026-01-05T00:00:00Z', isPinned: false, unread: 0 });
		expect([normalRead, pinnedRead, normalUnread, pinnedUnread].sort(compareChatItems).map((c) => c.id)).toEqual([
			'pu',
			'pr',
			'nu',
			'nr'
		]);
	});

	it('按最近活跃时间倒序：同为未读或同为已读时按时间倒序', () => {
		const olderUnread = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z', unread: 1 });
		const newerUnread = chat({ id: 'b', timestamp: '2026-01-02T00:00:00Z', unread: 1 });
		const olderRead = chat({ id: 'c', timestamp: '2026-01-01T00:00:00Z', unread: 0 });
		const newerRead = chat({ id: 'd', timestamp: '2026-01-02T00:00:00Z', unread: 0 });
		expect([olderUnread, newerUnread, newerRead, olderRead].sort(compareChatItems).map((c) => c.id)).toEqual([
			'b',
			'a',
			'd',
			'c'
		]);
	});

	it('时间相同时用 id 稳定排序', () => {
		const a = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z' });
		const b = chat({ id: 'b', timestamp: '2026-01-01T00:00:00Z' });
		expect([b, a].sort(compareChatItems).map((c) => c.id)).toEqual(['a', 'b']);
	});
});
