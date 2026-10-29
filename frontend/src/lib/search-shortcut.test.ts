// 全局搜索快捷键决策单测（2026-10-29 新增 ⌘K 功能；纯函数分支全覆盖）。
import { describe, it, expect } from 'vitest';
import { resolveSearchShortcut, type SearchShortcutContext } from './search-shortcut';

const baseCtx: SearchShortcutContext = {
	pathname: '/',
	searchPath: '/search',
	dialogOpen: false,
	editing: false
};

function ev(key: string, meta = true, ctrl = false) {
	return { key, metaKey: meta, ctrlKey: ctrl };
}

describe('resolveSearchShortcut', () => {
	it('⌘K 在其他页面 → 跳转搜索', () => {
		expect(resolveSearchShortcut(ev('k'), baseCtx)).toBe('goto-search');
	});

	it('Ctrl+K 同样生效（Windows/Linux）', () => {
		expect(resolveSearchShortcut(ev('k', false, true), baseCtx)).toBe('goto-search');
	});

	it('大写 K（Shift+⌘K）也识别', () => {
		expect(resolveSearchShortcut(ev('K'), baseCtx)).toBe('goto-search');
	});

	it('不带 ⌘/Ctrl 的 k → 不处理', () => {
		expect(resolveSearchShortcut(ev('k', false, false), baseCtx)).toBe('none');
	});

	it('其他组合键 → 不处理', () => {
		expect(resolveSearchShortcut(ev('j'), baseCtx)).toBe('none');
		expect(resolveSearchShortcut(ev('k', false, false), baseCtx)).toBe('none');
	});

	it('弹层打开时让位给弹层', () => {
		expect(resolveSearchShortcut(ev('k'), { ...baseCtx, dialogOpen: true })).toBe('none');
	});

	it('已在搜索页 → 聚焦输入框（即使焦点在表单控件内）', () => {
		expect(
			resolveSearchShortcut(ev('k'), { ...baseCtx, pathname: '/search', editing: true })
		).toBe('focus-search');
	});

	it('非搜索页且正在编辑表单 → 不劫持（避免未保存输入被带走）', () => {
		expect(resolveSearchShortcut(ev('k'), { ...baseCtx, editing: true })).toBe('none');
	});

	it('弹层优先于搜索页判断（弹层内按 ⌘K 不聚焦背景输入框）', () => {
		expect(
			resolveSearchShortcut(ev('k'), { ...baseCtx, pathname: '/search', dialogOpen: true })
		).toBe('none');
	});
});
