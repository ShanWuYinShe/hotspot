<script lang="ts">
	import type { Snippet } from 'svelte';
	import ContentColumn from './ContentColumn.svelte';

	// 布局组件（layout/page）。
	// StandardPageLayout — 三端共用的标准路由页面结构：
	// header（可选，与正文共用同一 ContentColumn gutter）+ 正文。
	// 对齐 Figma Make layout/page/StandardPageLayout 契约：
	// 页面不得自行组合 PageHeader/ContentColumn/页面级滚动/页面级横向 padding，
	// 一律把 header 传给本组件、正文放在 children。
	//
	// ┌──────────────── 路由页面 ──────────────────────────┐
	// │ header（sticky 可选，gutter 与正文对齐）              │
	// │ ┌──────── ContentColumn ────────┐                  │
	// │ │ 正文                          │                  │
	// │ └───────────────────────────────┘                  │
	// └────────────────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let {
		header,
		children,
		contentClassName = 'pt-4 pb-6',
		contentMaxWidth
	}: {
		header?: Snippet;
		children: Snippet;
		contentClassName?: string;
		contentMaxWidth?: number | string;
	} = $props();
</script>

<section data-standard-page-layout class="min-h-full w-full">
	{#if header}
		{@render header()}
	{/if}
	<ContentColumn maxWidth={contentMaxWidth} class={contentClassName}>
		{@render children()}
	</ContentColumn>
</section>
