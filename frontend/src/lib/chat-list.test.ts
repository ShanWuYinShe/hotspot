// 会话列表排序契约：置顶 > 未读 > 已读，组内按最近活跃时间倒序。
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
	it('置顶排在未读/已读之前', () => {
		const pinned = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z', isPinned: true, unread: 0 });
		const unread = chat({ id: 'b', timestamp: '2026-01-02T00:00:00Z', unread: 2 });
		const read = chat({ id: 'c', timestamp: '2026-01-03T00:00:00Z' });
		expect([read, pinned, unread].sort(compareChatItems).map((c) => c.id)).toEqual(['a', 'b', 'c']);
	});

	it('未读排在已读之前', () => {
		const unread = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z', unread: 1 });
		const read = chat({ id: 'b', timestamp: '2026-01-02T00:00:00Z' });
		expect([read, unread].sort(compareChatItems).map((c) => c.id)).toEqual(['a', 'b']);
	});

	it('同一组内按最近活跃时间倒序（后到在前）', () => {
		const older = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z', unread: 1 });
		const newer = chat({ id: 'b', timestamp: '2026-01-02T00:00:00Z', unread: 1 });
		const olderRead = chat({ id: 'c', timestamp: '2026-01-01T00:00:00Z' });
		const newerRead = chat({ id: 'd', timestamp: '2026-01-02T00:00:00Z' });
		expect([older, newer, newerRead, olderRead].sort(compareChatItems).map((c) => c.id)).toEqual([
			'b', 'a', 'd', 'c'
		]);
	});

	it('时间相同时用 id 稳定排序', () => {
		const a = chat({ id: 'a', timestamp: '2026-01-01T00:00:00Z' });
		const b = chat({ id: 'b', timestamp: '2026-01-01T00:00:00Z' });
		expect([b, a].sort(compareChatItems).map((c) => c.id)).toEqual(['a', 'b']);
	});
});
