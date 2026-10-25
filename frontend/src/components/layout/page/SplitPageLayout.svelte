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
		contentClassName = 'pt-4 pb-6',
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
	/** 移动端固定底栏高度：滚动内容据此补内距，末条内容不被底栏遮挡 */
	let footerHeight = $state(0);
</script>

<section data-split-page-layout class="flex h-full min-h-full w-full flex-col">
	{#if isDesktop}
		<!-- 桌面主区与 rail 呼吸留白分隔，同处统一氛围画布，无生硬实线与突兀白墙 -->
		<div class="flex h-full w-full" style="gap:var(--cst-context-rail-gap);">
			<!-- 主区：header + 滚动正文 + 固定 footer -->
			<div class="flex min-w-0 flex-1 flex-col">
				{#if header}
					<div class="shrink-0">{@render header()}</div>
				{/if}
				<div class="min-h-0 flex-1">
					<ContentColumn maxWidth={contentMaxWidth} class={contentClassName}>
						{@render children()}
					</ContentColumn>
				</div>
				{#if footer}
					<div class="shrink-0 sticky bottom-0 z-10">{@render footer()}</div>
				{/if}
			</div>
			<ContextRail attached>{@render rail()}</ContextRail>
		</div>
	{:else}
		<!-- 移动端：滚动交还 RoutedContentViewport（唯一滚动 owner），此处只承载
		     header 吸顶 / 内容列 / 固定 footer，避免双层嵌套滚动 -->
		{#if header}
			<!-- 移动端 header 吸顶（滚动时内容上探，避免固定 header 常驻挤压内容区） -->
			<div class="sticky top-0 z-[var(--cst-z-sticky)] shrink-0">{@render header()}</div>
		{/if}
		<div class="min-h-0 flex-1" style={footerHeight > 0 ? `padding-bottom:${footerHeight}px` : undefined}>
			<ContentColumn maxWidth={contentMaxWidth} class={contentClassName}>
				{@render children()}
			</ContentColumn>
		</div>
		{#if footer}
			<!-- 固定底栏（拇指热区）：真机键盘弹出时 position:sticky 会滞留在半屏
			     （iOS Safari 键盘收起后不回流，实机复现）——改用 fixed + 内容补偿内距，
			     配合 app.html 的 interactive-widget=resizes-content，键盘弹出时布局
			     视口收缩、底栏始终浮在键盘上方 -->
			<div
				class="fixed inset-x-0 bottom-0 z-[var(--cst-z-sticky)] shrink-0 pb-[env(safe-area-inset-bottom)]"
				bind:clientHeight={footerHeight}
			>
				{@render footer()}
			</div>
		{/if}
	{/if}
</section>
