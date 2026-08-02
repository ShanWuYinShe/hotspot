<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 布局组件（layout/page）。
	// ContentColumn — 标准路由页面唯一的外层宽度与 gutter 管理者。
	// 对齐 Figma Make layout/page/ContentColumn 契约：
	//   - mx-auto + px-[var(--page-gutter)]（移动 20px / 平板 24px / 桌面 28px 三档响应式）
	//   - 默认 maxWidth = var(--page-content-max)（100%），不设默认窄列；
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
</script>

<div
	data-page-content
	class={cn('mx-auto w-full px-[var(--page-gutter)]', className)}
	style={maxWidth ? `max-width:${maxWidth};` : 'max-width:var(--page-content-max);'}
>
	{@render children()}
</div>
