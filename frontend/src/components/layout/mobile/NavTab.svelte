<script lang="ts">
	import type { Component } from 'svelte';

	// 移动/平板底部导航项（对齐 Figma Make NavPill 契约）：
	// - flex-1 均分可用宽度，主导航不设固定 item 宽度
	// - 选中胶囊：inset 4px 6px + shape-panel 圆角 + navigation-active-indicator
	// - 图标/文字色走 navigation-content(-active) 语义 token

	let {
		label,
		isActive,
		onClick,
		Icon,
		iconSize = 22
	}: {
		label: string;
		isActive: boolean;
		onClick: () => void;
		Icon: Component;
		/** 图标尺寸 px（移动 22 / 平板 24） */
		iconSize?: number;
	} = $props();
</script>

<button
	type="button"
	class="hover-btn nav-tab relative flex flex-col items-center justify-center rounded-lg pt-2.5 pb-2 gap-1 focus-ring select-none
		{isActive ? '' : 'hover:bg-interactive-hover'}"
	style="flex:1;min-width:0;"
	aria-label={label}	aria-current={isActive ? 'page' : undefined}
	onclick={onClick}
>
	{#if isActive}
		<div
			class="transition-[opacity] duration-standard"
			style="position:absolute;inset:4px 6px;border-radius:var(--cst-shape-lg);background:var(--cst-navigation-active-indicator);opacity:var(--cst-opacity-active);"
		></div>
	{/if}

	<div style="display:flex;align-items:center;justify-content:center;position:relative;z-index:1;">
		<Icon
			style="width:{iconSize}px;height:{iconSize}px;color:{isActive ? 'var(--cst-navigation-content-active)' : 'var(--cst-navigation-content)'};transition:color var(--cst-motion-standard) var(--cst-ease-standard);"
			strokeWidth={isActive ? 2 : 1.5}
		/>
	</div>
	<span
		class="text-xs leading-none {isActive ? 'font-semibold' : 'font-normal'}"
		style="color:{isActive ? 'var(--cst-navigation-content-active)' : 'var(--cst-navigation-content)'};transition:color var(--cst-motion-standard) var(--cst-ease-standard);position:relative;z-index:1;"
	>
		{label}
	</span>
</button>
