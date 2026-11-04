// 「随便逛逛」语义契约（源码级守卫）：全站唯一语义 = 跳过登录/注册、去可浏览内容继续逛，
// **不创建任何账号**。历史上两个入口分叉过：
//   - 整页登录引导面板（NeedsAuthNotice）：「先随便逛逛」→ 回首页浏览；
//   - 登录拦截弹层（LoginPromptSheet 内嵌 AuthForm）：「随便逛逛（访客）」→ 却切到访客注册表单。
// 本用例锁死统一后的口径：
//   1) 弹层提供「先随便逛逛」，动作是关闭弹层（过 onClose，auth store 会同时清掉待执行操作），
//      留在当前公开页继续浏览，而不是跳首页；
//   2) 访客注册（临时账号）入口统一叫「访客模式」，禁止再命名「随便逛逛」；
//   3) 整页引导面板保持「先随便逛逛」→ 默认回首页（该页已被登录墙挡住）。
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';

const here = dirname(fileURLToPath(import.meta.url));
const read = (rel: string) => readFileSync(join(here, rel), 'utf8');

describe('「随便逛逛」语义契约', () => {
	it('登录弹层：提供「先随便逛逛」，关闭弹层留在当前页（不跳首页、不注册）', () => {
		const src = read('../LoginPromptSheet.svelte');
		expect(src).toContain('先随便逛逛');
		expect(src).toContain('onclick={onClose}');
		// 弹层挂在公开页上，「逛逛」= 关掉弹层继续在当前位置浏览，不做跳转
		expect(src).not.toContain('goto(ROUTES.home)');
		// 弹层里不能出现「随便逛逛」与注册动作绑定的旧文案
		expect(src).not.toContain('随便逛逛（访客）');
	});

	it('访客注册入口统一叫「访客模式」，不再叫「随便逛逛」', () => {
		const src = read('../AuthForm.svelte');
		expect(src).toContain('访客模式');
		expect(src).not.toContain('随便逛逛（访客）');
		expect(src).not.toMatch(/>\s*随便逛逛/);
	});

	it('整页登录引导面板：「先随便逛逛」默认回可浏览的首页', () => {
		const src = read('../NeedsAuthNotice.svelte');
		expect(src).toContain("secondaryLabel = '先随便逛逛'");
		expect(src).toContain('goto(ROUTES.home)');
	});
});
