<script lang="ts">
	import type { Component } from 'svelte';

	// 移动横屏左栏导航图标。
	// hover/active 反馈：主题感知的背景/阴影/缩放（CSS 类 + 主题类选择器）。

	let {
		Icon,
		label,
		isActive,
		onClick
	}: {
		Icon: Component;
		label: string;
		isActive: boolean;
		onClick: () => void;
	} = $props();
</script>

<button
	type="button"
	class="rail-icon relative flex flex-col items-center justify-center gap-0.75 focus-ring rounded-md {isActive ? 'is-active text-accent-default' : 'text-text-secondary'}"
	style="width:var(--cst-mobile-rail-item);height:var(--cst-mobile-rail-item);"
	aria-label={label}
	aria-current={isActive ? 'page' : undefined}
	onclick={onClick}
>
	{#if isActive}
		<div class="absolute inset-1 rounded-sm bg-accent-default opacity-active pointer-events-none"></div>
	{/if}
	<Icon
		class="relative z-1 size-5.5 transition-colors duration-snappy"
		strokeWidth={isActive ? 2 : 1.5}
	/>
	<span class="relative z-1 text-xxs leading-none transition-colors duration-snappy {isActive ? 'font-semibold' : 'font-normal'}">
		{label}
	</span>
	{#if isActive}
		<div class="absolute left-0 top-2.5 bottom-2.5 w-0.75 rounded-full bg-accent-default"></div>
	{/if}
</button>

<style>
	.rail-icon {
		transition: box-shadow var(--cst-motion-standard) var(--cst-ease-standard),
					transform var(--cst-motion-standard) var(--cst-ease-bounce),
					color var(--cst-motion-snappy) var(--cst-ease-standard);
	}

	@media (hover: hover) and (pointer: fine) {
		.rail-icon:hover {
			transform: scale(1.1);
			box-shadow: var(--cst-elevation-rail-hover);
			color: var(--cst-accent-default);
		}
		
		.rail-icon.is-active:hover {
			box-shadow: 0 0 0 1px var(--cst-accent-border);
		}
	}
</style>
