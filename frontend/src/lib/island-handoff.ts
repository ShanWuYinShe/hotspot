// 页头标题 → 顶栏灵动岛：滚动驱动的交接（scroll-linked handoff）。
//
// 为什么不用「时长动画」：标题到顶栏岛是「同一个东西被滚进顶栏」的空间变化，
// 市面上做得最好的同型交互都把它绑在滚动上，而不是点一下再放一段 300ms 动画——
//  · iOS 大标题 → 导航栏标题（Large Title collapse）：标题随内容上移、渐隐，导航栏标题渐显，
//    全程由滚动位置驱动，松手即停在半途，反向滚动完全可逆；
//  · Material 3 的 container transform 强调「共享元素在两点之间连续变形」；
//  · Apple HIG Motion 的原则是「motion 跟随手势、短而精确」，不要给高频交互加独立时长动画。
// 时长动画的固有问题：开始后不跟手（用户继续滚，动画还在按自己的节奏走）、反向滚动时要等它播完，
// 观感就是「不跟手、不自然」。因此这里把交接进度 p（0→1）直接由滚动位置算出，每帧写样式。
//
// 进度 p 的语义（由调用方计算）：
//  - p=0：页头标题完整可见，岛不存在；
//  - p=1：标题已完全进入顶栏，岛完全落位；
//  - 中间：岛的位置/尺寸在「锚点矩形 → 顶栏泊位」之间插值，透明度与内容另行分段。
//
// 只动 transform / opacity / 内容层透明度；不读布局（锚点绝对位置由调用方缓存），
// 因此每帧成本与一次 transform 写入相当。reduced-motion 下退化为「过闸即切换」。
import { islandFlight, type IslandRect } from './island-morph';
import { prefersReducedMotion } from './motion';

/** 一次交接帧：滚动进度 + 锚点矩形（锚点跟手移动时每帧更新）。 */
export interface HandoffState {
	p: number;
	source: IslandRect | null;
}

type Subscriber = (state: HandoffState) => void;

let state: HandoffState = { p: 0, source: null };
const subscribers = new Set<Subscriber>();

/** 由页头/详情页在滚动估算里调用；同步通知所有订阅者（岛、页头退场层）。 */
export function publishHandoff(next: HandoffState): void {
	state = { p: Math.min(1, Math.max(0, next.p)), source: next.source };
	for (const fn of subscribers) fn(state);
}

/** 订阅交接帧；返回取消订阅函数。 */
export function subscribeHandoff(fn: Subscriber): () => void {
	subscribers.add(fn);
	return () => subscribers.delete(fn);
}

export function currentHandoff(): HandoffState {
	return state;
}

/** 路由切换/组件卸载时归零，避免新页面误读到上一页的进度。 */
export function resetHandoff(): void {
	publishHandoff({ p: 0, source: null });
}

const clamp01 = (v: number): number => Math.min(1, Math.max(0, v));

/**
 * 吸顶页头：标题不动，用滚动窗口把交接进度从 0 推到 1。
 * 纯函数便于单测锁定「窗口外 0/1、窗口内线性」。
 */
export function scrollHandoffProgress(scrollTop: number, start: number, end: number): number {
	return clamp01((scrollTop - start) / Math.max(1, end - start));
}

/**
 * 非吸顶页头：标题随内容滚进顶栏，用锚点在视口中的位置算进度。
 * 从「标题即将碰到顶栏下沿」（top = barBottom + approach）开始，
 * 到「标题完全没入顶栏」（top = barBottom - height - tail）完成。
 */
export function anchorHandoffProgress(
	top: number,
	height: number,
	barBottom: number,
	approach = 24,
	tail = 10
): number {
	const startTop = barBottom + approach;
	const endTop = barBottom - height - tail;
	return clamp01((startTop - top) / Math.max(1, startTop - endTop));
}
/** smoothstep：两端速度为 0，滚动起/停处都不会有「硬拐点」。 */
const smoothstep = (t: number): number => t * t * (3 - 2 * t);

export interface HandoffCss {
	/** 空串表示已落位（交给 CSS hover 的 transform 接管）。 */
	transform: string;
	opacity: number;
	/** 内容层显字进度（CSS 变量 --cst-island-content）。 */
	content: number;
}

/**
 * 纯函数：给定目标泊位、锚点矩形与进度，算出胶囊这一帧的样式。
 * 单独导出便于用单测锁定「p=0 贴锚点 / p=1 归位 / 单调」三条契约。
 */
export function handoffCss(target: IslandRect, source: IslandRect, p: number, reduced = false): HandoffCss {
	const flight = islandFlight(target, source);
	const startTransform = `translate3d(${flight.dx.toFixed(2)}px, ${flight.dy.toFixed(2)}px, 0) scale(${flight.scaleX.toFixed(4)}, ${flight.scaleY.toFixed(4)})`;

	// reduced-motion：不做连续插值，过闸即切换（闸值 0.5，与页头淡出同源）。
	if (reduced) {
		return p > 0.5
			? { transform: '', opacity: 1, content: 1 }
			: { transform: startTransform, opacity: 0, content: 0 };
	}

	const eased = smoothstep(clamp01(p));
	if (eased >= 1) return { transform: '', opacity: 1, content: 1 };

	const inv = 1 - eased;
	const x = flight.dx * inv;
	const y = flight.dy * inv;
	const sx = 1 + (flight.scaleX - 1) * inv;
	const sy = 1 + (flight.scaleY - 1) * inv;
	// 壳的显影早于位移：标题还在被顶栏裁掉时，岛已经以半透明出现（空间连续性）。
	const opacity = clamp01(clamp01(p) / 0.35);
	// 文字只在后段显形：整壳 scale 期间正文是缩放态，先隐后显才不会有糊字。
	const content = clamp01((eased - 0.55) / 0.4);
	return {
		transform: `translate3d(${x.toFixed(2)}px, ${y.toFixed(2)}px, 0) scale(${sx.toFixed(4)}, ${sy.toFixed(4)})`,
		opacity,
		content
	};
}

