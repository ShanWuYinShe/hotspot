// 性能契约回归测试（2026-10-28）：防止把「全屏氛围动画 / 批量 will-change /
// 压图徽章的 backdrop-filter」加回来——首页实测这三项会让 GPU 进程在页面空闲时
// 持续工作（idle 24.6%、滚动 19.3%），修复后 idle 0%、滚动 14.9% → 6.1%。
// 说明：这是源码级契约（CSS 无法在 jsdom 里真实合成），断言规则块内容而非渲染结果。
import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';

// vitest 下 import.meta.url 不是 file://（jsdom 环境），直接用仓库内相对路径（cwd=frontend）。
const css = readFileSync('src/styles/theme/utilities.css', 'utf8');

/** 按「行首锚点」截取一条规则的声明体（避免后代选择器同时命中）。 */
function ruleBody(anchor: string): string {
	const idx = css.indexOf(anchor);
	if (idx < 0) throw new Error('未找到 CSS 规则锚点：' + JSON.stringify(anchor));
	const open = css.indexOf('{', idx);
	const close = css.indexOf('}', open);
	if (open < 0 || close < 0) throw new Error('规则块不完整：' + anchor);
	return css.slice(open + 1, close);
}

describe('氛围动画默认关闭（GPU 空闲占用）', () => {
	it.each(['\n  .ambient-glow--a {', '\n  .ambient-glow--b {', '\n  .ambient-stars {'])(
		'基础规则 %s 不得带无限动画',
		(anchor) => {
			expect(ruleBody(anchor)).not.toMatch(/animation\s*:/);
		}
	);

	it('动画只能通过 .ambient-backdrop--motion 显式 opt-in', () => {
		expect(ruleBody('\n  .ambient-backdrop--motion .ambient-glow--a {')).toMatch(/animation\s*:/);
		expect(ruleBody('\n  .ambient-backdrop--motion .ambient-glow--b {')).toMatch(/animation\s*:/);
		expect(ruleBody('\n  .ambient-backdrop--motion .ambient-stars {')).toMatch(/animation\s*:/);
	});
});

describe('合成层纪律', () => {
	it('.hover-action 不得常驻 will-change（首页 48 个实例会各占一个合成层）', () => {
		expect(ruleBody('\n  .hover-action {')).not.toMatch(/will-change/);
	});

	it('.media-badge 不得使用 backdrop-filter（首页 18 个实例逐帧重算模糊）', () => {
		expect(ruleBody('\n.media-badge {')).not.toMatch(/backdrop-filter/);
	});
});
