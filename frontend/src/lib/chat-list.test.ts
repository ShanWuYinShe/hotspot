// 会话列表排序契约：置顶优先，同级下未读优先，组内按最新消息时间倒序。
import { describe, it, expect } from 'vitest';
import { compareChatItems, type ChatItem } from './chat-list';
import { formatChatListTime } from './chat.svelte';

function chat(partial: Omit<Partial<ChatItem>, 'timestamp' | 'time'> & { id: string; timestamp: string }): ChatItem {
	return {
		peerUserId: '',
		name: '',
		avatar: '',
		lastMessage: '',
		time: '',
		unread: 0,
		isPinned: false,
		isMuted: false,
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

// 会话列表时间戳（审查 M1）：旧会话不得显示裸 HH:mm。
describe('formatChatListTime', () => {
	it('今天显示 HH:mm', () => {
		const now = new Date();
		const iso = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 9, 5).toISOString();
		expect(formatChatListTime(iso)).toBe('09:05');
	});

	it('昨天显示「昨天」', () => {
		const d = new Date();
		d.setDate(d.getDate() - 1);
		expect(formatChatListTime(d.toISOString())).toBe('昨天');
	});

	it('今年更早显示 M月D日', () => {
		const d = new Date();
		d.setMonth(0, 7); // 1 月 7 日（今年范围内，除非测试恰在 1 月 1-7 日跨年附近）
		if (d.getTime() > Date.now() - 2 * 86400000) return; // 跨年边缘跳过
		expect(formatChatListTime(d.toISOString())).toBe('1月7日');
	});

	it('空值与非法输入返回空串', () => {
		expect(formatChatListTime('')).toBe('');
		expect(formatChatListTime(undefined)).toBe('');
		expect(formatChatListTime('not-a-date')).toBe('');
	});
});
