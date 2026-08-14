<script lang="ts">
	import type { Component } from 'svelte';

	// 桌面侧栏导航项（对齐 Figma Make SidebarNavItem 契约）：
	// - 选中态：实心 accent-primary 背景 + content-on-accent 文字 + elevation-navigation-active 光晕
	// - hover（未选中）：surface-navigation-hover 背景 + 主题文字色 + 左侧细指示条
	// - 行高固定 40px；图标列固定 40px 居中，折叠时文字随侧栏宽度同步收起
	//   （max-width/opacity/margin 过渡与 aside 宽度过渡同时长，避免图标位移跳变）

	let {
		item,
		isActive,
		onClick,
		collapsed = false
	}: {
		item: { icon: Component; label: string };
		isActive: boolean;
		onClick: () => void;
		collapsed?: boolean;
	} = $props();

	const Icon = $derived(item.icon);
	let hovered = $state(false);
	const showHover = $derived(hovered && !isActive);

	const hoverBg = $derived(showHover ? 'var(--cst-surface-navigation-hover)' : 'transparent');
</script>

<button
	type="button"
	class="tap-97 relative flex h-10 w-full items-center rounded-sm focus-ring text-left {collapsed ? 'justify-center' : ''}"
	style="background:{isActive ? 'var(--cst-accent-default)' : hoverBg};color:{isActive ? 'var(--cst-content-on-accent)' : showHover ? 'var(--cst-text-primary)' : 'var(--cst-text-secondary)'};box-shadow:{isActive ? 'var(--cst-elevation-navigation-active)' : 'none'};transition:background var(--cst-motion-rapid) var(--cst-ease-standard), color var(--cst-motion-rapid) var(--cst-ease-standard), box-shadow var(--cst-motion-standard) var(--cst-ease-standard), transform var(--cst-motion-instant) var(--cst-ease-standard);"
	aria-current={isActive ? 'page' : undefined}
	data-active={isActive ? 'true' : undefined}
	aria-label={item.label}
	title={collapsed ? item.label : undefined}
	onclick={onClick}
	onmouseenter={() => (hovered = true)}
	onmouseleave={() => (hovered = false)}
>
	<!-- 左侧指示条 — 仅在 hover 时显示，主题色，细（选中态由实心填充表达） -->
	<span
		aria-hidden="true"
		class="rounded-full {showHover ? 'opacity-hover' : 'opacity-none'}"
		style="position:absolute;left:4px;top:50%;width:3px;height:{showHover ? 16 : 0}px;margin-top:{showHover ? -8 : 0}px;background:var(--cst-accent-default);transition:height var(--cst-motion-snappy) var(--cst-ease-standard), opacity var(--cst-motion-snappy) var(--cst-ease-standard), margin-top var(--cst-motion-snappy) var(--cst-ease-standard);pointer-events:none;"
	></span>
	<span class="flex size-10 shrink-0 items-center justify-center">
		<Icon
			class="size-4.5"
			style="color:{isActive ? 'var(--cst-content-on-accent)' : showHover ? 'var(--cst-accent-default)' : 'var(--cst-text-secondary)'};transition:color var(--cst-motion-rapid) var(--cst-ease-standard);"
			strokeWidth={isActive ? 2 : 1.5}
		/>
	</span>
	<span
		class="overflow-hidden whitespace-nowrap text-md transition-[max-width,opacity,margin] duration-standard ease-standard {collapsed ? 'ml-0 max-w-0 opacity-none' : 'ml-1.5 max-w-full opacity-full'} {isActive ? 'font-medium' : 'font-normal'}"
	>
		{item.label}
	</span>
</button>
