// 消息会话列表 ↑↓ 键盘导航的纯决策逻辑（DOM 无关，便于单测覆盖）。
//
// 契约（对齐 /notify 的列表键盘导航，PR #242）：
// - 候选集 = 会话列表行内的覆盖按钮（`[data-list-item] button[data-cover]`）；
// - 焦点不在行内（如搜索框）时：↓ 进入首行，↑ 不做跨区跳转；
// - 行内：↑↓ 逐项移动，两端收敛（返回 null 表示不接管，保留浏览器默认行为）。
export type ChatListNavKey = 'ArrowUp' | 'ArrowDown';

export function isChatListNavKey(key: string): key is ChatListNavKey {
	return key === 'ArrowUp' || key === 'ArrowDown';
}

/** @returns 下一个应聚焦的行下标；null 表示本次不接管 */
export function nextChatRowIndex(currentIndex: number, count: number, key: ChatListNavKey): number | null {
	if (count <= 0) return null;
	if (currentIndex < 0) return key === 'ArrowDown' ? 0 : null;
	const next = currentIndex + (key === 'ArrowDown' ? 1 : -1);
	if (next < 0 || next >= count) return null;
	return next;
}

/**
 * 搜索框 ↓ 进入会话列表首行（对齐 /notify 的「搜索框 ↓ 进列表首项」口径）。
 * 输入框自身不是行内元素，因此由页面把本函数接到 SearchInput 的 onKeydown 上。
 */
export function handleChatSearchArrowDown(event: KeyboardEvent): void {
	if (!isChatListNavKey(event.key) || event.key !== 'ArrowDown' || event.defaultPrevented) return;
	if (!(event.target instanceof HTMLInputElement)) return;
	const firstRow = document.querySelector<HTMLButtonElement>('[data-list-item] button[data-cover]');
	if (!firstRow) return;
	event.preventDefault();
	firstRow.focus({ preventScroll: true });
	// jsdom 等环境没有 scrollIntoView：可选用，避免单测环境炸掉
	firstRow.scrollIntoView?.({ block: 'nearest' });
}