function measureTarget(node: HTMLElement): IslandRect | null {
	const prev = node.style.transform;
	const prevOpacity = node.style.opacity;
	node.style.transform = '';
	node.style.opacity = '1';
	const rect = node.getBoundingClientRect();
	node.style.transform = prev;
	node.style.opacity = prevOpacity;
	if (rect.width <= 0 || rect.height <= 0) return null;
	return { x: rect.left, y: rect.top, width: rect.width, height: rect.height };
}

const applyFrame = (
	node: HTMLElement,
	target: IslandRect,
	state: HandoffState,
	reduced: boolean
): void => {
	if (!state.source) return;
	const css = handoffCss(target, state.source, state.p, reduced);
	if (css.transform) node.style.transform = css.transform;
	else node.style.removeProperty('transform');
	node.style.opacity = String(css.opacity);
	node.style.setProperty('--cst-island-content', String(css.content));
};

/**
 * 灵动岛动作：订阅滚动交接帧，把「锚点矩形 → 顶栏泊位」的插值写成 transform/opacity。
 * 用法：`<div use:islandHandoff={{ enabled: handoff }}>`（非交接模式下直接 no-op）。
 */
export function islandHandoff(node: HTMLElement, params: { enabled?: boolean } = {}) {
	// 默认启用：胶囊永远走滚动交接；显式传 enabled:false 才跳过（测试/特殊场景）。
	if (params.enabled === false) return;
	let target = measureTarget(node);
	let prevP = currentHandoff().p;
	let initialized = false;
	let landingTimer: ReturnType<typeof setTimeout> | null = null;

	const clearLanding = () => {
		if (landingTimer !== null) {
			clearTimeout(landingTimer);
			landingTimer = null;
		}
		node.classList.remove('island-land');
	};

	const triggerLanding = () => {
		if (prefersReducedMotion()) return;
		clearLanding();
		void node.offsetWidth;
		node.classList.add('island-land');
		landingTimer = setTimeout(clearLanding, 80);
	};

	const onAnimationEnd = (e: AnimationEvent) => {
		if (e.animationName === 'cst-island-land') {
			clearLanding();
		}
	};
	node.addEventListener('animationend', onAnimationEnd);

	const apply = (s: HandoffState) => {
		if (!target) target = measureTarget(node);
		if (!target || !s.source) {
			node.style.opacity = '0';
			clearLanding();
			prevP = s.p;
			return;
		}

		// 落地微动效（P1-7）：从飞行中（prevP < 1）刚刚抵达落位终态（s.p >= 1）瞬间触发；反向滚动时清空
		if (!initialized) {
			initialized = true;
		} else if (prevP < 1 && s.p >= 1) {
			triggerLanding();
		} else if (s.p < 1) {
			clearLanding();
		}
		prevP = s.p;

		applyFrame(node, target, s, prefersReducedMotion());
	};
	const unsubscribe = subscribeHandoff(apply);
	apply(currentHandoff());
	// 岛内容变化（标题/meta 更新）会改变自然宽度：只在岛可见期间重测，避免误伤 hover transform。
	const ro =
		typeof ResizeObserver === 'undefined'
			? null
			: new ResizeObserver(() => {
					target = measureTarget(node);
					apply(currentHandoff());
				});
	ro?.observe(node);
	return {
		destroy() {
			unsubscribe();
			node.removeEventListener('animationend', onAnimationEnd);
			clearLanding();
			ro?.disconnect();
		}
	};
}

/**
 * 页头退场动作：与交接进度同源的渐隐（整条页头 band）或渐隐+上移（内容层）。
 * 用 inline style 而不是 class 过渡：滚动驱动必须逐帧跟手，不能落后一个 transition 时长。
 */
export function handoffRecede(
	node: HTMLElement,
	params: { enabled?: boolean; mode?: 'band' | 'content' } = {}
) {
	if (!params.enabled) return;
	const apply = (s: HandoffState) => {
		const reduced = prefersReducedMotion();
		// 退场曲线：前 50% 保持不动（页头仍清晰可读，岛从标题位置升起并从它上面掠过），
		// 后 35% 才快速淡出——避免「半透明页头 + 下方内容滚动」长时间叠字的双曝光。
		const fade = reduced
			? s.p > 0.5
				? 1
				: 0
			: smoothstep(clamp01((s.p - 0.5) / 0.35));
		node.style.opacity = String(1 - clamp01(fade));
		if (params.mode === 'content') {
			node.style.transform = fade > 0 ? `translateY(${(-2 * clamp01(fade)).toFixed(2)}px)` : '';
		}
	};
	const unsubscribe = subscribeHandoff(apply);
	apply(currentHandoff());
	return {
		destroy() {
			unsubscribe();
		}
	};
}