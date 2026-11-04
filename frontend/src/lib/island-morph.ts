// 灵动岛交接动效（入口/出口共用同一份实现）。
//
// 目标：页面锚点（标题/meta、首页分段器、详情页身份）滚出视口时，不再「凭空弹出」一颗胶囊，
// 而是让这颗胶囊从锚点最后可见的位置收进 52px 顶栏中央——看起来就是「被滚走的 UI 被灵动岛
// 吸收」；滚回去时按同一路径反向吐出。只有动画这样走，信息交接才有连续感（对应
// docs/design-system.md §22「何时出现」）。
//
// 实现取向：
//   - 用 Svelte transition 挂在胶囊根节点上，进场/退场复用同一个 css 函数：Svelte 退场时
//     t 从 1 反向跑到 0，天然得到「原路返回」，不需要第二套关键帧；
//   - 只动 transform + opacity（合成器属性），不触发布局；胶囊宽度/内边距全程不变，
//     连滚带飞也不会引起顶栏抖动；
//   - 缩放比按来源矩形钳制（详情页锚点与胶囊长宽比差很大时不至于把胶囊拉成异形），
//     超出部分交给两端淡入淡出承担；
//   - reduced-motion 直接 0ms 到终态（与 motion.ts 的全局约定一致）。
import type { TransitionConfig } from 'svelte/transition';
import { cubicOut } from 'svelte/easing';
import { prefersReducedMotion } from './motion';

/** 锚点在视口中的矩形（getBoundingClientRect 的快照；必须按值传递，不能持有 DOM 引用）。 */
export interface IslandRect {
	x: number;
	y: number;
	width: number;
	height: number;
}

/** 收进/吐出时长：300ms——比 hover（220ms）略长，让「飞行」被感知，又不拖慢连续滚动。 */
const FLIGHT_MS = 300;
/** 无来源时的兜底淡入（直接刷新/无锚点矩形）。 */
const FADE_MS = 160;

const clamp = (value: number, min: number, max: number): number =>
	Math.min(max, Math.max(min, value));

/** 把 getBoundingClientRect 结果收敛成纯值快照。 */
export function islandRectOf(el: Element | null | undefined): IslandRect | null {
	if (!el) return null;
	const rect = el.getBoundingClientRect();
	if (rect.width <= 0 || rect.height <= 0) return null;
	return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
}

/**
 * 灵动岛交接 transition。
 * 用法：`<div transition:islandMorph={{ source: morphSource }}>`
 * source 由调用方在「显隐判定」那一刻抓取（锚点仍在原处时的矩形）。
 */
export function islandMorph(
	node: HTMLElement,
	params: { getSource?: () => IslandRect | null } = {}
): TransitionConfig {
	if (prefersReducedMotion()) return { duration: 0 };

	const target = node.getBoundingClientRect();
	// 用 getter 而不是值：退场时父组件会在同一帧内先把来源矩形写进 store 再卸载岛，
	// 值传递拿到的是上一帧的旧矩形（会退化成纯淡出）；getter 在动画真正开始时才读取。
	const source = params.getSource?.() ?? null;

	if (!source || target.width <= 0 || target.height <= 0) {
		// 没有可信来源：退化成短淡入，仍比直接闪现柔和。
		return {
			duration: FADE_MS,
			easing: cubicOut,
			css: (t) => `opacity:${clamp(t * 2.4, 0, 1)};`
		};
	}

	// 中心对齐的位移 + 长宽比缩放（transform-origin 默认 50% 50%，与此处的中心差值同口径）。
	const dx = source.x + source.width / 2 - (target.left + target.width / 2);
	const dy = source.y + source.height / 2 - (target.top + target.height / 2);
	const scaleX = clamp(source.width / target.width, 0.55, 1.4);
	const scaleY = clamp(source.height / target.height, 0.7, 1.3);

	return {
		duration: FLIGHT_MS,
		easing: cubicOut,
		css: (t, u) => {
			const tx = dx * u;
			const ty = dy * u;
			const sx = 1 + (scaleX - 1) * u;
			const sy = 1 + (scaleY - 1) * u;
			// t: 0（在锚点处）→ 1（就位）；u 为其反向余量。进场前 40% 完成淡入，
			// 退场时同一公式自动在最后 40% 淡出。
			const opacity = clamp(t * 2.5, 0, 1);
			return `transform:translate3d(${tx}px, ${ty}px, 0) scale(${sx}, ${sy});opacity:${opacity};`;
		}
	};
}
