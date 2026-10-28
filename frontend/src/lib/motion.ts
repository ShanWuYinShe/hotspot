// 动效基元（Round 7）：统一 enter/exit/pop 语义与 reduced-motion 显式判断。
// 约定：CSS 动画由全局 reduced 规则压制；JS 驱动路径（Svelte transition / class 定时器 / action）
// 必须先过 prefersReducedMotion()，reduced 时直接到终态、不留动画类。
import { fly } from 'svelte/transition';
import { cubicOut } from 'svelte/easing';

/** 动效时长语义（进入段与 CSS utilities.css 中的 .page-enter/.content-fade 同源，改一处必须同步另一处）。 */
export const MOTION = {
	/** 页面进场：120ms（R16 跟手微调，原 180ms；与 .page-enter 的 --cst-page-enter-duration 同步） */
	enter: 120,
	/** 骨架→内容 fade：110ms（R16 跟手微调，原 150ms；与 .content-fade 的 --cst-content-fade-duration 同步） */
	contentFade: 110,
	/** 退场（toast/移除）：rapid 150ms */
	exit: 150,
	/** 状态切换 pop：standard 220ms */
	pop: 220
} as const;

/** 当前是否偏好减弱动效；SSR/jsdom 无 matchMedia 时按「不减弱」处理。 */
export function prefersReducedMotion(): boolean {
	if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false;
	try {
		return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
	} catch {
		return false;
	}
}

/** 按 reduced 折算时长：reduced 直接 0ms（立即到终态）。 */
export function motionDuration(ms: number): number {
	return prefersReducedMotion() ? 0 : ms;
}

/**
 * 状态切换 pop action（点赞/收藏/关注等 false→true 才播）：
 * - 正常：加 .state-pop 类触发 220ms 缩放 keyframes，animationend 后移除；
 * - reduced：显式跳过，不添加任何动画类（直接终态）。
 * 用法：<span use:pop={liked}>…</span>
 */
export function pop(node: HTMLElement, active: boolean) {
	let prev = active;
	const onEnd = () => node.classList.remove('state-pop');
	const play = () => {
		if (prefersReducedMotion() || motionDuration(MOTION.pop) === 0) return;
		node.classList.remove('state-pop');
		void node.offsetWidth; // 强制 reflow，保证连续切换可重启动画
		node.classList.add('state-pop');
	};
	node.addEventListener('animationend', onEnd);
	return {
		update(next: boolean) {
			if (next && !prev) play();
			prev = next;
		},
		destroy() {
			node.removeEventListener('animationend', onEnd);
			node.classList.remove('state-pop');
		}
	};
}

/**
 * 页面进场（仅客户端导航时由 RoutedContentViewport 调用；首屏 enter 不调用）：
 * opacity 0.6→1 + translateY 2px→0，120ms——起点不再接近全透明，避免「先空白
 * 一拍再淡入」的可感知迟滞；reduced 直接跳过（不加类）。
 */
export function playPageEnter(node: HTMLElement | null): void {
	if (!node || prefersReducedMotion() || motionDuration(MOTION.enter) === 0) return;
	node.classList.remove('page-enter');
	void node.offsetWidth;
	node.classList.add('page-enter');
	const done = () => {
		node.classList.remove('page-enter');
		node.removeEventListener('animationend', done);
	};
	node.addEventListener('animationend', done);
	window.setTimeout(done, MOTION.enter + 80); // animationend 不触发时的兜底
}

/** Toast 退场：Svelte out transition（y −8 + fade 150ms）；reduced 时 duration 0（直接移除）。 */
export function toastExit(node: HTMLElement) {
	return fly(node, { y: -8, duration: motionDuration(MOTION.exit), easing: cubicOut });
}

/** 直接播放一次状态 pop（调用方必须在点击处理器里调用；reduced 跳过）。 */
export function playPop(node: HTMLElement | null): void {
	if (!node || prefersReducedMotion() || motionDuration(MOTION.pop) === 0) return;
	node.classList.remove('state-pop');
	void node.offsetWidth;
	node.classList.add('state-pop');
}

