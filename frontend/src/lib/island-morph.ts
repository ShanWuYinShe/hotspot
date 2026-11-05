// 灵动岛交接动效（入口/出口各自一份编排）。
//
// 目标：页面锚点（标题/meta、首页分段器、详情页身份/工具行）滚出视口时，不再「凭空弹出」一颗胶囊，
// 而是让这颗胶囊从锚点最后可见的位置收进 52px 顶栏中央——看起来就是「被滚走的 UI 被灵动岛
// 吸收」；滚回去时按同一路径反向吐出。只有动画这样走，信息交接才有连续感（对应
// docs/design-system.md §22「何时出现」）。
//
// 入口（in）三段式，总时长 HANDOFF_MS：
//   1) 显影（0–~35%）：容器从锚点矩形起跳，慢起、快速凝实——像页头那块 UI 自己变成胶囊；
//   2) 归位（~35–100%）：沿 S 形曲线滑入顶栏中央并收成最终尺寸；
//   3) 显字（~55–95%）：胶囊内文字/图标在飞行后段淡入——整胶囊 scale 期间正文是缩放态，
//      先隐后显才不会出现「糊字掠过顶栏」的廉价感。
//
// 出口（out）是同一路径的逆向：文字先散（前 25%）、壳随后飞回锚点并淡出，
// 与页头 150ms 的淡入错开，用户看到的是「胶囊把标题放回原位」而不是两段各飞各的。
//
// 三个细节决定「顺不顺」：
//   - **曲线不能用 cubicOut**：300ms 的 cubicOut 在头 3 帧就走完一半位移，肉眼读成「闪一下」；
//     这里用 sineInOut——慢起、中段加速、缓落，位移真正平均分布在整段时长里；
//   - **不钳制起止比例**：锚点矩形与胶囊矩形按中心对齐 + 精确缩放，起跳帧与锚点严格同框；
//     只对「极端长宽比」做宽松保护（缩到 0.2x 以下才会糊成一条线）；
//   - **只动 transform + opacity**：胶囊 contain:layout paint、面积小，不触发布局；
//     内容层用 CSS 自定义属性 --cst-island-content 单独控制，退场时内容先散、容器后散。
//
// reduced-motion 直接 0ms 到终态（与 motion.ts 的全局约定一致）。
import type { TransitionConfig } from 'svelte/transition';
import { sineInOut } from 'svelte/easing';
import { prefersReducedMotion } from './motion';

/** 锚点在视口中的矩形（getBoundingClientRect 的快照；必须按值传递，不能持有 DOM 引用）。 */
export interface IslandRect {
	x: number;
	y: number;
	width: number;
	height: number;
}

/** 锚点 → 胶囊的几何差（纯数据，便于单测）。 */
export interface IslandFlight {
	dx: number;
	dy: number;
	scaleX: number;
	scaleY: number;
	/** 中心点空间距离（px），用于断言/调试。 */
	distance: number;
}

/**
 * 入场时长：340ms。
 * 与既有 300ms「spring」档相比略长——慢起 S 形曲线的感知位移比 cubicOut 更均匀，
 * 需要多 ~40ms 才不显得赶；与 docs/design-system.md §22 的交接动效描述同源。
 */
const HANDOFF_MS = 340;
/** 出场时长：300ms（回程比入场略快，避免"送回去"拖沓）。 */
const RETURN_MS = 300;
/** 无来源矩形时的兜底淡入/淡出（直接刷新/无锚点矩形）：短到不打断阅读，又不至于硬切。 */
const FADE_MS = 160;
/** 入口：内容层开始显字的进度（缓动后 t）；0.55 之前是纯胶囊壳。 */
const CONTENT_IN = 0.55;
/** 入口：内容层显字跨度（到 t≈0.95 基本完全显现）。 */
const CONTENT_SPAN = 0.4;
/** 出口：内容层在前 25% 内散尽（文字不跟着缩回去）。 */
const CONTENT_OUT_SPAN = 0.25;

