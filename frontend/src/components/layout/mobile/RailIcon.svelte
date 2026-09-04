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

	let hovered = $state(false);
	// JS hover 仅键鼠设备启用（触屏防粘滞）；选中项 hover 保留轻增量（阴影）而非全抑制
	const canHover =
		typeof window !== 'undefined' &&
		typeof window.matchMedia === 'function' &&
		window.matchMedia('(hover: hover) and (pointer: fine)').matches;
	const showHover = $derived(canHover && hovered);

	// hover 背景恒为透明（vibrant/forest 两主题均无 rail hover 背景填充，
	// 反馈由 --elevation-rail-hover 阴影承载）；缩放统一 Figma btn 契约档 1.10
	const hoverShadow = $derived(
		showHover ? 'var(--cst-elevation-rail-hover)' : isActive && canHover && hovered ? '0 0 0 1px var(--cst-accent-border)' : 'none'
	);
	const hoverScale = $derived(canHover && hovered ? 1.1 : 1);
	const iconColor = $derived(
		isActive || showHover ? 'var(--cst-accent-default)' : 'var(--cst-text-secondary)'
	);
</script>

<button
	type="button"
	class="relative flex flex-col items-center justify-center focus-ring rounded-md"
	style="width:52px;height:52px;gap:3px;box-shadow:{hoverShadow};transform:scale({hoverScale});transition:box-shadow var(--cst-motion-standard) var(--cst-ease-standard), transform var(--cst-motion-standard) var(--cst-ease-bounce);"
	aria-label={label}
	aria-current={isActive ? 'page' : undefined}
	onclick={onClick}
	onmouseenter={() => canHover && (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	{#if isActive}
		<div
			style="position:absolute;inset:4px;border-radius:var(--cst-shape-sm);background:var(--cst-accent-default);opacity:var(--cst-opacity-active);"
		></div>
	{/if}
	<Icon
		class="size-5.5"
		style="color:{iconColor};transition:color var(--cst-motion-snappy) var(--cst-ease-standard);position:relative;z-index:1;"
		strokeWidth={isActive ? 2 : showHover ? 1.75 : 1.5}
	/>
	<span class="text-xxs leading-none {isActive ? 'font-semibold' : 'font-normal'}" style="color:{iconColor};position:relative;z-index:1;transition:color var(--cst-motion-snappy) var(--cst-ease-standard);">
		{label}
	</span>
	{#if isActive}
		<div class="w-0.75 rounded-full"  style="position:absolute;left:0;top:10px;bottom:10px;background:var(--cst-accent-default)"></div>
	{/if}
</button>