/**
 * 点击驱动的状态 pop action（推荐用于点赞/收藏/关注/加入）：
 * 只在点击处理器里递增 trigger（如 likedPop += 1）才播放，避免 like=true 首帧/数据同步误播。
 * 用法：<span use:popOn={likePop}>
 */
export function popOn(node: HTMLElement, trigger: number) {
	let prev = trigger;
	const onEnd = () => node.classList.remove('state-pop');
	const play = () => {
		if (prefersReducedMotion() || motionDuration(MOTION.pop) === 0) return;
		node.classList.remove('state-pop');
		void node.offsetWidth;
		node.classList.add('state-pop');
	};
	node.addEventListener('animationend', onEnd);
	return {
		update(next: number) {
			if (next !== prev) play();
			prev = next;
		},
		destroy() {
			node.removeEventListener('animationend', onEnd);
			node.classList.remove('state-pop');
		}
	};
}

/**
 * 骨架→内容 fade action：仅「曾经显示过骨架(true) → 数据到达(false)」时播 150ms fade；
 * 缓存命中/无骨架直出（初始就是 false）不播。reduced 显式跳过。
 */
export function contentFade(node: HTMLElement, showSkeleton: boolean) {
	let prev = showSkeleton;
	const onEnd = () => node.classList.remove('content-fade');
	const play = () => {
		if (prefersReducedMotion()) return;
		node.classList.remove('content-fade');
		void node.offsetWidth;
		node.classList.add('content-fade');
	};
	node.addEventListener('animationend', onEnd);
	return {
		update(next: boolean) {
			if (prev && !next) play();
			prev = next;
		},
		destroy() {
			node.removeEventListener('animationend', onEnd);
			node.classList.remove('content-fade');
		}
	};
}

/**
 * 只播一次的进场 action（评论/回复新增）：active 由调用方在「本次新增」时置 true，
 * 播放 160ms fade+translateY；reduced 显式跳过。初始渲染 active=false 不会误播。
 */
export function enterOnce(node: HTMLElement, active: boolean) {
	let played = false;
	const onEnd = () => node.classList.remove('comment-enter');
	const play = () => {
		if (played || prefersReducedMotion()) { played = true; return; }
		played = true;
		node.classList.add('comment-enter');
	};
	node.addEventListener('animationend', onEnd);
	if (active) play();
	return {
		update(next: boolean) {
			if (next) play();
		},
		destroy() {
			node.removeEventListener('animationend', onEnd);
			node.classList.remove('comment-enter');
		}
	};
}

/**
 * 卡片按压反馈 action（R14 卡片交互契约，所有图片内容卡共用）：
 * - 按下卡片本体（非行内按钮/链接）→ 根节点 data-pressed="true"，CSS 统一按
 *   --cst-card-tap-scale 缩放（vibrant 0.94 / forest·inkwash 0.97，随主题）；
 * - 在行内操作（点赞/收藏/关注/菜单/作者）上按下不缩放整卡，避免"点按钮整卡抖一下"；
 * - pointerup / cancel / leave 立即复位；iOS 滚动时 pointercancel 同样复位。
 * 使用：<article class="card-hover" use:cardPress>…</article>
 */
export function cardPress(node: HTMLElement) {
	const isInnerControl = (target: EventTarget | null) => {
		const el = target as HTMLElement | null;
		if (!el || typeof el.closest !== 'function') return false;
		return Boolean(
			el.closest(
				'button:not([data-cover]), a[href], input, textarea, select, [role="menuitem"], [role="slider"]'
			)
		);
	};
	const down = (e: PointerEvent) => {
		if (e.button !== 0 && e.pointerType === 'mouse') return;
		if (isInnerControl(e.target)) return;
		node.dataset.pressed = 'true';
	};
	const up = () => {
		delete node.dataset.pressed;
	};
	node.addEventListener('pointerdown', down);
	node.addEventListener('pointerup', up);
	node.addEventListener('pointercancel', up);
	node.addEventListener('pointerleave', up);
	return {
		destroy() {
			node.removeEventListener('pointerdown', down);
			node.removeEventListener('pointerup', up);
			node.removeEventListener('pointercancel', up);
			node.removeEventListener('pointerleave', up);
			delete node.dataset.pressed;
		}
	};
}
