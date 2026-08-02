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
		Icon
	}: {
		label: string;
		isActive: boolean;
		onClick: () => void;
		Icon: Component;
	} = $props();
</script>

<button
	type="button"
	class="neu-hover-btn neu-nav-tab relative flex flex-col items-center justify-center outline-none select-none rounded-[var(--shape-panel)]"
	style="flex:1;min-width:0;padding-top:10px;padding-bottom:8px;gap:4px;"
	aria-label={label}
	aria-current={isActive ? 'page' : undefined}
	onclick={onClick}
>
	{#if isActive}
		<div
			class="transition-[opacity] duration-200"
			style="position:absolute;inset:4px 6px;border-radius:var(--shape-panel);background:var(--navigation-active-indicator);opacity:0.13;"
		></div>
	{/if}

	<div style="display:flex;align-items:center;justify-content:center;position:relative;z-index:1;">
		<Icon
			style="width:22px;height:22px;color:{isActive ? 'var(--navigation-content-active)' : 'var(--navigation-content)'};transition:color 0.22s ease;"
			strokeWidth={isActive ? 2.1 : 1.5}
		/>
	</div>
	<span
		style="font-size:11px;line-height:1;color:{isActive ? 'var(--navigation-content-active)' : 'var(--navigation-content)'};font-weight:{isActive ? 600 : 400};transition:color 0.22s ease;letter-spacing:0.01em;position:relative;z-index:1;"
	>
		{label}
	</span>
</button>
