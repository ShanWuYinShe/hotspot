<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

	// 列表骨架组件（list）——列表只有两个公共组件：ListContainer（容器）+ ListItem（行）。
	// ListContainer 按所在区域二选一（对齐 Figma Make ListContainers 契约）：
	//   bordered=false（默认）：嵌入面板/Sheet/rail/分栏的集合（会话列表、地图附近栏等）。
	//                          无外框，仅提供安静的行内分隔线（divide-y）。
	//   bordered=true          ：直接位于开放页面区域的集合（朋友/关注/粉丝/通知等）。
	//                          拥有唯一的圆角外框与 1px 行分隔，行使用 --surface-primary 表面。
	// 两种形态都不得再给每行加第二层卡片边框。
	//
	// ┌──────────────── 页面内容区 ────────────┐
	// │ 分组标题（列表外）                       │
	// │ ┌──── ListContainer（bordered） ──────┐ │
	// │ │ ListItem 1                          │ │
	// │ │ ──────────────────────────────────  │ │
	// │ │ ListItem 2                          │ │
	// │ └─────────────────────────────────────┘ │
	// └─────────────────────────────────────────┘
	// 设备范围：桌面、平板和手机通用。

	let {
		children,
		bordered = false,
		class: className = '',
		style: styleProp = '',
		ariaLabel,
		inset = false
	}: {
		children: Snippet;
		/** 有边框形态（开放页面区域）还是无边框形态（面板/Sheet 内部）。 */
		bordered?: boolean;
		class?: string;
		style?: string;
		ariaLabel?: string;
		/** 内缩行形态（对齐 Telegram/macOS 设置的行高亮语言）：行间距分隔、
		 *  无分隔线，行 hover/选中填充自然离容器边缘 8px+（ListItem 高亮自带
		 *  12px 圆角）。桌面可 hover 表面（会话分栏、关系列表等）启用；
		 *  触屏为主的列表保持分隔线形态。 */
		inset?: boolean;
	} = $props();
</script>

{#if bordered}
	<!-- bordered：唯一外框。inset=false 行分隔（gap-px + 容器底色形成 1px 线）；
	     inset=true 行间距分隔、内缩圆角行（macOS 设置语言），行填充不顶卡边。 -->
	<div
		class={cn(
			'flex flex-col overflow-hidden rounded-md border',
			inset ? 'gap-y-0.5 p-2' : 'gap-px',
			className
		)}
		style={inset ? `border-color:var(--cst-border-default);${styleProp}` : `background:var(--cst-border-default);border-color:var(--cst-border-default);${styleProp}`}
		role="list"
		aria-label={ariaLabel}
	>
		{@render children()}
	</div>
{:else if inset}
	<!-- embedded+inset：无外框无分隔线，行内缩（容器水平 padding 即高亮离边距离） -->
	<div role="list" class={cn('flex flex-col gap-y-0.5 px-2 py-1', className)} aria-label={ariaLabel}>
		{@render children()}
	</div>
{:else}
	<!-- embedded：无外框，仅行内分隔线 -->
	<div role="list" class={cn('flex flex-col divide-y divide-border-default', className)} aria-label={ariaLabel}>
		{@render children()}
	</div>
{/if}
