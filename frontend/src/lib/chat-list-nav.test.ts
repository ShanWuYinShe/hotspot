// 会话列表键盘导航决策单测：进入/逐项移动/两端收敛。
import { describe, it, expect } from 'vitest';
import { handleChatSearchArrowDown, isChatListNavKey, nextChatRowIndex } from './chat-list-nav';

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

describe('handleChatSearchArrowDown', () => {
	const ROWS = `
		<input data-search-input aria-label="搜索会话" />
		<div role="list">
			<div data-list-item><button data-cover aria-label="打开与 A 的会话"></button></div>
			<div data-list-item><button data-cover aria-label="打开与 B 的会话"></button></div>
		</div>`;

	it('搜索框按 ↓：聚焦首个会话行并 preventDefault', () => {
		document.body.innerHTML = ROWS;
		const input = document.querySelector<HTMLInputElement>('input')!;
		input.focus();
		const event = new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true });
		input.dispatchEvent(event);
		handleChatSearchArrowDown(event);
		expect(document.activeElement?.getAttribute('aria-label')).toBe('打开与 A 的会话');
		expect(event.defaultPrevented).toBe(true);
	});

	it('列表为空时不接管（不 preventDefault、焦点留在输入框）', () => {
		document.body.innerHTML = '<input data-search-input aria-label="搜索会话" />';
		const input = document.querySelector<HTMLInputElement>('input')!;
		input.focus();
		const event = new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true });
		input.dispatchEvent(event);
		handleChatSearchArrowDown(event);
		expect(event.defaultPrevented).toBe(false);
		expect(document.activeElement).toBe(input);
	});

	it('非 ↓ 键 / 非输入框目标不接管', () => {
		document.body.innerHTML = ROWS;
		const input = document.querySelector<HTMLInputElement>('input')!;
		input.focus();
		const up = new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true, cancelable: true });
		input.dispatchEvent(up);
		handleChatSearchArrowDown(up);
		expect(up.defaultPrevented).toBe(false);
		const div = document.createElement('div');
		document.body.appendChild(div);
		div.focus();
		const down = new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true, cancelable: true });
		div.dispatchEvent(down);
		handleChatSearchArrowDown(down);
		expect(down.defaultPrevented).toBe(false);
	});
});
