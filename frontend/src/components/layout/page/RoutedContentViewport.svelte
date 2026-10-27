<script lang="ts">
	import type { Snippet } from 'svelte';
	import { afterNavigate, beforeNavigate } from '$app/navigation';
	import { cn } from '@/lib/utils';
	import { decideScroll, restoreScrollTop, savePosition } from '@/lib/scroll-positions';
	import { playPageEnter } from '@/lib/motion';
	import { announceRoute } from '@/lib/route-announcer';
	import BackToTop from '@/components/navigation/BackToTop.svelte';
	import PageViewport from './PageViewport.svelte';

	// 布局组件（layout/page）。
	// RoutedContentViewport — 三端唯一的路由内容与滚动承载区。
	// 对齐 Figma Make layout/page/RoutedContentViewport 契约（Guidelines 标记 REQUIRED）：
	//   - 每个平台 shell 必须渲染本组件恰好一次；它是唯一的页面滚动 owner
	//   - 包含 PageViewport（路由画布）+ Outlet 内容
	//   - 路由页面不得再渲染本组件或 PageViewport
	//   - desktopPane=true 时输出 data-desktop-main 属性，激活 utilities.css 中
	//     [data-desktop-main] .desktop-main-scroll > * 与 .sticky 的 surface-page 强制规则
	//
	// ┌──────────────── 应用壳 ────────────────┐
	// │ 导航 / 侧栏 / 底部栏                         │
	// ├─────────────────────────────────────────────┤
	// │ <main>（唯一滚动容器）                       │
	// │ ┌──────── 内容包裹 ────────┐                │
	// │ │ ┌── PageViewport ──┐     │                │
	// │ │ │ Outlet 路由页面  │     │                │
	// │ │ └──────────────────┘     │                │
	// │ └──────────────────────────┘                │
	// └─────────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let {
		children,
		class: className = '',
		contentClassName = '',
		style,
		desktopPane = false
	}: {
		children: Snippet;
		class?: string;
		contentClassName?: string;
		style?: string;
		desktopPane?: boolean;
	} = $props();

	// 路由级滚动存取（SvelteKit 只管 window，本项目滚动 owner 是本 <main>）：
	// 离开存位；popstate 返回按存位恢复；进新 pathname 置顶；同路由内导航
	//（如 ?tab= 切换）保持不动。
	// 恢复用追到为止（restoreScrollTop）：图片/瀑布流布局异步长高时单次赋值
	// 会被钳位，每帧重设直到到达目标或布局稳定。
	let mainEl: HTMLElement | null = $state(null);
	/** 内容层（不含 <main> 滚动容器本身）：仅客户端导航时做进场动画。 */
	let contentEl: HTMLElement | null = $state(null);
	const savedTops = new Map<string, number>();

	beforeNavigate((nav) => {
		if (mainEl && nav.from?.url) savePosition(savedTops, nav.from.url.pathname, mainEl.scrollTop);
	});

	afterNavigate((nav) => {
		// 页面进场：仅客户端导航（link/goto/form/popstate），首屏 enter 不播（保护 LCP）。
		if (nav.type !== 'enter') {
			playPageEnter(contentEl);
			// SPA 焦点与播报：优先聚焦页面 h1（退化为 main），preventScroll 不打断滚动恢复；
			// 首屏 enter 不聚焦/不播报；返回/前进同样生效。
			requestAnimationFrame(() => {
				const heading = contentEl?.querySelector('h1') as HTMLElement | null;
				const title = heading?.textContent ?? '';
				// 优先聚焦主内容区（main#main-content）：读屏从新页面开始，且避免
				// 「聚焦 h1 + announcer」重复念标题；main 不可聚焦时回退 h1 并跳过播报。
				const mainTarget = mainEl;
				if (mainTarget) {
					mainTarget.setAttribute('tabindex', '-1');
					mainTarget.focus({ preventScroll: true });
				}
				if (document.activeElement === mainTarget) {
					announceRoute(title);
					return;
				}
				const fallback: HTMLElement | null = heading ?? contentEl;
				if (fallback) {
					fallback.setAttribute('tabindex', '-1');
					fallback.focus({ preventScroll: true });
				}
				// 兜底路径不叠加播报（避免重复）
			});
		}
		const decision = decideScroll(nav.type, nav.from?.url.pathname ?? null, nav.to?.url.pathname ?? null, savedTops);
		if (decision.action === 'keep' || !mainEl) return;
		restoreScrollTop(mainEl, decision.action === 'restore' ? decision.top : 0);
	});
</script>

<main
	id="main-content"
	tabindex="-1"
	bind:this={mainEl}
	{...(desktopPane ? { 'data-desktop-main': '' } : {})}
	class={cn(
		'h-full min-h-0 flex-1 w-full overflow-y-auto overflow-x-hidden no-scrollbar scroll-smooth',
		className
	)}
	{style}
>
	<!-- 内容 div 需显式 h-full：PageViewport 的 h-full/min-h-full 相对 height:auto
	     父级时百分比高度失效（min-height 不算显式高度），导致内容不满时页面
	     无法撑满视口（消息页等短内容页底部出现空白）。h-full 相对 main
	     （flex item，高度由 shell 确定）始终有效；min-h-full 兜底超长内容。 -->
	<div bind:this={contentEl} class={cn('h-full min-h-full w-full', contentClassName)}>
		<PageViewport>
			{@render children()}
		</PageViewport>
	</div>
	<BackToTop scrollEl={() => mainEl} />
</main>
