<script lang="ts">
	import type { Component } from 'svelte';
	import { theme } from '@/stores/theme.svelte';

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
	const showHover = $derived(hovered && !isActive);

	// hover 背景恒为透明（vibrant/forest 两主题均无 rail hover 背景填充，
	// 反馈由 --elevation-rail-hover 阴影承载）
	const hoverShadow = $derived(showHover ? 'var(--cst-elevation-rail-hover)' : 'none');

	const hoverScale = $derived(showHover ? (theme.theme === 'forest' ? 1.08 : 1.14) : 1);
	const iconColor = $derived(
		isActive || showHover ? 'var(--cst-accent-default)' : 'var(--cst-text-secondary)'
	);
</script>

<button
	type="button"
	class="relative flex flex-col items-center justify-center outline-none rounded-control"
	style="width:52px;height:52px;gap:3px;box-shadow:{hoverShadow};transform:scale({hoverScale});transition:background 0.16s ease, box-shadow 0.2s cubic-bezier(0.22,1,0.36,1), transform 0.2s cubic-bezier(0.34,1.56,0.64,1);"
	aria-label={label}
	aria-current={isActive ? 'page' : undefined}
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	{#if isActive}
		<div
			style="position:absolute;inset:4px;border-radius:var(--cst-shape-small);background:var(--cst-accent-default);opacity:0.14;"
		></div>
	{/if}
	<Icon
		style="width:22px;height:22px;color:{iconColor};transition:color 0.18s ease;position:relative;z-index:1;"
		strokeWidth={isActive ? 2.1 : showHover ? 1.75 : 1.5}
	/>
	<span class="text-xxs" style="line-height:1;color:{iconColor};font-weight:{isActive ? 600 : 400};position:relative;z-index:1;transition:color 0.18s ease;">
		{label}
	</span>
	{#if isActive}
		<div class="w-0.75 rounded-[2px]"  style="position:absolute;left:0;top:10px;bottom:10px;background:var(--cst-accent-default)"></div>
	{/if}
</button>
