<script lang="ts">
	import { untrack, type Snippet } from 'svelte';
	import { cn } from '@/lib/utils';
	import { setInListContext } from './context';

	// a11y 契约：role=list 的直接子元素必须为 listitem（axe aria-required-children）。
	// 容器在此向子树声明列表上下文，ListItem/ListEnd/InlineLoading 在列表内
	// 自动包裹 role=listitem，调用方零感知。

// 列表骨架组件（list）——列表只有两个公共组件：ListContainer（容器）+ ListItem（行）。
// ListContainer 按所在区域二选一（对齐 Figma Make ListContainers 契约）：
//   bordered=false（默认）：嵌入面板/Sheet/rail/分栏的集合（会话列表、地图附近栏等）。
//                          无外框，默认行内分隔线（divide-y）；inset=true 时行内缩圆角。
//   bordered=true          ：开放页面区域的内容软岛（Round 3 语言）：无描边，
//                          表面 bg-surface-island + shadow-xs + 圆角 20；内缩圆角行
//                          （行间距分隔），行 hover/选中填充离岛边 8px。
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

	// bordered=true 时向子树声明软岛上下文（行背景透明，露出软岛表面）；
	// 必须在 props 解构之后调用（setContext 需在组件初始化期执行）。
	// 上下文按挂载时的形态写入一次即可（后续切 bordered 属组件替换场景），用 untrack 显式声明
	setInListContext(untrack(() => bordered));
</script>

{#if bordered}
	<!-- bordered：内容软岛 = 无描边 + 表面色 + 轻投影 + 圆角 20（Round 3 软岛语言） -->
	<div
		class={cn('flex flex-col gap-y-0.5 overflow-hidden rounded-lg bg-surface-island p-2 shadow-xs', className)}
		style={styleProp}
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
	<!-- embedded：无外框，细腻行内发丝分隔线 -->
	<div role="list" class={cn('flex flex-col divide-y divide-border-subtle/40', className)} aria-label={ariaLabel}>
		{@render children()}
	</div>
{/if}
