<script lang="ts">
	import type { Snippet } from 'svelte';

	// 布局组件（layout/page）。
	// PageViewport — 路由页面画布承载。提供 [data-page-viewport] 标记，激活
	// utilities.css 中 [data-page-viewport] 的 surface-page 背景与 text-primary 前景色契约。
	// 对齐 Figma Make layout/page/PageViewport：
	//   - 仅承载页面画布，绝不设置裁剪 overflow（滚动由三端 shell 独占）
	//   - h-full min-h-full w-full，让子页面撑满 RoutedContentViewport 的内容区
	//   - 底色透明：画布底色由外壳提供（桌面白浮卡 / 移动·平板 surface-page），
	//     氛围画布挂在外壳内容区，装饰层不会随页面滚动
	//
	// ┌──────────────── 应用壳 ────────────────┐
	// │ 导航 / 侧栏 / 底部栏                         │
	// ├─────────────────────────────────────────────┤
	// │ RoutedContentViewport（滚动容器）            │
	// │ ┌──────── PageViewport ──────────┐          │
	// │ │ 路由页面（Outlet 内容）         │          │
	// │ └─────────────────────────────────┘          │
	// └─────────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let { children }: { children: Snippet } = $props();
</script>

<div data-page-viewport class="h-full min-h-full w-full">
	{@render children()}
</div>
