// 锚点矩形与几何差工具：页头仍用 islandRectOf / islandTextRectOf 取交接锚点矩形；
// 飞行几何（islandFlight）当前无调用方（待清理）。
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
 * 严格施加视口天花板限位（缺省 minViewportY=14px），确保流体微胶囊始终在可见视口内优雅滑翔，绝不飞出屏幕上边界。
 */
export function computeFlightCurve(
	source: IslandRect,
	target: IslandRect,
	arcLift = 22,
	minViewportY = 14
): FlightCurve {
	const startX = source.x + source.width / 2;
	// 若起点为标题且因快速滚动导致负视口坐标，回退至顶栏下沿安全位置（56px）
	const startY = source.y < 0 ? 56 + source.height / 2 : source.y + source.height / 2;
	const targetX = target.x + target.width / 2;
	const targetY = target.y < 0 ? 26 : target.y + target.height / 2;

	const ctrlX = startX + (targetX - startX) * 0.38;
	const rawCtrlY = Math.min(startY, targetY) - arcLift;
	const ctrlY = Number.isFinite(minViewportY) ? Math.max(minViewportY, rawCtrlY) : rawCtrlY;

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

/** 算出贝塞尔曲线在参数 t 处的瞬时切线倾角（角度制 deg，限幅在 ±18° 内，使微胶囊随轨道自然流体微倾） */
export function evalBezierAngle(curve: FlightCurve, t: number): number {
	const clampedT = Math.min(1, Math.max(0, t));
	// 一阶导数（瞬时速度矢量）：B'(t) = 2*(1-t)*(p1 - p0) + 2*t*(p2 - p1)
	const dx = 2 * (1 - clampedT) * (curve.p1.x - curve.p0.x) + 2 * clampedT * (curve.p2.x - curve.p1.x);
	const dy = 2 * (1 - clampedT) * (curve.p1.y - curve.p0.y) + 2 * clampedT * (curve.p2.y - curve.p1.y);
	const rad = Math.atan2(dy, dx);
	const deg = (rad * 180) / Math.PI;
	return Math.max(-18, Math.min(18, deg));
}
