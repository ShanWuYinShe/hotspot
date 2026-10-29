// 全局搜索快捷键（⌘K / Ctrl+K）的纯决策逻辑：便于单测覆盖新功能分支，
// 也把「何时让位」的规则从 layout 里抽出来（弹层打开 / 非搜索页表单编辑中）。
//
// 说明：调用方负责在 action 非 none 时 preventDefault；goto-search 分支在导航后
// 还需 rAF 里显式聚焦搜索框（RoutedContentViewport 的 afterNavigate 会 focus main，
// 与搜索页 autofocus 存在时序竞争）。

export interface SearchShortcutEvent {
	metaKey: boolean;
	ctrlKey: boolean;
	key: string;
}

export interface SearchShortcutContext {
	/** 当前路径（客户端路由 pathname） */
	pathname: string;
	/** 搜索页路径（ROUTES.search） */
	searchPath: string;
	/** 是否有弹层打开（[role="dialog"]） */
	dialogOpen: boolean;
	/** 焦点是否在可编辑控件内（input/textarea/contenteditable） */
	editing: boolean;
}

export type SearchShortcutAction = 'none' | 'focus-search' | 'goto-search';

/**
 * 解析 ⌘K/Ctrl+K 应执行的动作：
 * - 非 ⌘/Ctrl+K → none；
 * - 弹层打开 → none（让位给弹层）；
 * - 已在搜索页 → focus-search（聚焦并全选输入框）；
 * - 其他页面且焦点在表单控件内 → none（避免把未保存输入带走）；
 * - 其他页面 → goto-search。
 */
export function resolveSearchShortcut(
	e: SearchShortcutEvent,
	ctx: SearchShortcutContext
): SearchShortcutAction {
	if (!(e.metaKey || e.ctrlKey) || e.key.toLowerCase() !== 'k') return 'none';
	if (ctx.dialogOpen) return 'none';
	if (ctx.pathname === ctx.searchPath) return 'focus-search';
	if (ctx.editing) return 'none';
	return 'goto-search';
}
