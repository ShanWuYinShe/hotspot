<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 布局组件（layout/page）。
	// ContentColumn — 标准路由页面唯一的外层宽度与 gutter 管理者。
	// 对齐 Figma Make layout/page/ContentColumn 契约：
	//   - mx-auto + px-[var(--cst-page-gutter)]（移动 16px / 平板 24px / 桌面 28px 三档响应式）
	//   - 默认 maxWidth = var(--cst-page-content-max)（100%），不设默认窄列；
	//     仅真正的阅读/详情列才传 maxWidth，不得用窄宽度来摆放标准页面。
	//
	// ┌──────────────── 应用壳（桌面右侧 / 平板画布 / 手机画布）────────┐
	// │ ┌──────────── ContentColumn ────────────┐                      │
	// │ │ 页面正文（header 与 body 共用同一 gutter）│                      │
	// │ └────────────────────────────────────────┘                      │
	// └─────────────────────────────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let {
		children,
		maxWidth,
		class: className = ''
	}: {
		children: Snippet;
		maxWidth?: number | string;
		class?: string;
	} = $props();

	// 数字 maxWidth 自动补 px：裸数字生成 max-width:560 会被 CSSOM 当非法值
	// 整条丢弃（style 静默变空），字符串值（'560px'/'50%'）原样透传。
	const columnStyle = $derived(
		maxWidth == null
			? 'max-width:var(--cst-page-content-max);'
			: `max-width:${typeof maxWidth === 'number' ? `${maxWidth}px` : maxWidth};`
	);
</script>

<div
	data-page-content
	class={cn('mx-auto w-full px-[var(--cst-page-gutter)]', className)}
	style={columnStyle}
>
	{@render children()}
</div>
