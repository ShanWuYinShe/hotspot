<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/state';
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
	const demoMode = $derived(page.url.searchParams.get('demo'));
	const isConstrained = $derived(contentMaxWidth != null);
</script>

<section data-split-page-layout class="flex h-full min-h-full w-full flex-col">
	{#if isDesktop}
		<!-- 桌面主区与 rail 聚合成自适应工作台，保持视觉邻近（gap: rail-gap），
		     在超宽/高分屏上居中聚合，杜绝内容孤悬与 ContextRail 飘到屏幕极右缘的撕裂感 -->
		{#if demoMode === 'a'}
			<!-- 备选方案 A：统一中轴舞台（对称居中，两端均匀呼吸） -->
			<div class="mx-auto flex h-full w-full max-w-6xl xl:max-w-7xl 2xl:max-w-[88rem] justify-center" style="gap:var(--cst-context-rail-gap);">
				<div class="flex min-w-0 flex-1 flex-col">
					{#if header}
						<div class="sticky top-0 z-[var(--cst-z-sticky)] shrink-0">{@render header()}</div>
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
		{:else if demoMode === 'b'}
			<!-- 备选方案 B：统一左锚定工站（左基准线对齐，内容列 680px，右侧跟随扩展画布） -->
			<div class="flex h-full w-full justify-start px-[var(--cst-page-gutter)]" style="gap:var(--cst-context-rail-gap);">
				<div class="flex min-w-0 max-w-2xl flex-1 flex-col">
					{#if header}
						<div class="sticky top-0 z-[var(--cst-z-sticky)] shrink-0">{@render header()}</div>
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
		{:else if demoMode === 'c'}
			<!-- 备选方案 C：超宽屏双列瀑布流（容器拓宽至 96rem，为双列卡片预留充足呼吸宽度） -->
			<div class="mx-auto flex h-full w-full max-w-7xl xl:max-w-[88rem] 2xl:max-w-[96rem] justify-center" style="gap:var(--cst-context-rail-gap);">
				<div class="flex min-w-0 flex-1 flex-col">
					{#if header}
						<div class="sticky top-0 z-[var(--cst-z-sticky)] shrink-0">{@render header()}</div>
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
		{:else if demoMode === 'd'}
			<!-- 备选方案 D：LBS 双核分屏（左侧 58% 社交动态流 + 右侧 42% 实时雷达与地图画板） -->
			<div class="flex h-full w-full px-[var(--cst-page-gutter)]" style="gap:var(--cst-context-rail-gap);">
				<div class="flex min-w-0 flex-1 max-w-[58%] flex-col">
					{#if header}
						<div class="sticky top-0 z-[var(--cst-z-sticky)] shrink-0">{@render header()}</div>
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
				<section class="flex min-w-0 flex-1 max-w-[42%] h-full flex-col py-2 overflow-y-auto" aria-label="雷达探索中心">
					{@render rail()}
				</section>
			</div>
		{:else}
			<!-- 方案 E（默认标准）：超宽屏流式 3 列满铺画廊 + 右侧 Bento 侧栏（满幅无死角）
			     若页面指定了 contentMaxWidth（如详情/个人主页/通知），则在超宽屏下居中聚合，
			     杜绝内容孤悬与 ContextRail 飘到屏幕极右缘的 1000px 撕裂断层 -->
			<div
				class={isConstrained
					? 'mx-auto flex h-full w-full justify-center px-[var(--cst-page-gutter)]'
					: 'flex h-full w-full justify-between px-[var(--cst-page-gutter)]'}
				style={isConstrained
					? `max-width:calc(${typeof contentMaxWidth === 'number' ? `${contentMaxWidth}px` : contentMaxWidth} + 21rem + var(--cst-context-rail-gap) + (var(--cst-page-gutter) * 2));gap:var(--cst-context-rail-gap);`
					: 'gap:var(--cst-context-rail-gap);'}
			>
				<div
					class="flex min-w-0 flex-1 flex-col"
					style={isConstrained
						? `max-width:${typeof contentMaxWidth === 'number' ? `${contentMaxWidth}px` : contentMaxWidth};`
						: undefined}
				>
					{#if header}
						<div class="sticky top-0 z-[var(--cst-z-sticky)] shrink-0">
							{#if isConstrained}
								<ContentColumn maxWidth={contentMaxWidth} class="p-0">
									{@render header()}
								</ContentColumn>
							{:else}
								{@render header()}
							{/if}
						</div>
					{/if}
					<div class="min-h-0 flex-1">
						<ContentColumn maxWidth={contentMaxWidth} class={contentClassName}>
							{@render children()}
						</ContentColumn>
					</div>
					{#if footer}
						<div class="shrink-0 sticky bottom-0 z-10 w-full bg-surface-primary">
							{#if isConstrained}
								<ContentColumn maxWidth={contentMaxWidth} class="p-0">
									{@render footer()}
								</ContentColumn>
							{:else}
								{@render footer()}
							{/if}
						</div>
					{/if}
				</div>
				<ContextRail attached class="w-84 min-w-84 shrink-0">{@render rail()}</ContextRail>
			</div>
		{/if}
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
