// 搜索页 ↑↓ 键盘导航的纯决策逻辑（DOM 无关，便于单测覆盖）。
//
// 契约（R6「搜索页 ↑↓ 焦点移动」+ 2026-10-31 修正）：
// - 候选集 = 正文内容区里的真实链接/按钮（页头返回钮、Tab 栏、输入框内清空钮都不在内）；
// - 焦点在候选集之外（如搜索框本身）时：↓ 进入候选区首个条目，↑ 不做跨区跳转
//   （原实现 ↑ 直接落到最后一个候选，视觉上等于跳到页底）；
// - 候选集内：↑↓ 逐项移动，两端收敛（不循环）；
// - 候选集为空时交还浏览器默认行为。

export type SearchNavKey = 'ArrowUp' | 'ArrowDown';

export function isSearchNavKey(key: string): key is SearchNavKey {
	return key === 'ArrowUp' || key === 'ArrowDown';
}

/**
 * 计算下一个应聚焦的候选下标。
 * @param currentIndex 当前焦点在候选集中的下标；-1 表示焦点不在候选集内
 * @param count 候选集长度
 * @returns 下一个下标；null 表示本次不接管（不 preventDefault，保留浏览器默认行为）
 */
export function nextSearchNavIndex(currentIndex: number, count: number, key: SearchNavKey): number | null {
	if (count <= 0) return null;
	if (currentIndex < 0) return key === 'ArrowDown' ? 0 : null;
	const delta = key === 'ArrowDown' ? 1 : -1;
	return Math.min(count - 1, Math.max(0, currentIndex + delta));
}
