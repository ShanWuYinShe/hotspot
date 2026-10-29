// 输入族表面契约（2026-10-29 重构）：锁住 .input-surface 的统一表面语言，
// 防止回退到「双层描边 + 双向内凹阴影 + vibrant 毛玻璃」的旧实现，也防止主题文件
// 再次按主题覆写输入表面（输入表面必须单一来源）。
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';

// cwd = frontend（与 perf-contract.test.ts 同口径）。
const utilities = readFileSync('src/styles/theme/utilities.css', 'utf8');
const semantic = readFileSync('src/styles/theme/semantic.css', 'utf8');
const base = readFileSync('src/styles/theme/base.css', 'utf8');
const themeFiles = ['vibrant', 'forest', 'inkwash'].map(
	(t) => [t, readFileSync(`src/styles/theme/themes/${t}.css`, 'utf8')] as const
);

/** 按「行首锚点」截取一条规则的声明体（避免后代选择器同时命中）。 */
function ruleBody(anchor: string): string {
	const idx = utilities.indexOf(anchor);
	if (idx < 0) throw new Error('未找到 CSS 规则锚点：' + JSON.stringify(anchor));
	const open = utilities.indexOf('{', idx);
	const close = utilities.indexOf('}', open);
	if (open < 0 || close < 0) throw new Error('规则块不完整：' + anchor);
	return utilities.slice(open + 1, close);
}

describe('input-surface 静态表面', () => {
	it('单层边界 + 字段表面填充，不再有内缘描边/双向内凹阴影', () => {
		const body = ruleBody('\n  .input-surface {');
		expect(body).toMatch(/background:\s*var\(--cst-field-bg\)\s*!important/);
		expect(body).toMatch(/border:\s*var\(--cst-border-width-hairline\) solid var\(--cst-field-border\)/);
		expect(body).not.toMatch(/inset 0 0 0 1px/);
		expect(body).not.toMatch(/--shadow-inset/);
		expect(body).not.toMatch(/backdrop-filter/);
		expect(body).toMatch(/transition:[\s\S]*border-color/);
	});

	it('hover 边界加深（补齐此前空缺的 hover 态）；禁用/聚焦时让位', () => {
		const body = ruleBody('\n    .input-surface:not(:has(:disabled)):not(:focus-within):hover,');
		expect(body).toMatch(/border-color:\s*var\(--cst-field-border-hover\)/);
		// 直接挂 .input-surface 的紧凑控件（input/select）同样排除自身 disabled；
		// 两种选择器都必须排除 :focus-within，否则指针聚焦时 hover 会盖掉 accent 焦点边。
		expect(utilities).toContain('\n    .input-surface:not(:disabled):not(:focus-within):hover {');
	});
});

describe('input-surface 焦点/错误/禁用态', () => {
	it('焦点 = accent 边界 + 内缘 1px（视觉 2px）+ 3px 柔光环', () => {
		const body = ruleBody('\n  .input-surface:focus-within {');
		expect(body).toMatch(/border-color:\s*var\(--cst-field-border-focus\)/);
		expect(body).toMatch(/inset 0 0 0 1px var\(--cst-field-border-focus\)/);
		expect(body).toMatch(/0 0 0 3px var\(--cst-field-ring\)/);
	});

	it('错误态 = danger 边界 + danger 光环；错误行由组件就近渲染', () => {
		const body = ruleBody("\n  .input-surface[aria-invalid='true'],");
		expect(body).toMatch(/border-color:\s*var\(--cst-status-danger\)/);
		expect(body).not.toMatch(/shadow-inset/);
		const focusBody = ruleBody(
			"\n  .input-surface[aria-invalid='true']:focus-within,"
		);
		expect(focusBody).toMatch(/var\(--cst-field-ring-danger\)/);
	});

	it('禁用态回落到弱化嵌入面', () => {
		const body = ruleBody('\n  .input-surface:has(input:disabled),');
		expect(body).toMatch(/background:\s*var\(--cst-field-bg-disabled\)/);
	});

	it('占位文字锁 text-secondary 且不叠 opacity（此前 0.6 透明度实测 2.43–4.0:1）', () => {
		const body = ruleBody('\n  .input-surface input::placeholder,');
		expect(body).toMatch(/color:\s*var\(--cst-text-secondary\)/);
		expect(body).toMatch(/opacity:\s*1/);
	});
});

describe('输入表面单一来源 / token 定义', () => {
	it('主题文件不得再按主题覆写 .input-surface（vibrant 毛玻璃已移除）', () => {
		for (const [name, css] of themeFiles) {
			expect(css, `themes/${name}.css 不应出现 .input-surface 覆写`).not.toMatch(/\.input-surface/);
		}
	});

	it('semantic.css 定义字段语义 token（颜色全部由主题派生）', () => {
		for (const token of [
			'--cst-field-bg:',
			'--cst-field-bg-disabled:',
			'--cst-field-border:',
			'--cst-field-border-hover:',
			'--cst-field-border-focus:',
			'--cst-field-ring:',
			'--cst-field-ring-danger:'
		]) {
			expect(semantic).toContain(token);
		}
	});

	it('autofill 内嵌底色与字段表面色同源', () => {
		expect(base).toMatch(/-webkit-box-shadow: 0 0 0 1000px var\(--cst-field-bg\) inset/);
	});
});
