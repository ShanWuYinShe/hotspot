<script lang="ts">
	import type { Component } from 'svelte';

	// TabBar — 胶囊分类 Tab 栏（对齐鸿蒙 TabsPillGroup 分段式视觉，全站统一）。
	// 只负责 Tab 行 + 激活项 accent 实底胶囊 + aria-pressed，随 PageHeader below 吸顶；
	// 底部边界线由上下文提供（PageHeader below 场景由 header 底线承担，
	// 正文场景由调用方外层 border-b 承担），避免与 header 形成双线。
	// 用 data-tab-bar 标记，全局 button:hover 规则据此排除（导航 Tab 不应触发实色按压填充），
	// 改用极轻背景 + 胶囊圆角的 subtle hover（见 utilities.css 的 .tab-item 规则）。

	export type TabItem = { id: string; label: string; count?: number; icon?: Component };

	let {
		items,
		value,
		onChange,
		class: className = ''
	}: {
		items: TabItem[];
		value: string;
		onChange: (id: string) => void;
		class?: string;
	} = $props();
</script>

<div data-tab-bar class="no-scrollbar flex overflow-x-auto {className}">
	{#each items as tab}
		{@const isActive = value === tab.id}
		<button
			type="button"
			onclick={() => onChange(tab.id)}
			aria-pressed={isActive}
			class="tab-item flex shrink-0 items-center gap-1.5 rounded-full px-4 py-2 text-sm focus-ring transition-colors duration-standard {isActive
				? 'font-medium bg-accent-default text-text-on-accent shadow-glow-sm'
				: 'font-normal text-text-secondary hover:bg-accent-soft hover:text-text-primary'}"
		>
			{#if tab.icon}
				<tab.icon class="size-3.5"  strokeWidth={isActive ? 2 : 1.5} />
			{/if}
			{tab.label}
			{#if tab.count !== undefined}
				<span class="text-xxs opacity-emphasis">{tab.count}</span>
			{/if}
		</button>
	{/each}
</div>
