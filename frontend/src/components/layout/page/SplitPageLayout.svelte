<script lang="ts">
	import type { Snippet } from 'svelte';
	import { platform } from '@/stores/platform.svelte';
	import ContextRail from '@/components/layout/ContextRail.svelte';
	import ContentColumn from './ContentColumn.svelte';

	// 布局组件（layout/page）。
	// SplitPageLayout — 分栏页面的统一外层布局（桌面主区 + 右 rail）。
	// 与 StandardPageLayout 的分工：
	//   - StandardPageLayout：常规单列页面（header + ContentColumn），
	//     滚动由 RoutedContentViewport 承担，页面正文不得自建滚动容器；
	//   - SplitPageLayout：有分栏需求（桌面主区 + 右 rail）的页面，主区与 rail 的
	//     滚动由本组件统一管理，页面正文不得自建滚动容器。
	// 桌面 = header + [主区滚动正文 | 右 rail]；平板/手机 = header + 主区滚动正文
	// （rail 不渲染）。footer 固定在主区滚动区下方，不随正文滚动。
	// 正文内普通多列 grid（如群组详情 main + aside）不属于分栏需求，
	// 仍应使用 StandardPageLayout，由 ContentColumn 统一 gutter。

	let {
		header,
		children,
		rail,
		footer,
		contentClassName = 'py-5',
		contentMaxWidth
	}: {
		header?: Snippet;
		children: Snippet;
		rail: Snippet;
		footer?: Snippet;
		contentClassName?: string;
		contentMaxWidth?: number | string;
	} = $props();

	const isDesktop = $derived(platform.platform === 'desktop');
</script>

<section
	data-split-page-layout
	class="flex h-full min-h-full w-full flex-col"
	style="background-color:var(--surface-primary);"
>
	{#if isDesktop}
		<div class="flex h-full w-full">
			<!-- 主区：header + 滚动正文 + 固定 footer -->
			<div class="flex min-w-0 flex-1 flex-col">
				{#if header}
					<div class="shrink-0">{@render header()}</div>
				{/if}
				<div class="no-scrollbar min-h-0 flex-1 overflow-y-auto">
					<ContentColumn maxWidth={contentMaxWidth} class={contentClassName}>
						{@render children()}
					</ContentColumn>
				</div>
				{#if footer}
					<div class="shrink-0">{@render footer()}</div>
				{/if}
			</div>
			<ContextRail attached>{@render rail()}</ContextRail>
		</div>
	{:else}
		{#if header}
			<div class="shrink-0">{@render header()}</div>
		{/if}
		<div class="no-scrollbar min-h-0 flex-1 overflow-y-auto">
			<ContentColumn maxWidth={contentMaxWidth} class={contentClassName}>
				{@render children()}
			</ContentColumn>
		</div>
		{#if footer}
			<div class="shrink-0">{@render footer()}</div>
		{/if}
	{/if}
</section>
