// Toast store — 轻量全局反馈层（runes 模块单例）。
// 用法：showToast('已加入群组', 'success')；视图由 ToastViewport 渲染，
// 容器带 aria-live="polite"，供屏幕阅读器播报；自动 3.5s 消失，最多保留 3 条。

export type ToastKind = 'info' | 'success' | 'danger';

export interface ToastItem {
	id: number;
	message: string;
	kind: ToastKind;
}

let items = $state<ToastItem[]>([]);
let nextId = 1;
const DURATION_MS = 3500;
const MAX_VISIBLE = 3;

export function showToast(message: string, kind: ToastKind = 'info'): void {
	const id = nextId++;
	items = [...items.slice(-(MAX_VISIBLE - 1)), { id, message, kind }];
	setTimeout(() => dismissToast(id), DURATION_MS);
}

export function dismissToast(id: number): void {
	items = items.filter((t) => t.id !== id);
}

export const toast = {
	get items() {
		return items;
	}
};
