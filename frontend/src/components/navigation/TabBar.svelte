<script lang="ts">
	import type { Component } from 'svelte';

	// TabBar — 下划线分类 Tab 栏（统一 nearby/moments、关系聚合页、用户详情等手写重复实现）。
	// 容器 border-bottom + 激活项底部 2px 指示条 + aria-pressed，随 PageHeader below 吸顶。
	// 用 data-tab-bar 标记，全局 button:hover 规则据此排除（导航 Tab 不应触发实色按压填充），
	// 改用极轻背景 + 圆角的 subtle hover（见 utilities.css 的 .tab-item 规则）。

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

<div data-tab-bar class="no-scrollbar flex overflow-x-auto border-b border-border-default {className}">
	{#each items as tab}
		{@const isActive = value === tab.id}
		<button
			type="button"
			onclick={() => onChange(tab.id)}
			aria-pressed={isActive}
			class="tab-item relative flex shrink-0 items-center gap-1.5 px-4 py-2.5 text-sm outline-none transition-colors duration-200 {isActive ? 'font-medium text-accent-default' : 'font-normal text-text-secondary hover:text-text-primary'}"
		>
			{#if tab.icon}
				<tab.icon class="size-3.25"  strokeWidth={isActive ? 2 : 1.5} />
			{/if}
			{tab.label}
			{#if tab.count !== undefined}
				<span class="text-xxs opacity-75">{tab.count}</span>
			{/if}
			{#if isActive}
				<span class="absolute inset-x-3 bottom-[-1px] h-0.5 rounded-t-full bg-accent-default"></span>
			{/if}
		</button>
	{/each}
</div>
