<script lang="ts">
	import type { Component } from 'svelte';

	// TabBar — 胶囊分类/分段 Tab 栏（对齐鸿蒙 TabsPillGroup 分段式视觉，全站统一）。
	// 支持两种形态（对齐鸿蒙 TabsPillGroup）：
	// - variant="pills"（默认，流式胶囊导航）：透明容器，横向滚动，选项带 gap-1.5 安全呼吸间距，杜绝贴边粘连。
	// - variant="segmented"（系统级分段控制器）：surface-subtle 浅色底轨容器 + ring-1 ring-border-default + p-1 内距，
	//   选项等宽网格（grid grid-flow-col auto-cols-fr gap-1），激活项浮起胶囊，消除粘连推挤感。
	// 底部边界线由上下文提供（PageHeader below 场景由 header 底线承担，
	// 正文场景由调用方外层 border-b 承担），避免与 header 形成双线。
	// 用 data-tab-bar 标记，全局 button:hover 规则据此排除（导航 Tab 不应触发实色按压填充），
	// 改用极轻背景 + 胶囊圆角的 subtle hover（见 utilities.css 的 .tab-item 规则）。

	export type TabItem = { id: string; label: string; count?: number; icon?: Component };

	let {
		items,
		value,
		onChange,
		variant = 'pills',
		class: className = ''
	}: {
		items: TabItem[];
		value: string;
		onChange: (id: string) => void;
		variant?: 'pills' | 'segmented';
		class?: string;
	} = $props();
</script>

{#if variant === 'segmented'}
	<div
		data-tab-bar
		role="tablist"
		class="grid grid-flow-col auto-cols-fr gap-1 rounded-full bg-surface-subtle p-1 ring-1 ring-border-default/70 {className}"
	>
		{#each items as tab}
			{@const isActive = value === tab.id}
			<button
				type="button"
				role="tab"
				onclick={() => onChange(tab.id)}
				aria-selected={isActive}
				data-active={isActive ? 'true' : undefined}
				class="tab-item flex items-center justify-center gap-1.5 rounded-full px-3 py-1.5 text-xs focus-ring transition-all duration-standard {isActive
					? 'font-medium bg-accent-default text-text-on-accent shadow-raised-sm'
					: 'font-normal text-text-secondary hover:text-text-primary'}"
			>
				{#if tab.icon}
					<tab.icon class="size-3.5 shrink-0" strokeWidth={isActive ? 2 : 1.5} />
				{/if}
				<span class="truncate">{tab.label}</span>
				{#if tab.count !== undefined}
					<span class="text-xxs opacity-emphasis">{tab.count}</span>
				{/if}
			</button>
		{/each}
	</div>
{:else}
	<div data-tab-bar class="no-scrollbar flex items-center gap-1.5 overflow-x-auto {className}">
		{#each items as tab}
			{@const isActive = value === tab.id}
			<button
				type="button"
				onclick={() => onChange(tab.id)}
				aria-pressed={isActive}
				class="tab-item flex shrink-0 items-center gap-1.5 rounded-full px-3.5 py-1.5 text-sm focus-ring transition-colors duration-standard {isActive
					? 'font-medium bg-accent-default text-text-on-accent shadow-raised-sm'
					: 'font-normal text-text-secondary hover:text-text-primary'}"
			>
				{#if tab.icon}
					<tab.icon class="size-3.5" strokeWidth={isActive ? 2 : 1.5} />
				{/if}
				{tab.label}
				{#if tab.count !== undefined}
					<span class="text-xxs opacity-emphasis">{tab.count}</span>
				{/if}
			</button>
		{/each}
	</div>
{/if}
