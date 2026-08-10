// 会话列表排序契约：置顶优先，其余按最新消息时间倒序；未读/已读不参与排序。
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
	it('置顶优先，其余按最新消息时间倒序', () => {
		const pinned = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z', isPinned: true, unread: 0 });
		const newest = chat({ id: 'b', timestamp: '2026-01-03T00:00:00Z', unread: 2 });
		const middle = chat({ id: 'c', timestamp: '2026-01-02T00:00:00Z' });
		expect([middle, newest, pinned].sort(compareChatItems).map((c) => c.id)).toEqual(['a', 'b', 'c']);
	});

	it('未读不参与排序：较旧的未读排在较新的已读之后（读后不跳动）', () => {
		const olderUnread = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z', unread: 1 });
		const newerRead = chat({ id: 'b', timestamp: '2026-01-02T00:00:00Z' });
		expect([olderUnread, newerRead].sort(compareChatItems).map((c) => c.id)).toEqual(['b', 'a']);
	});

	it('按最近活跃时间倒序（后到在前），未读/已读混合仍只看时间', () => {
		const older = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z', unread: 1 });
		const newer = chat({ id: 'b', timestamp: '2026-01-02T00:00:00Z', unread: 1 });
		const olderRead = chat({ id: 'c', timestamp: '2026-01-01T00:00:00Z' });
		const newerRead = chat({ id: 'd', timestamp: '2026-01-02T00:00:00Z' });
		expect([older, newer, newerRead, olderRead].sort(compareChatItems).map((c) => c.id)).toEqual([
			'b', 'd', 'a', 'c'
		]);
	});

	it('时间相同时用 id 稳定排序', () => {
		const a = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z' });
		const b = chat({ id: 'b', timestamp: '2026-01-01T00:00:00Z' });
		expect([b, a].sort(compareChatItems).map((c) => c.id)).toEqual(['a', 'b']);
	});
});
