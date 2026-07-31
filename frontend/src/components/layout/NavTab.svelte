<script lang="ts">
	import type { Component } from 'svelte';

	// 移动/平板底部导航项：滑动胶囊（layoutId 动画简化为条件渲染 + CSS transition）
	// 迁移自 React MainLayout.tsx 内 NavTab。

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
	class="neu-hover-btn relative flex flex-col items-center justify-center outline-none select-none"
	style="width:68px;padding-top:10px;padding-bottom:8px;gap:4px;"
	aria-label={label}
	aria-current={isActive ? 'page' : undefined}
	onclick={onClick}
>
	{#if isActive}
		<div
			class="transition-[opacity] duration-200"
			style="position:absolute;top:5px;width:60px;height:48px;border-radius:24px;background:var(--neu-accent-primary);opacity:0.13;"
		/>
	{/if}

	<div style="display:flex;align-items:center;justify-content:center;position:relative;z-index:1;">
		<Icon
			style="width:22px;height:22px;color:{isActive ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};transition:color 0.22s ease;"
			strokeWidth={isActive ? 2.1 : 1.5}
		/>
	</div>
	<span
		style="font-size:11px;line-height:1;color:{isActive ? 'var(--neu-accent-primary)' : 'var(--neu-text-muted)'};font-weight:{isActive ? 600 : 400};transition:color 0.22s ease;letter-spacing:0.01em;position:relative;z-index:1;"
	>
		{label}
	</span>
</button>
