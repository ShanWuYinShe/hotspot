// 性能契约回归测试（2026-10-28）：防止把「全屏氛围动画 / 批量 will-change /
// 压图徽章的 backdrop-filter」加回来——首页实测这三项会让 GPU 进程在页面空闲时
// 持续工作（idle 24.6%、滚动 19.3%），修复后 idle 0%、滚动 14.9% → 6.1%。
// 说明：这是源码级契约（CSS 无法在 jsdom 里真实合成），断言规则块内容而非渲染结果。
import { describe, it, expect } from 'vitest';
import { existsSync, readFileSync } from 'node:fs';

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

// 低对比全屏渐变在 8-bit sRGB 下会量化成 2–3 级色阶（暗底上肉眼呈竖条纹）；
// 氛围层用一层静态细颗粒做抖动打断台阶。两条硬约束：
//  ① 必须静态（全屏噪声层若带动画 = 持续合成，性能红线）；
//  ② 不得用 mix-blend-mode —— .ambient-backdrop 是 z-index:-1 的隔离合成层，
//     混合背景板是层内透明渐变，soft-light 会退化成正片叠底式 source-over，
//     实测把整页提亮 5 级（暗底被冲淡）。
describe('氛围颗粒抖动层（banding 抑制）', () => {
	it('颗粒层静态、走 token 透明度、不使用 mix-blend-mode', () => {
		const body = ruleBody('\n  .ambient-backdrop::after {');
		expect(body).toMatch(/opacity:\s*var\(--cst-ambient-grain-opacity\)/);
		expect(body).not.toMatch(/animation\s*:/);
		expect(body).not.toMatch(/mix-blend-mode/);
	});

	it('颗粒纹理资产存在且只在氛围装饰层引用', () => {
		expect(css).toContain("url('/textures/ambient-grain.png')");
		expect(existsSync('static/textures/ambient-grain.png')).toBe(true);
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

// R18（2026-10-29）：快速跨卡晃动时 GPU/CPU 飙升的两个根因——
//  ① 整卡覆盖按钮的 background-color 过渡 = 整卡面积逐帧重绘；
//  ② 卡片 hover 层反复提升/销毁并重栅格化。
// 契约：底色改由 [data-cover]::before 的 opacity 承载；卡片/配图在 fine pointer
// 下常驻合成层（仅这两类卡面允许，且必须限定在 hover 媒体查询内）。
describe('卡片 hover 只走合成器属性（R18）', () => {
	it('覆盖按钮 hover 底色改为伪元素 opacity 过渡（不再让按钮自身背景过渡）', () => {
		const tint = ruleBody('\n[data-cover]::before {');
		expect(tint).toMatch(/background\s*:/);
		expect(tint).toMatch(/opacity\s*:\s*0/);
		expect(tint).toMatch(/pointer-events\s*:\s*none/);
		expect(tint).toMatch(/transition\s*:\s*opacity/);
		expect(ruleBody('\n  [data-cover]:hover {')).toMatch(/background-color\s*:\s*transparent/);
		// 2026-10-31：图片内容卡（.card-hover）已有上浮 + 图片缩放，不再叠整卡底色；
		// opt-out 必须存在、且排在上面的全局规则之后（同层无 layer 区，靠特异性覆盖）。
		expect(ruleBody('\n  .card-hover [data-cover]:hover::before {')).toMatch(/opacity\s*:\s*0/);
		expect(css.indexOf('.card-hover [data-cover]:hover::before')).toBeGreaterThan(
			css.indexOf('[data-cover]:hover::before')
		);
	});

	it('卡片与配图在 fine pointer 下常驻合成层；基础规则不带 will-change', () => {
		expect(ruleBody('\n  .card-hover {')).not.toMatch(/will-change/);
		expect(ruleBody('\n    .card-hover {')).toMatch(/will-change\s*:\s*transform/);
		expect(ruleBody('\n  .image-hover-frame__image {')).not.toMatch(/will-change/);
		expect(ruleBody('\n    .image-hover-frame__image {')).toMatch(/will-change\s*:\s*transform/);
	});
});
