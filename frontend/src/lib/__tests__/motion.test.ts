import { afterEach, describe, expect, it, vi } from 'vitest';
import { readFileSync } from 'node:fs';
import { MOTION, contentFade, enterOnce, motionDuration, playPageEnter, playPop, pop, popOn, prefersReducedMotion, toastExit } from '../motion';

// 源码级契约（jsdom 无法真实合成动画）：CSS 关键帧/时长必须与 lib/motion.ts 同源，
// 且 reduced-motion 压制不得回归。R16 跟手微调后锁定 120ms / 110ms。
const utilitiesCss = readFileSync('src/styles/theme/utilities.css', 'utf8');
function cssRuleBody(anchor: string): string {
	const idx = utilitiesCss.indexOf(anchor);
	if (idx < 0) throw new Error('未找到 CSS 规则锚点：' + JSON.stringify(anchor));
	const open = utilitiesCss.indexOf('{', idx);
	const close = utilitiesCss.indexOf('}', open);
	if (open < 0 || close < 0) throw new Error('规则块不完整：' + anchor);
	return utilitiesCss.slice(open + 1, close);
}

function mockReduced(reduced: boolean) {
	vi.stubGlobal('matchMedia', (query: string) => ({
		matches: reduced && query.includes('prefers-reduced-motion: reduce'),
		media: query,
		onchange: null,
		addListener: () => {},
		removeListener: () => {},
		addEventListener: () => {},
		removeEventListener: () => {},
		dispatchEvent: () => false
	}));
}

afterEach(() => {
	vi.unstubAllGlobals();
	document.body.innerHTML = '';
});

describe('prefersReducedMotion / motionDuration', () => {
	it('matchMedia 不可用时按不减弱处理', () => {
		expect(prefersReducedMotion()).toBe(false);
		expect(motionDuration(MOTION.enter)).toBe(120);
	});

	it('reduced 时 duration=0，正常时返回原值', () => {
		mockReduced(true);
		expect(prefersReducedMotion()).toBe(true);
		expect(motionDuration(MOTION.pop)).toBe(0);
		mockReduced(false);
		expect(motionDuration(MOTION.pop)).toBe(220);
	});
});

describe('pop action', () => {
	it('正常：false→true 添加 .state-pop，animationend 后移除', () => {
		mockReduced(false);
		const node = document.createElement('span');
		const action = pop(node, false);
		action.update(true);
		expect(node.classList.contains('state-pop')).toBe(true);
		node.dispatchEvent(new Event('animationend'));
		expect(node.classList.contains('state-pop')).toBe(false);
		action.destroy?.();
	});

	it('reduced：不添加动画类（直接终态）', () => {
		mockReduced(true);
		const node = document.createElement('span');
		const action = pop(node, false);
		action.update(true);
		expect(node.classList.contains('state-pop')).toBe(false);
	});

	it('取消激活（true→false）不播放 pop', () => {
		mockReduced(false);
		const node = document.createElement('span');
		const action = pop(node, true);
		action.update(false);
		expect(node.classList.contains('state-pop')).toBe(false);
		action.destroy?.();
	});
});

describe('playPageEnter', () => {
	it('正常：加 .page-enter 并在动画后移除兜底', () => {
		mockReduced(false);
		const node = document.createElement('div');
		playPageEnter(node);
		expect(node.classList.contains('page-enter')).toBe(true);
		node.dispatchEvent(new Event('animationend'));
		expect(node.classList.contains('page-enter')).toBe(false);
	});

	it('reduced：不添加 .page-enter', () => {
		mockReduced(true);
		const node = document.createElement('div');
		playPageEnter(node);
		expect(node.classList.contains('page-enter')).toBe(false);
	});
});

describe('toastExit', () => {
	it('正常 duration=150，reduced duration=0', () => {
		mockReduced(false);
		const node = document.createElement('div');
		const normal = toastExit(node) as { duration: number };
		expect(normal.duration).toBe(150);
		mockReduced(true);
		const reduced = toastExit(node) as { duration: number };
		expect(reduced.duration).toBe(0);
	});
});

describe('popOn / playPop（点击驱动，避免数据同步误播）', () => {
	it('正常：trigger 变化才添加 .state-pop；初始不播', () => {
		mockReduced(false);
		const node = document.createElement('span');
		const action = popOn(node, 0);
		expect(node.classList.contains('state-pop')).toBe(false);
		action.update(0);
		expect(node.classList.contains('state-pop')).toBe(false);
		action.update(1);
		expect(node.classList.contains('state-pop')).toBe(true);
		node.dispatchEvent(new Event('animationend'));
		expect(node.classList.contains('state-pop')).toBe(false);
		action.destroy?.();
	});
	it('reduced：trigger 变化不添加动画类', () => {
		mockReduced(true);
		const node = document.createElement('span');
		const action = popOn(node, 0);
		action.update(1);
		expect(node.classList.contains('state-pop')).toBe(false);
	});
	it('playPop：reduced 直接跳过', () => {
		mockReduced(true);
		const node = document.createElement('span');
		playPop(node);
		expect(node.classList.contains('state-pop')).toBe(false);
	});
});

describe('contentFade（骨架→内容）', () => {
	it('true→false 播 .content-fade；初始 false 不播', () => {
		mockReduced(false);
		const node = document.createElement('div');
		const action = contentFade(node, false);
		expect(node.classList.contains('content-fade')).toBe(false);
		action.update(false);
		expect(node.classList.contains('content-fade')).toBe(false);
		const node2 = document.createElement('div');
		const a2 = contentFade(node2, true);
		a2.update(false);
		expect(node2.classList.contains('content-fade')).toBe(true);
		node2.dispatchEvent(new Event('animationend'));
		expect(node2.classList.contains('content-fade')).toBe(false);
	});
	it('reduced：不播', () => {
		mockReduced(true);
		const node = document.createElement('div');
		const action = contentFade(node, true);
		action.update(false);
		expect(node.classList.contains('content-fade')).toBe(false);
	});
});

describe('CSS 动效与 JS 时长同源（R16，防漂移）', () => {
	it('page-enter 时长 = MOTION.enter，起点不是接近空白（opacity ≥ 0.5）', () => {
		expect(cssRuleBody('\n  .page-enter {')).toContain(MOTION.enter + 'ms');
		const keyframes = utilitiesCss.slice(utilitiesCss.indexOf('@keyframes cst-page-enter'));
		expect(keyframes).toMatch(/from\s*\{\s*opacity:\s*0\.[5-9]/);
		expect(keyframes).toMatch(/translateY\(2px\)/);
	});

	it('content-fade 时长 = MOTION.contentFade', () => {
		expect(cssRuleBody('\n  .content-fade {')).toContain(MOTION.contentFade + 'ms');
	});

	it('reduced-motion 仍直接压制动画并禁用平滑滚动（不得回归）', () => {
		expect(utilitiesCss).toMatch(/@media \(prefers-reduced-motion: reduce\)/);
		expect(utilitiesCss).toMatch(/animation-duration: 0\.01ms !important/);
		expect(utilitiesCss).toMatch(/scroll-behavior: auto !important/);
	});
});

describe('enterOnce（评论新增）', () => {
	it('active=true 播一次；reduced 不播', () => {
		mockReduced(false);
		const node = document.createElement('div');
		enterOnce(node, true);
		expect(node.classList.contains('comment-enter')).toBe(true);
		mockReduced(true);
		const node2 = document.createElement('div');
		enterOnce(node2, true);
		expect(node2.classList.contains('comment-enter')).toBe(false);
	});
});
