// 聊天输入草稿（Web）契约：按会话键持久化、空白等同清空、清空全部只动本功能前缀。
import { describe, it, expect, beforeEach } from 'vitest';
import {
	chatDraftKey,
	clearAllChatDrafts,
	clearChatDraft,
	loadChatDraft,
	saveChatDraft
} from './chat-draft';

const PREFIX = 'hotspot.chatDraft.';

describe('chat-draft', () => {
	beforeEach(() => localStorage.clear());

	it('私聊/群聊按不同作用域分键，互不覆盖', () => {
		expect(chatDraftKey('private', 'u1')).toBe('private:u1');
		expect(chatDraftKey('group', 'u1')).toBe('group:u1');
		saveChatDraft(chatDraftKey('private', 'u1'), '私聊草稿');
		saveChatDraft(chatDraftKey('group', 'u1'), '群聊草稿');
		expect(loadChatDraft(chatDraftKey('private', 'u1'))).toBe('私聊草稿');
		expect(loadChatDraft(chatDraftKey('group', 'u1'))).toBe('群聊草稿');
	});

	it('空白文本不落盘：保存空白等价清空', () => {
		saveChatDraft('private:u1', '内容');
		saveChatDraft('private:u1', '   ');
		expect(loadChatDraft('private:u1')).toBe('');
		expect(localStorage.getItem(PREFIX + 'private:u1')).toBeNull();
	});

	it('clearChatDraft 只清指定会话', () => {
		saveChatDraft('private:u1', 'a');
		saveChatDraft('private:u2', 'b');
		clearChatDraft('private:u1');
		expect(loadChatDraft('private:u1')).toBe('');
		expect(loadChatDraft('private:u2')).toBe('b');
	});

	it('clearAllChatDrafts 只清本功能前缀，保留其他本地数据', () => {
		saveChatDraft('private:u1', 'a');
		saveChatDraft('group:g1', 'b');
		localStorage.setItem('appTheme', 'forest');
		clearAllChatDrafts();
		expect(loadChatDraft('private:u1')).toBe('');
		expect(loadChatDraft('group:g1')).toBe('');
		expect(localStorage.getItem('appTheme')).toBe('forest');
	});

	it('空键不读写，避免串档', () => {
		saveChatDraft('', 'x');
		expect(localStorage.length).toBe(0);
		expect(loadChatDraft('')).toBe('');
	});
});
