<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';
	import PageViewport from './PageViewport.svelte';

	// 布局组件（layout/page）。
	// RoutedContentViewport — 三端唯一的路由内容与滚动承载区。
	// 对齐 Figma Make layout/page/RoutedContentViewport 契约（Guidelines 标记 REQUIRED）：
	//   - 每个平台 shell 必须渲染本组件恰好一次；它是唯一的页面滚动 owner
	//   - 包含 PageViewport（路由画布）+ Outlet 内容
	//   - 路由页面不得再渲染本组件或 PageViewport
	//   - desktopPane=true 时输出 data-desktop-main 属性，激活 utilities.css 中
	//     [data-desktop-main] .desktop-main-scroll > * 与 .sticky 的 page-bg 强制规则
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
</script>

<main
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
	<div class={cn('h-full min-h-full w-full', contentClassName)}>
		<PageViewport>
			{@render children()}
		</PageViewport>
	</div>
</main>
