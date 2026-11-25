// 设置页分段控件（role="radiogroup" 的三选一卡片：主题/明暗/底图）的纯决策逻辑。
// 抽成纯函数便于单测；页面侧只负责 DOM 焦点与选中，不重复实现方向键语义。
//
// WAI-ARIA radio group：方向键在组内移动焦点并即时选中（本文件只返回目标下标），
// Home/End 跳到首/末项；不支持 RTL 方向反转（页面暂无 RTL 需求）。

export type RovingRadioKey =
	| 'ArrowRight'
	| 'ArrowDown'
	| 'ArrowLeft'
	| 'ArrowUp'
	| 'Home'
	| 'End';

/**
 * 返回方向键应移动到的下标；非方向键返回 null（调用方不 preventDefault）。
 * 边界循环：首项 ← 末项、末项 → 首项（WAI-ARIA 推荐行为，三选一卡片不会「卡住」）。
 */
export function resolveRovingRadioIndex(currentIndex: number, count: number, key: string): number | null {
	if (count <= 0) return null;
	const index = Math.min(Math.max(currentIndex, 0), count - 1);
	switch (key) {
		case 'ArrowRight':
		case 'ArrowDown':
			return (index + 1) % count;
		case 'ArrowLeft':
		case 'ArrowUp':
			return (index - 1 + count) % count;
		case 'Home':
			return 0;
		case 'End':
			return count - 1;
		default:
			return null;
	}
}
