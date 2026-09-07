<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '@/lib/utils';

// 列表骨架组件（list）——列表只有两个公共组件：ListContainer（容器）+ ListItem（行）。
// ListContainer 按所在区域二选一（对齐 Figma Make ListContainers 契约）：
//   bordered=false（默认）：嵌入面板/Sheet/rail/分栏的集合（会话列表、地图附近栏等）。
//                          无外框，默认行内分隔线（divide-y）；inset=true 时行内缩圆角。
//   bordered=true          ：直接位于开放页面区域的集合（朋友/关注/粉丝/通知等）。
//                          唯一的圆角外框卡 + 内缩圆角行（行间距分隔，对齐
//                          Telegram/macOS 设置语言），行 hover/选中填充天然离卡边 8px。
// 两种形态都不得再给每行加第二层卡片边框；开放页面的分组卡只有 bordered 一种，
// 禁止再用 card-raised div 手工包列表（历史遗留已清除）。
//
// ┌──────────────── 页面内容区 ────────────┐
// │ 分组标题（列表外）                       │
// │ ┌──── ListContainer（bordered） ──────┐ │
// │ │  ListItem 1（内缩圆角行）            │ │
// │ │  ListItem 2                          │ │
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
	/** 仅 embedded 生效：内缩圆角行（行间距分隔、无分隔线，行 hover/选中填充
	 *  离面板边缘 8px）。bordered 恒为内缩行为，此 prop 忽略。 */
	inset?: boolean;
} = $props();
</script>

{#if bordered}
	<!-- bordered：唯一分组卡 = 白底描边外框 + 内缩圆角行（macOS 设置语言） -->
	<div
		class={cn('flex flex-col gap-y-0.5 overflow-hidden rounded-md border bg-surface-primary p-2', className)}
		style={`border-color:var(--cst-border-default);${styleProp}`}
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
