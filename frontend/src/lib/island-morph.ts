// 锚点矩形与几何差（灵动岛交接共用的小工具）。
//
// 交接动效本身在 lib/island-handoff.ts：进度 p 由滚动位置算出、逐帧写样式，
// 滚动停在哪交接就停在哪，反向滚动完全可逆（对齐 iOS 大标题收进导航栏的交互模型）。
// 本文件只保留两件事，便于单测锁定几何契约：
//   - 把 DOM 锚点变成「纯值矩形」快照（不持有 DOM 引用，可跨帧比较/插值）；
//   - 算出「锚点矩形 ↔ 胶囊泊位」的中心位移与长宽比缩放。
//
// 为什么要 Range 取文字矩形：页头 h1 是块级元素（占满整列宽），直接取元素 rect 会让
// 匣从一块 786px 宽的「巨幕」起飞——那不是用户看到的标题。取文字盒子，起跳点才与肉眼一致；
// jsdom/空节点取不到时回退元素矩形。

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

/** 缩放保护区间：锚点与胶囊长宽比极端时不至于把胶囊压成一条线/拉成巨幕。 */
const SCALE_X_MIN = 0.2;
const SCALE_X_MAX = 2.2;
const SCALE_Y_MIN = 0.45;
const SCALE_Y_MAX = 2.0;

const clamp = (value: number, min: number, max: number): number =>
	Math.min(max, Math.max(min, value));

/** 把 getBoundingClientRect 结果收敛成纯值快照（零尺寸视为取不到）。 */
export function islandRectOf(el: Element | null | undefined): IslandRect | null {
	if (!el) return null;
	const rect = el.getBoundingClientRect();
	if (rect.width <= 0 || rect.height <= 0) return null;
	return { x: rect.x, y: rect.y, width: rect.width, height: rect.height };
}

/** 取元素「文字内容」的绘制矩形（Range 包围盒）；取不到时回退元素矩形。 */
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

/** 纯几何：算出「从锚点矩形 ↔ 胶囊矩形」的中心位移与长宽比缩放。 */
export function islandFlight(target: IslandRect, source: IslandRect): IslandFlight {
	const dx = source.x + source.width / 2 - (target.x + target.width / 2);
	const dy = source.y + source.height / 2 - (target.y + target.height / 2);
	const scaleX = clamp(source.width / Math.max(1, target.width), SCALE_X_MIN, SCALE_X_MAX);
	const scaleY = clamp(source.height / Math.max(1, target.height), SCALE_Y_MIN, SCALE_Y_MAX);
	return { dx, dy, scaleX, scaleY, distance: Math.hypot(dx, dy) };
}

/** 二维平面坐标点 */
export interface BezierPoint {
	x: number;
	y: number;
}

/** 贝塞尔抛物线飞行轨道：起点、拱形控制点、终点 */
export interface FlightCurve {
	p0: BezierPoint;
	p1: BezierPoint;
	p2: BezierPoint;
	arcLift: number;
}

/**
 * 算出从「锚点矩形中心」到「顶栏灵动岛泊位中心」的贝塞尔拱形飞行曲线。
 * 控制点向上微拱（缺省 48px），使流体粒子呈现如露珠轻盈升空的视觉抛物线。
 */
export function computeFlightCurve(source: IslandRect, target: IslandRect, arcLift = 48): FlightCurve {
	const startX = source.x + source.width / 2;
	const startY = source.y + source.height / 2;
	const targetX = target.x + target.width / 2;
	const targetY = target.y + target.height / 2;

	const ctrlX = startX + (targetX - startX) * 0.35;
	const ctrlY = Math.min(startY, targetY) - arcLift;

	return {
		p0: { x: startX, y: startY },
		p1: { x: ctrlX, y: ctrlY },
		p2: { x: targetX, y: targetY },
		arcLift
	};
}

/** 求二次贝塞尔曲线在参数 t ∈ [0, 1] 处的瞬时坐标 */
export function evalBezier(curve: FlightCurve, t: number): BezierPoint {
	const clampedT = Math.min(1, Math.max(0, t));
	const inv = 1 - clampedT;
	return {
		x: inv * inv * curve.p0.x + 2 * inv * clampedT * curve.p1.x + clampedT * clampedT * curve.p2.x,
		y: inv * inv * curve.p0.y + 2 * inv * clampedT * curve.p1.y + clampedT * clampedT * curve.p2.y
	};
}
