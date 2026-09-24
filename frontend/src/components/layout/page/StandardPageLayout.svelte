<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';
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

<section data-standard-page-layout class="flex min-h-full w-full flex-col">
	{#if header}
		{@render header()}
	{/if}
	<!-- flex flex-col + ContentColumn flex-1：正文列至少撑满视口剩余高度。
		 页面总高不变（section 本就 min-h-full），但让需要「贯穿到底」的分栏
		 分割线等布局在短内容页成立（如首页桌面双列的 border-r）。
		 header 为 sticky flex 子项，行为不变。 -->
	<ContentColumn maxWidth={contentMaxWidth} class={cn(contentClassName, 'flex-1')}>
		{@render children()}
	</ContentColumn>
</section>