/** 缩放保护区间：锚点与胶囊长宽比极端时不至于把胶囊压成一条线/拉成巨幕。 */
const SCALE_X_MIN = 0.2;
const SCALE_X_MAX = 2.2;
const SCALE_Y_MIN = 0.45;
const SCALE_Y_MAX = 2.0;

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
 * 取元素「文字内容」的绘制矩形（Range 包围盒）。
 * 页面标题 h1 在默认页头里是块级元素（占满整列宽），直接拿 h1 的 rect 会让灵动岛
 * 从一个 786px 宽的「巨幕」起飞/落回——那不是用户看到的标题。用 Range 取文字盒子，
 * 起跳点才和肉眼看到的标题一致；取不到（jsdom/空节点）时回退元素矩形。
 */
export function islandTextRectOf(el: Element | null | undefined): IslandRect | null {
	if (!el) return null;
	if (typeof document !== 'undefined' && typeof document.createRange === 'function') {
		try {
			const range = document.createRange();
			range.selectNodeContents(el);
			const rect = range.getBoundingClientRect();
			if (rect.width > 0 && rect.height > 0) {
				return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
			}
		} catch {
			// 跨文档/空节点等异常：走元素矩形兜底
		}
	}
	return islandRectOf(el);
}

function rectOf(node: HTMLElement): IslandRect | null {
	const rect = node.getBoundingClientRect();
	if (rect.width <= 0 || rect.height <= 0) return null;
	return { x: rect.left, y: rect.top, width: rect.width, height: rect.height };
}

/**
 * 锚点矩形收敛到视口内：非吸顶锚点（integrated 页头/首页分段器）离场那一帧可能已经
 * 被顶栏裁掉一半，直接按原始坐标起跳会出现「从屏幕外飞进来」的断档感。
 * 这里把中心点钳制回视口（顶部留一点顶栏高度），保证胶囊永远从「最后可见的位置」起飞。
 */
export function sanitizeIslandSource(
	source: IslandRect | null,
	viewport: { width: number; height: number } | null = null
): IslandRect | null {
	if (!source) return null;
	const vw = viewport?.width ?? (typeof window === 'undefined' ? 0 : window.innerWidth);
	const vh = viewport?.height ?? (typeof window === 'undefined' ? 0 : window.innerHeight);
	if (!vw || !vh) return source;
	// 顶栏高 52（rem 化后随根字号放大，这里取 52px 的保守下界即可：只用于钳制，不参与布局）。
	const topLimit = 40;
	const bottomLimit = vh - 24;
	const centerY = clamp(source.y + source.height / 2, topLimit, bottomLimit);
	const centerX = clamp(source.x + source.width / 2, 24, vw - 24);
	return {
		x: centerX - source.width / 2,
		y: centerY - source.height / 2,
		width: source.width,
		height: source.height
	};
}

/** 纯几何：算出「从锚点矩形 ↔ 胶囊矩形」的中心位移与长宽比缩放。 */
export function islandFlight(target: IslandRect, source: IslandRect): IslandFlight {
	const dx = source.x + source.width / 2 - (target.x + target.width / 2);
	const dy = source.y + source.height / 2 - (target.y + target.height / 2);
	const scaleX = clamp(source.width / Math.max(1, target.width), SCALE_X_MIN, SCALE_X_MAX);
	const scaleY = clamp(source.height / Math.max(1, target.height), SCALE_Y_MIN, SCALE_Y_MAX);
	return { dx, dy, scaleX, scaleY, distance: Math.hypot(dx, dy) };
}

/** 无来源时入口退化：短淡入 + 内容同步显影。 */
function islandFadeIn(): TransitionConfig {
	return {
		duration: FADE_MS,
		easing: sineInOut,
		css: (t) => {
			const opacity = clamp(t * 2.2, 0, 1);
			const content = clamp(t * 2.4, 0, 1);
			return `--cst-island-content:${content};opacity:${opacity};`;
		}
	};
}

