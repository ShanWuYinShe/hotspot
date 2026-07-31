<script lang="ts">
	import type { Component } from 'svelte';
	import { themeStore } from '@/stores/theme.svelte';

	// 移动横屏左栏导航图标（迁移自 React MainLayout.tsx 内 RailIcon）。
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

	// 主题感知 hover 背景（RailIcon 专用：forest/vibrant 透明、neu 纸面）
	const hoverBg = $derived.by(() => {
		if (!showHover) return 'transparent';
		const t = themeStore.theme;
		return t === 'forest' || t === 'vibrant' ? 'transparent' : 'var(--neu-bg)';
	});

	const hoverShadow = $derived.by(() => {
		if (!showHover) return 'none';
		const t = themeStore.theme;
		const isDark = themeStore.isDark;
		if (t === 'forest') {
			return isDark ? '0 0 0 1px rgba(82,184,120,0.28)' : '0 0 0 1px rgba(61,122,85,0.22)';
		}
		if (t === 'vibrant') {
			return isDark
				? '0 6px 16px rgba(129,140,248,0.22), 0 2px 6px rgba(129,140,248,0.12)'
				: '0 6px 16px rgba(99,102,241,0.18), 0 2px 6px rgba(99,102,241,0.10)';
		}
		return '5px 5px 12px var(--neu-shadow-dark), -5px -5px 12px var(--neu-shadow-light)';
	});

	const hoverScale = $derived(showHover ? (themeStore.theme === 'forest' ? 1.08 : 1.14) : 1);
	const iconColor = $derived(
		isActive || showHover ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'
	);
</script>

<button
	type="button"
	class="relative flex flex-col items-center justify-center outline-none rounded-[16px]"
	style="width:52px;height:52px;gap:3px;background:{hoverBg};box-shadow:{hoverShadow};transform:scale({hoverScale});transition:background 0.16s ease, box-shadow 0.2s cubic-bezier(0.22,1,0.36,1), transform 0.2s cubic-bezier(0.34,1.56,0.64,1);"
	aria-label={label}
	aria-current={isActive ? 'page' : undefined}
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	{#if isActive}
		<div
			style="position:absolute;inset:4px;border-radius:12px;background:var(--neu-accent-primary);opacity:0.14;"
		/>
	{/if}
	<Icon
		style="width:22px;height:22px;color:{iconColor};transition:color 0.18s ease;position:relative;z-index:1;"
		strokeWidth={isActive ? 2.1 : showHover ? 1.75 : 1.5}
	/>
	<span style="font-size:10px;line-height:1;color:{iconColor};font-weight:{isActive ? 600 : 400};position:relative;z-index:1;transition:color 0.18s ease;">
		{label}
	</span>
	{#if isActive}
		<div style="position:absolute;left:0;top:10px;bottom:10px;width:3px;border-radius:2px;background:var(--neu-accent-primary);" />
	{/if}
</button>
