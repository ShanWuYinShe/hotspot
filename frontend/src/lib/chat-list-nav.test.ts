// 会话列表键盘导航决策单测：进入/逐项移动/两端收敛。
import { describe, it, expect } from 'vitest';
import { isChatListNavKey, nextChatRowIndex } from './chat-list-nav';

describe('isChatListNavKey', () => {
	it('只识别上下方向键', () => {
		expect(isChatListNavKey('ArrowUp')).toBe(true);
		expect(isChatListNavKey('ArrowDown')).toBe(true);
		expect(isChatListNavKey('ArrowLeft')).toBe(false);
		expect(isChatListNavKey('Enter')).toBe(false);
	});
});

describe('nextChatRowIndex', () => {
	it('焦点不在行内时：↓ 进入首行，↑ 不接管', () => {
		expect(nextChatRowIndex(-1, 3, 'ArrowDown')).toBe(0);
		expect(nextChatRowIndex(-1, 3, 'ArrowUp')).toBeNull();
	});

	it('行内逐项移动', () => {
		expect(nextChatRowIndex(0, 3, 'ArrowDown')).toBe(1);
		expect(nextChatRowIndex(2, 3, 'ArrowUp')).toBe(1);
	});

	it('两端收敛：不接管、不循环', () => {
		expect(nextChatRowIndex(2, 3, 'ArrowDown')).toBeNull();
		expect(nextChatRowIndex(0, 3, 'ArrowUp')).toBeNull();
	});

	it('空列表返回 null', () => {
		expect(nextChatRowIndex(0, 0, 'ArrowDown')).toBeNull();
	});
});