/**
 * 无来源时出口退化：短淡出（内容先散）。
 * 注意 Svelte 的 out transition 传入的 t 是从 1 → 0（1=当前可见态，0=已移除态），
 * 所以这里的公式与 in 相反，用 t 本身表达「还剩多少」。
 */
function islandFadeOut(): TransitionConfig {
	return {
		duration: FADE_MS,
		easing: sineInOut,
		css: (t) => {
			const opacity = clamp(t * 1.4, 0, 1);
			const content = clamp((t - 0.6) / 0.4, 0, 1);
			return `--cst-island-content:${content};opacity:${opacity};`;
		}
	};
}

/**
 * 灵动岛入场交接 transition。
 * 用法：`<div in:islandMorphIn={{ getSource: () => sourceRect }} ...>`
 * source 由调用方在「显隐判定」那一刻抓取（锚点仍在原处时的矩形）。
 */
export function islandMorphIn(
	node: HTMLElement,
	params: { getSource?: () => IslandRect | null } = {}
): TransitionConfig {
	if (prefersReducedMotion()) return { duration: 0 };
	const target = rectOf(node);
	// 用 getter 而不是值：调用方在切换显隐的同一帧里才写入来源矩形。
	const source = sanitizeIslandSource(params.getSource?.() ?? null);
	if (!source || !target) return islandFadeIn();

	const flight = islandFlight(target, source);
	return {
		duration: HANDOFF_MS,
		// 缓动曲线只作用于几何进度 t；显影/显字按 t 做分段映射，三者共用一条时间轴。
		easing: sineInOut,
		css: (t) => {
			const inv = 1 - t;
			const x = flight.dx * inv;
			const y = flight.dy * inv;
			const sx = 1 + (flight.scaleX - 1) * inv;
			const sy = 1 + (flight.scaleY - 1) * inv;
			// 显影：sqrt 曲线让胶囊在前 1/3 就凝实（慢起阶段也有可见反馈），
			// 又不会像线性那样「一半路程就全不透明」。
			const opacity = clamp(Math.sqrt(t) * 1.6, 0, 1);
			const content = clamp((t - CONTENT_IN) / CONTENT_SPAN, 0, 1);
			return `--cst-island-content:${content};transform:translate3d(${x}px, ${y}px, 0) scale(${sx}, ${sy});opacity:${opacity};`;
		}
	};
}

/**
 * 灵动岛出场交接 transition。
 * 用法：`<div out:islandMorphOut={{ getSource: () => sourceRect }} ...>`
 */
export function islandMorphOut(
	node: HTMLElement,
	params: { getSource?: () => IslandRect | null } = {}
): TransitionConfig {
	if (prefersReducedMotion()) return { duration: 0 };
	const target = rectOf(node);
	const source = sanitizeIslandSource(params.getSource?.() ?? null);
	if (!source || !target) return islandFadeOut();

	const flight = islandFlight(target, source);
	// Svelte 的 out transition 传入 t 从 1 → 0（1 = 当前可见态，0 = 已移除态）；
	// 这里把 t 换成「离场进度 p」再走与入口对称的几何，回程才真正从顶栏飞回标题位置。
	return {
		duration: RETURN_MS,
		easing: sineInOut,
		css: (t) => {
			const p = 1 - t;
			const x = flight.dx * p;
			const y = flight.dy * p;
			const sx = 1 + (flight.scaleX - 1) * p;
			const sy = 1 + (flight.scaleY - 1) * p;
			// 文字先散（离场前 25%），壳随后一路飞回锚点并淡出：回程末尾正好落在标题位置上。
			const content = clamp(1 - p / CONTENT_OUT_SPAN, 0, 1);
			const opacity = clamp(1 - p * 1.05, 0, 1);
			return `--cst-island-content:${content};transform:translate3d(${x}px, ${y}px, 0) scale(${sx}, ${sy});opacity:${opacity};`;
		}
	};
}
